package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/micahco/musli/pkg/musli"
)

func main() {
	configFile, err := musli.ConfigFile()
	if err != nil {
		log.Fatal(err)
	}

	cFlag := flag.String("c", configFile, "Use config file")
	qFlag := flag.String("q", "", "Search library for query")
	rFlag := flag.Bool("r", false, "List random albums")
	sFlag := flag.Bool("s", false, "Scan music directory to database")
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.Usage()
		return
	}

	err = musli.Init(*cFlag)
	if err != nil {
		log.Fatal(err)
	}

	if len(*qFlag) > 0 {
		albums, err := musli.SearchAlbums(*qFlag)
		if err != nil {
			log.Fatal(err)
		}
		err = musli.ShowAlbums(albums)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if *rFlag {
		albums, err := musli.RandomAlbums()
		if err != nil {
			log.Fatal(err)
		}
		if len(albums) == 0 {
			fmt.Println("No entries in database.\nTo scan music directory to databse, run: musli -s")
			return
		}
		err = musli.ShowAlbums(albums)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if *sFlag {
		err = musli.ScanLibrary()
		if err != nil {
			log.Fatal(err)
		}
		return
	}
}
