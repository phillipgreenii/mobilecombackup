package coalescer

import (
)

type Result struct {
	Total    int
	New      int
}

type Coalescer interface {
	Coalesce(filePath string) (Result, error)
  Supports(filePath string) (bool, error)
  Flush() error
}

