package main

import (
	"flag"
	"fmt"
	"github.com/jbowtie/ratago/xslt"
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

func main() {
	flag.Parse()
	//set some prefs based on flags
	xslfile := flag.Arg(0)
	inxml := flag.Arg(1)

	style, err := xmlReadFile(xslfile)
	if err != nil {
		fmt.Println(err)
		return
	}

	doc, err := xmlReadFile(inxml)
	if err != nil {
		fmt.Println(err)
		return
	}

	//TODO: register some extensions (EXSLT, testing, debug)
	//TODO: process XInclude if enabled
	stylesheet := xslt.ParseStylesheet(style, xslfile)
	output := stylesheet.Process(doc)
	fmt.Println(output)
}
