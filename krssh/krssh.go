package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net"
	"os"
	"os/exec"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krd"

	"github.com/keybase/saltpack/encoding/basex"
	"github.com/op/go-logging"
	"golang.org/x/crypto/ssh"
)

var silenceWarnings = os.Getenv("KR_SILENCE_WARNINGS") != ""

func fatal(msg string) {
	os.Stderr.WriteString(msg + "\r\n")
	os.Exit(1)
}

func useSyslog() bool {
	env := os.Getenv("KR_LOG_SYSLOG")
	if env != "" {
		return env == "true"
	}
	return true
}

var logger *logging.Logger = kr.SetupLogging("krssh", logging.WARNING, useSyslog())

//	from https://github.com/golang/crypto/blob/master/ssh/messages.go#L98-L102
type kexECDHReplyMsg struct {
	HostKey         []byte `sshtype:"31|33"` //	handle SSH2_MSG_KEX_DH_GEX_REPLY as well
	EphemeralPubKey []byte
	Signature       []byte
}

func sendHostAuth(hostAuth kr.HostAuth) {
	conn, err := kr.HostAuthDial()
	if err != nil {
		os.Stderr.WriteString(kr.Red("Kryptonite ▶ Could not connect to Kryptonite daemon. Make sure it is running by typing \"kr restart\".\r\n"))
		return
	}
	defer conn.Close()
	json.NewEncoder(conn).Encode(hostAuth)
}

func tryParse(hostname string, onHostPrefix chan string, buf []byte) (err error) {
	kexECDHReplyTemplate := kexECDHReplyMsg{}
	err = ssh.Unmarshal(buf, &kexECDHReplyTemplate)

	hostnameWithNonDefaultPort := hostname
	if port != "22" && port != "" {
		hostnameWithNonDefaultPort = net.JoinHostPort(hostname, port)
	}

	if err == nil {
		hostAuth := kr.HostAuth{
			HostKey:   kexECDHReplyTemplate.HostKey,
			Signature: kexECDHReplyTemplate.Signature,
			HostNames: []string{hostnameWithNonDefaultPort},
		}
		sigHash := sha256.Sum256(hostAuth.Signature)
		select {
		case onHostPrefix <- "[" + basex.Base62StdEncoding.EncodeToString(sigHash[:]) + "]":
		default:
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
	if payloadLen > (1<<18) || payloadLen < 1 || len(b) <= int(5+payloadLen) {
		return
	}
	packet = make([]byte, payloadLen)
	copy(packet, b[5:5+payloadLen])
	return
}

func startLogger(prefix string, checkForUpdate bool) (r kr.NotificationReader, err error) {
	r, err = kr.OpenNotificationReader(prefix)
	if err != nil {
		return
	}
	go func() {
		if prefix != "" {
			defer os.Remove(r.Name())
		}

		go func() {
			if checkForUpdate && !silenceWarnings && krd.CheckIfUpdateAvailable(logger) {
				os.Stderr.WriteString(kr.Yellow("Kryptonite ▶ A new version of Kryptonite is available. Run \"kr upgrade\" to install it. You can view the changelog at https://krypt.co/app/krd_changelog/\r\n"))
			}
		}()

		printedNotifications := map[string]bool{}
		for {
			notification, err := r.Read()
			switch err {
			case nil:
				notificationStr := string(notification)
				if _, ok := printedNotifications[notificationStr]; ok {
					continue
				}
				if silenceWarnings {
					if strings.Contains(notificationStr, kr.ErrNotPaired.Error()) {
						continue
					}
				}
				if strings.HasPrefix(notificationStr, "[") {
					if prefix != "" && strings.HasPrefix(notificationStr, prefix) {
						trimmed := strings.TrimPrefix(notificationStr, prefix)
						if strings.HasPrefix(trimmed, "STOP") {
							return
						}
						if strings.HasPrefix(trimmed, "HOST_KEY_MISMATCH") {
							os.Stderr.WriteString(kr.Red(
								fmt.Sprintf("Kryptonite ▶ Public key for %s does not match pinned key. If the host key has actually changed, remove the pinned key in Kryptonite.\r\n", host),
							))
							os.Exit(1)
						}
						if strings.HasPrefix(trimmed, "REJECTED") {
							os.Stderr.WriteString(kr.Red("Kryptonite ▶ " + kr.ErrRejected.Error() + "\r\n"))
							os.Exit(1)
						}
						os.Stderr.WriteString(trimmed)
					}
				} else {
					if strings.Contains(notificationStr, "]") {
						//	skip malformed notification
						continue
					}
					os.Stderr.WriteString(notificationStr)
				}
				printedNotifications[notificationStr] = true
			case io.EOF:
				<-time.After(50 * time.Millisecond)
			default:
				return
			}
		}
	}()
	return
}

type StdIOReadWriter struct {
	io.Reader
	io.Writer
}

func startRemoteOutputParsing(remoteConn io.Reader, doneChan chan bool, notifyPrefixChan chan string) {
	go func() {
		func() {
			buf := make([]byte, 1<<18)
			packetNum := 0
			for {
				n, err := remoteConn.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					packetNum++
					if packetNum > 1 {
						sshPacket := parseSSHPacket(buf)
						tryParse(host, notifyPrefixChan, sshPacket)
					}
					byteBuf := bytes.NewBuffer(buf[:n])
					_, err := byteBuf.WriteTo(os.Stdout)
					if err != nil {
						return
					}
				}
			}
		}()
		doneChan <- true
	}()
}

func startRemoteInputFowarding(remoteInput io.Writer, doneChan chan bool) {
	go func() {
		func() {
			buf := make([]byte, 1<<18)
			for {
				n, err := os.Stdin.Read(buf)
				if err != nil {
					return
				}
				if n > 0 {
					byteBuf := bytes.NewBuffer(buf[:n])
					_, err := byteBuf.WriteTo(remoteInput)
					if err != nil {
						return
					}
				}
			}
		}()
		doneChan <- true
	}()
}

var host, port string

func main() {
	log.SetFlags(0)

	proxyCommand := flag.String("p", "", "ssh proxy command")
	proxyHost := flag.String("h", "", "ssh destination host")
	flag.Parse()
	if proxyCommand != nil && *proxyCommand == "" {
		proxyCommand = nil
	}

	var remoteConn io.ReadWriter

	if proxyCommand != nil {
		if proxyHost == nil || *proxyHost == "" {
			os.Stderr.WriteString(kr.Red("No proxy host specified. Please pass proxy host with the '-h' flag to krssh, i.e. 'ProxyCommand krssh -p \"ssh -W %h:%p destination\" -h %h'\r\n"))
			os.Exit(1)
		}
		host = *proxyHost

		toks := strings.Split(*proxyCommand, " ")
		sshCommand := exec.Command(toks[0], toks[1:]...)

		proxyStdinR, proxyStdinW := io.Pipe()
		proxyStdoutR, proxyStdoutW := io.Pipe()

		sshCommand.Stdin = proxyStdinR
		sshCommand.Stdout = proxyStdoutW
		sshCommand.Stderr = os.Stderr
		sshCommand.Start()

		remoteConn = StdIOReadWriter{
			proxyStdoutR,
			proxyStdinW,
		}
	} else {
		host = os.Args[1]
		if len(os.Args) >= 3 {
			port = os.Args[2]
		} else {
			port = "22"
		}

		var err error
		remoteConn, err = net.Dial("tcp", net.JoinHostPort(host, port))
		if err != nil {
			fatal(kr.Red("could not connect to remote: " + err.Error()))
		}
	}

	notifyPrefix := make(chan string, 1)
	startLogger("", true)
	go func() {
		prefix := <-notifyPrefix
		startLogger(prefix, false)
	}()

	remoteDoneChan := make(chan bool)
	startRemoteOutputParsing(remoteConn, remoteDoneChan, notifyPrefix)

	localDoneChan := make(chan bool)
	startRemoteInputFowarding(remoteConn, localDoneChan)

	stopSignal := make(chan os.Signal, 1)
	signal.Notify(stopSignal, os.Interrupt, os.Kill, syscall.SIGHUP, syscall.SIGQUIT, syscall.SIGTERM)
	for {
		select {
		case <-stopSignal:
			return
		case <-localDoneChan:
			os.Stdout.Sync()
			<-time.After(500 * time.Millisecond)
			return
		case <-remoteDoneChan:
			os.Stdout.Sync()
			<-time.After(500 * time.Millisecond)
			return
		}
	}
}
