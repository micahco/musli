package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/eiannone/keyboard"
	"github.com/micahco/musli"
)

func main() {
	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	err := root(os.Args[1:])
	if err != nil {
		printMsg("error", err.Error())
		exitCode = 1
	}
}

func root(args []string) error {
	if len(args) < 1 {
		printUsage()
		return nil
	}

	conf, err := loadConfig()
	if err != nil {
		return err
	}

	db, err := loadDB()
	if err != nil {
		return err
	}
	defer musli.CloseDB(db)

	switch args[0] {
		case "-h", "--help":
			printUsage()
		case "-q", "--query":
			err = execQuery(args[1:], conf, db)
		case "-r", "--random":
			err = execRandom(conf, db)
		case "-s", "--scan":
			err = execScan(conf, db)
		case "-t", "--tidy":
			err = execTidy(db)
		case "-y", "--year":
			err = execYear(args[1:], conf, db)
		default:
			return fmt.Errorf("invalid option: '%s'", args[0])
	}
	if err != nil {
		return err
	}
	return nil
}

func loadConfig() (*musli.Config, error) {
	path, err := musli.GetDefaultConfigPath()
	if err != nil {
		return nil, err
	}

	conf, err := musli.ReadConfig(path)
	if err != nil {
		return nil, err
	}

	return conf, nil
}

func loadDB() (*sql.DB, error) {
	path, err := musli.GetDBPath()
	if err != nil {
		return nil, err
	}

	db, err := musli.OpenDB(path)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func printMsg(s ...string) {
	msg := os.Args[0]
	for _, m := range s {
		msg += ": " + m
	}
	fmt.Println(msg)
}

func printNoResults() {
	printMsg("no results")
}

func printUsage() {
	fmt.Printf("Usage of %s:", os.Args[0])
	fmt.Println(`
-q, --query <query>: find albums by <query>
-r, --random: list random albums
-s, --scan: scan music directory for new files
-t, --tidy: scrub library for entries that no longer exist
-y, --year <year> [year]: find albums from <year> or between <year> [year]`)
}

func execQuery(args []string, conf *musli.Config, db *sql.DB) error {
	if len(args) < 1 {
		return errors.New("no query")
	}

	albums, err := musli.FindAlbumsByNameOrArtist(args[1], db)
	if err != nil {
		return err
	}
	if len(albums) == 0 {
		printNoResults()
		return nil
	}
	err = listAlbums(albums, conf, db)
	if err != nil {
		return err
	}
	return nil
}

func execRandom(conf *musli.Config, db *sql.DB) error {
	albums, err := musli.FetchRandomAlbums(db)
	if err != nil {
		return err
	}
	
	if len(albums) == 0 {
		printNoResults()
		return nil
	}

	err = listAlbums(albums, conf, db)
	if err != nil {
		return err
	}

	return nil
}

func execScan(conf *musli.Config, db *sql.DB) error {
	fmt.Print("Scanning library...")
	
	paths, err := musli.WalkLibrary(conf)
	if err != nil {
		return err
	}
	total := len(paths)

	err = musli.AddPathsToLibrary(paths, db, func(i int) {
		clearLine(i, "/", total)
	})
	if err != nil {
		return err
	}

	clearLine("Scanned ", total, " files\n")
	return nil
}

func execTidy(db *sql.DB) error {
	fmt.Println("Cleaning library...")

	paths, err := musli.FetchTrackPaths(db)
	if err != nil {
		return err
	}

	err = musli.RemoveNotExistPaths(paths, db)
	if err != nil {
		return err
	}

	err = musli.RemoveEmptyAlbums(db)
	if err != nil {
		return err
	}

	return nil
}

func execYear(args []string, conf *musli.Config, db *sql.DB) error {
	if len(args) < 1 {
		return errors.New("no query")
	}

	albums, err := musli.FindAlbumsByYear(args, db)
	if err != nil {
		return err
	}
	
	if len(albums) == 0 {
		printNoResults()
		return nil
	}

	err = listAlbums(albums, conf, db)
	if err != nil {
		return err
	}

	return nil
}

const escape = "\033"

func hideCursor() {
	fmt.Printf("%s[?25l", escape)
}

func showCursor() {
	fmt.Printf("%s[?25h", escape)
}

func clearScreen() {
	fmt.Printf("%s[H%s[2J", escape, escape)
}

func clearLine(a ...any) {
	fmt.Printf("%s[2K\r%s", escape, fmt.Sprint(a...))
}

func sprintSGR(text string, sgr ...int) string {
	var p []string
	for _, i := range sgr {
		p = append(p, strconv.Itoa(i))
	}
	return fmt.Sprintf("%s[%sm%s%s[1;0m", escape, strings.Join(p, ";"), text, escape)
}

func listAlbums(albums []musli.Album, conf *musli.Config, db *sql.DB) error {
	err := keyboard.Open()
	if err != nil {
		return err
	}

	hideCursor()
	defer func() {
		_ = keyboard.Close()
		clearScreen()
		showCursor()
	}()

	pageLength := conf.PageLength
	start := 0
	sel := 0
	max := len(albums)
	for {
		clearScreen()
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
				l = sprintSGR(l, conf.HiglightSGR...)
			}
			fmt.Println(l)
		}
		char, key, err := keyboard.GetKey()
		if err != nil {
			return err
		}
		if key == keyboard.KeySpace || key == keyboard.KeyEnter {
			err := musli.PlayAlbum(pageAlbums[sel].ID, conf, db)
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
