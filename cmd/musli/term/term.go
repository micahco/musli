package term

import (
	"database/sql"
	"fmt"
	"strconv"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/micahco/musli"
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

func CLI(albums []musli.Album, conf *musli.Config, db *sql.DB) error {
	err := keyboard.Open()
	if err != nil {
		return err
	}

	HideCursor()
	defer func() {
		_ = keyboard.Close()
		Clear()
		ShowCursor()
	}()

	pageLength := conf.PageLength
	start := 0
	sel := 0
	max := len(albums)
	for {
		Clear()
		if start < 0 {
			start = 0
		}
		if sel < 0 || start+sel >= max {
			sel = 0
		}
		var pageAlbums []musli.Album
		for i := 0; i < pageLength && start+i < len(albums); i++ {
			pos := start + i
			a := albums[pos]
			pageAlbums = append(pageAlbums, a)
			l := conf.ListTemplate
			l = strings.Replace(l, "%album%", a.Name, -1)
			l = strings.Replace(l, "%artist%", a.AlbumArtist, -1)
			l = strings.Replace(l, "%year%", strconv.Itoa(a.Year), -1)
			if sel == i {
				l = SprintSGR(l, conf.HiglightSGR...)
			}
			fmt.Println(l)
		}
		char, key, err := keyboard.GetKey()
		if err != nil {
			return err
		}
		if key == keyboard.KeySpace || key == keyboard.KeyEnter {
			err := musli.PlayAlbum(pageAlbums[sel], conf, db)
			if err != nil {
				return err
			}
			return nil
		}
		if (char == 'j' || key == keyboard.KeyArrowDown) && sel < pageLength-1 && start+sel < len(albums)-1 {
			sel += 1
			continue
		}
		if (char == 'k' || key == keyboard.KeyArrowUp) && sel > 0 {
			sel -= 1
			continue
		}
		if (char == 'h' || key == keyboard.KeyArrowLeft) && start-pageLength >= 0 {
			start -= pageLength
			continue
		}
		if (char == 'l' || key == keyboard.KeyArrowRight) && start+pageLength <= max {
			start += pageLength
			continue
		}
		if char == 'q' || key == keyboard.KeyCtrlC || key == keyboard.KeyCtrlD {
			break
		}
	}
	return nil
}
