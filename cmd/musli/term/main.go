package term

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/inancgumus/screen"
)

const escape = "\033"

func Open() error {
	err := keyboard.Open()
	if err != nil {
		return err
	}
	hideCursor()
	return nil
}

func Close() {
	_ = keyboard.Close()
	ClearScreen()
	showCursor()
}

func GetSize() (int, int) {
	w, h := screen.Size()
	return w, h
}

func GetInput() (map[string]bool, error) {
	m := make(map[string]bool)

	char, key, err := keyboard.GetKey()
	if err != nil {
		return m, err
	}

	switch {
	case key == keyboard.KeySpace || key == keyboard.KeyEnter:
		m["enter"] = true
	case (char == 'j' || key == keyboard.KeyArrowDown):
		m["down"] = true
	case (char == 'k' || key == keyboard.KeyArrowUp):
		m["up"] = true
	case (char == 'h' || key == keyboard.KeyArrowLeft):
		m["left"] = true
	case (char == 'l' || key == keyboard.KeyArrowRight):
		m["right"] = true
	case char == 'q' || key == keyboard.KeyCtrlC || key == keyboard.KeyCtrlD || key == keyboard.KeyEsc:
		m["quit"] = true
	}

	return m, nil
}

func ClearLine(a ...any) {
	fmt.Printf("%s[1A%s[K", escape, escape)
	fmt.Println(a...)
}

func ClearScreen() {
	screen.Clear()
}

func SprintSGR(text string, sgr ...int) string {
	var p []string
	for _, i := range sgr {
		p = append(p, strconv.Itoa(i))
	}
	return fmt.Sprintf("%s[%sm%s%s[1;0m", escape, strings.Join(p, ";"), text, escape)
}

func hideCursor() {
	fmt.Printf("%s[?25l", escape)
}

func showCursor() {
	fmt.Printf("%s[?25h", escape)
}
