package main

import (
	"errors"
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

type (
	C = layout.Context
	D = layout.Dimensions
)

const (
	// Number of columns
	ncolsMax     int = 31
	ncolsMin     int = 3
	ncolsInitial int = 9
	ncolsStep    int = 2
)

func draw(w *app.Window, api API, conf Config) error {
	th := material.NewTheme()
	var (
		ops op.Ops

		// Components
		grid component.GridState

		// Buttons
		updateButton widget.Clickable
		cleanButton  widget.Clickable
		sortButton   widget.Clickable

		// State
		albums []Album
		//pictures []image.Image
		clickers []widget.Clickable
		updating bool
		cleaning bool
		progress int
		total    int
	)

	// Initial albums state
	var err error
	albums, err = api.GetRandomAlbums()
	if err != nil {
		log.Fatal(err)
	}

	for i := 0; i < len(albums); i++ {
		clickers = append(clickers, widget.Clickable{})
	}

	colorTransparent := color.NRGBA{R: 0, G: 0, B: 0, A: 0}

	ncols := ncolsInitial

	for {
		switch e := w.Event().(type) {
		case app.DestroyEvent:
			return e.Err
		case app.FrameEvent:
			gtx := app.NewContext(&ops, e)

			for {
				// Define which keypress to monitor
				ev, ok := gtx.Event(
					key.Filter{Optional: key.ModShift, Name: "0"},
					key.Filter{Optional: key.ModShift, Name: "+"},
					key.Filter{Optional: key.ModShift, Name: "="},
					key.Filter{Optional: key.ModShift, Name: "-"},
				)
				if !ok {
					break
				}

				// Handle keypress
				if ev.(key.Event).State == key.Press {
					name := ev.(key.Event).Name
					if name == "0" {
						// Reset ncols to initial state
						ncols = ncolsInitial
					}
					if name == "+" || name == "=" {
						// Zoom in by decreasing the ncols
						if ncols > ncolsMin {
							ncols -= ncolsStep
						}
					}
					if name == "-" {
						// Zoom out by incrasing the ncols
						if ncols < ncolsMax {
							ncols += ncolsStep
						}
					}
				}
			}

			// Handle buttons and clickers
			if updateButton.Clicked(gtx) && !(updating || cleaning) {
				// Update library
				go func() {
					updating = true

					paths, err := findAudioFilePaths(conf.MusicDir)
					if err != nil {
						log.Fatal(err)
					}
					total = len(paths)

					for i, path := range paths {
						progress = i
						w.Invalidate()

						err = api.AddPathToLibrary(path)
						if err != nil {
							log.Fatal(err)
						}
					}

					// Reset state
					total = 0
					progress = 0
					updating = false

					albums, err = api.GetRandomAlbums()
					if err != nil {
						log.Fatal(err)
					}
				}()
			}

			if cleanButton.Clicked(gtx) && !(updating || cleaning) {
				// Clean library
				go func() {
					cleaning = true

					paths, err := api.AllTrackPaths()
					if err != nil {
						log.Fatal(err)
					}
					total = len(paths)

					for i, path := range paths {
						progress = i
						w.Invalidate()

						// Delete tracks that no longer exist
						_, err := os.Stat(path)
						if errors.Is(err, os.ErrNotExist) {
							err = api.DeleteTrack(path)
						}

						if err != nil {
							log.Fatal(err)
						}
					}

					err = api.RemoveEmptyAlbums()
					if err != nil {
						log.Fatal(err)
					}

					// Reset state
					total = 0
					progress = 0
					cleaning = false

					albums, err = api.GetRandomAlbums()
					if err != nil {
						log.Fatal(err)
					}
				}()
			}

			for i := range clickers {
				if clickers[i].Clicked(gtx) {
					fmt.Println("You clicked button", i)
				}
			}

			cellSize := e.Size.X / ncols
			nrows := (len(clickers) / ncols) + 1

			// Root flex container
			layout.Flex{
				Axis: layout.Vertical,
			}.Layout(gtx,

				// Padding
				layout.Rigid(
					layout.Spacer{Height: unit.Dp(5)}.Layout,
				),

				// Top bar
				layout.Rigid(
					func(gtx C) D {
						return layout.Flex{
							Axis:    layout.Horizontal,
							Spacing: layout.SpaceBetween,
						}.Layout(gtx,

							layout.Rigid(
								func(gtx C) D {
									return layout.Flex{
										Axis: layout.Horizontal,
									}.Layout(gtx,

										// Update button
										layout.Rigid(
											func(gtx C) D {
												var text string
												if updating {
													text = "Updating"
													if total == 0 {
														text += "..."
													} else {
														text += fmt.Sprintf(" (%d/%d)", progress, total)
													}
												} else {
													text = "Update library"
												}

												btn := material.Button(th, &updateButton, text)

												return btn.Layout(gtx)
											},
										),

										// Padding
										layout.Rigid(
											layout.Spacer{Width: unit.Dp(5)}.Layout,
										),

										// Clean button
										layout.Rigid(
											func(gtx C) D {
												var text string
												if cleaning {
													text = "Cleaning"
													if total == 0 {
														text += "..."
													} else {
														text += fmt.Sprintf(" (%d/%d)", progress, total)
													}
												} else {
													text = "Clean library"
												}

												btn := material.Button(th, &cleanButton, text)

												return btn.Layout(gtx)
											},
										),
									)
								},
							),

							// TODO: Query text input

							layout.Rigid(
								func(gtx C) D {
									return layout.Flex{
										Axis: layout.Horizontal,
									}.Layout(gtx,

										// TODO: sort
										layout.Rigid(
											func(gtx C) D {
												text := "Sort: Random"
												btn := material.Button(th, &sortButton, text)

												return btn.Layout(gtx)
											},
										),
									)
								},
							),
						)
					},
				),

				// Padding
				layout.Rigid(
					layout.Spacer{Height: unit.Dp(5)}.Layout,
				),

				// Grid
				layout.Rigid(
					func(gtx C) D {
						return component.Grid(th, &grid).Layout(gtx, nrows, ncols,
							func(axis layout.Axis, index, constraint int) int {
								return gtx.Dp(unit.Dp(cellSize))
							},
							func(gtx C, row, col int) D {
								// Calculate index
								i := (row * ncols) + col

								// Check if index is in bounds
								if i >= len(clickers) {
									return D{}
								}

								// Stack container
								return layout.Stack{}.Layout(gtx,
									// Album picture
									layout.Stacked(func(gtx C) D {
										return layout.Dimensions{}
									}),

									// Transparent button
									layout.Stacked(func(gtx C) D {
										txt := fmt.Sprintf("%s - %s", albums[i].AlbumArtist, albums[i].Name)
										btn := material.Button(th, &clickers[i], txt)
										btn.Background = colorTransparent
										btn.CornerRadius = 0
										return btn.Layout(gtx)
									}),
								)
							},
						)
					},
				),
			)

			e.Frame(gtx.Ops)
		}
	}
}
