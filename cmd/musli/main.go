package main

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/micahco/musli"
)

const APP_NAME = "musli"
const ESCAPE = "\033"

func main() {
	exitCode := 0
	defer func() {
		os.Exit(exitCode)
	}()

	err := root(os.Args[1:])
	if err != nil {
		fmt.Println(APP_NAME + ": " + err.Error())
		exitCode = 1
	}
}

func root(args []string) error {
	conf, err := loadConfig()
	if err != nil {
		return err
	}

	db, err := loadDB()
	if err != nil {
		return err
	}
	defer musli.CloseDB(db)

	if len(args) == 0 {
		m, err := initialModel(conf, db)
		if err != nil {
			return err
		}
		p := tea.NewProgram(m)
		_, err = p.Run()
		return err
	}

	switch arg := args[0]; arg {
	case "-h", "--help":
		printUsage()
	case "-r", "--random":
		// play random album
	case "-s", "--scan":
		err = execScan(conf, db)
	case "-t", "--tidy":
		err = execTidy(db)
	default:
		return fmt.Errorf("invalid option: '%s'", arg)
	}
	if err != nil {
		return err
	}
	return nil
}

func printUsage() {
	fmt.Printf("Usage of %s:", APP_NAME)
	fmt.Println(`
-s, --scan: scan music directory for new files
-t, --tidy: scrub library for entries that no longer exist`)
}

func getAppPath(filename string) (string, error) {
	home, err := os.UserHomeDir()
	return filepath.Join(home, ".musli", filename), err
}

func loadConfig() (*musli.Config, error) {
	path, err := getAppPath("config.toml")
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
	path, err := getAppPath("library.db")
	if err != nil {
		return nil, err
	}

	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return nil, err
	}

	db, err := musli.OpenDB(path)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func clearLine(a ...any) {
	fmt.Printf("%s[1A%s[K", ESCAPE, ESCAPE)
	fmt.Println(a...)
}

func execScan(conf *musli.Config, db *sql.DB) error {
	fmt.Println("Scanning directory")
	paths, err := musli.GetMusicDirPaths(conf)
	if err != nil {
		return err
	}
	total := len(paths)

	for i, path := range paths {
		err = musli.AddPathToLibrary(path, db)
		clearLine(i, "/", total)
		if err != nil {
			return err
		}
	}

	clearLine("Scanned", total, "files")
	return nil
}

func execTidy(db *sql.DB) error {
	fmt.Println("Scrubbing library")
	paths, err := musli.FetchTrackPaths(db)
	if err != nil {
		return err
	}
	total := len(paths)

	for i, path := range paths {
		err = musli.RemoveNotExistPath(path, db)
		clearLine(i, "/", total)
		if err != nil {
			return err
		}
	}

	clearLine("Cleaning up")
	err = musli.RemoveEmptyAlbums(db)
	if err != nil {
		return err
	}
	clearLine("Scrubbed", total, "files")
	return nil
}

var sortMethods = [3]string{"random", "artist", "year"}

const (
	sortMethodRandom = iota
	sortMethodArtist
	sortMethodYear
)

func fetchAlbums(db *sql.DB, sortMethod int, asc bool) ([]musli.Album, error) {
	var albums []musli.Album
	var err error
	switch sortMethod {
	case sortMethodRandom:
		albums, err = musli.FetchAlbumsByRandom(db)
	case sortMethodArtist:
		albums, err = musli.FetchAlbumsByAlbumArtist(asc, db)
	case sortMethodYear:
		albums, err = musli.FetchAlbumsByYear(asc, db)
	}
	return albums, err
}

type model struct {
	albums     []musli.Album
	conf       *musli.Config
	db         *sql.DB
	filter     bool
	sortMethod int
	sortAsc    bool // true = ascending; false = descending
	query      string
	start      int // page start index
	cursor     int // page cursor index
}

var style = lipgloss.NewStyle().Foreground(lipgloss.Color("#ff0000"))

func initialModel(conf *musli.Config, db *sql.DB) (*model, error) {
	albums, err := musli.FetchAlbumsByRandom(db)
	if err != nil {
		return nil, err
	}
	m := &model{
		albums:     albums,
		conf:       conf,
		db:         db,
		filter:     false,
		sortMethod: 0,
		sortAsc:    true,
		query:      "",
		start:      0,
		cursor:     0,
	}
	return m, nil
}

func (m *model) Init() tea.Cmd {
	return nil
}

func (m *model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		key := msg.String()
		if key == "ctrl+c" {
			return m, tea.Quit
		}
		var cmd tea.Cmd
		var err error
		if m.filter {
			cmd, err = m.controllerFilter(key)
		} else {
			cmd, err = m.controllerMain(key)
		}
		if err != nil {
			fmt.Println(err.Error())
		}
		if cmd != nil {
			return m, cmd
		}
	}
	if m.filter && len(m.query) > 0 {
		m.start = 0
		albums, err := musli.FetchAlbumsByQuery(m.query, m.db)
		if err != nil {
			fmt.Println(err.Error())
		}
		m.albums = albums
	}
	return m, nil
}

func (m *model) View() string {
	s := ""
	if m.filter {
		s += "Query: " + m.query + "\n"
	}
	for i := m.start; i <= m.start+m.conf.PageLength && i < len(m.albums); i++ {
		a := m.albums[i]
		t := m.conf.ListTemplate
		t = strings.Replace(t, "%album%", a.Name, -1)
		t = strings.Replace(t, "%artist%", a.AlbumArtist, -1)
		t = strings.Replace(t, "%year%", strconv.Itoa(a.Year), -1)
		if m.start+m.cursor == i {
			t = style.Render(t)
		}
		s += t + "\n"
	}
	if !m.filter && len(m.query) == 0 {
		s += "\nsort: " + sortMethods[m.sortMethod]
		if m.sortMethod != sortMethodRandom {
			s += "\t order: "
			if m.sortAsc {
				s += "ascending"
			} else {
				s += "descending"
			}
		}
	}
	return s
}

func (m *model) controllerMain(key string) (tea.Cmd, error) {
	switch key {
	case "q":
		return tea.Quit, nil
	case "left", "h":
		m.moveLeft()
	case "up", "k":
		m.moveUp()
	case "down", "j":
		m.moveDown()
	case "right", "l":
		m.moveRight()
	case "o":
		if len(m.query) == 0 && m.sortMethod != sortMethodRandom {
			m.sortAsc = !m.sortAsc
			albums, err := fetchAlbums(m.db, m.sortMethod, m.sortAsc)
			if err != nil {
				return nil, err
			}
			m.albums = albums
		}
	case "s":
		if len(m.query) == 0 {
			err := m.toggleSortMethod()
			if err != nil {
				return nil, err
			}
		}
	case "/":
		m.cursor = m.start
		m.filter = true
	case "enter", " ":
		album := m.albums[m.cursor]
		err := musli.PlayAlbum(album.ID, m.conf, m.db)
		if err != nil {
			return nil, err
		}
		return tea.Quit, nil
	}
	return nil, nil
}

func (m *model) controllerFilter(key string) (tea.Cmd, error) {
	switch key {
	case "backspace":
		if len(m.query) > 0 {
			// remove last character from query
			m.query = m.query[:len(m.query)-1]
		}
	case "enter":
		m.filter = false
		m.start = 0
		m.cursor = 0
	case "esc":
		m.filter = false
		m.query = ""
		albums, err := fetchAlbums(m.db, m.sortMethod, m.sortAsc)
		if err != nil {
			return nil, err
		}
		m.albums = albums
	default:
		if len(key) == 1 {
			m.query += key
		}
	}
	return nil, nil
}

func (m *model) toggleSortMethod() error {
	m.sortMethod++
	if m.sortMethod >= len(sortMethods) {
		m.sortMethod = 0
	}
	albums, err := fetchAlbums(m.db, m.sortMethod, m.sortAsc)
	if err != nil {
		return err
	}
	m.albums = albums
	return nil
}

func (m *model) moveLeft() {
	m.start -= m.conf.PageLength
	if m.start < 0 {
		m.start = 0
	}
}

func (m *model) moveUp() {
	if m.cursor > 0 {
		m.cursor--
	}
}

func (m *model) moveDown() {
	if m.cursor < m.conf.PageLength && m.start+m.cursor < len(m.albums)-1 {
		m.cursor++
	}
}

func (m *model) moveRight() {
	if m.start+m.conf.PageLength < len(m.albums)-1 {
		m.start += m.conf.PageLength
	}
}
