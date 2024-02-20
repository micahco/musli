package main

import (
	"database/sql"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
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
		panic(err)
	}

	sqlite(db, `CREATE TABLE IF NOT EXISTS albums(
					id integer PRIMARY KEY,
					album_artist TEXT,
					name TEXT NOT NULL,
					year INTEGER
				);`)

	sqlite(db, `CREATE TABLE IF NOT EXISTS tracks(
					id integer PRIMARY KEY,
					album_id INTEGER NOT NULL,
					track INTEGER NOT NULL,
					path TEXT NOT NULL
				);`)

	if *scanMode {
		fmt.Println("Scanning library")
		scanLibrary(db)
		fmt.Println("Scan complete")
	}

	if *randMode {

	}

	db.Close()
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

func scanLibrary(db *sql.DB) {
	err := filepath.Walk("/home/micah/Music", func(path string, info fs.FileInfo, err error) error {
		if !info.IsDir() && hasValidExt(path) {
			f, err := os.OpenFile(path, os.O_RDONLY, 0444)
			if err != nil {
				panic(err)
			}
			t, err := tag.ReadFrom(f)
			if err != nil {
				panic(err)
			}
			a := Album{
				albumArtist: t.AlbumArtist(),
				name:        t.Album(),
				year:        t.Year(),
			}

			albumId, err := selectAlbum(db, a)
			if err != nil {
				panic(err)
			}
			if albumId == -1 {
				albumId, err = insertAlbum(db, a)
				if err != nil {
					panic(err)
				}
			}
			fmt.Println(albumId)
		}
		return nil
	})
	if err != nil {
		panic(err)
	}
}

type Album struct {
	albumArtist string
	name        string
	year        int
}

func insertAlbum(db *sql.DB, a Album) (int64, error) {
	res, err := db.Exec(`INSERT INTO albums(album_artist,name,year)
						VALUES(?,?,?);`, a.albumArtist, a.name, a.year)
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

func sqlite(db *sql.DB, query string) {
	_, err := db.Exec(query)
	if err != nil {
		panic(err)
	}
}
