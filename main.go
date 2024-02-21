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

	sqlite(db, `CREATE TABLE IF NOT EXISTS albums(
					id integer PRIMARY KEY,
					album_artist TEXT NOT NULL,
					name TEXT NOT NULL,
					path TEXT NOT NULL,
					year INTEGER
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
	for _, filename := range filenames {
		f, err := os.OpenFile(filename, os.O_RDONLY, 0444)
		if err != nil {
			log.Fatal(err)
		}
		m, err := tag.ReadFrom(f)
		if err != nil {
			log.Fatal(err)
		}
		a := Album{
			albumArtist: m.AlbumArtist(),
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

		fmt.Println(albumId)
	}
	if err != nil {
		log.Fatal(err)
	}
}

type Album struct {
	albumArtist string
	name        string
	path        string
	year        int
}

func insertAlbum(db *sql.DB, a Album) (int64, error) {
	res, err := db.Exec(`INSERT INTO albums(album_artist,name,path,year)
						VALUES(?,?,?);`, a.albumArtist, a.name, a.path, a.year)
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
			WHERE album_artist = ? AND name = ? p AND year = ?`
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
		log.Fatal(err)
	}
}
