package main

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/micahco/musli"
	"github.com/micahco/musli/cmd/musli/term"
)

func main() {
	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	err := root(os.Args[1:])
	if err != nil {
		printMsg(err.Error())
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

func execQuery(args []string, conf *musli.Config, db *sql.DB) error {
	if len(args) < 1 {
		return errors.New("no query")
	}

	albums, err := musli.FindAlbumsByNameOrArtist(args[0], db)
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
	fmt.Println("Scanning directory")
	paths, err := musli.GetMusicDirPaths(conf)
	if err != nil {
		return err
	}

	err = musli.AddPathsToLibrary(paths, db, func(n int) {
		term.ClearLine(n, "/", len(paths))
	})
	if err != nil {
		return err
	}

	term.ClearLine("Scanned ", len(paths), " files\n")
	return nil
}

func execTidy(db *sql.DB) error {
	fmt.Print("Scrubbing library")
	paths, err := musli.FetchTrackPaths(db)
	if err != nil {
		return err
	}

	err = musli.RemoveNotExistPaths(paths, db, func(n int) {
		term.ClearLine(n, "/", len(paths))
	})
	if err != nil {
		return err
	}

	term.ClearLine("Cleaning up...")
	err = musli.RemoveEmptyAlbums(db)
	if err != nil {
		return err
	}
	term.ClearLine("done\n")

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

func printAlbum(a musli.Album, t string, highlight bool, sgr []int) {
	t = strings.Replace(t, "%album%", a.Name, -1)
	t = strings.Replace(t, "%artist%", a.AlbumArtist, -1)
	t = strings.Replace(t, "%year%", strconv.Itoa(a.Year), -1)
	if highlight {
		t = term.SprintSGR(t, sgr...)
	}
	fmt.Println(t)
}

func validateListIndex(i, p, l, max int) int {
	end := p + l - 1
	if i < p {
		return p
	} else if i > end {
		return end
	} else if i > max {
		return max
	}
	return i
}

func listAlbums(albums []musli.Album, conf *musli.Config, db *sql.DB) error {
	err := term.Open()
	if err != nil {
		return err
	}
	defer term.Close()

	l := conf.PageLength
	max := len(albums) - 1
	var p, i int // page start, index
	for {
		term.ClearScreen()
		i = validateListIndex(i, p, l, max)
		for j := p; j < p+l && j <= max; j++ {
			printAlbum(albums[j], conf.ListTemplate, j == i, conf.HiglightSGR)
		}

		in, err := term.GetInput()
		if err != nil {
			return err
		}
		switch {
		case in["enter"]:
			err = musli.PlayAlbum(albums[i].ID, conf, db)
			if err != nil {
				return err
			}
			return nil
		case in["down"]:
			i++
		case in["up"]:
			i--
		case in["left"] && p-l >= 0:
			i -= l
			p -= l
		case in["right"] && p+l <= max:
			i += l
			p += l
		case in["quit"]:
			return nil
		}
	}
}
