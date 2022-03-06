package mobilecombackup

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"os"
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

	mcb, err := Init(conf.repoPath)
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
		return fmt.Errorf("Had %d failures", errorCount)
	} else {
		return nil
	}
}

func Run(args []string) (exitCode int, output *string, err error) {
	conf, o, err := parseFlags(args[0], args[1:])
	if err == flag.ErrHelp {
		return 4, nil, err
	} else if err != nil {
		return 3, &o, err
	}

	err = validateConfig(conf)
	if err != nil {
		return 2, nil, err
	}

	err = doWork(conf)
	if err != nil {
		return 1, nil, err
	}

	return 0, nil, nil
}
