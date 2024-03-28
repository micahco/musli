# musli

A music library interface. Unlike most consumer music programs, musli doesn't include an audio player. Instead, it relies on the media player you already have installed (see: [Compatible media players](#compatible-media-players)). Likewise, the program doesn't include any methods to edit metadata, requiring that your music files already be properly tagged.

What musli *does* do is provide a fast and efficient way to browse and search your local music libary.

![demo](./demo.webm)

## Install
```
go install github.com/micahco/musli/cmd/musli
```

## Options

Run `musli --help` for a list of options and how to use them.

## Configuration

The program uses a [TOML](https://toml.io/en/v1.0.0) configuration file. Each key has a default value.

The config file is located at: `~/.config/musli/config.toml` on Unix systems. This file will not be created automatically.

See [config.toml](https://github.com/micahco/musli/blob/main/config.toml) for an example.

### Parameters

#### MusicDir

Default: `"~/Music"`

Recursively find music files in said directory.

#### ExecCmd

Default: `"mpv"`

Command to be executed for album playback. The individual paths for each track of the selected album will be passed as arguments to the command.

The command executed will look something like this:

`mpv /path/to/ablum/track1.mp3 /path/to/ablum/track2.mp3 ...`

#### ListTemplate

Default: `"%artist% - %album%"`

Customize the album list. The available variables are `%artist%` for album artist, `%album%` for album name, and `%year%` for the release year.

#### HiglightSGR

Default: `[ 7 ]`

Select Graphic Rendition (SGR) parameters for the highlighted albun. Must be a valid integer array. See [SGR parameters (Wikipedia)](https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_(Select_Graphic_Rendition)_parameters) for valid values.

Examples:
* Bold, magenta foreground: `[ 1, 35 ]`
* Bright red background `[ 101 ]`

#### ShowStdout / ShowStderr:

Default: both `false`

Prints the command's stdout/stderr while the media player is running. Useful for debugging. Should only enable one at a time. If both are set to `true`, stdout takes precedence and stderr will not be printed.

## Compatible media players

Any media player that follows the following pattern should work.

`cmd [options] files...`

Tested:

* mpv
* vlc
* mplayer
* parole
