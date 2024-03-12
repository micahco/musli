package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/micahco/musli"
	"github.com/micahco/musli/cmd/musli/term"
)

func main() {
	var exitCode int
	defer func() {
		os.Exit(exitCode)
	}()

	configFile, err := musli.GetDefaultConfigPath()
	if err != nil {
		fmt.Println(err)
		exitCode = 1
		return
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
	usageS := "scan music directory for new files"
	flag.BoolVar(&flagS, "s", false, usageS)
	flag.BoolVar(&flagS, "scan", false, usageS)

	var flagT bool
	usageT := "scrub library for entries that no longer exist"
	flag.BoolVar(&flagT, "t", false, usageT)
	flag.BoolVar(&flagT, "tidy", false, usageT)

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
 -t, --tidy: %s
 -y, --year <year(-end)>: %s
`
		f := fmt.Sprintf(u, usageC, usageQ, usageR, usageS, usageT, usageY)
		fmt.Print(f)
	}
	flag.Parse()

	if flag.NFlag() == 0 {
		flag.Usage()
		return
	}

	conf, db, err := musli.Init(flagC)
	if err != nil {
		fmt.Println(err)
		exitCode = 1
		return
	}

	defer musli.CloseDB(db)

	if len(flagQ) > 0 {
		albums, err := musli.FindAlbumsByNameOrAlbumArtist(flagQ, db)
		if err != nil {
			fmt.Println(err)
			exitCode = 1
			return
		}
		if len(albums) == 0 {
			fmt.Println("musli: no results")
			return
		}
		err = term.CLI(albums, conf, db)
		if err != nil {
			fmt.Println(err)
			exitCode = 1
		}
		return
	}

	if flagR {
		albums, err := musli.RandomAlbums(db)
		if err != nil {
			fmt.Println(err)
			exitCode = 1
			return
		}
		if len(albums) == 0 {
			fmt.Println("musli: no results")
			return
		}
		err = term.CLI(albums, conf, db)
		if err != nil {
			fmt.Println(err)
			exitCode = 1
		}
		return
	}

	if flagS {
		fmt.Println("Scanning library...")
		err = musli.ScanLibrary(conf, db, term.ClearLine)
		if err != nil {
			fmt.Println(err)
			exitCode = 1
		}
		return
	}

	if flagT {
		fmt.Println("Cleaning library...")
		err = musli.CleanLibrary(conf, db)
		if err != nil {
			fmt.Println(err)
			exitCode = 1
		}
		return
	}

	if len(flagY) > 0 {
		albums, err := musli.FindAlbumsByYear(flagY, db)
		if err != nil {
			fmt.Println(err)
			exitCode = 1
			return
		}
		if len(albums) == 0 {
			fmt.Println("musli: no results")
			return
		}
		err = term.CLI(albums, conf, db)
		if err != nil {
			fmt.Println(err)
			exitCode = 1
		}
		return
	}
}
