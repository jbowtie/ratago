package xslt

import (
	"github.com/ThomsonReutersEikon/gokogiri/xml"
	"io/ioutil"
)

func xmlReadFile(filename string) (doc *xml.XmlDocument, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	doc, err = xml.Parse(data, xml.DefaultEncodingBytes, nil, xml.StrictParseOption, xml.DefaultEncodingBytes)
	return
}
