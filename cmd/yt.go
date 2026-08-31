package cmd

import (
	"context"
	"fmt"
	"math/rand/v2"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/adrg/xdg"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"medley/internal/youtube"
)

// Helper function to enforce a shutdown timeout
func waitForCleanup(wg *sync.WaitGroup, timeout time.Duration) {
	waitCh := make(chan struct{})
	go func() {
		wg.Wait()
		close(waitCh)
	}()

	select {
	case <-waitCh:
		GetLogger().Debugf("Workers cleaned up successfully.")
	case <-time.After(timeout):
		GetLogger().Debugf("Worker cleanup timed out after %v. Forcing exit.", timeout)
	}
}

func init() {
	var cmdYT = &cobra.Command{
		Use:     "yt",
		Aliases: []string{"youtube"},
		Short:   "Pipeline for automated media downloading and post-processing from YouTube.",
		Example: "\ntctl youtube --help\ntctl yt -f music.yaml -V\n",
		Long: `
The "youtube" command automates the steps of fetching media, converting formats with ffmpeg,
embedding artwork, and injecting tags into your music library — rendered inside a live-updating
terminal interface.
`,
		RunE:         yt,
		SilenceUsage: true,
	}

	rootCmd.AddCommand(cmdYT)
	cmdYT.Flags().StringVarP(&tokens.mediaConfigFileName, "file", "f", "", "Media config file name path.")
}

// YTDownloader uses the internal/youtube domain structures
type YTDownloader struct {
	ConfigFile string
	Tasks      []youtube.TrackTask
}

func (y *YTDownloader) Validate() error {
	if y.ConfigFile == "" {
		return fmt.Errorf("Media conf file name is not specified or empty.\n")
	}

	tasks, err := youtube.ParseMediaFile(y.ConfigFile, tokens.mediaHome)
	if err != nil {
		return fmt.Errorf("Initialization Failure parsing configuration matrix: %v\n", err)
	}

	y.Tasks = tasks
	return nil
}

func (y *YTDownloader) TotalTasks() int {
	return len(y.Tasks)
}

func (y *YTDownloader) Download(ctx context.Context, p *tea.Program,
	logChan chan string, errChan chan error) error {

	logChan <- "Checking Cookies file ..."

	resolvedPath, err := xdg.ConfigFile(filepath.Join(appName, cookieFileBasename))
	if err != nil {
		GetLogger().Errorf("Failed to resolve cookies file: %s\n%w", resolvedPath, err)
		errChan <- err
		p.Send(doneMsg{})
		return err
	}

	// os.Open(resolvedPath) equivelent to: os.OpenFile(resolvedPath, os.O_RDONLY, 0)
	// if cookieFile, err := os.Open(resolvedPath); os.IsNotExist(err) {}
	// cookieFile, err := os.Open(resolvedPath)
	// if os.IsNotExist(err) {}

	var cookieFilePathPtr *string

	if cookieFile, err := os.Open(resolvedPath); os.IsNotExist(err) {
		GetLogger().Warning("Cookies file %s does not exist.", resolvedPath)
	} else if err != nil {
		GetLogger().Errorf("Cookies file exists, but unable to open.\n%v", err)
		errChan <- err
		p.Send(doneMsg{})
		return err
	} else {
		cookieFilePathPtr = &resolvedPath
		cookieFile.Close()
	}

	GetLogger().Debugf("Number of tracks to process: %d\n", len(y.Tasks))

	taskPauseTimer := time.NewTimer(time.Second)
	defer taskPauseTimer.Stop()

	batchPauseTimer := time.NewTimer(time.Second)
	defer batchPauseTimer.Stop()

	for i, task := range y.Tasks {
		select {
		case <-ctx.Done():
			// logChan <- "Shutdown signal received. Cancelling remaining tasks..."
			GetLogger().Debugf("Shutdown signal received. Aborting tasks cleanly...")
			return ctx.Err() // Exit loop cleanly if TUI closed
		default:
		}

		p.Send(statusMsg(fmt.Sprintf("%d|Processing item: %s", i+1, task.Track.Title)))

		GetLogger().Debugf(fmt.Sprintf("%d|Processing track: %s", i+1, task.Track.Title))

		taskCtx, taskCancel := context.WithTimeout(ctx, TASK_TIMEOUT_MINUTES*time.Minute)

		ytdlpTimeout := time.Duration(YT_DLP_TIMEOUT_MINUTES) * time.Minute
		ffmpegTimeout := time.Duration(FFMPEG_TIMEOUT_MINUTES) * time.Second
		convTimeout := time.Duration(CONVERT_TIMEOUT_SEC) * time.Second
		eyed3Timeout := time.Duration(EYED3_TIMEOUT_SEC) * time.Second
		atomicparsleyTimeout := time.Duration(ATOMICPARSLEY_TIMEOUT_SEC) * time.Second

		if err := youtube.ProcessTask(taskCtx,
			appName, task, logChan, tokens.verbose, cookieFilePathPtr,
			ytdlpTimeout, ffmpegTimeout, convTimeout, eyed3Timeout, atomicparsleyTimeout,
		); err != nil {

			errChan <- err
			p.Send(doneMsg{})
			return err
		}

		taskCancel()

		if !taskPauseTimer.Stop() {
			select {
			case <-taskPauseTimer.C:
			default:
			}
		}

		baseDuration := YT_BASE_TRACK_PAUSE_SEC * time.Second
		taskPauseDuration := baseDuration + rand.N(1500*time.Millisecond)
		taskPauseTimer.Reset(taskPauseDuration)

		GetLogger().Debugf(fmt.Sprintf("Pausing %f seconds.", taskPauseDuration.Seconds()))

		select {
		case <-taskPauseTimer.C:
			GetLogger().Debugf("Pause ended...")
		case <-ctx.Done():
			GetLogger().Debugf("Shutdown signal received. Aborting tasks cleanly...")
			return ctx.Err()
		}

		if (i+1)%YT_BATCH_PAUSE_INTERVAL == 0 && i != len(y.Tasks)-1 {
			if !batchPauseTimer.Stop() {
				select {
				case <-batchPauseTimer.C:
				default:
				}
			}

			batchPauseDuration := YT_BATCH_PAUSE_SEC * time.Second
			batchPauseTimer.Reset(batchPauseDuration)

			GetLogger().Debugf(fmt.Sprintf("%d tracks completed. Pausing %f seconds.",
				YT_BATCH_PAUSE_INTERVAL, batchPauseDuration.Seconds()))

			select {
			case <-batchPauseTimer.C:
				GetLogger().Debugf("Pause ended...")
			case <-ctx.Done():
				GetLogger().Debugf("Shutdown signal received. Aborting tasks cleanly...")
				return ctx.Err()
			}
		}
	}

	// Wipe the entire tmp and downloads directories after all tasks conclude
	if len(y.Tasks) > 0 {
		GetLogger().Debugf("Cleaning up downloads and tmp directories.")

		mediaHome := y.Tasks[0].MediaHome
		_ = os.RemoveAll(filepath.Join(mediaHome, "downloads"))
		_ = os.RemoveAll(filepath.Join(mediaHome, "tmp"))
	}

	p.Send(doneMsg{})

	GetLogger().Debugf("Tracks processing complete.")

	return nil
}

func yt(cmd *cobra.Command, args []string) error {
	// Create a context that cancels when yt() finishes (e.g. when TUI quits)
	ctx, cancel := context.WithCancel(cmd.Context())
	defer cancel() // Triggers cancellation as soon as p.Run() unblocks and yt() returns

	mediaConfigFileName, err := cmd.Flags().GetString("file")
	if err != nil {
		return err
	}

	downloader := &YTDownloader{
		ConfigFile: mediaConfigFileName,
	}
	if err := downloader.Validate(); err != nil {
		return err
	}

	logChan := make(chan string)
	errChan := make(chan error)

	m := initialModel(downloader.TotalTasks(), logChan, errChan)
	p := tea.NewProgram(m)

	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		_ = downloader.Download(ctx, p, logChan, errChan)
	}()

	GetLogger().Debugf("Initializing Medley engine...")

	// p.Run() blocks here until 'q' or 'ctrl+c' is pressed
	finalModel, err := p.Run()
	if err != nil {
		cancel()
		waitForCleanup(&wg, WG_CLEANUP_TIMEOUT_SEC*time.Second)

		return fmt.Errorf("TUI Error: %v\n", err)
	}

	// Cast the returned tea.Model back to concrete model struct
	if m, ok := finalModel.(model); ok {
		if !m.done {
			// cmd.Println("\nProcess interrupted by user before completion.")
			GetLogger().Debugf("Process interrupted by user before completion.")

			cancel()
			waitForCleanup(&wg, WG_CLEANUP_TIMEOUT_SEC*time.Second)

			return nil
		}
	}

	// cmd.Println("\nAll tasks completed successfully!")
	GetLogger().Debugf("All tasks completed successfully!")

	cancel()
	waitForCleanup(&wg, WG_CLEANUP_TIMEOUT_SEC*time.Second)

	// Entering here triggers 'defer cancel()', signaling background tasks to stop immediately
	return nil
}
