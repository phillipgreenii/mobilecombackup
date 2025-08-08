package mobilecombackup

// TODO: This is legacy code that needs to be refactored to use the new import architecture
// Temporarily commenting out to allow build

/*
import (
	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
)

type Result struct {
	Calls coalescer.Result
}
*/

type Result struct {
	// Placeholder for now
}

type Processor interface {
	Process(fileRoot string) (Result, error)
}
