package main

import (
	"fmt"
  "os"
  "github.com/phillipgreen/mobilecombackup/pkg/mobilecombackup"
)

func main() {
  exitCode := 0
  defer func() { os.Exit(exitCode) }()
  
  mcb := mobilecombackup.Init("./testdata/archive")

  result,err := mcb.Process("./testdata/to_process")
  if err != nil {
    fmt.Fprintf(os.Stderr, "Failure: %v\n", err.Error())
    exitCode = 1
  } else {
    fmt.Printf("Success: %v\n",result)
  }
}
