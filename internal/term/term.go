package term

import "fmt"

const escape = "\033"

func Clear() string {
	return fmt.Sprintf("%s[H%s[2J", escape, escape)
}

func HideCursor() string {
	return fmt.Sprintf("%s[?25l", escape)
}

func ShowCursor() string {
	return fmt.Sprintf("%s[?25h", escape)
}

func Highlight(text string, sgr int) string {
	return fmt.Sprintf("%s[1;%dm%s%s[1;0m", escape, sgr, text, escape)
}
