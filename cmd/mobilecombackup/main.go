package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/phillipgreen/mobilecombackup/pkg/mobilecombackup"
)

func main() {
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	repoPath := flag.String("repo", ".", "path which contains repository")

	flag.Parse()
	pathsToProcess := flag.Args()

  if len(pathsToProcess) <= 0 {
    	fmt.Fprintf(os.Stderr, "Atleast one path to process must be specified\n")
			exitCode = 1
    return
  }

	mcb := mobilecombackup.Init(*repoPath)

	for _, path := range pathsToProcess {
		result, err := mcb.Process(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failure: %v\n", err.Error())
			exitCode = 1
		} else {
			fmt.Printf("Success: %v\n", result)
		}
	}
}
