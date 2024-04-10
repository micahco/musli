package term

// #include "Windows.h"
import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"

	"github.com/eiannone/keyboard"
	"golang.org/x/term"
)

const ESCAPE = "\033"

func OpenCLI() error {
	err := keyboard.Open()
	if err != nil {
		return err
	}
	hideCursor()
	return nil
}

func CloseCLI() {
	_ = keyboard.Close()
	ClearScreen()
	showCursor()
}

func ClearScreen() {
	switch runtime.GOOS {
	case "windows":
		cmd := exec.Command("cmd", "/c", "cls")
		cmd.Stdout = os.Stdout
		cmd.Run()
	default:
		fmt.Printf("%s[H%s[2J", ESCAPE, ESCAPE)
	}
}

func ClearLine(a ...any) {
	switch runtime.GOOS {
	default:
		fmt.Printf("%s[1A%s[K", ESCAPE, ESCAPE)
		fmt.Println(a...)
	}
}

func GetSize() (int, int, error) {
	if !term.IsTerminal(0) {
		return 0, 0, nil
	}
	w, h, err := term.GetSize(int(os.Stdin.Fd()))
	if err != nil {
		return 0, 0, err
	}
	return w, h, nil
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

func SprintSGR(text string, sgr ...int) string {
	var p []string
	for _, i := range sgr {
		p = append(p, strconv.Itoa(i))
	}
	return fmt.Sprintf("%s[%sm%s%s[1;0m", ESCAPE, strings.Join(p, ";"), text, ESCAPE)
}

func hideCursor() {
	fmt.Printf("%s[?25l", ESCAPE)
}

func showCursor() {
	fmt.Printf("%s[?25h", ESCAPE)
}
