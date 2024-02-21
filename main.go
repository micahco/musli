package main

import (
	"bufio"
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
	_ "github.com/mattn/go-sqlite3"
	"github.com/schollz/progressbar/v3"
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
	albumID int
	disc    int
	path    string
	track   int
}

func main() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	scanMode := flag.Bool("s", false, "Scan library")
	randMode := flag.Bool("r", false, "Query random")
	flag.Parse()

	db, err := sql.Open("sqlite3", "musli.db")
	if err != nil {
		log.Fatal(err)
	}

	executeQuery(db, `PRAGMA journal_mode = OFF;
					PRAGMA synchronous = 0;
					PRAGMA cache_size = 1000000;
					PRAGMA locking_mode = EXCLUSIVE;
					PRAGMA temp_store = MEMORY;`)

	executeQuery(db, `CREATE TABLE IF NOT EXISTS albums(
					id integer PRIMARY KEY,
					album_artist TEXT,
					discs INTEGER,
					name TEXT,
					year INTEGER
				);`)

	executeQuery(db, `CREATE TABLE IF NOT EXISTS tracks(
					id integer PRIMARY KEY,
					album_id INTEGER,
					disc INTEGER,
					path TEXT,
					track INTEGER
				);`)

	if *scanMode {
		clear()
		fmt.Println("Scanning library\n")
		scanLibrary(db)
		fmt.Println("\nScan complete")
	}

	if *randMode {
		albums := randomAlbums(db)
		start := 0
		size := 9
		scanner := bufio.NewScanner(os.Stdin)
		for {
			clear()
			var m = make([]Album, size)
			for i := 0; i < size; i++ {
				pos := start + i
				a := albums[pos]
				m[i] = a
				fmt.Println("[" + strconv.Itoa(i+1) + "] " + a.albumArtist + " - " + a.name)
			}
			fmt.Print("Choose album: ")
			scanner.Scan()
			text := scanner.Text()
			i, err := strconv.Atoi(text)
			if len(text) != 0 && err == nil && i >= 0 && len(m) > i-1 {
				clear()
				t := albumTracks(db, m[i-1])
				playTracks(t)
				break
			}
			start += size
		}
	}

	db.Close()
}

func clear() {
	cmd := exec.Command("clear")
	cmd.Stdout = os.Stdout
	cmd.Run()
}

func scanLibrary(db *sql.DB) {
	var filenames []string
	err := filepath.Walk("/nfs/m600/music", func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			fmt.Println(err)
			return nil
		}

		if !info.IsDir() && hasValidExt(path) {
			filenames = append(filenames, path)
		}

		return nil
	})

	bar := progressbar.Default(int64(len(filenames)))
	for _, filename := range filenames {
		bar.Add(1)

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

		albumID, err := findalbumID(db, a)
		if err != nil {
			log.Fatal(err)
		}

		if albumID == -1 {
			albumID, err = insertAlbum(db, a)
			if err != nil {
				log.Fatal(err)
			}
		}

		trackNumber, _ := m.Track()

		t := Track{
			albumID: int(albumID),
			disc:    disc,
			path:    filename,
			track:   trackNumber,
		}

		trackID, err := findTrackID(db, filename)
		if err != nil {
			log.Fatal(err)
		}

		if trackID == -1 {
			_, err = insertTrack(db, t)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
	if err != nil {
		log.Fatal(err)
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

func executeQuery(db *sql.DB, query string) {
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}

func insertAlbum(db *sql.DB, a Album) (int64, error) {
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

func findalbumID(db *sql.DB, a Album) (int64, error) {
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

func insertTrack(db *sql.DB, t Track) (int64, error) {
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

func findTrackID(db *sql.DB, path string) (int64, error) {
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

func randomAlbums(db *sql.DB) []Album {
	rows, err := db.Query("SELECT * FROM albums ORDER BY RANDOM();")
	if err != nil {
		log.Fatal(err)
	}
	var albums []Album
	for rows.Next() {
		var r Album
		err := rows.Scan(&r.id, &r.albumArtist, &r.discs, &r.name, &r.year)
		if err != nil {
			log.Fatal(err)
		}
		albums = append(albums, r)
	}
	return albums
}

func albumTracks(db *sql.DB, a Album) []Track {
	query := `SELECT * FROM tracks
			WHERE album_id = ? ORDER BY track ASC, disc ASC;`
	rows, err := db.Query(query, a.id)
	if err != nil {
		log.Fatal(err)
	}
	var tracks []Track
	for rows.Next() {
		var t Track
		err := rows.Scan(&t.id, &t.albumID, &t.disc, &t.path, &t.track)
		if err != nil {
			log.Fatal(err)
		}
		tracks = append(tracks, t)
	}
	return tracks
}

func playTracks(tracks []Track) {
	var paths string
	for _, t := range tracks {
		paths += t.path + "\n"
	}

	cmd := exec.Command("mpv", "--playlist=-", "&")
	cmd.Stdin = strings.NewReader(paths)

	err := cmd.Start()
	if err != nil {
		log.Fatal(err)
	}
}
