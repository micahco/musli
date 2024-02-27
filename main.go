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
	flag.StringVar(&flagC, "config", configFile, usageC)
	flag.StringVar(&flagC, "c", configFile, usageC)

	var flagQ string
	usageQ := "search library for <query>"
	flag.StringVar(&flagQ, "query", "", usageQ)
	flag.StringVar(&flagQ, "q", "", usageQ)

	var flagR bool
	usageR := "list random albums"
	flag.BoolVar(&flagR, "random", false, usageR)
	flag.BoolVar(&flagR, "r", false, usageR)

	var flagS bool
	usageS := "scan music directory to database"
	flag.BoolVar(&flagS, "scan", false, usageS)
	flag.BoolVar(&flagS, "s", false, usageS)

	flag.Usage = func() {
		u := `Usage of musli:
 -c, --config <path>: %s
 -q, --query <query>: %s
 -r, --random: %s
 -s, --scan: %s
`
		f := fmt.Sprintf(u, usageC, usageQ, usageR, usageS)
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

	if len(flagQ) > 0 {
		albums, err := musli.SearchAlbums(flagQ)
		if err != nil {
			log.Fatal(err)
		}
		if len(albums) == 0 {
			fmt.Println("musli: no results")
			return
		}
		err = musli.Present(albums)
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
		err = musli.Present(albums)
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
}
