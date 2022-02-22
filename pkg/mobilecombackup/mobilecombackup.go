package mobilecombackup

import (
	"log"
	"os"
	"path/filepath"

	"github.com/phillipgreen/mobilecombackup/pkg/calls"
	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
)

type Result struct {
	Calls coalescer.Result
}

type Processor interface {
	Process(fileRoot string) (Result, error)
}

type processorState struct {
	outputDir     string
	callCoalescer coalescer.Coalescer
}

func coalesce(c coalescer.Coalescer, fileRoot string) (coalescer.Result, error) {
	var res coalescer.Result = coalescer.Result{Total: 0, New: 0}
	paths := make(chan string, 10)
	results := make(chan coalescer.Result, 10)

	// find all files to process
	go func() {
		filepath.Walk(fileRoot, func(path string, info os.FileInfo, err error) error {

			if info.IsDir() {
				return nil
			}
			var supports, serr = c.Supports(path)

			if err != nil {
				return serr
			}

			if supports {
				log.Printf("found path: %s\n", path)
				paths <- path
			}

			return nil
		})
		close(paths)
	}()

	go func() {
		// var keepGoing bool = true
		for {
			p, ok := <-paths
			if !ok {
				break
			}
			// keepGoing = ok
			var r, err = c.Coalesce(p)
			log.Printf("Coalesced [%s]: %v;%v", p, r, err)
			if err != nil {
				// todo
			} else {
				results <- r
			}
		}
    var err = c.Flush()
    if err != nil {
      log.Printf("Error on Flush: %v", err)
      
    }
		close(results)
	}()

	for {
		r, ok := <-results
		if !ok {
			break
		}
		res.Total = r.Total
		res.New += r.New

	}

	return res, nil
}

func (s *processorState) Process(fileRoot string) (Result, error) {
	var result Result

	var cResult, err = coalesce(s.callCoalescer, fileRoot)
	if err != nil {
		return result, err
	}

	return Result{cResult}, nil
}

func Init(rootPath string) Processor {
	return &processorState{
		rootPath,
		calls.Init(rootPath),
	}
}
