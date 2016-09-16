package hci

import (
	"bytes"
	"encoding/binary"
	"fmt"
)

const (
	pairingRequest           = 0x01 // Pairing Request LE-U, ACL-U
	pairingResponse          = 0x02 // Pairing Response LE-U, ACL-U
	pairingConfirm           = 0x03 // Pairing Confirm LE-U
	pairingRandom            = 0x04 // Pairing Random LE-U
	pairingFailed            = 0x05 // Pairing Failed LE-U, ACL-U
	encryptionInformation    = 0x06 // Encryption Information LE-U
	masterIdentification     = 0x07 // Master Identification LE-U
	identiInformation        = 0x08 // Identity Information LE-U, ACL-U
	identityAddreInformation = 0x09 // Identity Address Information LE-U, ACL-U
	signingInformation       = 0x0A // Signing Information LE-U, ACL-U
	securityRequest          = 0x0B // Security Request LE-U
	pairingPublicKey         = 0x0C // Pairing Public Key LE-U
	pairingDHKeyCheck        = 0x0D // Pairing DHKey Check LE-U
	pairingKeypress          = 0x0E // Pairing Keypress Notification LE-U
)

func (c *Conn) sendSMP(p pdu) error {
	buf := bytes.NewBuffer(make([]byte, 0))
	binary.Write(buf, binary.LittleEndian, uint16(4+len(p)))
	binary.Write(buf, binary.LittleEndian, cidSMP)
	binary.Write(buf, binary.LittleEndian, p)
	_, err := c.writePDU(buf.Bytes())
	logger.Debug("smp", "send", fmt.Sprintf("[%X]", buf.Bytes()))
	return err
}

func (c *Conn) handleSMP(p pdu) error {
	logger.Debug("smp", "recv", fmt.Sprintf("[%X]", p))
	code := p[0]
	switch code {
	case pairingRequest:
	case pairingResponse:
	case pairingConfirm:
	case pairingRandom:
	case pairingFailed:
	case encryptionInformation:
	case masterIdentification:
	case identiInformation:
	case identityAddreInformation:
	case signingInformation:
	case securityRequest:
	case pairingPublicKey:
	case pairingDHKeyCheck:
	case pairingKeypress:
	default:
		// If a packet is received with a reserved Code it shall be ignored. [Vol 3, Part H, 3.3]
		return nil
	}
	// FIXME: work aound to the lack of SMP implementation - always return non-supported.
	// C.5.1 Pairing Not Supported by Slave
	return c.sendSMP([]byte{pairingFailed, 0x05})
}
