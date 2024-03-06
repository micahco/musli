package musli

import (
	"bufio"
	"database/sql"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/dhowden/tag"
	"github.com/eiannone/keyboard"
	_ "github.com/mattn/go-sqlite3"
)

type Album struct {
	id          int64
	albumArtist string
	name        string
	year        int
}

type Track struct {
	albumID     int64
	disc        int
	path        string
	trackNumber int
}

type Config struct {
	MusicDir     string
	ExecCmd      string
	ListTemplate string
	SelectColor  int
	PageLength   int
	ShowStdout   bool
	ShowStderr   bool
}

func GetDefaultConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(configDir, "musli", "config.toml")
	return path, nil
}

func GetAppDir() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	var appDir string
	appName := "musli"
	if runtime.GOOS == "linux" {
		xdgUserDir := os.Getenv("XDG_STATE_HOME")
		if xdgUserDir != "" {
			appDir = filepath.Join(xdgUserDir, appName)
		} else {
			appDir = filepath.Join(homeDir, ".state", appName)
		}
	} else {
		appDir = filepath.Join(homeDir, "."+appName)
	}

	err = os.MkdirAll(appDir, os.ModePerm)
	if err != nil {
		return "", err
	}

	return appDir, nil
}

func Init(configFile string) (*Config, *sql.DB, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, nil, err
	}

	conf := Config{ // Default values
		MusicDir:     filepath.Join(homeDir, "Music"),
		ExecCmd:      "mpv",
		ListTemplate: "%artist% - %album%",
		SelectColor:  3,
		PageLength:   10,
		ShowStdout:   false,
		ShowStderr:   false,
	}

	_, err = toml.DecodeFile(configFile, &conf)
	if err != nil {
		return nil, nil, err
	}

	appDir, err := GetAppDir()
	if err != nil {
		return nil, nil, err
	}

	db, err := sql.Open("sqlite3", filepath.Join(appDir, "library.db"))
	if err != nil {
		return nil, nil, err
	}

	_, err = db.Exec(`PRAGMA journal_mode = wal;
					PRAGMA synchronous = normal;
					PRAGMA foreign_keys = on;`)
	if err != nil {
		return nil, nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS albums(
						id integer PRIMARY KEY,
						album_artist TEXT,
						name TEXT,
						year INTEGER
					);`)
	if err != nil {
		return nil, nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tracks(
						id integer PRIMARY KEY,
						album_id INTEGER,
						disc INTEGER,
						path TEXT,
						track_number INTEGER
					);`)
	if err != nil {
		return nil, nil, err
	}

	return &conf, db, nil
}

func ScanLibraryToDB(conf *Config, db *sql.DB) error {
	var filenames []string
	err := filepath.Walk(conf.MusicDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() && isValidFileType(path) {
			filenames = append(filenames, path)
		}

		return nil
	})

	total := len(filenames)
	if total == 0 {
		return nil
	}
	for i, filename := range filenames {
		fmt.Print("\033[2K\r", i, "/", total)

		trackID, err := findTrackID(filename, db)
		if err != nil {
			return err
		}
		if trackID != -1 {
			continue // path already in db
		}

		f, err := os.OpenFile(filename, os.O_RDONLY, 0444)
		if err != nil {
			return err
		}

		m, err := tag.ReadFrom(f)
		if err != nil {
			return err
		}

		a := Album{
			albumArtist: m.AlbumArtist(),
			name:        m.Album(),
			year:        m.Year(),
		}
		if m.Year() == 0 {
			a.year = readAltYearMetadata(m)
		}

		albumID, err := findAlbumID(a, db)
		if err != nil {
			return err
		}
		if albumID == -1 {
			albumID, err = insertAlbum(a, db)
			if err != nil {
				return err
			}
		}

		disc, _ := m.Disc()
		trackNumber, _ := m.Track()
		t := Track{
			albumID:     int64(albumID),
			disc:        disc,
			path:        filename,
			trackNumber: trackNumber,
		}

		_, err = insertTrack(t, db)
		if err != nil {
			return err
		}
	}
	if err != nil {
		return err
	}
	fmt.Println("\033[2K\rScanned", total, "files")
	return nil
}

func RandomAlbums(db *sql.DB) ([]Album, error) {
	rows, err := db.Query("SELECT * FROM albums ORDER BY RANDOM();")
	if err != nil {
		return nil, err
	}

	albums, err := parseRowsToAlbums(rows)
	if err != nil {
		return nil, err
	}

	return albums, nil
}

func FindAlbumsByNameOrAlbumArtist(query string, db *sql.DB) ([]Album, error) {
	a := "%" + query + "%"
	rows, err := db.Query(`SELECT * FROM albums WHERE
						name LIKE ? OR album_artist LIKE ?
						ORDER BY album_artist ASC, name ASC;`, a, a)
	if err != nil {
		return nil, err
	}

	albums, err := parseRowsToAlbums(rows)
	if err != nil {
		return nil, err
	}

	return albums, nil
}

func FindAlbumsByYear(query string, db *sql.DB) ([]Album, error) {
	var rows *sql.Rows
	var err error

	if len(query) != 4 && len(query) != 9 {
		return nil, fmt.Errorf("invalid query: %s", query)
	}

	s := strings.Split(query, "-")
	a1, _ := strconv.Atoi(s[0])
	if a1 == 0 {
		return nil, fmt.Errorf("invalid query: %s", s[0])
	}

	if len(s) > 1 {
		a2, _ := strconv.Atoi(s[1])
		if a2 == 0 {
			return nil, fmt.Errorf("invalid query: %s", s[1])
		}

		rows, err = db.Query(`SELECT * FROM albums WHERE
							year BETWEEN ? AND ?
							ORDER BY year ASC, album_artist ASC, name ASC;`, a1, a2)
	} else {
		rows, err = db.Query(`SELECT * FROM albums WHERE
							year = ?
							ORDER BY album_artist ASC, name ASC;`, a1)
	}

	if err != nil {
		return nil, err
	}

	albums, err := parseRowsToAlbums(rows)
	if err != nil {
		return nil, err
	}

	return albums, nil
}

func CloseDB(db *sql.DB) error {
	if db == nil {
		return nil
	}

	_, err := db.Exec(`PRAGMA analysis_limit=400;
					PRAGMA optimize;`)
	if err != nil {
		return err
	}

	err = db.Close()
	if err != nil {
		return err
	}

	return nil
}

func ListAlbums(albums []Album, conf *Config, db *sql.DB) error {
	pageLength := conf.PageLength
	start := 0
	sel := 0
	max := len(albums)

	hideCursor()

	if err := keyboard.Open(); err != nil {
		panic(err)
	}
	defer func() {
		clearScreen()
		showCursor()
		_ = keyboard.Close()
	}()

	for {
		clearScreen()
		if start < 0 {
			start = 0
		}
		if sel < 0 || start+sel >= max {
			sel = 0
		}
		var pageAlbums []Album
		for i := 0; i < pageLength && start+i < len(albums); i++ {
			pos := start + i
			a := albums[pos]
			pageAlbums = append(pageAlbums, a)
			l := conf.ListTemplate
			l = strings.Replace(l, "%album%", a.name, -1)
			l = strings.Replace(l, "%artist%", a.albumArtist, -1)
			l = strings.Replace(l, "%year%", strconv.Itoa(a.year), -1)
			if sel == i {
				l = highlight(l, conf.SelectColor)
			}
			fmt.Println(l)
		}

		char, key, err := keyboard.GetKey()
		if err != nil {
			return err
		}
		if key == keyboard.KeySpace || key == keyboard.KeyEnter {
			err := PlayAlbum(pageAlbums[sel], conf, db)
			if err != nil {
				return err
			}
			return nil
		}
		if (char == 'j' || key == keyboard.KeyArrowDown) && sel < pageLength-1 && start+sel < len(albums)-1 {
			sel += 1
			continue
		}
		if (char == 'k' || key == keyboard.KeyArrowUp) && sel > 0 {
			sel -= 1
			continue
		}
		if (char == 'h' || key == keyboard.KeyArrowLeft) && start-pageLength >= 0 {
			start -= pageLength
			continue
		}
		if (char == 'l' || key == keyboard.KeyArrowRight) && start+pageLength <= max {
			start += pageLength
			continue
		}
		if char == 'q' || key == keyboard.KeyCtrlC || key == keyboard.KeyCtrlD {
			break
		}
	}
	return nil
}

func PlayAlbum(a Album, conf *Config, db *sql.DB) error {
	paths, err := selectPathsFromTracks(a, db)
	if err != nil {
		return err
	}
	cmd := exec.Command(conf.ExecCmd, paths...)
	if conf.ShowStdout {
		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return err
		}
		err = startCmdWithOutput(cmd, stdout)
		if err != nil {
			return err
		}
	} else if conf.ShowStderr {
		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}
		err = startCmdWithOutput(cmd, stderr)
		if err != nil {
			return err
		}
	} else {
		err := cmd.Start()
		if err != nil {
			return err
		}
	}
	return nil
}

func startCmdWithOutput(cmd *exec.Cmd, r io.ReadCloser) error {
	err := cmd.Start()
	if err != nil {
		return err
	}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		m := scanner.Text()
		fmt.Println(m)
	}
	err = cmd.Wait()
	if err != nil {
		return err
	}
	return nil
}

func hideCursor() {
	fmt.Print("\x1b[?25l")
}

func showCursor() {
	fmt.Print("\x1b[?25h")
}

func clearScreen() {
	fmt.Print("\033[H\033[2J")
}

func colorOutput(c int) string {
	escape := "\x1b"
	if c == 0 {
		return fmt.Sprintf("%s[%dm", escape, c)
	}

	return fmt.Sprintf("%s[3%dm", escape, c)
}

func highlight(text string, color int) string {
	if runtime.GOOS == "windows" || color == 0 {
		return "> " + text
	}
	return colorOutput(color) + text + colorOutput(0)
}

func readAltYearMetadata(m tag.Metadata) int {
	// https://eyed3.readthedocs.io/en/latest/compliance.html
	r := m.Raw()
	tdor := (r["TDOR"]) // ID3 v2.4 orig release date
	if tdorStr, ok := tdor.(string); ok {
		yearStr := strings.Split(tdorStr, "-")[0]
		year, err := strconv.Atoi(yearStr)
		if err == nil {
			return year
		}
	}
	tdrl := (r["TDRL"]) // ID3 v2.4 release date
	if tdrlStr, ok := tdrl.(string); ok {
		fmt.Println("TDRL", tdrlStr)
	}
	xdor := (r["XDOR"]) // ID3 v2.3 orig release year
	if xdorStr, ok := xdor.(string); ok {
		fmt.Println("XDOR", xdorStr)
	}
	tory := (r["TORY"]) // ID3 v2.3 orig release year
	if toryStr, ok := tory.(string); ok {
		fmt.Println("TORY", toryStr)
	}
	return 0
}

func isValidFileType(path string) bool {
	ext := filepath.Ext(path)
	switch strings.ToUpper(ext) {
	case
		".MP3",
		".M4A",
		".M4B",
		".M4P",
		".ALAC",
		".FLAC",
		".OGG",
		".DSF":
		return true
	}
	return false
}

func insertAlbum(a Album, db *sql.DB) (int64, error) {
	res, err := db.Exec(`INSERT INTO albums(album_artist,name,year)
						VALUES(?,?,?);`, a.albumArtist, a.name, a.year)
	if err != nil {
		return -1, err
	}
	albumID, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return albumID, nil
}

func findAlbumID(a Album, db *sql.DB) (int64, error) {
	query := `SELECT id FROM albums
			WHERE album_artist = ? AND name = ? AND year = ?;`
	row := db.QueryRow(query, a.albumArtist, a.name, a.year)
	var albumID int64
	err := row.Scan(&albumID)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return albumID, err
}

func insertTrack(t Track, db *sql.DB) (int64, error) {
	res, err := db.Exec(`INSERT INTO tracks(album_id,disc,path,track_number)
						VALUES(?,?,?,?);`, t.albumID, t.disc, t.path, t.trackNumber)
	if err != nil {
		return -1, err
	}
	trackID, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return trackID, nil
}

func findTrackID(path string, db *sql.DB) (int64, error) {
	query := `SELECT id FROM tracks
			WHERE path = ?;`
	row := db.QueryRow(query, path)
	var trackID int64
	err := row.Scan(&trackID)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return trackID, err
}

func parseRowsToAlbums(rows *sql.Rows) ([]Album, error) {
	var albums []Album
	for rows.Next() {
		var a Album
		err := rows.Scan(&a.id, &a.albumArtist, &a.name, &a.year)
		if err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, nil
}

func selectPathsFromTracks(a Album, db *sql.DB) ([]string, error) {
	query := `SELECT path FROM tracks
			WHERE album_id = ?
			ORDER BY track_number ASC, disc ASC;`
	rows, err := db.Query(query, a.id)
	if err != nil {
		return nil, err
	}
	var paths []string
	for rows.Next() {
		var p string
		err := rows.Scan(&p)
		if err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}
