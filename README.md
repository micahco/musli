# musli

`musli` is a CLI music library. 

`musli` is **not**:

* A music player. Instead, `musli` uses the the media player you already have to handle playback. See [Compatible media players](#compatible-media-players).

* A metadata editor. `musli` requires that your music file be properly tagged.

* Built for songs or playlist. `musli` is exclusively made for album playback. This allows it to be very fast and simple.

## Install  

Refer to the Go programming language documentation: [Compile and install the application](https://go.dev/doc/tutorial/compile-install).

## Configuration file

Location: `/home/micah/.config/musli/config.toml`

Default values:

```
DbFile = "/home/micah/.state/musli/library.db"
MusicDir = "/home/micah/Music"
ExecCmd = "mpv"
ShowStdout = false
ShowStderr = false
```

**Note**: use full directory paths `/home/micah/Music`, not: `$HOME/Music` or `~/Music`

### Options

#### DbFile

Sets the file path to where the SQLite database is stored. You probably won't need to change this.

#### MusicDir

Recursively find music files in said directory.

#### ExecCmd

Set which command to execute when an album is selected. The individual paths for each track of the selected album will be passed as arguments to the command.

For example, if `ExecCmd` was set to `mpv`, a command such as the following would be executed:

`/sbin/mpv "/home/micah/Music/Fiona Apple/Tidal/01 Sleep to Dream.flac" "/home/micah/Music/Fiona Apple/Tidal/02 Sullen Girl.flac" "/home/micah/Music/Fiona Apple/Tidal/03 Shadowboxer.flac" ...`

#### ShowStdout / ShowStderr

Prints the command's stdout/stderr while the media player is running. Useful for debugging. Should only enable one at a time. If both are set to `true`, stdout takes precedence and stderr will not be printed.

## Compatible media players

The basic premise is that musli simply launches the medea player with a list of filepaths as arguments. Theoretically, any media player that has a CLI with the pattern should work.

I have tested it with the following packages on Debian:

* mpv
* mplayer
* vlc
* parole
