package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

// Define the YAML schema natively
type Config struct {
	MediaHome string  `yaml:"media-home"`
	Genres    []Genre `yaml:"genres"`
}

type Genre struct {
	Name   string  `yaml:"name"`
	Albums []Album `yaml:"albums"`
}

type Album struct {
	Name   string  `yaml:"name"`
	Tracks []Track `yaml:"tracks"`
}

type Track struct {
	Title      string          `yaml:"title"`
	TrackNum   int             `yaml:"track"`
	Ytid       string          `yaml:"ytid"`
	Filename   string          `yaml:"filename"`
	Folder     string          `yaml:"folder"`
	Formats    []string        `yaml:"formats"`
	Thumbnails map[string]bool `yaml:"thumbnails"`
	Section    struct {
		Start string `yaml:"start"`
		End   string `yaml:"end"`
	} `yaml:"section"`
	Artist struct {
		Name      string `yaml:"name"`
		Shortname string `yaml:"shortname"`
	} `yaml:"artist"`
	Composer string `yaml:"composer"`
}

// TrackTask aggregates contextual data needed for processing an entry
type TrackTask struct {
	Track     Track
	GenreName string
	AlbumName string
	MediaHome string
}

// ParseMediaFile completely replaces the complex internal yq execution block
func ParseMediaFile(path string) ([]TrackTask, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	var cfg Config
	decoder := yaml.NewDecoder(file)
	if err := decoder.Decode(&cfg); err != nil {
		return nil, err
	}

	var tasks []TrackTask
	for _, genre := range cfg.Genres {
		for _, album := range genre.Albums {
			for _, track := range album.Tracks {
				// Apply validation fallbacks equivalent to the Bash script defaults
				if track.Filename == "" {
					track.Filename = track.Title
				}
				if track.Folder == "" {
					track.Folder = filepath.Join(genre.Name, album.Name)
				}
				if len(track.Formats) == 0 {
					track.Formats = []string{"m4a"}
				}
				if track.Thumbnails == nil {
					track.Thumbnails = map[string]bool{"mp3": false, "m4a": true}
				}

				tasks = append(tasks, TrackTask{
					Track:     track,
					GenreName: genre.Name,
					AlbumName: album.Name,
					MediaHome: cfg.MediaHome,
				})
			}
		}
	}
	return tasks, nil
}

// ProcessTask handles the processing sequence for an individual task
func ProcessTask(task TrackTask, logChan chan<- string) error {
	dldir := filepath.Join(task.MediaHome, "downloads")
	tmpdir := filepath.Join(task.MediaHome, "tmp")
	_ = os.MkdirAll(dldir, 0755)
	_ = os.MkdirAll(tmpdir, 0755)

	logChan <- fmt.Sprintf("Processing track: %s", task.Track.Title)

	// --- 1. Fetch Media via yt-dlp ---
	ytargs := []string{"--no-warnings", "--quiet", "-f", "ba[ext=webm],ba"}

	// Determine if either target format requires an image download
	needImg := task.Track.Thumbnails["mp3"] || task.Track.Thumbnails["m4a"]
	if needImg {
		ytargs = append(ytargs, "--write-thumbnail")
	}
	ytargs = append(ytargs, "-o", filepath.Join(dldir, task.Track.Filename+".%(ext)s"))
	ytargs = append(ytargs, "https://www.youtube.com/watch?v="+task.Track.Ytid)

	logChan <- fmt.Sprintf("Downloading asset %s via yt-dlp...", task.Track.Filename)
	cmd := exec.Command("yt-dlp", ytargs...)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("yt-dlp failed: %w", err)
	}

	// Dynamic extension discovery equivalent to get_audio_format()
	var dlext string
	for _, ext := range []string{"webm", "m4a", "mp3"} {
		if _, err := os.Stat(filepath.Join(dldir, task.Track.Filename+"."+ext)); err == nil {
			dlext = ext
			break
		}
	}
	if dlext == "" {
		return fmt.Errorf("no downloaded asset discovered for %s", task.Track.Filename)
	}

	// --- Loop over individual format targets ---
	for _, fmtTarget := range task.Track.Formats {
		logChan <- fmt.Sprintf("Extracting audio target format: %s", fmtTarget)

		afile := filepath.Join(tmpdir, task.Track.Filename+"."+fmtTarget)
		dlfile := filepath.Join(dldir, task.Track.Filename+"."+dlext)

		// Incorporate the phase-cancellation fix along with script args
		ffargs := []string{"-nostdin", "-y", "-loglevel", "error"}
		if task.Track.Section.Start != "" {
			ffargs = append(ffargs, "-ss", task.Track.Section.Start)
		}
		if task.Track.Section.End != "" {
			ffargs = append(ffargs, "-to", task.Track.Section.End)
		}
		ffargs = append(ffargs, "-i", dlfile, "-map", "0:a", "-vn", "-dn")

		// Address Phase Cancellation on downloads while respecting copy logic
		if fmtTarget == "mp3" {
			if dlext == "webm" || dlext == "m4a" {
				ffargs = append(ffargs, "-af", "pan=stereo|c0=FL|c1=FL", "-c:a", "libmp3lame", "-b:a", "256k", "-ar", "44100", "-map_metadata", "-1", "-bitexact")
			} else {
				ffargs = append(ffargs, "-c:a", "copy")
			}
		} else if fmtTarget == "m4a" {
			if dlext == "webm" {
				ffargs = append(ffargs, "-af", "pan=stereo|c0=FL|c1=FL", "-c:a", "aac", "-b:a", "256k", "-ar", "44100", "-map_metadata", "-1", "-bitexact")
			} else {
				ffargs = append(ffargs, "-c:a", "copy")
			}
		}
		ffargs = append(ffargs, afile)

		if err := exec.Command("ffmpeg", ffargs...).Run(); err != nil {
			return fmt.Errorf("ffmpeg extraction error: %w", err)
		}

		// --- Handle Image/Thumbnail Resizing via ImageMagick Convert ---
		imgfile := filepath.Join(tmpdir, fmtTarget+".jpg")
		if task.Track.Thumbnails[fmtTarget] {
			logChan <- "Converting downscaled artwork layout..."
			var sizeArg string
			if fmtTarget == "m4a" {
				sizeArg = "600>"
			} else {
				sizeArg = "320>"
			}

			// Find downloaded image
			var srcImg string
			for _, ext := range []string{"jpg", "jpeg", "webp", "png"} {
				testPath := filepath.Join(dldir, task.Track.Filename+"."+ext)
				if _, err := os.Stat(testPath); err == nil {
					srcImg = testPath
					break
				}
			}

			if srcImg != "" {
				convArgs := []string{srcImg, "-resize", sizeArg, "-interlace", "none", "-strip", "-quality", "80", imgfile}
				_ = exec.Command("convert", convArgs...).Run()
			}
		}

		// --- Tagging Pass via eyeD3 / AtomicParsley ---
		destDir := filepath.Join(task.MediaHome, fmtTarget, task.Track.Folder)
		_ = os.MkdirAll(destDir, 0755)
		finalDest := filepath.Join(destDir, task.Track.Filename+"."+fmtTarget)

		// Copy working raw asset to destination
		if err := copyFile(afile, finalDest); err != nil {
			return err
		}

		if fmtTarget == "mp3" {
			tagArgs := []string{
				"--to-v2.3", "--encoding", "latin1",
				"--album", task.AlbumName,
				"--track", fmt.Sprintf("%d", task.Track.TrackNum),
				"--title", task.Track.Title,
			}
			if task.GenreName != "" {
				tagArgs = append(tagArgs, "--genre", task.GenreName)
			}
			if task.Track.Artist.Name != "" {
				tagArgs = append(tagArgs, "--artist", task.Track.Artist.Name)
			}
			if task.Track.Composer != "" {
				tagArgs = append(tagArgs, "--composer", task.Track.Composer)
			}
			if task.Track.Thumbnails["mp3"] {
				if _, err := os.Stat(imgfile); err == nil {
					tagArgs = append(tagArgs, "--add-image", fmt.Sprintf("%s:FRONT_COVER", imgfile))
				}
			}
			tagArgs = append(tagArgs, finalDest)
			_ = exec.Command("eyeD3", tagArgs...).Run()

		} else if fmtTarget == "m4a" {
			// Leverage shortname fallback for tight M4A memory limits
			aname := task.Track.Artist.Shortname
			if aname == "" {
				aname = task.Track.Artist.Name
			}

			tagArgs := []string{finalDest, "--album", task.AlbumName, "--tracknum", fmt.Sprintf("%d", task.Track.TrackNum), "--title", task.Track.Title}
			if task.GenreName != "" {
				tagArgs = append(tagArgs, "--genre", task.GenreName)
			}
			if aname != "" {
				tagArgs = append(tagArgs, "--artist", aname)
			}
			if task.Track.Composer != "" {
				tagArgs = append(tagArgs, "--composer", task.Track.Composer)
			}
			if task.Track.Thumbnails["m4a"] {
				if _, err := os.Stat(imgfile); err == nil {
					tagArgs = append(tagArgs, "--artwork", imgfile)
				}
			}
			tagArgs = append(tagArgs, "--overWrite")
			_ = exec.Command("AtomicParsley", tagArgs...).Run()
		}
	}
	return nil
}

func copyFile(src, dst string) error {
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0644)
}
