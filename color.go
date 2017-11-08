package kr

import (
	"github.com/fatih/color"
)

func Cyan(s string) string {
	cyan := color.New(color.FgHiCyan)
	cyan.EnableColor()
	return cyan.SprintFunc()(s)
}

func Green(s string) string {
	green := color.New(color.FgHiGreen)
	green.EnableColor()
	return green.SprintFunc()(s)
}

func Magenta(s string) string {
	magenta := color.New(color.FgHiMagenta)
	magenta.EnableColor()
	return magenta.SprintFunc()(s)
}

func Yellow(s string) string {
	yellow := color.New(color.FgHiYellow)
	yellow.EnableColor()
	return yellow.SprintFunc()(s)
}

func Red(s string) string {
	red := color.New(color.FgHiRed)
	red.EnableColor()
	return red.SprintFunc()(s)
}
