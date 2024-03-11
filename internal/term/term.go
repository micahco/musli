package term

import (
	"fmt"
	"strconv"
	"strings"
)

const escape = "\033"

func HideCursor() {
	fmt.Printf("%s[?25l", escape)
}

func ShowCursor() {
	fmt.Printf("%s[?25h", escape)
}

func Clear() {
	fmt.Printf("%s[H%s[2J", escape, escape)
}

func ClearLine(a ...any) {
	fmt.Printf("%s[2K\r%s", escape, fmt.Sprint(a...))
}

func SprintSGR(text string, sgr ...int) string {
	var p []string
	for _, i := range sgr {
		p = append(p, strconv.Itoa(i))
	}
	return fmt.Sprintf("%s[%sm%s%s[1;0m", escape, strings.Join(p, ";"), text, escape)
}
