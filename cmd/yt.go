package cmd

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

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

func (y *YTDownloader) Download(ctx context.Context, p *tea.Program, logChan chan string,
	errChan chan error) error {
	GetLogger().Debugf("Number of tracks to process: %d\n", len(y.Tasks))

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

		if err := youtube.ProcessTask(taskCtx, task, logChan); err != nil {
			errChan <- err
			p.Send(doneMsg{})
			return err
		}

		taskCancel()
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
