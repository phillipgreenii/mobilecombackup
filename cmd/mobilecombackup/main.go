package main

import (
	"fmt"
  "os"
  "github.com/phillipgreen/mobilecombackup/pkg/coalescer"
)

func main() {
  exitCode := 0
  defer func() { os.Exit(exitCode) }()
  
  c := coalescer.Init("./testdata/archive/")
  result, err := c.Coalesce("./testdata/to_process/calls-test.xml")
  if err != nil {
    fmt.Fprintf(os.Stderr, "Failure: %v\n", err.Error())
    exitCode = 1
  } else {
    fmt.Printf("Success: %v\n",result)
  }
}
