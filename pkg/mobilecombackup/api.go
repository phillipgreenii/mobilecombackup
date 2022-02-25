package mobilecombackup

import (
	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
)

type Result struct {
	Calls coalescer.Result
}

type Processor interface {
	Process(fileRoot string) (Result, error)
}
