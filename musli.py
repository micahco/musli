#!/usr/bin/env python3

import argparse
import glob
import os
import sqlite3
from tinytag import TinyTag

APP_NAME = 'musli'

class Counter:
    def __init__(self):
        self.count = 0
        
    def getCount(self):
        return self.count

    def inc(self):
        self.count += 1

def create_connection(database):
    conn = None
    try:
        conn = sqlite3.connect(database)
    except sqlite3.Error as e:
        print(e)
    
    return conn

def create_table(conn, sql):
    try:
        cur = conn.cursor()
        cur.execute(sql)
    except sqlite3.Error as e:
        print(e)

def create_album(conn, album):
    sql = ''' INSERT INTO albums(album_artist,name,year)
              VALUES(?,?,?) '''
    cur = conn.cursor()
    cur.execute(sql, album)
    conn.commit()
    return cur.lastrowid

def select_album(conn, album):
    sql = ''' SELECT id FROM albums
              WHERE album_artist = ? AND name = ? AND year = ? '''
    cur = conn.cursor()
    res = cur.execute(sql, album)
    album = res.fetchone()
    if album:
        return album[0]
    return None

def create_track(conn, track):
    sql = ''' INSERT INTO tracks(album_id,track,path)
              VALUES(?,?,?) '''
    cur = conn.cursor()
    cur.execute(sql, track)
    conn.commit()
    return cur.lastrowid

def select_track(conn, path):
    sql = ''' SELECT * FROM tracks
              WHERE path = ? '''
    cur = conn.cursor()
    res = cur.execute(sql, (path,))
    track = res.fetchone()
    if track:
        return track
    return None

def select_random_albums(conn):
    cur = conn.cursor()
    res = cur.execute('SELECT * FROM albums ORDER BY RANDOM()')
    albums = res.fetchall()
    return albums

def select_album_tracks(conn, album_id):
    sql = ''' SELECT * FROM tracks WHERE album_id = ? 
              ORDER BY track ASC'''
    cur = conn.cursor()
    res = cur.execute(sql, (album_id,))
    albums = res.fetchall()
    return albums

def scan_library(conn):
    count = 0
    for track_path in glob.glob('/nfs/m600/music/**/*.*', recursive = True):
        if TinyTag.is_supported(track_path):
            tag = TinyTag.get(track_path)
            album = (tag.albumartist, tag.album, tag.year)
            album_id = select_album(conn, album)
            if not album_id:
                album_id = create_album(conn, album)

            track = (album_id, tag.track, track_path)
            track_id = select_track(conn, track_path)
            if not track_id:
                track_id = create_track(conn, track)
            
            count += 1
            if count > 200:
                return
            print(f'Scanned {count} tracks', end="\r", flush=True)

def main():
    parser = argparse.ArgumentParser(
                        prog=APP_NAME,
                        description=f'{APP_NAME}: an opinionated, read-only music library utility',
                        epilog='Created by Micah Cowell')
    parser.add_argument('-s', '--scan', action='store_true')
    parser.add_argument('-r', '--random', action='store_true')
    args = parser.parse_args()

    sql_albums_table = ''' CREATE TABLE IF NOT EXISTS albums(
                            id integer PRIMARY KEY,
                            album_artist TEXT,
                            name TEXT NOT NULL,
                            year TEXT
                        )'''
    
    sql_tracks_table = ''' CREATE TABLE IF NOT EXISTS tracks(
                            id integer PRIMARY KEY,
                            album_id INTEGER NOT NULL,
                            track INTEGER NOT NULL,
                            path TEXT NOT NULL
                        )'''

    conn = create_connection(f'{APP_NAME}.db')

    if conn is not None:
        create_table(conn, sql_albums_table)
        create_table(conn, sql_tracks_table)

        if args.scan:
            print('Scanning library')
            scan_library(conn)
            print('\nScan complete')
        if args.random:
            albums = select_random_albums(conn)
            if albums:
                print(albums[0][1] + ' - ' + albums[0][2])
                tracks = select_album_tracks(conn, albums[0][0])
                for track in tracks:
                    print(track[3])
        
        conn.close()

if __name__ == '__main__':
    main()
