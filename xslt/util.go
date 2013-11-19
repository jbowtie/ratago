package xslt

import (
	"github.com/moovweb/gokogiri/xml"
	"io/ioutil"
)

/*
   XML_PARSE_RECOVER = 1 // recover on errors
   XML_PARSE_NOENT = 2 // substitute entities
   XML_PARSE_DTDLOAD = 4 // load the external subset
   XML_PARSE_DTDATTR = 8 // default DTD attributes
   XML_PARSE_DTDVALID = 16 // validate with the DTD
   XML_PARSE_NOERROR = 32 // suppress error reports
   XML_PARSE_NOWARNING = 64 // suppress warning reports
   XML_PARSE_PEDANTIC = 128 // pedantic error reporting
   XML_PARSE_NOBLANKS = 256 // remove blank nodes
   XML_PARSE_SAX1 = 512 // use the SAX1 interface internally
   XML_PARSE_XINCLUDE = 1024 // Implement XInclude substitition
   XML_PARSE_NONET = 2048 // Forbid network access
   XML_PARSE_NODICT = 4096 // Do not reuse the context dictionnary
   XML_PARSE_NSCLEAN = 8192 // remove redundant namespaces declarations
   XML_PARSE_NOCDATA = 16384 // merge CDATA as text nodes
   XML_PARSE_NOXINCNODE = 32768 // do not generate XINCLUDE START/END nodes
   XML_PARSE_COMPACT = 65536 // compact small text nodes; no modification of the tree allowed afterwards (will possibly crash if you try to modify the tree)
   XML_PARSE_OLD10 = 131072 // parse using XML-1.0 before update 5
   XML_PARSE_NOBASEFIX = 262144 // do not fixup XINCLUDE xml//base uris
   XML_PARSE_HUGE = 524288 // relax any hardcoded limit from the parser
   XML_PARSE_OLDSAX = 1048576 // parse using SAX2 interface before 2.7.0
   XML_PARSE_IGNORE_ENC = 2097152 // ignore internal document encoding hint
   XML_PARSE_BIG_LINES = 4194304 // Store big lines numbers in text PSVI field
*/
const XSLT_PARSE_OPTIONS xml.ParseOption = 2 | 4 | 8 | 16384

func xmlReadFile(filename string) (doc *xml.XmlDocument, err error) {
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		return
	}
	doc, err = xml.Parse(data, xml.DefaultEncodingBytes, nil, XSLT_PARSE_OPTIONS, xml.DefaultEncodingBytes)
	return
}
