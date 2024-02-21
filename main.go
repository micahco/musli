package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
	_ "github.com/mattn/go-sqlite3"
)

func main() {
	scanMode := flag.Bool("s", false, "Scan library")
	randMode := flag.Bool("r", false, "Query random")
	flag.Parse()

	db, err := sql.Open("sqlite3", "musli.db")
	if err != nil {
		log.Fatal(err)
	}

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
		fmt.Println("Scanning library")
		scanLibrary(db)
		fmt.Println("Scan complete")
	}

	if *randMode {
		rows := selectRandomAlbums(db)
		var albums []Album
		for rows.Next() {
			var i int
			var r Album
			err := rows.Scan(&i, &r.albumArtist, &r.discs, &r.name, &r.year)
			if err != nil {
				log.Fatal(err)
			}
			albums = append(albums, r)
		}
		for count, album := range albums {
			if count > 10 {
				return
			}
			fmt.Println(album.name)
		}
	}

	db.Close()
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
	for count, filename := range filenames {
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

		albumId, err := selectAlbum(db, a)
		if err != nil {
			log.Fatal(err)
		}

		if albumId == -1 {
			albumId, err = insertAlbum(db, a)
			if err != nil {
				log.Fatal(err)
			}
		}

		trackNumber, _ := m.Track()

		t := Track{
			albumId: int(albumId),
			disc:    disc,
			path:    filename,
			track:   trackNumber,
		}

		trackId, err := selectTrack(db, filename)
		if err != nil {
			log.Fatal(err)
		}

		if trackId == -1 {
			trackId, err = insertTrack(db, t)
			if err != nil {
				log.Fatal(err)
			}
		}

		fmt.Println(count)
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

type Album struct {
	albumArtist string
	discs       int
	name        string
	year        int
}

type Track struct {
	albumId int
	disc    int
	path    string
	track   int
}

func insertAlbum(db *sql.DB, a Album) (int64, error) {
	res, err := db.Exec(`INSERT INTO albums(album_artist,discs,name,year)
						VALUES(?,?,?,?);`, a.albumArtist, a.discs, a.name, a.year)
	if err != nil {
		return -1, err
	}
	albumId, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return albumId, nil
}

func selectAlbum(db *sql.DB, a Album) (int64, error) {
	query := `SELECT id FROM albums
			WHERE album_artist = ? AND name = ? AND year = ?`
	row := db.QueryRow(query, a.albumArtist, a.name, a.year)
	var albumId int64
	err := row.Scan(&albumId)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return albumId, err
}

func insertTrack(db *sql.DB, t Track) (int64, error) {
	res, err := db.Exec(`INSERT INTO tracks(album_id,disc,path,track)
						VALUES(?,?,?,?);`, t.albumId, t.disc, t.path, t.track)
	if err != nil {
		return -1, err
	}
	trackId, err := res.LastInsertId()
	if err != nil {
		return -1, err
	}
	return trackId, nil
}

func selectTrack(db *sql.DB, path string) (int64, error) {
	query := `SELECT id FROM tracks
			WHERE path = ?`
	row := db.QueryRow(query, path)
	var trackId int64
	err := row.Scan(&trackId)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return trackId, err
}

func selectRandomAlbums(db *sql.DB) *sql.Rows {
	rows, err := db.Query("SELECT * FROM albums ORDER BY RANDOM()")
	if err != nil {
		log.Fatal(err)
	}
	return rows
}

func executeQuery(db *sql.DB, query string) {
	_, err := db.Exec(query)
	if err != nil {
		log.Fatal(err)
	}
}
