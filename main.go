package main

import (
	"fmt"
	"image/color"
	"log"
	"os"

	"gioui.org/app"
	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/micahco/musli/api"
	"github.com/micahco/musli/config"
)

const APP_NAME = "musli"

func main() {
	log.SetPrefix(fmt.Sprintf("%s: ", APP_NAME))
	log.SetFlags(0)

	conf, err := config.Load(APP_NAME)
	if err != nil {
		log.Fatal(err)
	}

	api, err := api.New(APP_NAME)
	if err != nil {
		log.Fatal(err)
	}
	defer api.Close()

	go func() {
		w := new(app.Window)
		w.Option(app.Title("Musli"))
		err := draw(w, api, conf)
		if err != nil {
			log.Fatal(err)
		}
		os.Exit(0)
	}()
	app.Main()
}

const (
	initalCellSize = 100
	cellSizeStep   = 50
	minCellSize    = 100
	maxCellSize    = 500
)

type (
	C = layout.Context
	D = layout.Dimensions
)

func draw(w *app.Window, api api.API, conf config.Config) error {
	th := material.NewTheme()
	var (
		ops          op.Ops
		grid         component.GridState
		updateButton widget.Clickable
	)

	clickers := []widget.Clickable{}
	for i := 0; i < 1; i++ {
		clickers = append(clickers, widget.Clickable{})
	}

	cellSize := initalCellSize

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			for {
				ev, ok := gtx.Event(
					key.Filter{Optional: key.ModShift, Name: "+"},
					key.Filter{Optional: key.ModShift, Name: "="},
					key.Filter{Optional: key.ModShift, Name: "-"},
				)
				if !ok {
					break
				}

				if ev.(key.Event).State == key.Press {
					name := ev.(key.Event).Name
					if name == "+" || name == "=" {
						if cellSize < maxCellSize {
							cellSize += cellSizeStep
						}
					}
					if name == "-" {
						if cellSize > minCellSize {
							cellSize -= cellSizeStep
						}
					}
				}
			}

			for i := range clickers {
				if clickers[i].Clicked(gtx) {
					fmt.Println("You clicked button", i)
				}
			}

			windowWidth := e.Size.X
			colCount := windowWidth / cellSize
			rowCount := len(clickers)/colCount + 1

			layout.Flex{
				Axis: layout.Horizontal,
			}.Layout(gtx,
				// Update button
				layout.Rigid(
					func(gtx C) D {
						// The text on the button depends on program state
						text := "Update"
						btn := material.Button(th, &updateButton, text)
						return btn.Layout(gtx)
					},
				),

				layout.Rigid(
					func(gtx C) D {
						// The text on the button depends on program state
						text := "Update"
						btn := material.Button(th, &updateButton, text)
						return btn.Layout(gtx)
					},
				),
			)

			component.Grid(th, &grid).Layout(gtx, rowCount, colCount,
				func(axis layout.Axis, index, constraint int) int {
					return gtx.Dp(unit.Dp(cellSize))
				},
				func(gtx C, row, col int) D {
					// Calculate index
					i := row*colCount + col

					// Check if index is in bounds
					if i >= len(clickers) {
						return D{}
					}

					clk := &clickers[i]
					btn := material.Button(th, clk, fmt.Sprintf("%d", i))
					color := color.NRGBA{
						R: uint8(255 / colCount * row),
						G: uint8(255 / colCount * col),
						B: uint8(255 * row * col / (colCount * colCount)),
						A: 255,
					}
					btn.Background = color
					btn.CornerRadius = 0
					return btn.Layout(gtx)
				})

			e.Frame(gtx.Ops)
		}
	}
}
