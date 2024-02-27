# musli

A music library interface. Unlike most consumer music programs, musli doesn't include an audio player. Instead, it relies on the media player you already have installed (see: [Compatible media players](#compatible-media-players)). Likewise, the program doesn't include any methods to edit metadata, requiring that your music files already be properly tagged.

What musli *does* do is provide a fast and efficient way to browse and search your local music libary.

## Options

<code><strong>-c, --config</strong> path</code>

Use config file at `path`. Must be a valid `toml` configuration file (see: [Configuration](#configuration)).

<code><strong>-q, --query</strong> query</code>

Search library for album name or album artist that contains `query`.

<code><strong>-r, --random</strong></code>

List random albums from your music library. Convenient for when you don't know what to listen to.

<code><strong>-s, --scan</strong></code>

Scans your music directory for compatible music files. This may take a while the first time, but subsequent scans (like when you add new files to your music directory) shouldn't take long.

<code><strong>-y, --year</strong> year(-end)</code>

List albums from `year`. Or, list albums from range [`year`, `end`] by using `musli -y 1968-1971`

## Configuration

Location: `~/.config/musli/config.toml`

**MusicDir**

Recursively find music files in said directory. *Default*: `~/Music`

**ExecCmd**

Set which command to execute when an album is selected. The individual paths for each track of the selected album will be passed as arguments to the command.

For example, if `ExecCmd` was set to `mpv`, a command such as the following would be executed:

`/sbin/mpv "/home/micah/Music/Fiona Apple/Tidal/01 Sleep to Dream.flac" "/home/micah/Music/Fiona Apple/Tidal/02 Sullen Girl.flac" "/home/micah/Music/Fiona Apple/Tidal/03 Shadowboxer.flac" ...`

**ShowStdout** / **ShowStderr**

Prints the command's stdout/stderr while the media player is running. Useful for debugging. Should only enable one at a time. If both are set to `true`, stdout takes precedence and stderr will not be printed.

## Compatible media players

Any media player that follows the following pattern should work.

`cmd [options] files...`

I have tested it with the following packages on Debian:

* mpv
* mplayer
* vlc
* parole
