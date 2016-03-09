package main

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	kingpin "github.com/alecthomas/kingpin"
	itl "github.com/dhowden/itl"
	yaml "github.com/go-yaml/yaml"
	"html"
	"io"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	"strings"
)

type Configuration struct {
	Log struct {
		OutputFile string `yaml:"output"`
		Level      string `yaml:"level"`
	}
	ITunes struct {
		Playlists          []string `yaml:"playlists"`
		TrackOutputFolder  string   `yaml:"trackOutputFolder"`
		LibraryFile        string   `yaml:"libraryFile"`
		CreateM3U          bool     `yaml:"createM3U"`
		CustomFileLocation string   `yaml:"customFileLocation"`
		WatchChanges       bool     `yaml:"watchChanges"`
	}
}

var (
	ConfigFile = kingpin.Flag("config", "YAML configuration file").Default("config.yml").Short('c').String()
	Config     = setDefaultConfig()
)

func setDefaultConfig() Configuration {
	defaultConfig := Configuration{}
	defaultConfig.Log.Level = "info"
	defaultConfig.Log.OutputFile = "ipte.log"

	defaultConfig.ITunes.Playlists = []string{}
	defaultConfig.ITunes.TrackOutputFolder = "."
	defaultConfig.ITunes.LibraryFile = "/Users/mo/Music/iTunes/iTunes Music Library.xml"
	defaultConfig.ITunes.CustomFileLocation = ""
	defaultConfig.ITunes.CreateM3U = true
	defaultConfig.ITunes.WatchChanges = false
	return defaultConfig
}

func checkError(msg string, e error) {
	if e != nil {
		log.Fatalf("%s: %s", msg, e)
	}
}

func setupLogger() *os.File {
	level, err := log.ParseLevel(Config.Log.Level)
	checkError("config.level", err)
	log.SetLevel(level)

	if Config.Log.OutputFile == "stdout" {
		return os.Stdout
	} else {
		os.MkdirAll(path.Dir(Config.Log.OutputFile), 0777)
		file, err := os.Create(Config.Log.OutputFile)
		checkError("config.log", err)
		log.SetOutput(file)
		return file
	}
}

func main() {
	kingpin.Parse()
	data, err := ioutil.ReadFile(*ConfigFile)
	checkError("config.read", err)
	err = yaml.Unmarshal(data, &Config)
	checkError("config.load", err)

	logFile := setupLogger()
	defer logFile.Close()

	log.Info("Reading iTunes Library")

	file, err := os.Open(Config.ITunes.LibraryFile)
	checkError("config.library", err)
	lib, err := itl.ReadFromXML(file)
	checkError("itl.library", err)

	tracks := make([]itl.Track, 0)
	for _, wantedPlaylist := range Config.ITunes.Playlists {
		for _, playlist := range lib.Playlists {
			if playlist.Name == wantedPlaylist {
				log.Info("Getting Track Information for '" + playlist.Name + "'")
				tracks = extractPlaylist(playlist, lib, tracks)
			}
		}
	}

	log.Info("Copying " + strconv.Itoa(len(tracks)) + " tracks")
	newFiles := copyTracks(lib.MusicFolder, tracks)
	fileList := getFileLists()
	log.Info("Syncing files ")
	deletedFiles := deleteFiles(fileList, newFiles)
	log.Info("Sync Successful. Copied " + strconv.Itoa(len(tracks)) + " tracks - Deleted " + strconv.Itoa(len(deletedFiles)) + " files.")
}

func extractPlaylist(playlist itl.Playlist, lib itl.Library, tracks []itl.Track) []itl.Track {
	var buffer bytes.Buffer
	buffer.WriteString(fmt.Sprintln("#EXTM3U"))
	for _, item := range playlist.PlaylistItems {
		track := lib.Tracks[strconv.Itoa(item.TrackID)]
		tracks = append(tracks, track)
		trackLocation, _ := UnescapeToString(track.Location)
		trackLocation = html.UnescapeString(trackLocation)

		filePath := "/" + strings.Replace(trackLocation, lib.MusicFolder, "", -1)
		buffer.WriteString(fmt.Sprintln(filePath))
	}

	if Config.ITunes.CreateM3U {
		path := Config.ITunes.TrackOutputFolder + playlist.Name + ".m3u"
		ioutil.WriteFile(path, buffer.Bytes(), 0777)
	}

	return tracks
}

func copyTracks(musicFolder string, tracks []itl.Track) []string {
	dstFiles := make([]string, len(tracks))
	for _, track := range tracks {
		trackLocation, _ := UnescapeToString(track.Location)
		trackLocation = html.UnescapeString(trackLocation)

		if !strings.HasPrefix(trackLocation, "file://") {
			break
		}

		srcFile := strings.Replace(trackLocation, "file://", "", 1)
		if Config.ITunes.CustomFileLocation != "" {
			srcFile = strings.Replace(trackLocation, musicFolder, Config.ITunes.CustomFileLocation, 1)
		}
		dstFile := Config.ITunes.TrackOutputFolder + strings.Replace(trackLocation, musicFolder, "", 1)
		dstPath := path.Dir(dstFile)
		dstFiles = append(dstFiles, dstFile)
		os.MkdirAll(dstPath, 0777)
		copyFile(srcFile, dstFile)
	}
	return dstFiles
}

func getFileLists() []string {
	fileList := []string{}
	err := filepath.Walk(Config.ITunes.TrackOutputFolder, func(path string, f os.FileInfo, err error) error {
		if !f.IsDir() {
			fileList = append(fileList, path)
		}
		return nil
	})
	checkError("file.list", err)
	return fileList
}

func deleteFiles(allFiles []string, newFiles []string) []string {
	deletedFiles := []string{}

	for _, file := range allFiles {
		found := false
		for _, newFile := range newFiles {
			if file == newFile {
				found = true
			}
		}

		if strings.ToLower(filepath.Ext(file)) != ".mp3" {
			found = true
		}

		if !found {
			log.Debug("Deleting '" + file + "'")
			err := os.Remove(file)
			checkError("file.Remove", err)
			deletedFiles = append(deletedFiles, file)
		}
	}

	return deletedFiles
}

func copyFile(src, dst string) (err error) {
	sfi, err := os.Stat(src)
	checkError("src.Stat", err)
	if !sfi.Mode().IsRegular() {
		log.Warn("Skipping file " + src + ": Non-regular source file")
		return
	}
	dfi, err := os.Stat(dst)
	if err != nil {
		if !os.IsNotExist(err) {
			checkError("dst.Stat", err)
		}
	} else {
		if !(dfi.Mode().IsRegular()) {
			log.Warn("Skipping file " + src + ": Non-regular destination file")
			return
		}

		if dfi.Name() == sfi.Name() && dfi.Size() == sfi.Size() {
			log.Debug("Skipping file " + src + ": File is the same")
			return
		}
	}
	err = copyFileContents(src, dst)
	return
}

func copyFileContents(src, dst string) (err error) {
	in, err := os.Open(src)
	checkError("src.Open", err)
	defer in.Close()
	out, err := os.Create(dst)
	checkError("dst.Create", err)
	defer func() {
		cerr := out.Close()
		if err == nil {
			err = cerr
		}
	}()
	if _, err = io.Copy(out, in); err != nil {
		return
	}
	err = out.Sync()
	return
}
