package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"
	"krypt.co/kr/common/socket"
	"os"
	"os/exec"
	"strings"
	"time"

	"github.com/urfave/cli/v2"

	. "krypt.co/kr/common/protocol"
	. "krypt.co/kr/common/util"
	krdclient "krypt.co/kr/daemon/client"
)

var stderr *os.File

func setupTTY() {
	var err error
	stderr, err = os.OpenFile(os.Getenv("GPG_TTY"), os.O_RDWR, 0)
	if err != nil {
		stderr, err = os.OpenFile(os.Getenv("TTY"), os.O_RDWR, 0)
		if err != nil {
			stderr = os.Stderr
		}
	}
}

func readLineSplittingFirstToken(reader *bufio.Reader) (firstToken string, rest string, err error) {
	line, err := reader.ReadString('\n')
	if err != nil {
		return
	}
	toks := strings.Fields(line)
	if len(toks) == 0 {
		err = fmt.Errorf("no tokens")
		return
	}
	firstToken = toks[0]
	rest = strings.Join(toks[1:], " ")
	return
}

func main() {
	setupTTY()
	app := cli.NewApp()
	app.Name = "krgpg"
	app.Usage = "Sign git commits with your Krypton key"
	app.Flags = []cli.Flag{
		&cli.BoolFlag{
			Name:  "a",
			Usage: "Ouput ascii armor",
		},
		&cli.BoolFlag{
			Name:  "b,detach-sign",
			Usage: "Create a detached signature",
		},
		&cli.BoolFlag{
			Name:  "s,sign",
			Usage: "Create a signature",
		},
		&cli.StringFlag{
			Name:  "u,local-user",
			Value: "",
			Usage: "User ID",
		},
		&cli.StringFlag{
			Name:  "status-fd",
			Value: "",
			Usage: "status file descriptor",
		},
		&cli.BoolFlag{
			Name:  "bsau",
			Usage: "Git method of passing in detach-sign ascii armor flags",
		},
		&cli.BoolFlag{
			Name:  "verify",
			Usage: "Verify a signature",
		},
		&cli.StringFlag{
			Name:  "keyid-format",
			Usage: "Key ID format",
		},
		&cli.BoolFlag{
			Name:  "batch",
			Hidden: true,
		},
		&cli.BoolFlag{
			Name:  "no-tty",
			Hidden: true,
		},
		&cli.BoolFlag{
			Name:  "yes",
			Hidden: true,
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("bsau") || (c.Bool("s") && c.Bool("b") && c.Bool("a")) {
			//	TODO: verify userID matches stored kryptonite PGP key
			signGit()
		} else {
			redirectToGPG(os.Stdin)
		}
		return nil
	}
	app.OnUsageError = func(c *cli.Context, err error, isSubcommand bool) error {
		stderr.WriteString(Red(err.Error() + "\n"))
		redirectToGPG(os.Stdin)
		return nil
	}

	app.Run(os.Args)
	return
}

func redirectToGPG(stdin io.Reader) {
	gpgCommand := exec.Command("gpg", os.Args[1:]...)
	gpgCommand.Stdin = stdin
	gpgCommand.Stdout = os.Stdout
	gpgCommand.Stderr = os.Stderr
	err := gpgCommand.Run()
	if err == nil {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func signGit() {
	stdinBytes, err := ioutil.ReadAll(os.Stdin)
	if err != nil {
		stderr.WriteString("error reading stdin")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	reader := bufio.NewReader(bytes.NewReader(stdinBytes))
	tag, firstLine, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing first line")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	switch tag {
	case "tree":
		err = signGitCommit(firstLine, reader)
	case "object":
		err = signGitTag(firstLine, reader)
	default:
		err = fmt.Errorf("error parsing commit tree, wrong tag")
		stderr.WriteString(err.Error())
	}
	if err != nil {
		if HasGPG() {
			stderr.WriteString(Yellow("Krypton ▶ Falling back to local gpg keychain") + "\r\n")
			stderr.WriteString(string(stdinBytes))
			redirectToGPG(bytes.NewReader(stdinBytes))
		} else {
			os.Exit(1)
		}
	}
}

func signGitCommit(tree string, reader *bufio.Reader) (err error) {
	var firstParent *string
	var mergeParents *[]string
	var author string
	err = func() (err error) {
		for {
			tag, contents, err := readLineSplittingFirstToken(reader)
			if err != nil {
				stderr.WriteString("error parsing commit")
				stderr.WriteString(err.Error())
				return err
			}
			switch tag {
			case "parent":
				if firstParent == nil {
					firstParent = &contents
					continue
				}
				if mergeParents == nil {
					mergeParents = &[]string{}
				}
				newMergeParents := append(*mergeParents, contents)
				mergeParents = &newMergeParents
			case "author":
				author = contents
				return nil
			default:
				err = fmt.Errorf("unexpected tag: " + tag)
				stderr.WriteString(err.Error())
				return err
			}
		}
	}()
	if err != nil {
		return
	}

	_, committer, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing commit committer")
		stderr.WriteString(err.Error())
		return
	}
	message, err := ioutil.ReadAll(reader)
	if err != nil {
		stderr.WriteString("error parsing commit message")
		stderr.WriteString(err.Error())
		return
	}
	commit := CommitInfo{
		Tree:         tree,
		Parent:       firstParent,
		MergeParents: mergeParents,
		Author:       author,
		Committer:    committer,
		Message:      message,
	}
	request, err := NewRequest()
	if err != nil {
		stderr.WriteString(err.Error())
		return
	}
	startLogger(request.NotifyPrefix())
	request.GitSignRequest = &GitSignRequest{
		Commit: &commit,
		UserId: os.Args[len(os.Args)-1],
	}
	stderr.WriteString(Cyan("Krypton ▶ Requesting git commit signature from phone") + "\r\n")
	response, err := requestSignature(request)
	if err != nil {
		return
	}
	sig, err := response.GitSignResponse.AsciiArmorSignature(response.Version)
	if err != nil {
		stderr.WriteString(err.Error())
		return
	}
	os.Stdout.WriteString(sig)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Close()
	os.Stderr.WriteString("\n[GNUPG:] SIG_CREATED ")
	os.Exit(0)
	return
}

func signGitTag(object string, reader *bufio.Reader) (err error) {
	_, _type, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing type")
		stderr.WriteString(err.Error())
		return
	}
	_, tag, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing tag")
		stderr.WriteString(err.Error())
		return
	}
	_, tagger, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing tagger")
		stderr.WriteString(err.Error())
		return
	}
	message, err := ioutil.ReadAll(reader)
	if err != nil {
		stderr.WriteString("error parsing commit message")
		stderr.WriteString(err.Error())
		return
	}
	tagInfo := TagInfo{
		Object:  object,
		Type:    _type,
		Tag:     tag,
		Tagger:  tagger,
		Message: message,
	}
	request, err := NewRequest()
	if err != nil {
		stderr.WriteString(err.Error())
		return
	}
	startLogger(request.NotifyPrefix())
	request.GitSignRequest = &GitSignRequest{
		Tag:    &tagInfo,
		UserId: os.Args[len(os.Args)-1],
	}
	stderr.WriteString(Cyan("Krypton ▶ Requesting git tag signature from phone") + "\r\n")
	response, err := requestSignature(request)
	if err != nil {
		return
	}
	sig, err := response.GitSignResponse.AsciiArmorSignature(response.Version)
	if err != nil {
		stderr.WriteString(err.Error())
		return
	}
	os.Stdout.WriteString(sig)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Close()
	os.Stderr.WriteString("\n[GNUPG:] SIG_CREATED ")
	os.Exit(0)
	return
}

func requestSignature(request Request) (sig Response, err error) {
	response, err := krdclient.RequestGitSignature(request)
	if err != nil {
		switch err {
		case ErrNotPaired:
			stderr.WriteString(Yellow("Krypton ▶ "+ErrNotPaired.Error()) + "\r\n")
			return
		case ErrConnectingToDaemon:
			stderr.WriteString(Red("Krypton ▶ Could not connect to Krypton daemon. Make sure it is running by typing \"kr restart\"\r\n"))
			return
		default:
			stderr.WriteString(Red("Krypton ▶ Unknown error: " + err.Error() + "\r\n"))
			return
		}
	}
	if response.GitSignResponse == nil {
		err = fmt.Errorf("no GitSignResponse")
		return
	}
	gitSignResponse := response.GitSignResponse
	if gitSignResponse.Error != nil {
		switch *gitSignResponse.Error {
		case "rejected":
			stderr.WriteString(Red("Krypton ▶ " + ErrRejected.Error() + "\r\n"))
			err = fmt.Errorf("%s", *gitSignResponse.Error)
			return
		}
	}
	stderr.WriteString(Green("Krypton ▶ Success. Request Allowed ✔") + "\r\n")
	return response, nil
}

func startLogger(prefix string) (r socket.NotificationReader, err error) {
	r, err = socket.OpenNotificationReader(prefix)
	if err != nil {
		return
	}
	go func() {
		if prefix != "" {
			defer os.Remove(r.Name())
		}

		printedNotifications := map[string]bool{}
		for {
			notification, err := r.Read()
			switch err {
			case nil:
				notificationStr := string(notification)
				if _, ok := printedNotifications[notificationStr]; ok {
					continue
				}
				stderr.WriteString(notificationStr)
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
