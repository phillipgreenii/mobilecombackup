package calls

import (
	"encoding/xml"
	//"os"
	//"strings"
  //"path/filepath"
	//"errors"
)

type Calls struct {
	XMLName xml.Name `xml:"calls"`
	Calls   []Call   `xml:"call"`
}

type Call struct {
	XMLName      xml.Name `xml:"call"`
	Number       string   `xml:"number,attr"`
	Duration     string   `xml:"duration,attr"`
	Date         int      `xml:"date,attr"`
	Type         string   `xml:"type,attr"`
	ReadableDate string   `xml:"readable_date,attr"`
	ContactName  string   `xml:"contact_name,attr"`
}

