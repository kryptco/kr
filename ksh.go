package main

import (
	cautil "bitbucket.org/kryptco/enclave/ca/util"

	"github.com/urfave/cli"

	"crypto"
	//"errors"
	//"crypto/ecdsa"
	//"crypto/sha256"
	//"crypto/x509"
	//"encoding/asn1"
	"bytes"
	"encoding/base64"
	"fmt"
	"golang.org/x/crypto/ssh"
	"golang.org/x/crypto/ssh/terminal"
	"io"
	"io/ioutil"
	"log"
	"math/big"
	"net"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

type SSHParams struct {
	User         string
	Host         string
	Port         string
	IdentityFile string
}

type CertifiedSigner struct {
	ssh.Signer
	cert ssh.PublicKey
}

func (cs CertifiedSigner) PublicKey() ssh.PublicKey {
	return cs.cert
}

type CertifiedPublicKey struct {
	ssh.PublicKey
	certBytes []byte
}

func (cpk CertifiedPublicKey) Marshal() []byte {
	b := &bytes.Buffer{}
	toks := strings.Split(string(cpk.certBytes), " ")
	b64Decoded, _ := base64.StdEncoding.DecodeString(toks[1])
	b.WriteString(cpk.Type())
	b.WriteByte(' ')
	b.Write(b64Decoded)
	b.WriteByte('\n')

	log.Println("marhsaling", string(b.String()))
	log.Println("instead of ", string(cpk.PublicKey.Marshal()))
	return b.Bytes()
}

func (cpk CertifiedPublicKey) Type() string {
	return strings.Split(string(cpk.certBytes), " ")[0]
}

func commandMain(c *cli.Context) (err error) {
	identityFile := c.String("i")
	if identityFile == "" {
		identityFile = os.Getenv("HOME") + "/.ssh/id_rsa"
	}
	key, err := ioutil.ReadFile(identityFile)
	if err != nil {
		log.Fatalf("unable to read private key: %v", err)
	}

	signer, err := ssh.ParsePrivateKey(key)
	if err != nil {
		log.Fatalf("unable to parse private key: %v", err)
	}

	key2, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_ecdsa")
	if err != nil {
		log.Fatalf("unable to read private key2: %v", err)
	}
	signer2, err := ssh.ParsePrivateKey(key2)
	if err != nil {
		log.Fatalf("unable to parse private key2: %v", err)
	}
	cert2Bytes, err := ioutil.ReadFile(os.Getenv("HOME") + "/.ssh/id_ecdsa-cert.pub")
	if err != nil {
		log.Fatalf("unable to read cert file: %v", err)
	}
	cert2, _, _, _, err := ssh.ParseAuthorizedKey(cert2Bytes)
	if err != nil {
		log.Fatalf("unable to parse cert: %v", err)
	}
	certifiedSigner2, err := ssh.NewCertSigner(cert2.(*ssh.Certificate), signer2)
	if err != nil {
		log.Fatalf("unable to make cert: %v", err)
	}

	sshHost := c.Args()[0]
	user := os.Getenv("USER")

	if strings.Contains(c.Args()[0], "@") {
		userHost := strings.Split(c.Args()[0], "@")
		user = userHost[0]
		sshHost = userHost[1]
	}

	if _, _, err = net.SplitHostPort(sshHost); err != nil {
		sshHost += ":22"
	}

	_ = signer
	config := &ssh.ClientConfig{
		User: user,
		Auth: []ssh.AuthMethod{
			ssh.PublicKeys(certifiedSigner2),
		},
	}

	log.Println(c.Args())
	if len(c.Args()) == 0 {
		log.Fatal("No host provided")
	}

	if len(c.Args()) >= 2 {
		runCommand(c, config, sshHost)
	} else {
		pty(c, config, sshHost)
	}
	return
}

type ecdsaSignature struct {
	R, S *big.Int
}

func main() {
	//sec1DER, err := base64.StdEncoding.DecodeString("BGaUAW48IrrOc7uIqX97ENI5tl4u0uKRwhHnhoKcmsIGW0PzDWlnfAInQywFwySs8qRrMzW9ZgOlSrGwQf/eogk47k+sNP+VbARMf53r3Ey3FaFbpxCd46ijXnQLNOSUmA==")
	////sec1DER, err := base64.StdEncoding.DecodeString("MEQCIF9yEnOGr8VQFkZeFN938Hy/JPYJURfWbmpyTNNHply6AiAvXEq229cJO5PPHSG5Ql6c1Slx6omWyN8iWvQ/9R5PJw==")
	//if err != nil {
	//log.Fatal(err)
	//}
	//ecKey, err := x509.ParseECPrivateKey(sec1DER)
	//if err != nil {
	//log.Fatal(err)
	//}

	//sigDER, _ := base64.StdEncoding.DecodeString("MEQCIF9yEnOGr8VQFkZeFN938Hy/JPYJURfWbmpyTNNHply6AiAvXEq229cJO5PPHSG5Ql6c1Slx6omWyN8iWvQ/9R5PJw==")
	//var sig ecdsaSignature
	//asn1.Unmarshal(sigDER, &sig)
	//digest := sha256.Sum256([]byte("aaaaaaaaaaaaaaaaa"))
	//log.Println(ecdsa.Verify(&ecKey.PublicKey, digest[:], sig.R, sig.S))

	app := cli.NewApp()
	app.Name = "ksh"
	app.Usage = "SSH using keys stored in the Kryptonite mobile app"
	app.Action = commandMain
	app.Flags = []cli.Flag{
		cli.StringFlag{
			Name:  "i",
			Usage: "identity file",
		},
	}

	app.Run(os.Args)

}

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
	if err := session.RequestPty(os.Getenv("TERM"), int(height), int(width), modes); err != nil {
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

func runCommand(c *cli.Context, sshConfig *ssh.ClientConfig, sshHost string) {
	// Connect to the remote server and perform the SSH handshake.
	client, err := ssh.Dial("tcp", sshHost, sshConfig)
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

	err = session.Run(strings.Join(c.Args()[1:], " "))
	if err != nil {
		log.Fatalf("error running command: %v", err)
	}
}

func pty(c *cli.Context, sshConfig *ssh.ClientConfig, sshHost string) {
	cli, err := ssh.Dial("tcp", sshHost, sshConfig)
	if err != nil {
		panic(err)
	}

	session, err := RequestPseudoTerminal(cli)
	if err != nil {
		panic(err)
	}

	StartSessionAndWait(session)

}
