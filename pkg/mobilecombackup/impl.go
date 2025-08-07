package mobilecombackup

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
)

type processorState struct {
	outputDir     string
	callCoalescer coalescer.Coalescer
}

func coalesce(c coalescer.Coalescer, fileRoot string) (coalescer.Result, error) {
	res := coalescer.Result{Total: 0, New: 0}

	// find all files to process
	paths := searchPath(c, fileRoot)
	results := coalescePaths(c, paths)

	for r := range results {
		res.Total = r.Total
		res.New += r.New
	}

	return res, nil
}

func searchPath(c coalescer.Coalescer, root string) <-chan string {
	paths := make(chan string, 10)

	go func() {
		err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {

			if info.IsDir() {
				// skip directories
				return nil
			}
			var supports, serr = c.Supports(path)

			if err != nil {
				return serr
			}

			if supports {
				paths <- path
			}

			return nil
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "while walking", root, "got error:", err)
		}
		close(paths)
	}()

	return paths
}

func coalescePaths(c coalescer.Coalescer, paths <-chan string) <-chan coalescer.Result {
	results := make(chan coalescer.Result, 10)

	go func() {
		for {
			p, ok := <-paths
			if !ok {
				break
			}
			var r, err = c.Coalesce(p)
			if err != nil {
				log.Printf("Error on Coalescing [%s]: %v", p, err)
			} else {
				log.Printf("Coalesced [%s]: %v", p, r)
				results <- r
			}
		}
		var err = c.Flush()
		if err != nil {
			log.Printf("Error on Flush: %v", err)
		}
		close(results)
	}()
	return results
}

func (s *processorState) Process(fileRoot string) (Result, error) {
	var result Result

	var cResult, err = coalesce(s.callCoalescer, fileRoot)
	if err != nil {
		return result, err
	}

	return Result{cResult}, nil
}

func Init(rootPath string) (Processor, error) {
	// TODO: This implementation needs to be updated to match the new architecture
	// The old coalescer pattern doesn't match the new CallsReader/SMSReader pattern
	return nil, fmt.Errorf("Init not implemented - architecture mismatch")
}
