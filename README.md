## iTunes Playlist Track Syncer (IPTS)

A command-line tool to sync iTunes playlist tracks with a local folder.

- This works by reading your `iTunes Music Library.xml` file and then copying the files to a local folder.
- You can safely repeat the process as only new files are copied.
- If a file is no longer in a playlist, it is deleted from the given folder.
- A M3U file is created for each playlist automatically.
- You can list multiple playlists to sync to the same folder.
- You can setup multiple config files for each playlists to sync to their own folders.
- You can set a custom location for files (than the one in iTunes Music Library)
- This is tested under OS X El Capitan with iTunes 12.

### Usage

Personally, I will use this with [Syncthing](https://syncthing.net/) to sync my iTunes playlist with my Android Phone (and other devices).

Tweak this [YAML file](https://github.com/mofirouz/iTunesPlaylistTrackSyncer/blob/master/config.yml) with your own configuration and then run `bin/ipts`. If you'd like to run it against different config files, simply run:

```
usage: ipts [<flags>]
Flags:
      --help                 Show context-sensitive help (also try --help-long and --help-man).
  -c, --config="config.yml"  YAML configuration file
```

The YAML config file definition:

```yml
log:
  level: debug #panic, fatal, error, warn, info, debug
  output: stdout # stdout or path to a file
itunes:
  playlists: ["Test"] #Required. List of playlist names. Case sensitive
  trackOutputFolder: "/Users/<user_name>/Desktop/IPTS/" #Required. Path to sync files to.
  libraryFile: "/Users/<user_name>/Music/iTunes/iTunes Music Library.xml" #Required. Path to iTunes Music Library.xml.
  customFileLocation: "" #Optional. Path to custom location for source file.
  createM3U: true #Optional. Create M3U Playlist
  watchChanges: false #TODO - Optional. Watch iTunes Library for changes.
```

#### For Developers

1. `./install.sh`
2. `go build -v -o bin/ipts *.go`

##### TODO:

- Sync to folder automatically by watching `iTunes Music Library.xml` for changes.
