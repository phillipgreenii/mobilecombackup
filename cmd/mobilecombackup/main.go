// Package main provides the mobilecombackup CLI application.
package main

import (
	"os"

	"github.com/phillipgreenii/mobilecombackup/cmd/mobilecombackup/cmd"
)

var (
	// Version is set via ldflags during build
	Version   = "dev"
	BuildTime = "unknown"
)

func main() {
	cmd.SetVersion(Version)
	if err := cmd.Execute(); err != nil {
		os.Exit(1)
	}
}
