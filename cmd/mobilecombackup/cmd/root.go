package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var (
	version  string
	quiet    bool
	repoRoot string
)

var rootCmd = &cobra.Command{
	Use:   "mobilecombackup",
	Short: "A tool for processing mobile phone backup files",
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

func Execute() error {
	return rootCmd.Execute()
}

func SetVersion(v string) {
	version = v
	rootCmd.Version = v
}

func init() {
	rootCmd.SetVersionTemplate("mobilecombackup version {{.Version}}\n")

	// Global flags
	rootCmd.PersistentFlags().BoolVar(&quiet, "quiet", false, "Suppress non-error output")
	rootCmd.PersistentFlags().StringVar(&repoRoot, "repo-root", ".", "Path to repository root")
}

// Helper functions for error handling
func PrintError(format string, args ...interface{}) {
	fmt.Fprintf(os.Stderr, "Error: "+format+"\n", args...)
}
