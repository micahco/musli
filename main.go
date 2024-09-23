package main

import (
	"fmt"
	"log"
	"os"

	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/widget"
)

const APP_NAME = "musli"

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", APP_NAME))
	log.SetFlags(0)

	conf, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	api, err := NewAPI()
	if err != nil {
		log.Fatal(err)
	}
	defer api.Close()

	a := app.New()
	w := a.NewWindow("Hello World")

	w.SetContent(widget.NewLabel("Hello World!"))
	w.ShowAndRun()

	os.Exit(0)
}
