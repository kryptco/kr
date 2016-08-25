package main

import (
	cautil "bitbucket.org/kryptco/enclave/ca/util"

	"crypto"
	//"errors"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type ProxiedSSHKey struct {
	publicKey crypto.PublicKey
	sk        crypto.Signer
}

func (key ProxiedSSHKey) Sign(rand io.Reader, digest []byte, opts crypto.SignerOpts) (signature []byte, err error) {
	fmt.Println("signing", digest)
	//err = errors.New("not yet implemented")
	return key.sk.Sign(rand, digest, opts)
}

func (key ProxiedSSHKey) Public() crypto.PublicKey {
	return key.publicKey
}

func NewProxiedSSHKey() ProxiedSSHKey {
	_, sk, _ := cautil.GenOrgCA("test")
	return ProxiedSSHKey{
		publicKey: sk.(crypto.Signer).Public(),
		sk:        sk.(crypto.Signer),
	}
}

func RequestPseudoTerminal(client *ssh.Client) (session *ssh.Session, err error) {
	session, err = client.NewSession()
	if err != nil {
		log.Println("error creating new session:", err)
		return
	}

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr
	// Set up terminal modes
	modes := ssh.TerminalModes{
		ssh.TTY_OP_ISPEED: 14400, // input speed = 14.4kbaud
		ssh.TTY_OP_OSPEED: 14400, // output speed = 14.4kbaud
		ssh.VSTATUS:       1,
	}

	//	get current tty size
	cmd := exec.Command("stty", "size")
	cmd.Stdin = os.Stdin
	out, err := cmd.Output()
	if err != nil {
		log.Fatal(err)
	}
	fields := strings.Fields(string(out))
	height, _ := strconv.ParseInt(fields[0], 10, 16)
	width, _ := strconv.ParseInt(fields[1], 10, 16)
	// Request pseudo terminal
	if err := session.RequestPty("xterm-256color", int(height), int(width), modes); err != nil {
		log.Fatalf("request for pseudo terminal failed: %s", err)
	}
	return
}

func StartSessionAndWait(session *ssh.Session) {
	oldState, err := terminal.MakeRaw(0)
	if err != nil {
		panic(err)
	}
	defer terminal.Restore(0, oldState)
	// Start remote shell
	if err := session.Shell(); err != nil {
		log.Fatalf("failed to start shell: %s", err)
	}
	session.Wait()
}

func main() {
	log.Println(os.Args)
	key, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_rsa")
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	// Create the Signer for this private key.
	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	config := &ssh.ClientConfig{
		User: strings.Split(os.Args[1], "@")[0],
		Auth: []ssh.AuthMethod{
			// Use the PublicKeys method for remote authentication.
			ssh.PublicKeys(signer),
		},
	}

	// Connect to the remote server and perform the SSH handshake.
	client, err := ssh.Dial("tcp", strings.Split(os.Args[1], "@")[1]+":22", config)
	if err != nil {
		log.Fatalf("unable to connect: %v", err)
	}

	session, err := client.NewSession()
	if err != nil {
		log.Fatalf("unable to create session: %v", err)
	}

	session.Stdin = os.Stdin
	session.Stdout = os.Stdout
	session.Stderr = os.Stderr

	err = session.Run(strings.Join(os.Args[2:], " "))
	if err != nil {
		log.Fatalf("error running command: %v", err)
	}
}

func printArgs() {
	log.Println(os.Args)
	os.Exit(1)
	signer, err := ssh.NewSignerFromSigner(NewProxiedSSHKey())
	if err != nil {
		panic(err)
	}

	auth := ssh.PublicKeys(signer)
	config := &ssh.ClientConfig{
		Auth: []ssh.AuthMethod{auth},
		User: os.Getenv("USER"),
	}

	cli, err := ssh.Dial("tcp", "chat.shazow.net:22", config)
	if err != nil {
		panic(err)
	}

	session, err := RequestPseudoTerminal(cli)
	if err != nil {
		panic(err)
	}

	StartSessionAndWait(session)

}
