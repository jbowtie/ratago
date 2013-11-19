// The ratago command-line utility runs an input file through an XSLT stylesheet.
package main

import (
	"flag"
	"fmt"
	"github.com/jbowtie/ratago/xslt"
	"github.com/moovweb/gokogiri/xml"
	"io/ioutil"
	"os"
)

func xmlReadFile(filename string) (doc *xml.XmlDocument, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	doc, err = xml.Parse(data, xml.DefaultEncodingBytes, nil, xslt.XSLT_PARSE_OPTIONS, xml.DefaultEncodingBytes)
	return
}

func usage() {
	fmt.Fprintf(os.Stderr, "Usage: %s [options] STYLESHEET INPUT\n", os.Args[0])
	flag.PrintDefaults()
}

var indent = flag.Bool("indent", false, "Attempt to indent any XML output")

func main() {
	flag.Usage = usage
	flag.Parse()
	if flag.NArg() < 2 {
		usage()
		return
	}
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
	stylesheet, err := xslt.ParseStylesheet(style, xslfile)
	if err != nil {
		fmt.Println(err)
		return
	}

	options := xslt.StylesheetOptions{*indent, nil}

	output, err := stylesheet.Process(doc, options)
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(output)
}
