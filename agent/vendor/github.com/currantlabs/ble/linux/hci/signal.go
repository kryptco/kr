package hci

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"time"

	"github.com/currantlabs/ble/linux/hci/cmd"
)

// Signal ...
type Signal interface {
	Code() int
	Marshal() []byte
	Unmarshal([]byte) error
}

type sigCmd []byte

func (s sigCmd) code() int    { return int(s[0]) }
func (s sigCmd) id() uint8    { return s[1] }
func (s sigCmd) len() int     { return int(binary.LittleEndian.Uint16(s[2:4])) }
func (s sigCmd) data() []byte { return s[4 : 4+s.len()] }

// Signal ...
func (c *Conn) Signal(req Signal, rsp Signal) error {
	data := req.Marshal()
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, uint16(4+len(data)))
	binary.Write(buf, binary.LittleEndian, uint16(cidLESignal))

	binary.Write(buf, binary.LittleEndian, uint8(req.Code()))
	binary.Write(buf, binary.LittleEndian, uint8(c.sigID))
	binary.Write(buf, binary.LittleEndian, uint16(len(data)))
	binary.Write(buf, binary.LittleEndian, data)

	c.sigSent = make(chan []byte)
	defer close(c.sigSent)
	if _, err := c.writePDU(buf.Bytes()); err != nil {
		return err
	}
	var s sigCmd
	select {
	case s = <-c.sigSent:
	case <-time.After(time.Second):
		// TODO: Find the proper timed out defined in spec, if any.
		return errors.New("signaling request timed out")
	}

	if s.code() != req.Code() {
		return errors.New("mismatched signaling response")
	}
	if s.id() != c.sigID {
		return errors.New("mismatched signaling id")
	}
	c.sigID++
	if rsp == nil {
		return nil
	}
	return rsp.Unmarshal(s.data())
}

func (c *Conn) sendResponse(code uint8, id uint8, r Signal) (int, error) {
	data := r.Marshal()
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, uint16(4+len(data)))
	binary.Write(buf, binary.LittleEndian, uint16(cidLESignal))
	binary.Write(buf, binary.LittleEndian, uint8(code))
	binary.Write(buf, binary.LittleEndian, uint8(id))
	binary.Write(buf, binary.LittleEndian, uint16(len(data)))
	if err := binary.Write(buf, binary.LittleEndian, data); err != nil {
		return 0, err
	}
	logger.Debug("sig", "send", fmt.Sprintf("[%X]", buf.Bytes()))
	return c.writePDU(buf.Bytes())
}

func (c *Conn) handleSignal(p pdu) error {
	logger.Debug("sig", "recv", fmt.Sprintf("[%X]", p))
	// When multiple commands are included in an L2CAP packet and the packet
	// exceeds the signaling MTU (MTUsig) of the receiver, a single Command Reject
	// packet shall be sent in response. The identifier shall match the first Request
	// command in the L2CAP packet. If only Responses are recognized, the packet
	// shall be silently discarded. [Vol3, Part A, 4.1]
	if p.dlen() > c.sigRxMTU {
		c.sendResponse(
			SignalCommandReject,
			sigCmd(p.payload()).id(),
			&CommandReject{
				Reason: 0x0001,                                            // Signaling MTU exceeded.
				Data:   []byte{uint8(c.sigRxMTU), uint8(c.sigRxMTU >> 8)}, // Actual MTUsig.
			})
		return nil
	}

	s := sigCmd(p.payload())
	for len(s) > 0 {
		// Check if it's a supported request.
		switch s.code() {
		case SignalDisconnectRequest:
			c.handleDisconnectRequest(s)
		case SignalConnectionParameterUpdateRequest:
			c.handleConnectionParameterUpdateRequest(s)
		case SignalLECreditBasedConnectionRequest:
			c.LECreditBasedConnectionRequest(s)
		case SignalLEFlowControlCredit:
			c.LEFlowControlCredit(s)
		default:
			// Check if it's a response to a sent command.
			select {
			case c.sigSent <- s:
				continue
			default:
			}

			c.sendResponse(
				SignalCommandReject,
				s.id(),
				&CommandReject{
					Reason: 0x0000, // Command not understood.
				})
		}
		s = s[4+s.len():] // advance to next the packet.

	}
	return nil
}

// DisconnectRequest implements Disconnect Request (0x06) [Vol 3, Part A, 4.6].
func (c *Conn) handleDisconnectRequest(s sigCmd) {
	var req DisconnectRequest
	if err := req.Unmarshal(s.data()); err != nil {
		return
	}

	// Send Command Reject when the DCID is unrecognized.
	if req.DestinationCID != cidLEAtt {
		endpoints := make([]byte, 4)
		binary.LittleEndian.PutUint16(endpoints, req.SourceCID)
		binary.LittleEndian.PutUint16(endpoints, req.DestinationCID)
		c.sendResponse(
			SignalCommandReject,
			s.id(),
			&CommandReject{
				Reason: 0x0002, // Invalid CID in request
				Data:   endpoints,
			})
		return
	}

	// Silently discard the request if SCID failed to find the same match.
	if req.SourceCID != cidLEAtt {
		return
	}

	c.sendResponse(
		SignalDisconnectResponse,
		s.id(),
		&DisconnectResponse{
			DestinationCID: req.DestinationCID,
			SourceCID:      req.SourceCID,
		})
}

// ConnectionParameterUpdateRequest implements Connection Parameter Update Request (0x12) [Vol 3, Part A, 4.20].
func (c *Conn) handleConnectionParameterUpdateRequest(s sigCmd) {
	// This command shall only be sent from the LE slave device to the LE master
	// device and only if one or more of the LE slave Controller, the LE master
	// Controller, the LE slave Host and the LE master Host do not support the
	// Connection Parameters Request Link Layer Control Procedure ([Vol. 6] Part B,
	// Section 5.1.7). If an LE slave Host receives a Connection Parameter Update
	// Request packet it shall respond with a Command Reject packet with reason
	// 0x0000 (Command not understood).
	if c.param.Role() != roleMaster {
		c.sendResponse(
			SignalCommandReject,
			s.id(),
			&CommandReject{
				Reason: 0x0000, // Command not understood.
			})

		return
	}
	var req ConnectionParameterUpdateRequest
	if err := req.Unmarshal(s.data()); err != nil {
		return
	}

	// LE Connection Update (0x08|0x0013) [Vol 2, Part E, 7.8.18]
	c.hci.Send(&cmd.LEConnectionUpdate{
		ConnectionHandle:   c.param.ConnectionHandle(),
		ConnIntervalMin:    req.IntervalMin,
		ConnIntervalMax:    req.IntervalMax,
		ConnLatency:        req.SlaveLatency,
		SupervisionTimeout: req.TimeoutMultiplier,
		MinimumCELength:    0, // Informational, and spec doesn't specify the use.
		MaximumCELength:    0, // Informational, and spec doesn't specify the use.
	}, nil)

	// Currently, we (as a slave host) accept all the parameters and forward
	// it to the controller. The controller might update all, partial or even
	// none (ignore) of the parameters. The slave(remote) host will be indicated
	// by its controller if the update actually happens.
	// TODO: allow users to implement what parameters to accept.
	c.sendResponse(
		SignalConnectionParameterUpdateResponse,
		s.id(),
		&ConnectionParameterUpdateResponse{
			Result: 0, // Accept.
		})
}

// LECreditBasedConnectionRequest ...
func (c *Conn) LECreditBasedConnectionRequest(s sigCmd) {
	// TODO:
}

// LEFlowControlCredit ...
func (c *Conn) LEFlowControlCredit(s sigCmd) {
	// TODO:
}
