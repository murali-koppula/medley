package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	logging "gopkg.in/op/go-logging.v1"
)

var rootCmd = &cobra.Command{
	Version: commandVersion,
	Use:     "medley",
	Short:   "Interactive pipeline for automated media downloading and post-processing.",
	Long: `
A high-performance, interactive Go pipeline for downloading, re-encoding, and metadata-tagging
media streams.

MeDLey automates the tedious steps of fetching media, converting formats, embedding artwork, and
injecting tags into your music library — all rendered inside a live-updating terminal interface.
`,
	Example: `
# See "medley youtube --help" for help specific to youtube command.
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println(cmd.UsageString())
		return nil
	},
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		cmd.SetOut(cmd.OutOrStdout())

		cacheDir, err := os.UserCacheDir()
		if err != nil {
			return fmt.Errorf("failed to get cache dir: %w", err)
		}

		logDir := filepath.Join(cacheDir, "medley")
		if err := os.MkdirAll(logDir, 0700); err != nil {
			return fmt.Errorf("failed to create log dir: %w", err)
		}

		logFile, err := os.OpenFile(
			filepath.Join(logDir, "medley.log"),
			os.O_CREATE|os.O_WRONLY|os.O_TRUNC,
			0666,
		)

		if err != nil {
			return fmt.Errorf("failed to open log file: %w", err)
		}

		var format = logging.MustStringFormatter(
			`%{time:2006-01-02 15:04:05} %{shortfunc} [%{level:.4s}] %{message}`,
		)

		fileBackend := logging.NewLogBackend(logFile, "", 0)
		fileFormatter := logging.NewBackendFormatter(fileBackend, format)
		backendLeveled := logging.AddModuleLevel(fileFormatter)

		if tokens.verbose {
			backendLeveled.SetLevel(logging.DEBUG, "")
		} else {
			backendLeveled.SetLevel(logging.INFO, "")
		}

		logging.SetBackend(backendLeveled)
		return nil
	},
	SilenceUsage: true,
}

func Command() *cobra.Command {
	return rootCmd
}

func Execute() error {
	if err := rootCmd.Execute(); err != nil {
		GetLogger().Errorf("Error: %s\n", err.Error())
		os.Exit(1)
	}
	return nil
}

func init() {
	rootCmd.SetUsageTemplate(strings.Replace((&cobra.Command{}).UsageTemplate(),
		"Global Flags", "Flags Applying to All Commands", -1))
	rootCmd.SetVersionTemplate("Current Version: {{.Version}}\n")

	rootCmd.PersistentFlags().BoolVarP(&tokens.verbose, "verbose", "V", false, "Verbose mode")
	rootCmd.PersistentFlags().StringVarP(&tokens.mediaHome, "home", "H", "./media",
		"Root directory where media will be saved")
}
