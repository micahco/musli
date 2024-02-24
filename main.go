package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/micahco/musli/pkg/musli"
)

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile) // debugging

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Fatal(err)
	}

	cFlag := flag.String("c", filepath.Join(homeDir, ".musli", "config.toml"), "Use config file")
	qFlag := flag.String("q", "", "Search query")
	rFlag := flag.Bool("r", false, "Random albums")
	sFlag := flag.Bool("s", false, "Scan library")
	flag.Parse()

	musli.Init(*cFlag)

	if len(*qFlag) > 0 {
		albums := musli.SearchAlbums(*qFlag, -1, -1)
		musli.ShowAlbums(albums, 10)
		return
	}

	if *rFlag {
		albums := musli.RandomAlbums()
		musli.ShowAlbums(albums, 10)
		return
	}

	if *sFlag {
		musli.ScanLibrary()
		return
	}
}
