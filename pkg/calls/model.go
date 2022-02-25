package calls

import (
	"encoding/xml"
)

type Calls struct {
	XMLName xml.Name `xml:"calls"`
	Calls   []Call   `xml:"call"`
	Count   int      `xml:"count,attr"`
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
