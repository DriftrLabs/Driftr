package ioutil

import "os"

func colorEnabled() bool {
	if os.Getenv("NO_COLOR") != "" {
		return false
	}
	return IsTerminal(os.Stdout)
}

func colorize(s, ansiCode string) string {
	return "\033[" + ansiCode + "m" + s + "\033[0m"
}

func Green(s string) string {
	if !colorEnabled() {
		return s
	}
	return colorize(s, "32")
}

func Bold(s string) string {
	if !colorEnabled() {
		return s
	}
	return colorize(s, "1")
}

func Dim(s string) string {
	if !colorEnabled() {
		return s
	}
	return colorize(s, "2")
}
