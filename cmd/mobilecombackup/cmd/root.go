package cmd

import (
	"fmt"

	"github.com/phillipgreenii/mobilecombackup/pkg/logging"
	"github.com/spf13/cobra"
)

var (
	version   string
	quiet     bool
	verbose   bool
	repoRoot  string
	logFormat string
	logger    logging.Logger
)

var rootCmd = &cobra.Command{
	Use:   "mobilecombackup",
	Short: "mobilecombackup processes call logs and SMS/MMS messages",
	Long: `mobilecombackup processes call logs and SMS/MMS messages from 
mobile phone backup files, removing duplicates and organizing by year.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		// If args provided without a valid subcommand, show error
		if len(args) > 0 {
			return fmt.Errorf("unknown command %q for %q", args[0], cmd.CommandPath())
		}
		// Show help when no subcommand provided
		return cmd.Help()
	},
	SilenceErrors: false,
	SilenceUsage:  false,
}

// Execute runs the root command for the CLI
func Execute() error {
	return rootCmd.Execute()
}

// SetVersion sets the version for the CLI
func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.SetVersionTemplate("mobilecombackup version {{.Version}}\n")

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress non-error output")
	rootCmd.PersistentFlags().BoolVar(&verbose, "verbose", false, "Enable verbose logging")
	rootCmd.PersistentFlags().StringVar(&repoRoot, "repo-root", ".", "Path to repository root")
	rootCmd.PersistentFlags().StringVar(&logFormat, "log-format", "console", "Log output format (console or json)")

	// Initialize logger
	initLogger()
}

// initLogger initializes the global logger based on command-line flags
func initLogger() {
	config := &logging.LogConfig{
		Format:     logging.FormatConsole,
		TimeStamps: true,
		Color:      true,
	}

	// Set log level based on flags
	switch {
	case quiet:
		config.Level = logging.LevelError
	case verbose:
		config.Level = logging.LevelDebug
	default:
		config.Level = logging.LevelInfo
	}

	// Set log format
	switch logFormat {
	case "json":
		config.Format = logging.FormatJSON
		config.Color = false
	case "console":
		config.Format = logging.FormatConsole
	}

	logger = logging.NewLogger(config).WithComponent("cli")
}

// GetLogger returns the configured logger for use by subcommands
func GetLogger() logging.Logger {
	return logger
}

// UpdateLogger re-initializes the logger with current flag values
func UpdateLogger() {
	initLogger()
}

// PrintError prints an error message to the logger
// Deprecated: Use GetLogger().Error() instead
func PrintError(format string, args ...interface{}) {
	logger.Error().Msgf(format, args...)
}

// PrintInfo prints informational messages
// Deprecated: Use GetLogger().Info() instead
func PrintInfo(format string, args ...interface{}) {
	logger.Info().Msgf(format, args...)
}

// PrintVerbose prints verbose messages
// Deprecated: Use GetLogger().Debug() instead
func PrintVerbose(format string, args ...interface{}) {
	logger.Debug().Msgf(format, args...)
}

// PrintDebug prints debug messages
// Deprecated: Use GetLogger().Debug() instead
func PrintDebug(format string, args ...interface{}) {
	logger.Debug().Msgf(format, args...)
}
