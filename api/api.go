package api

import (
	"database/sql"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dhowden/tag"
	_ "github.com/mattn/go-sqlite3"
)

type API struct {
	db *sql.DB
}

type Album struct {
	ID          int64
	AlbumArtist string
	Name        string
	Year        int
}

type Track struct {
	AlbumID     int64
	Disc        int
	TrackNumber int
	Path        string
}

func New(appDir string) (API, error) {
	db, err := loadDB(appDir)
	if err != nil {
		return API{}, err
	}

	return API{db}, nil
}

func loadDB(appDir string) (*sql.DB, error) {
	dir, err := os.UserCacheDir()
	if err != nil {
		return nil, err
	}

	path := filepath.Join(dir, appDir, "library.db")

	err = os.MkdirAll(filepath.Dir(path), os.ModePerm)
	if err != nil {
		return nil, err
	}

	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`PRAGMA journal_mode = wal;
					PRAGMA synchronous = normal;
					PRAGMA foreign_keys = on;`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS albums(
						id integer PRIMARY KEY,
						album_artist TEXT,
						name TEXT,
						year INTEGER
					);`)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(`CREATE TABLE IF NOT EXISTS tracks(
						id integer PRIMARY KEY,
						album_id INTEGER REFERENCES albums(id),
						disc INTEGER,
						path TEXT,
						track_number INTEGER
					);`)
	if err != nil {
		return nil, err
	}

	return db, nil
}

func (api API) Close() error {
	if api.db == nil {
		return nil
	}

	_, err := api.db.Exec(`PRAGMA analysis_limit=400;
					PRAGMA optimize;`)
	if err != nil {
		return err
	}

	err = api.db.Close()
	if err != nil {
		return err
	}

	return nil
}

func readMetadata(path string) (*Album, *Track, error) {
	f, err := os.OpenFile(path, os.O_RDONLY, 0444)
	if err != nil {
		return nil, nil, err
	}
	defer f.Close()

	m, err := tag.ReadFrom(f)
	if err != nil {
		return nil, nil, err
	}

	a := Album{
		AlbumArtist: m.AlbumArtist(),
		Name:        m.Album(),
		Year:        m.Year(),
	}
	if m.Year() == 0 {
		a.Year = readAltYearMetadata(m)
	}

	disc, _ := m.Disc()
	trackNumber, _ := m.Track()
	t := Track{
		Disc:        disc,
		Path:        path,
		TrackNumber: trackNumber,
	}

	return &a, &t, nil
}

// Add track (and album, if not present) to library
func (api API) AddPathToLibrary(path string) error {
	trackID, err := api.findTrackID(path)
	if err != nil {
		return err
	}
	if trackID != -1 { // path already in db
		return nil
	}

	a, t, err := readMetadata(path)
	if err != nil {
		return err
	}

	albumID, err := api.findAlbumID(a)
	if err != nil {
		return err
	}
	if albumID == -1 { // album doesn't exist
		albumID, err = api.insertAlbum(a)
		if err != nil {
			return err
		}
	}
	t.AlbumID = albumID

	_, err = api.insertTrack(t)
	if err != nil {
		return err
	}
	return nil
}

func (api API) findTrackID(path string) (int64, error) {
	query := `SELECT id FROM tracks WHERE path = ?;`
	row := api.db.QueryRow(query, path)
	var trackID int64
	err := row.Scan(&trackID)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return trackID, err
}

func (api API) findAlbumID(a *Album) (int64, error) {
	query := `SELECT id FROM albums
			WHERE album_artist = ? AND name = ? AND year = ?;`
	row := api.db.QueryRow(query, a.AlbumArtist, a.Name, a.Year)
	var albumID int64
	err := row.Scan(&albumID)
	if err == sql.ErrNoRows {
		return -1, nil
	}
	return albumID, err
}

func (api API) insertAlbum(a *Album) (int64, error) {
	res, err := api.db.Exec(`INSERT INTO albums(album_artist,name,year)
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

func (api API) insertTrack(t *Track) (int64, error) {
	res, err := api.db.Exec(`INSERT INTO tracks(album_id,disc,path,track_number)
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

func (api API) DeleteTrack(path string) error {
	_, err := api.db.Exec(`DELETE FROM tracks WHERE path = ?`, path)
	return err
}

// Delete all albums with no tracks
func (api API) RemoveEmptyAlbums() error {
	tx, err := api.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM albums
					WHERE id NOT IN (
						SELECT DISTINCT album_id
						FROM tracks
					);`)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func (api API) GetRandomAlbums() ([]Album, error) {
	rows, err := api.db.Query("SELECT * FROM albums ORDER BY RANDOM();")
	if err != nil {
		return nil, err
	}

	albums, err := parseRowsToAlbums(rows)
	if err != nil {
		return nil, err
	}

	return albums, nil
}

func (api API) GetOneRandomAlbum() (*Album, error) {
	row := api.db.QueryRow("SELECT * FROM albums ORDER BY RANDOM() LIMIT 1;")

	var a Album
	err := row.Scan(&a.ID, &a.AlbumArtist, &a.Name, &a.Year)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil // No albums found
		}
		return nil, err
	}

	return &a, nil
}

func sqliteOrder(asc bool) string {
	if asc {
		return "ASC"
	}
	return "DESC"
}

func (api API) GetAlbumsByArtist(asc bool) ([]Album, error) {
	rows, err := api.db.Query(`SELECT * FROM albums
						ORDER BY album_artist ` + sqliteOrder(asc))
	if err != nil {
		return nil, err
	}

	albums, err := parseRowsToAlbums(rows)
	if err != nil {
		return nil, err
	}

	return albums, nil
}

func (api API) GetAlbumsByYear(asc bool) ([]Album, error) {
	rows, err := api.db.Query(`SELECT * FROM albums
						ORDER BY year ` + sqliteOrder(asc))
	if err != nil {
		return nil, err
	}

	albums, err := parseRowsToAlbums(rows)
	if err != nil {
		return nil, err
	}

	return albums, nil
}

func (api API) SearchAlbums(query string) ([]Album, error) {
	a := "%" + query + "%"
	rows, err := api.db.Query(`SELECT * FROM albums WHERE
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
		log.Println("TDRL", tdrlStr)
	}
	xdor := (r["XDOR"]) // ID3 v2.3 orig release year
	if xdorStr, ok := xdor.(string); ok {
		log.Println("XDOR", xdorStr)
	}
	tory := (r["TORY"]) // ID3 v2.3 orig release year
	if toryStr, ok := tory.(string); ok {
		log.Println("TORY", toryStr)
	}
	return 0
}

func (api API) AlbumTrackPaths(albumID int64) ([]string, error) {
	query := `SELECT path FROM tracks
			WHERE album_id = ?
			ORDER BY track_number ASC, disc ASC;`
	rows, err := api.db.Query(query, albumID)
	if err != nil {
		return nil, err
	}

	paths, err := parseRowsToTrackPaths(rows)
	if err != nil {
		return nil, err
	}

	return paths, nil
}

func (api API) AllTrackPaths() ([]string, error) {
	query := `SELECT path FROM tracks`
	rows, err := api.db.Query(query)
	if err != nil {
		return nil, err
	}

	paths, err := parseRowsToTrackPaths(rows)
	if err != nil {
		return nil, err
	}

	return paths, nil
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
