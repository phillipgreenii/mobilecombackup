package main

import (
	"fmt"
	"os"

	"github.com/phillipgreen/mobilecombackup/pkg/mobilecombackup"
)


func main() {
	exitCode, output, err := mobilecombackup.Run(os.Args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "got error:", err)
		if output != nil {
			fmt.Fprintln(os.Stderr, "output:\n", output)
		}
		os.Exit(exitCode)
	}
}
