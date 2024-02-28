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

	var flagC string
	usageC := "use config at <path>"
	flag.StringVar(&flagC, "c", configFile, usageC)
	flag.StringVar(&flagC, "config", configFile, usageC)

	var flagQ string
	usageQ := "find albums with name or artist that contains <query>"
	flag.StringVar(&flagQ, "q", "", usageQ)
	flag.StringVar(&flagQ, "query", "", usageQ)

	var flagR bool
	usageR := "list random albums"
	flag.BoolVar(&flagR, "r", false, usageR)
	flag.BoolVar(&flagR, "random", false, usageR)

	var flagS bool
	usageS := "scan music directory to database"
	flag.BoolVar(&flagS, "s", false, usageS)
	flag.BoolVar(&flagS, "scan", false, usageS)

	var flagY string
	usageY := "find albums from <year> or range <year-end>"
	flag.StringVar(&flagY, "y", "", usageY)
	flag.StringVar(&flagY, "year", "", usageY)

	flag.Usage = func() {
		u := `Usage of musli:
 -c, --config <path>: %s
 -q, --query <query>: %s
 -r, --random: %s
 -s, --scan: %s
 -y, --year <year(-end)>: %s
`
		f := fmt.Sprintf(u, usageC, usageQ, usageR, usageS, usageY)
		fmt.Print(f)
	}
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.Usage()
		return
	}

	err = musli.Init(flagC)
	if err != nil {
		log.Fatal(err)
	}

	defer func() {
		err := musli.CloseDB()
		if err != nil {
			log.Fatal(err)
		}
	}()

	if len(flagQ) > 0 {
		albums, err := musli.FindAlbumsByNameOrAlbumArtist(flagQ)
		if err != nil {
			log.Fatal(err)
		}
		if len(albums) == 0 {
			fmt.Println("musli: no results")
			return
		}
		err = musli.ListAlbums(albums)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if flagR {
		albums, err := musli.RandomAlbums()
		if err != nil {
			log.Fatal(err)
		}
		if len(albums) == 0 {
			fmt.Println("musli: no results")
			return
		}
		err = musli.ListAlbums(albums)
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if flagS {
		err = musli.ScanLibrary()
		if err != nil {
			log.Fatal(err)
		}
		return
	}

	if len(flagY) > 0 {
		albums, err := musli.FindAlbumsByYear(flagY)
		if err != nil {
			log.Fatal(err)
		}
		if len(albums) == 0 {
			fmt.Println("musli: no results")
			return
		}
		err = musli.ListAlbums(albums)
		if err != nil {
			log.Fatal(err)
		}
		return
	}
}
