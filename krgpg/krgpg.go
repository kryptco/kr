package main

import (
	"bufio"
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	"github.com/urfave/cli"

	"github.com/kryptco/kr"
	"github.com/kryptco/kr/krdclient"
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
	app.Usage = "Sign git commits with your Kryptonite key"
	app.Flags = []cli.Flag{
		cli.BoolFlag{
			Name:  "a",
			Usage: "Ouput ascii armor",
		},
		cli.BoolFlag{
			Name:  "b,detach-sign",
			Usage: "Create a detached signature",
		},
		cli.BoolFlag{
			Name:  "s,sign",
			Usage: "Create a signature",
		},
		cli.StringFlag{
			Name:  "u,local-user",
			Value: "",
			Usage: "User ID",
		},
		cli.StringFlag{
			Name:  "status-fd",
			Value: "",
			Usage: "status file descriptor",
		},
		cli.BoolFlag{
			Name:  "bsau",
			Usage: "Git method of passing in detach-sign ascii armor flags",
		},
		cli.BoolFlag{
			Name:  "verify",
			Usage: "Verify a signature",
		},
		cli.StringFlag{
			Name:  "keyid-format",
			Usage: "Key ID format",
		},
	}
	app.Action = func(c *cli.Context) error {
		if c.Bool("bsau") || (c.Bool("s") && c.Bool("b") && c.Bool("a")) {
			//	TODO: verify userID matches stored kryptonite PGP key
			signGit()
		} else {
			redirectToGPG()
		}
		return nil
	}
	app.OnUsageError = func(c *cli.Context, err error, isSubcommand bool) error {
		stderr.WriteString(kr.Red(err.Error() + "\n"))
		redirectToGPG()
		return nil
	}

	app.Run(os.Args)
	return
}

func redirectToGPG() {
	gpgCommand := exec.Command("gpg", os.Args[1:]...)
	gpgCommand.Stdin = os.Stdin
	gpgCommand.Stdout = os.Stdout
	gpgCommand.Stderr = os.Stderr
	gpgCommand.Run()
}

func signGit() {
	stdinBytes, _ := ioutil.ReadAll(os.Stdin)
	stderr.WriteString(string(stdinBytes))
	reader := bufio.NewReader(bytes.NewReader(stdinBytes))
	tag, firstLine, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing first line")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	switch tag {
	case "tree":
		signGitCommit(firstLine, reader)
	case "object":
		signGitTag(firstLine, reader)
	default:
		stderr.WriteString("error parsing commit tree, wrong tag")
		os.Exit(1)
	}
}

func signGitCommit(tree string, reader *bufio.Reader) {
	var parent *string
	var author string
	secondTag, secondContents, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing commit second line")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	switch secondTag {
	case "parent":
		parent = &secondContents
		_, author, err = readLineSplittingFirstToken(reader)
		if err != nil {
			stderr.WriteString("error parsing commit author")
			stderr.WriteString(err.Error())
			os.Exit(1)
		}
	case "author":
		author = secondContents
	default:
		stderr.WriteString("error parsing commit second line, wrong tag")
		os.Exit(1)
	}

	_, committer, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing commit committer")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	message, err := ioutil.ReadAll(reader)
	if err != nil {
		stderr.WriteString("error parsing commit message")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	commit := kr.CommitInfo{
		Tree:      tree,
		Parent:    parent,
		Author:    author,
		Committer: committer,
		Message:   message,
	}
	request := kr.GitSignRequest{
		Commit: &commit,
		UserId: os.Args[len(os.Args)-1],
	}
	response, err := krdclient.RequestGitSignature(request)
	if err != nil {
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	sig, err := response.AsciiArmorSignature()
	if err != nil {
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	os.Stdout.WriteString(sig)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Close()
	os.Stderr.WriteString("\n[GNUPG:] SIG_CREATED ")
	os.Exit(0)
}

func signGitTag(object string, reader *bufio.Reader) {
	_, _type, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing type")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	_, tag, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing tag")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	_, tagger, err := readLineSplittingFirstToken(reader)
	if err != nil {
		stderr.WriteString("error parsing tagger")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	message, err := ioutil.ReadAll(reader)
	if err != nil {
		stderr.WriteString("error parsing commit message")
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	tagInfo := kr.TagInfo{
		Object:  object,
		Type:    _type,
		Tag:     tag,
		Tagger:  tagger,
		Message: message,
	}
	request := kr.GitSignRequest{
		Tag:    &tagInfo,
		UserId: os.Args[len(os.Args)-1],
	}
	response, err := krdclient.RequestGitSignature(request)
	if err != nil {
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	sig, err := response.AsciiArmorSignature()
	if err != nil {
		stderr.WriteString(err.Error())
		os.Exit(1)
	}
	os.Stdout.WriteString(sig)
	os.Stdout.Write([]byte("\n"))
	os.Stdout.Close()
	os.Stderr.WriteString("\n[GNUPG:] SIG_CREATED ")
	os.Exit(0)
}
