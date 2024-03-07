package term

import "fmt"

const escape = "\033"

func Clear() {
	fmt.Printf("%s[H%s[2J", escape, escape)
}

func HideCursor() {
	fmt.Printf("%s[?25l", escape)
}

func ShowCursor() {
	fmt.Printf("%s[?25h", escape)
}

func Highlight(text string, sgr int) string {
	return fmt.Sprintf("%s[1;%dm%s%s[1;0m", escape, sgr, text, escape)
}
