package musli

import (
	"bufio"
	"database/sql"
	"errors"
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
	_ "github.com/mattn/go-sqlite3"
)

type Album struct {
	ID          int64
	AlbumArtist string
	Name        string
	Year        int
}

type Track struct {
	AlbumID     int64
	Disc        int
	Path        string
	TrackNumber int
}

type Config struct {
	MusicDir     string
	ExecCmd      string
	ListTemplate string
	HiglightSGR  []int
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

func GetLibraryPath() (string, error) {
	appDir, err := GetAppDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(appDir, "library.db"), nil
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
		HiglightSGR:  []int{1},
		PageLength:   10,
		ShowStdout:   false,
		ShowStderr:   false,
	}

	_, err = toml.DecodeFile(configFile, &conf)
	if err != nil {
		return nil, nil, err
	}

	libraryPath, err := GetLibraryPath()
	if err != nil {
		return nil, nil, err
	}

	db, err := sql.Open("sqlite3", libraryPath)
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

func WalkLibrary(conf *Config) ([]string, error) {
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
	return filenames, err
}

func ScanLibrary(filenames []string, db *sql.DB, counter func(count int)) error {
	for i, filename := range filenames {
		counter(i)

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
			AlbumArtist: m.AlbumArtist(),
			Name:        m.Album(),
			Year:        m.Year(),
		}
		if m.Year() == 0 {
			a.Year = readAltYearMetadata(m)
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
			AlbumID:     int64(albumID),
			Disc:        disc,
			Path:        filename,
			TrackNumber: trackNumber,
		}

		_, err = insertTrack(t, db)
		if err != nil {
			return err
		}
	}
	return nil
}

func CleanLibrary(conf *Config, db *sql.DB) error {
	paths, err := getAllTrackPaths(db)
	if err != nil {
		return err
	}

	for _, p := range paths {
		_, err := os.Stat(p)
		if errors.Is(err, os.ErrNotExist) {
			_, err := db.Exec(`DELETE FROM tracks WHERE path = ?`, p)
			if err != nil {
				return err
			}
		}
	}

	albumIDs, err := GetAllAlbumIDs(db)
	if err != nil {
		return err
	}

	for _, id := range albumIDs {
		a := Album{ID: id}
		t, err := GetAlbumTrackPaths(a.ID, db)
		if err != nil {
			return err
		}
		if len(t) < 1 {
			_, err := db.Exec(`DELETE FROM albums WHERE id = ?`, id)
			if err != nil {
				return err
			}
		}
	}

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

func GetAlbums(db *sql.DB) ([]Album, error) {
	rows, err := db.Query("SELECT * FROM albums;")
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

type YearQueryError struct {
	Query string
}

func (err *YearQueryError) Error() string {
	return fmt.Sprintf("musli: invalid year query: %s", err.Query)
}

func FindAlbumsByYear(query string, db *sql.DB) ([]Album, error) {
	var rows *sql.Rows
	var err error

	if len(query) != 4 && len(query) != 9 {
		return nil, &YearQueryError{query}
	}

	s := strings.Split(query, "-")
	a1, _ := strconv.Atoi(s[0])
	if a1 == 0 {
		return nil, &YearQueryError{s[0]}
	}

	if len(s) > 1 {
		a2, _ := strconv.Atoi(s[1])
		if a2 == 0 {
			return nil, &YearQueryError{s[1]}
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

func PlayAlbum(albumID int64, conf *Config, db *sql.DB) error {
	paths, err := GetAlbumTrackPaths(albumID, db)
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
						VALUES(?,?,?);`, a.AlbumArtist, a.Name, a.Year)
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
	row := db.QueryRow(query, a.AlbumArtist, a.Name, a.Year)
	var albumID int64
	err := row.Scan(&albumID)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return albumID, err
}

func insertTrack(t Track, db *sql.DB) (int64, error) {
	res, err := db.Exec(`INSERT INTO tracks(album_id,disc,path,track_number)
						VALUES(?,?,?,?);`, t.AlbumID, t.Disc, t.Path, t.TrackNumber)
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

func GetAlbumTrackPaths(albumID int64, db *sql.DB) ([]string, error) {
	query := `SELECT path FROM tracks
			WHERE album_id = ?
			ORDER BY track_number ASC, disc ASC;`
	rows, err := db.Query(query, albumID)
	if err != nil {
		return nil, err
	}

	paths, err := parseRowsToTrackPaths(rows)
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func getAllTrackPaths(db *sql.DB) ([]string, error) {
	query := `SELECT path FROM tracks`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	paths, err := parseRowsToTrackPaths(rows)
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func GetAllAlbumIDs(db *sql.DB) ([]int64, error) {
	query := `SELECT id FROM albums`
	rows, err := db.Query(query)
	if err != nil {
		return nil, err
	}

	albumIDs, err := parseRowsToAlbumIDs(rows)
	if err != nil {
		return nil, err
	}

	return albumIDs, nil
}

func parseRowsToAlbums(rows *sql.Rows) ([]Album, error) {
	var albums []Album
	for rows.Next() {
		var a Album
		err := rows.Scan(&a.ID, &a.AlbumArtist, &a.Name, &a.Year)
		if err != nil {
			return nil, err
		}
		albums = append(albums, a)
	}
	return albums, nil
}

func parseRowsToAlbumIDs(rows *sql.Rows) ([]int64, error) {
	var albumIDs []int64
	for rows.Next() {
		var a int64
		err := rows.Scan(&a)
		if err != nil {
			return nil, err
		}
		albumIDs = append(albumIDs, a)
	}
	return albumIDs, nil
}

func parseRowsToTrackPaths(rows *sql.Rows) ([]string, error) {
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
