package calls

import (
	"encoding/xml"
	"fmt"
	"github.com/phillipgreen/mobilecombackup/pkg/coalescer"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

type Key struct {
	Number   string
	Duration string
	Date     int
	Type     string
}

func (call *Call) key() Key {
	return Key{call.Number, call.Duration, call.Date, call.Type}
}

type backup struct {
	outputDir string
	calls     map[Key]Call
}

type multierror struct {
	msg    string
	errors []error
}

func (m *multierror) Error() string {
	var sb strings.Builder
	sb.WriteString(m.msg)
	for _, m := range m.errors {
		sb.WriteString("\n\t")
		sb.WriteString(m.Error())
	}
	return sb.String()
}

func (b *backup) ingest(file *os.File) error {
	// load file
	decoder := xml.NewDecoder(file)
	errs := make([]error, 20)
	for {
		t, err := decoder.Token()
		if err == io.EOF || t == nil {
			break
		}
		if err != nil {
			errs = append(errs, err)
			break
		}

		switch se := t.(type) {
		case xml.StartElement:
			if se.Name.Local == "call" {
				var call Call
				err := decoder.DecodeElement(&call, &se)
				if err != nil {
					errs = append(errs, err)
					break
				}
				var k = call.key()
				if _, ok := b.calls[k]; !ok {
					b.calls[k] = call
				}
			}
		default:
		}
	}
	if len(errs) > 0 {
		return &multierror{msg: fmt.Sprintf("Error parsing %s", file.Name()), errors: errs}
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

type ByDate []Call

func (a ByDate) Len() int           { return len(a) }
func (a ByDate) Less(i, j int) bool { return a[i].Date < a[j].Date }
func (a ByDate) Swap(i, j int)      { a[i], a[j] = a[j], a[i] }

func (b *backup) Flush() error {
	xmlFile, err := os.Create(b.BackingFile())
	// if we os.Open returns an error then handle it
	if err != nil {
		return err
	}
	defer xmlFile.Close()

	// convert map to list
	var calls []Call = make([]Call, 0, len(b.calls))
	for _, value := range b.calls {
		calls = append(calls, value)
	}
	// sort list
	sort.Sort(ByDate(calls))
	// build xml container
	var wrappedData = Calls{Calls: calls, Count: len(calls)}
	out, err := xml.MarshalIndent(wrappedData, "", "\t")
	if err != nil {
		return err
	}
	_, err = xmlFile.WriteString(xml.Header)
	if err != nil {
		return err
	}
	_, err = xmlFile.WriteString("<?xml-stylesheet type=\"text/xsl\" href=\"calls.xsl\"?>\n")
	if err != nil {
		return err
	}
	_, err = xmlFile.Write(out)
	if err != nil {
		return err
	}

	return nil
}

func (b *backup) BackingFile() string {
	return filepath.Join(b.outputDir, "calls.xml")
}

func Init(rootDir string) coalescer.Coalescer {
	var backup = backup{rootDir, map[Key]Call{}}
	var cf = backup.BackingFile()
	var err error
	if _, err := os.Stat(cf); err == nil {
		_, err = backup.Coalesce(cf)
	}
	if err != nil {
		panic(err.Error())
	}

	return &backup
}
