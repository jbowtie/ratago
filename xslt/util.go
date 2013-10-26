package xslt

import (
	"github.com/moovweb/gokogiri"
	"github.com/moovweb/gokogiri/xml"
	"io/ioutil"
)

func xmlReadFile(filename string) (doc *xml.XmlDocument, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	doc, err = gokogiri.ParseXml(data)
	return
}
