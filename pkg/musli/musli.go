package musli

import (
	"bufio"
	"database/sql"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/dhowden/tag"
	_ "github.com/mattn/go-sqlite3"
	"github.com/mitchellh/go-homedir"
)

type Album struct {
	id          int64
	albumArtist string
	discs       int
	name        string
	year        int
}

type Track struct {
	id      int64
	albumID int64
	disc    int
	path    string
	track   int
}

type Config struct {
	DbFile     string
	MusicDir   string
	ExecCmd    string
	ShowOutput bool
}

var db *sql.DB
var conf Config

func Init(configFile string) {
	homeDir, err := homedir.Dir()
	if err != nil {
		log.Fatal(err)
	}

	conf = Config{ // Default values
		DbFile:     filepath.Join(homeDir, ".musli/library.db"),
		MusicDir:   filepath.Join(homeDir, "Music"),
		ExecCmd:    "mpv",
		ShowOutput: false,
	}

	_, err = toml.DecodeFile(configFile, &conf)
	if err != nil {
		log.Fatal(err)
	}

	db, err = sql.Open("sqlite3", conf.DbFile)
	if err != nil {
		log.Fatal(err)
	}

	/*	OPTIMIZATIONS
		executeQuery(db, `PRAGMA journal_mode = OFF;
						PRAGMA synchronous = 0;
						PRAGMA cache_size = 1000000;
						PRAGMA locking_mode = EXCLUSIVE;
						PRAGMA temp_store = MEMORY;`)
	*/

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS albums(
		id integer PRIMARY KEY,
		album_artist TEXT,
		discs INTEGER,
		name TEXT,
		year INTEGER
	);`)
	if err != nil {
		log.Fatal(err)
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tracks(
		id integer PRIMARY KEY,
		album_id INTEGER,
		disc INTEGER,
		path TEXT,
		track INTEGER
	);`)
	if err != nil {
		log.Fatal(err)
	}
}

func Exit() {
	db.Close()
}

func ScanLibrary() {
	var filenames []string
	err := filepath.Walk(conf.MusicDir, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			log.Fatal(err)
		}

		if !info.IsDir() && hasValidExt(path) {
			filenames = append(filenames, path)
		}

		return nil
	})

	total := len(filenames)
	if total == 0 {
		log.Fatal("No readable files in directory")
	}
	for _, filename := range filenames {

		f, err := os.OpenFile(filename, os.O_RDONLY, 0444)
		if err != nil {
			log.Fatal(err)
		}

		m, err := tag.ReadFrom(f)
		if err != nil {
			log.Fatal(err)
		}

		disc, totalDiscs := m.Disc()

		a := Album{
			albumArtist: m.AlbumArtist(),
			discs:       totalDiscs,
			name:        m.Album(),
			year:        m.Year(),
		}

		if m.Year() == 0 {
			r := m.Raw()
			tdor := (r["TDOR"])
			if tdorStr, ok := tdor.(string); ok {
				yearStr := strings.Split(tdorStr, "-")[0]
				year, err := strconv.Atoi(yearStr)
				if err == nil {
					a.year = year
				}
			}
		}

		albumID, err := findAlbumID(a)
		if err != nil {
			log.Fatal(err)
		}

		if albumID == -1 {
			albumID, err = insertAlbum(a)
			if err != nil {
				log.Fatal(err)
			}
		}

		trackNumber, _ := m.Track()

		t := Track{
			albumID: int64(albumID),
			disc:    disc,
			path:    filename,
			track:   trackNumber,
		}

		trackID, err := findTrackID(filename)
		if err != nil {
			log.Fatal(err)
		}

		if trackID == -1 {
			_, err = insertTrack(t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if err != nil {
		log.Fatal(err)
	}
}

func RandomAlbums() []Album {
	rows, err := db.Query("SELECT * FROM albums ORDER BY RANDOM();")
	if err != nil {
		log.Fatal(err)
	}
	albums := parseAlbumsResults(rows)
	return albums
}

func SearchAlbums(query string, year1 int, year2 int) []Album {
	query = "%" + query + "%"
	var rows *sql.Rows
	var err error
	if year1 > 0 && year2 > 0 {
		rows, err = db.Query(`SELECT * FROM albums WHERE 
						(year BETWEEN ? AND ?) AND
						(name LIKE ? OR album_artist LIKE ?);`, year1, year2, query, query)
	} else if year1 > 0 {
		rows, err = db.Query(`SELECT * FROM albums WHERE 
						(year = ?) AND
						(name LIKE ? OR album_artist LIKE ?);`, year1, query, query)
	} else {
		rows, err = db.Query(`SELECT * FROM albums WHERE 
						name LIKE ? OR album_artist LIKE ?;`, query, query)
	}
	if err != nil {
		log.Fatal(err)
	}
	albums := parseAlbumsResults(rows)
	return albums
}

func ShowAlbums(albums []Album, pageLength int) {
	start := 0
	max := len(albums)
	scanner := bufio.NewScanner(os.Stdin)
	for start < max {
		var albumIDs = make([]Album, pageLength)
		for i := 0; i < pageLength && i < max; i++ {
			pos := start + i
			a := albums[pos]
			albumIDs[i] = a
			fmt.Println("[" + strconv.Itoa(i+1) + "] " + a.albumArtist + " - " + a.name)
		}
		fmt.Print("sel: ")
		scanner.Scan()
		in := scanner.Text()
		i, err := strconv.Atoi(in)
		i -= 1
		if len(in) != 0 && err == nil && i >= 0 && len(albumIDs) > i {
			paths := selectPathsFromTracks(albumIDs[i])
			cmd := exec.Command(conf.ExecCmd, paths...)
			if conf.ShowOutput {
				out, err := cmd.Output()
				if err != nil {
					log.Fatal(err)
				}
				fmt.Printf("%s", out)
			} else {
				err := cmd.Start()
				if err != nil {
					log.Fatal(err)
				}
			}
			break
		}
		start += max
	}
}

func hasValidExt(path string) bool {
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

func insertAlbum(a Album) (int64, error) {
	res, err := db.Exec(`INSERT INTO albums(album_artist,discs,name,year)
						VALUES(?,?,?,?);`, a.albumArtist, a.discs, a.name, a.year)
	if err != nil {
		return -1, err
	}
	albumID, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return albumID, nil
}

func findAlbumID(a Album) (int64, error) {
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

func insertTrack(t Track) (int64, error) {
	res, err := db.Exec(`INSERT INTO tracks(album_id,disc,path,track)
						VALUES(?,?,?,?);`, t.albumID, t.disc, t.path, t.track)
	if err != nil {
		return -1, err
	}
	trackID, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return trackID, nil
}

func findTrackID(path string) (int64, error) {
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

func parseAlbumsResults(rows *sql.Rows) []Album {
	var albums []Album
	for rows.Next() {
		var a Album
		err := rows.Scan(&a.id, &a.albumArtist, &a.discs, &a.name, &a.year)
		if err != nil {
			log.Fatal(err)
		}
		albums = append(albums, a)
	}
	return albums
}

func selectPathsFromTracks(a Album) []string {
	query := `SELECT path FROM tracks
				WHERE album_id = ? ORDER BY track ASC, disc ASC;`
	rows, err := db.Query(query, a.id)
	if err != nil {
		log.Fatal(err)
	}
	var paths []string
	for rows.Next() {
		var p string
		err := rows.Scan(&p)
		if err != nil {
			log.Fatal(err)
		}
		paths = append(paths, p)
	}
	return paths
}
