package main

import (
	"io/ioutil"
	"math"
	"os"
	"strings"
	"time"
)

type command struct {
	command string
	time    time.Time
}

func getLastZshCommand() *command {
	file := os.Getenv("HOME") + "/.zsh_history"
	hist, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}
	histStr := string(hist)
	lines := strings.Split(histStr, "\n")
	if len(lines) < 2 {
		return nil
	}

	lastCommandLog := lines[len(lines)-2]
	logTokens := strings.Split(lastCommandLog, ";")
	if len(logTokens) < 2 {
		return nil
	}
	stat, err := os.Stat(file)
	if err != nil {
		return nil
	}

	lastCommand := logTokens[1]
	return &command{lastCommand, stat.ModTime()}
}

func getLastBashCommand() *command {
	file := os.Getenv("HOME") + "/.bash_history"
	hist, err := ioutil.ReadFile(file)
	if err != nil {
		return nil
	}
	histStr := string(hist)
	lines := strings.Split(histStr, "\n")
	if len(lines) == 0 {
		return nil
	}

	lastCommand := lines[len(lines)-1]
	stat, err := os.Stat(file)
	if err != nil {
		return nil
	}
	return &command{lastCommand, stat.ModTime()}
}

func getLastCommand() *string {
	commands := []*command{getLastZshCommand(), getLastBashCommand()}
	var latestCommand *command
	for _, command := range commands {
		if command == nil {
			continue
		}
		if latestCommand == nil || command.time.After(latestCommand.time) {
			latestCommand = command
		}
	}
	if latestCommand != nil {
		if math.Abs(float64(latestCommand.time.Sub(time.Now()))) < float64(30*time.Second) {
			return &latestCommand.command
		}
	}
	return nil
}
