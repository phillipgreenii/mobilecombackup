package coalescer

import (
    "errors"
)

type Result struct {
  totalCalls int
  newCalls int
  totalMessages int
  newMessages int
}

type Coalescer interface {
  Coalesce(rootDir string) (Result, error)
}

type backup struct {
  outputDir string
}

func (b *backup) Coalesce(rootDir string) (Result, error) {
  var result Result
  return result,  errors.New("not implemented")
} 

func Init(rootDir string) Coalescer {
  return &backup{rootDir}
}