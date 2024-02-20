#!/usr/bin/env python3

import os
import sqlite3
import argparse
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

def scan_library(dir_path, counter, conn):
    for entry in os.scandir(dir_path):
        if entry.is_dir(follow_symlinks=False):
            scan_library(entry.path, counter, conn)
        elif TinyTag.is_supported(entry.name):
            track_path = os.path.join(dir_path, entry.name)
            tag = TinyTag.get(track_path)

            album = (tag.albumartist, tag.album, tag.year)
            album_id = select_album(conn, album)
            if not album_id:
                album_id = create_album(conn, album)

            track = (album_id, tag.track, track_path)
            track_id = select_track(conn, track_path)
            
            counter.inc()
            if counter.getCount() > 100:
                return
            print(f'Scanned {counter.getCount()} tracks', end="\r", flush=True)
            

def main():
    parser = argparse.ArgumentParser(
                        prog=APP_NAME,
                        description=f'{APP_NAME}: an opinionated, read-only music library utility',
                        epilog='Created by Micah Cowell')
    parser.add_argument('-s', '--scan', action='store_true')
    args = parser.parse_args()

    dir_path = '/nfs/m600/music/'

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
            scan_library(dir_path, Counter(), conn)
            print('\nScan complete')
        
        conn.close()

if __name__ == '__main__':
    main()
