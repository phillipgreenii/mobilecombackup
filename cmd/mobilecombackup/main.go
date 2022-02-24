package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/phillipgreen/mobilecombackup/pkg/mobilecombackup"
)

type config struct {
	repoPath       string
	pathsToProcess []string
}

func parseFlags(progname string, args []string) (conf *config, output string, err error) {
	flags := flag.NewFlagSet(progname, flag.ContinueOnError)
	var buf bytes.Buffer
	flags.SetOutput(&buf)

  flags.Usage = func() {
        fmt.Fprintf(flags.Output(), "Usage of %s [options] [pathToProcess1 ... pathToProcessN]:\n", progname)

        flags.PrintDefaults()
  }
  
	var c config
	flags.StringVar(&c.repoPath, "repo", ".", "path which contains repository")

	err = flags.Parse(args)
	if err != nil {
		return nil, buf.String(), err
	}
	c.pathsToProcess = flags.Args()
	return &c, buf.String(), nil
}

func validateConfig(conf *config) error {
	if len(conf.pathsToProcess) <= 0 {
		return errors.New("Atleast one path to process must be specified")
	}
	return nil
}

func doWork(conf *config) error {

	mcb, err := mobilecombackup.Init(conf.repoPath)
	if err != nil {
		return err
	}

	var errorCount int
	for _, path := range conf.pathsToProcess {
		result, err := mcb.Process(path)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Failure: %v\n", err.Error())
			errorCount += 1
		} else {
			fmt.Printf("Success: %v\n", result)
		}
	}
	if errorCount > 0 {
		return errors.New(fmt.Sprintf("Had %d failures", errorCount))
	} else {
		return nil
	}
}

func main() {
	exitCode := 0
	defer func() { os.Exit(exitCode) }()

	conf, output, err := parseFlags(os.Args[0], os.Args[1:])
	if err == flag.ErrHelp {
		fmt.Fprintln(os.Stderr, output)
		os.Exit(4)
	} else if err != nil {
		fmt.Fprintln(os.Stderr, "got error:", err)
		fmt.Fprintln(os.Stderr, "output:\n", output)
		os.Exit(3)
	}

	err = validateConfig(conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "got error:", err)
		os.Exit(2)
	}

	err = doWork(conf)
	if err != nil {
		fmt.Fprintln(os.Stderr, "got error:", err)
		os.Exit(1)
	}
}
