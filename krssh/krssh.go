package main

import (
	"bytes"
	"encoding/binary"
	"encoding/json"
	"io"
	"log"
	"math/big"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/kryptco/kr"

	"golang.org/x/crypto/ssh"
)

//	from https://github.com/golang/crypto/blob/master/ssh/messages.go#L98-L102
type kexDHReplyMsg struct {
	HostKey   []byte `sshtype:"31"`
	Y         *big.Int
	Signature []byte
}

type kexECDHReplyMsg struct {
	HostKey         []byte `sshtype:"31"`
	EphemeralPubKey []byte
	Signature       []byte
}

func sendHostAuth(hostAuth kr.HostAuth) {
	conn, err := kr.HostAuthDial()
	if err != nil {
		log.Println(kr.Red("Kryptonite ▶ Could not connect to Kryptonite daemon. Make sure it is running by typing \"kr restart\"."))
		return
	}
	defer conn.Close()
	json.NewEncoder(conn).Encode(hostAuth)
}

func tryParse(buf []byte) (err error) {
	kexDHReplyTemplate := kexDHReplyMsg{}
	kexECDHReplyTemplate := kexECDHReplyMsg{}
	err = ssh.Unmarshal(buf, &kexDHReplyTemplate)
	if err == nil {
		hostAuth := kr.HostAuth{
			HostKey:   kexDHReplyTemplate.HostKey,
			Signature: kexDHReplyTemplate.Signature,
		}
		sendHostAuth(hostAuth)
	}
	err = ssh.Unmarshal(buf, &kexECDHReplyTemplate)
	if err == nil {
		hostAuth := kr.HostAuth{
			HostKey:   kexDHReplyTemplate.HostKey,
			Signature: kexDHReplyTemplate.Signature,
		}
		sendHostAuth(hostAuth)
	}
	return
}

func parseSSHPacket(b []byte) (packet []byte) {
	if len(b) <= 4 {
		return
	}
	packetLen := binary.BigEndian.Uint32(b[:4])
	paddingLen := b[4]
	payloadLen := packetLen - uint32(paddingLen) - 1
	if len(b) <= int(5+payloadLen) {
		return
	}
	packet = make([]byte, payloadLen)
	copy(packet, b[5:5+payloadLen])
	return
}

func startLogger() {
	r, err := kr.OpenNotificationReader()
	if err != nil {
		return
	}
	go func() {
		for {
			notification, err := r.Read()
			switch err {
			case nil:
				os.Stderr.Write(notification)
			case io.EOF:
				<-time.After(250 * time.Millisecond)
			default:
				return
			}
		}
	}()
}

func main() {
	log.SetFlags(0)
	if len(os.Args) < 2 {
		log.Fatal("not enough arguments")
	}
	var host, port string
	host = os.Args[1]
	if len(os.Args) >= 3 {
		port = os.Args[2]
	} else {
		port = "22"
	}

	startLogger()

	remoteConn, err := net.Dial("tcp", host+":"+port)
	if err != nil {
		log.Fatal(kr.Red("could not connect to remote: " + err.Error()))
	}

	remoteDoneChan := make(chan bool)

	go func() {
		func() {
			buf := make([]byte, 1<<18)
			packetNum := 0
			for {
				n, err := remoteConn.Read(buf)
				if err != nil && err != io.EOF {
					log.Println("remote write err:", err.Error())
					return
				}
				if n > 0 {
					packetNum++
					if packetNum > 1 {
						sshPacket := parseSSHPacket(buf)
						tryParse(sshPacket)
					}
					byteBuf := bytes.NewBuffer(buf[:n])
					wroteN, err := byteBuf.WriteTo(os.Stdout)
					if wroteN != int64(n) {
						log.Println("not all bytes written to stdout")
					}
					if err != nil {
						log.Println("err writing remote to stdout", err.Error())
						return
					}
				}
			}
		}()
		remoteDoneChan <- true
	}()

	localDoneChan := make(chan bool)

	go func() {
		func() {
			buf := make([]byte, 1<<18)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					if err != io.EOF {
						log.Println("stdin read err:", err.Error())
					}
					return
				}
				if n > 0 {
					byteBuf := bytes.NewBuffer(buf[:n])
					wroteN, err := byteBuf.WriteTo(remoteConn)
					if wroteN != int64(n) {
						log.Println("not all bytes written to remote")
					}
					if err != nil {
						log.Println("err writing stdin to remote", err.Error())
						return
					}
				}
			}
		}()
		localDoneChan <- true
	}()

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	var localDone, remoteDone bool
	for {
		select {
		case <-stopSignal:
			return
		case <-localDoneChan:
			localDone = true
			if remoteDone {
				return
			}
		case <-remoteDoneChan:
			remoteDone = true
			if localDone {
				return
			}
		}
	}
}
