package youtube

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"medley/internal/shared"

	"github.com/adrg/xdg"
	"gopkg.in/yaml.v3"
)

type Config struct {
	Genres []Genre `yaml:"genres"`
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

type TrackTask struct {
	Track     Track
	GenreName string
	AlbumName string
	MediaHome string
}

func ParseMediaFile(path string, mediaHome string) ([]TrackTask, error) {
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
					MediaHome: mediaHome,
				})
			}
		}
	}
	return tasks, nil
}

func ProcessTask(ctx context.Context,
	appName string, task TrackTask, logChan chan<- string, verbose bool,
	cookieFilePathPtr *string, ytdlpTimeout time.Duration, ffmpegTimeout time.Duration,
	convTimeout time.Duration, eyed3Timeout time.Duration,
	atomicparsleyTimeout time.Duration) error {

	dldir := filepath.Join(task.MediaHome, "downloads")
	tmpdir := filepath.Join(task.MediaHome, "tmp")

	if err := os.MkdirAll(dldir, 0755); err != nil {
		return fmt.Errorf("fatal: cannot create download directory: %w", err)
	}

	if err := os.MkdirAll(tmpdir, 0755); err != nil {
		return fmt.Errorf("fatal: cannot create tmp directory: %w", err)
	}

	// Check if all requested formats already exist in their final destinations
	allFormatsExist := true
	for _, fmtTarget := range task.Track.Formats {
		destDir := filepath.Join(task.MediaHome, fmtTarget, task.Track.Folder)
		finalDest := filepath.Join(destDir, task.Track.Filename+"."+fmtTarget)

		if _, err := os.Stat(finalDest); os.IsNotExist(err) {
			allFormatsExist = false
			break // At least one format is missing, so we must process the track
		}
	}

	if allFormatsExist {
		logChan <- fmt.Sprintf("Skipping track: %s (Already exists)", task.Track.Title)
		return nil // Exit early, skipping download and extraction
	}

	logChan <- fmt.Sprintf("Processing track: %s", task.Track.Title)

	var ytargs []string
	if verbose {
		ytargs = []string{"--verbose"}
	} else {
		ytargs = []string{"--no-warnings", "--quiet"}
	}

	// Full absolute path of the command "bgutil-pot" is not required for the
	// "youtubepot-bgutilcli:cli_path" if the command is in the $PATH.

	ytargs = append(ytargs,
		"--no-cache-dir",
		"-f", "ba[ext=webm],ba",
		"--extractor-args", "youtubepot-bgutilcli:cli_path=bgutil-pot")

	if cookieFilePathPtr != nil {
		ytargs = append(ytargs, "--cookies", *cookieFilePathPtr)
	}

	needImg := task.Track.Thumbnails["mp3"] || task.Track.Thumbnails["m4a"]
	if needImg {
		ytargs = append(ytargs, "--write-thumbnail")
	}
	ytargs = append(ytargs, "-o", filepath.Join(dldir, task.Track.Filename+".%(ext)s"))
	ytargs = append(ytargs, "https://www.youtube.com/watch?v="+task.Track.Ytid)

	logChan <- fmt.Sprintf("Downloading asset %s via yt-dlp...", task.Track.Filename)

	cmd := exec.CommandContext(ctx, "yt-dlp", ytargs...)
	cmd.WaitDelay = ytdlpTimeout

	ytdlpLogFilename, err := xdg.StateFile(filepath.Join(appName, "yt-dlp.log"))
	if err != nil {
		return fmt.Errorf("Failed to resolve yt-dlp log file: %s\n%w", ytdlpLogFilename, err)
	}

	ytdlpLogFile, err := os.OpenFile(ytdlpLogFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
	if err != nil {
		return fmt.Errorf("Failed to open yt-dlp log file: %w", err)
	}
	defer ytdlpLogFile.Close()

	cmd.Stdout = ytdlpLogFile
	cmd.Stderr = ytdlpLogFile
	err = cmd.Run()
	if err != nil {
		return fmt.Errorf("yt-dlp failed: %w (logfile: %s)", err, ytdlpLogFilename)
	}

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

	for _, fmtTarget := range task.Track.Formats {
		destDir := filepath.Join(task.MediaHome, fmtTarget, task.Track.Folder)
		finalDest := filepath.Join(destDir, task.Track.Filename+"."+fmtTarget)

		if _, err := os.Stat(finalDest); err == nil {
			logChan <- fmt.Sprintf("Skipping target format %s for %s (Already exists)", fmtTarget,
				task.Track.Filename)
			continue
		}

		logChan <- fmt.Sprintf("Extracting audio target format: %s", fmtTarget)

		afile := filepath.Join(tmpdir, task.Track.Filename+"."+fmtTarget)
		dlfile := filepath.Join(dldir, task.Track.Filename+"."+dlext)

		ffargs := []string{"-nostdin", "-y"}

		// -loglevel [error | verbose | debug]
		if verbose {
			ffargs = append(ffargs, "-loglevel", "verbose")
		} else {
			ffargs = append(ffargs, "-hide_banner", "-loglevel", "error")
		}

		if task.Track.Section.Start != "" {
			ffargs = append(ffargs, "-ss", task.Track.Section.Start)
		}
		if task.Track.Section.End != "" {
			ffargs = append(ffargs, "-to", task.Track.Section.End)
		}
		ffargs = append(ffargs, "-i", dlfile, "-map", "0:a", "-vn", "-dn")

		if fmtTarget == "mp3" {
			if dlext == "webm" || dlext == "m4a" {
				ffargs = append(ffargs, "-af", "pan=stereo|c0=FL|c1=FL", "-c:a", "libmp3lame",
					"-b:a", "256k", "-ar", "44100", "-map_metadata", "-1", "-bitexact")
			} else {
				ffargs = append(ffargs, "-c:a", "copy")
			}
		} else if fmtTarget == "m4a" {
			if dlext == "webm" {
				ffargs = append(ffargs, "-af", "pan=stereo|c0=FL|c1=FL", "-c:a", "aac", "-b:a",
					"256k", "-ar", "44100", "-map_metadata", "-1", "-bitexact")
			} else {
				ffargs = append(ffargs, "-c:a", "copy")
			}
		}
		ffargs = append(ffargs, afile)

		cmd := exec.CommandContext(ctx, "ffmpeg", ffargs...)
		cmd.WaitDelay = ffmpegTimeout

		ffmpegLogFilename, err := xdg.StateFile(filepath.Join(appName, "ffmpeg.log"))
		if err != nil {
			return fmt.Errorf("Failed to resolve ffmpeg log file: %s\n%w", ffmpegLogFilename, err)
		}

		ffmpegLogFile, err := os.OpenFile(ffmpegLogFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return fmt.Errorf("Failed to open ffmpeg log file: %w", err)
		}
		defer ffmpegLogFile.Close()

		cmd.Stdout = ffmpegLogFile
		cmd.Stderr = ffmpegLogFile
		err = cmd.Run()
		if err != nil {
			return fmt.Errorf("ffmpeg failed: %w (logfile: %s)", err, ffmpegLogFilename)
		}

		imgfile := filepath.Join(tmpdir, fmt.Sprintf("%s.%s.jpg", task.Track.Filename, fmtTarget))
		if task.Track.Thumbnails[fmtTarget] {
			logChan <- "Converting downscaled artwork layout..."
			var sizeArg string
			if fmtTarget == "m4a" {
				sizeArg = "600>"
			} else {
				sizeArg = "320>"
			}

			var srcImg string
			for _, ext := range []string{"jpg", "jpeg", "webp", "png"} {
				testPath := filepath.Join(dldir, task.Track.Filename+"."+ext)
				if _, err := os.Stat(testPath); err == nil {
					srcImg = testPath
					break
				}
			}

			if srcImg != "" {

				// convArgs := []string{srcImg, "-resize", sizeArg, "-interlace", "none", "-strip",
				// 	"-quality", "80", imgfile}

				var convArgs []string
				if verbose {
					convArgs = []string{"-verbose"}
				} else {
					convArgs = []string{"-quiet"}
				}
				convArgs = append(convArgs, srcImg, "-resize", sizeArg, "-interlace", "none",
					"-strip", "-quality", "80", imgfile)

				cmd := exec.CommandContext(ctx, "convert", convArgs...)
				cmd.WaitDelay = convTimeout

				convLogFilename, err := xdg.StateFile(filepath.Join(appName, "convert.log"))
				if err != nil {
					return fmt.Errorf("Failed to resolve convert log file: %s\n%w", convLogFilename, err)
				}

				convLogFile, err := os.OpenFile(convLogFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
				if err != nil {
					return fmt.Errorf("Failed to open convert log file: %w", err)
				}
				defer convLogFile.Close()

				cmd.Stdout = convLogFile
				cmd.Stderr = convLogFile
				err = cmd.Run()
				if err != nil {
					return fmt.Errorf("convert failed: %w (logfile: %s)", err, convLogFilename)
				}
			}
		}

		_ = os.MkdirAll(destDir, 0755)

		// Rely on the internal/shared package for common file interactions
		if err := shared.CopyFile(afile, finalDest); err != nil {
			return err
		}

		if fmtTarget == "mp3" {
			var tagArgs []string
			if verbose {
				tagArgs = []string{"--verbose", "--log-level", "verbose"}
			} else {
				tagArgs = []string{"--quiet", "--log-level=error"}
			}

			tagArgs = append(tagArgs,
				"--to-v2.3", "--encoding", "latin1",
				"--album", task.AlbumName,
				"--track", fmt.Sprintf("%d", task.Track.TrackNum),
				"--title", task.Track.Title,
			)

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

			cmd := exec.CommandContext(ctx, "eyeD3", tagArgs...)
			cmd.WaitDelay = eyed3Timeout

			eyed3LogFilename, err := xdg.StateFile(filepath.Join(appName, "eyed3.log"))
			if err != nil {
				return fmt.Errorf("Failed to resolve eyed3 log file: %s\n%w", eyed3LogFilename, err)
			}

			eyed3LogFile, err := os.OpenFile(eyed3LogFilename, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("Failed to open eyed3 log file: %w", err)
			}
			defer eyed3LogFile.Close()

			cmd.Stdout = eyed3LogFile
			cmd.Stderr = eyed3LogFile
			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("eyed3 failed: %w (logfile: %s)", err, eyed3LogFilename)
			}

		} else if fmtTarget == "m4a" {

			aname := task.Track.Artist.Shortname
			if aname == "" {
				aname = task.Track.Artist.Name
			}
			tagArgs := []string{finalDest, "--album", task.AlbumName, "--tracknum",
				fmt.Sprintf("%d", task.Track.TrackNum), "--title", task.Track.Title}

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

			cmd := exec.CommandContext(ctx, "AtomicParsley", tagArgs...)
			cmd.WaitDelay = atomicparsleyTimeout

			atomicparsleyLogFilename, err := xdg.StateFile(filepath.Join(appName,
				"atomicparsley.log"))
			if err != nil {
				return fmt.Errorf("Failed to resolve atomicparsley log file: %s\n%w",
					atomicparsleyLogFilename, err)
			}

			atomicparsleyLogFile, err := os.OpenFile(atomicparsleyLogFilename,
				os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				return fmt.Errorf("Failed to open atomicparsley log file: %w", err)
			}
			defer atomicparsleyLogFile.Close()

			cmd.Stdout = atomicparsleyLogFile
			cmd.Stderr = atomicparsleyLogFile
			err = cmd.Run()
			if err != nil {
				return fmt.Errorf("atomicparsley failed: %w (logfile: %s)", err,
					atomicparsleyLogFilename)
			}
		}
	}

	logChan <- "Cleaning up download and tmp directories..."

	if dlFiles, err := filepath.Glob(filepath.Join(dldir, task.Track.Filename+".*")); err == nil {
		for _, f := range dlFiles {
			_ = os.Remove(f)
		}
	}

	if tmpFiles, err := filepath.Glob(filepath.Join(tmpdir, task.Track.Filename+".*")); err == nil {
		for _, f := range tmpFiles {
			_ = os.Remove(f)
		}
	}

	return nil
}
