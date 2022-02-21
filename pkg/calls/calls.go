package calls

import (
	"encoding/xml"
	"os"
	"strings"
  "path"
  "path/filepath"
	//"errors"
  "github.com/phillipgreen/mobilecombackup/pkg/coalescer"
)

type Key struct {
  	Number       string  
	Duration     string 
	Date         int  
	Type         string 
}

func (call *Call) key() Key {
	return Key{call.Number, call.Duration, call.Date,call.Type}
}

type backup struct {
	outputDir string
	calls     map[Key]Call
}

func (b *backup) ingest(file *os.File) error {
	// load file
	decoder := xml.NewDecoder(file)
	for {
		t, _ := decoder.Token()
		if t == nil {
			break
		}
		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "call" {
				var call Call
				decoder.DecodeElement(&call, &se)
				var k = call.key()
				if _, ok := b.calls[k]; !ok {
					b.calls[k] = call
				}
			}
		default:
		}
	}
  return nil
}

func (b *backup) Supports(filePath string) (bool, error) {

	return strings.Contains(path.Base(filePath), "call"), nil
}

func (b *backup) Coalesce(filePath string) (coalescer.Result, error) {
	var result coalescer.Result
	var initialTotalCalls int = len(b.calls)

	xmlFile, err := os.Open(filePath)
	// if we os.Open returns an error then handle it
	if err != nil {
		return result, err
	}
	defer xmlFile.Close()

  err = b.ingest(xmlFile)
  	if err != nil {
		return result, err
	}

	result.Total = len(b.calls)
	result.New = len(b.calls) - initialTotalCalls
	return result, nil
}

func Init(rootDir string) coalescer.Coalescer {
  var backup = backup{rootDir, map[Key]Call{}}
  var cf = filepath.Join(rootDir,"calls.xml")
  if _, err := os.Stat(cf); err == nil {
    backup.Coalesce(cf)
  }
 
	return &backup
}
