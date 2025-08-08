package main

import (
	"os"

	"github.com/phillipgreen/mobilecombackup/cmd/mobilecombackup/cmd"
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
