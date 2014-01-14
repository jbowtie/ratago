package xslt

import (
	"fmt"
	"github.com/moovweb/gokogiri/xml"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

// Simple naive test; primarily exists as a canary in case test helpers break
func TestNaive(t *testing.T) {
	xslFile := "testdata/test.xsl"
	inputXml := "testdata/test.xml"
	outputXml := "testdata/test.out"

	runXslTest(t, xslFile, inputXml, outputXml)
}

// Helper where actual checking occurs
func runXslTest(t *testing.T, xslFile, inputXmlFile, outputXmlFile string) bool {
	style, _ := xml.ReadFile(xslFile, xml.StrictParseOption)
	input, _ := xml.ReadFile(inputXmlFile, xml.StrictParseOption)
	outData, _ := ioutil.ReadFile(outputXmlFile)
	expected := string(outData)
	stylesheet, _ := ParseStylesheet(style, xslFile)
	testOptions := StylesheetOptions{false, nil}
	output, _ := stylesheet.Process(input, testOptions)
	if output != expected {
		t.Error(xslFile, "failed")
		fmt.Println("---- EXPECTED  ", xslFile, "----")
		fmt.Println(expected)
		fmt.Println("---- ACTUAL  ", xslFile, "----")
		fmt.Println(output)
		return false
	}
	return true
}

// check whether a file exists
func exists(path string) (bool, error) {
	_, err := os.Stat(path)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

// Runs the tests derived from the XSLT 1.0 specification examples
func TestXsltREC(t *testing.T) {
	var passed []string
	d, _ := os.Open("testdata/REC")
	fi, _ := d.Readdir(-1)
	for _, f := range fi {
		if f.Mode().IsRegular() && path.Ext(f.Name()) == ".xsl" {
			xslname := path.Join("testdata/REC", f.Name())
			b := xslname[0 : len(xslname)-4]
			inName := b + ".xml"
			outName := b + ".out"
			ii, _ := exists(inName)
			oo, _ := exists(outName)
			if ii && oo {
				ok := runXslTest(t, xslname, inName, outName)
				if ok {
					passed = append(passed, xslname)
				}
			} else {
				// can use this to debug if tests suddenly disappear
				//fmt.Println("Cannot test", xslname)
			}
		}
	}
	fmt.Println("passed", len(passed), "tests")
}

// Tests the first full example presented in the XSLT 1.0 spec
func TestXsltRECexample1(t *testing.T) {
	xslFile := "testdata/REC1/doc.xsl"
	inputXml := "testdata/REC1/doc.xml"
	outputXml := "testdata/REC1/result.xml"

	runXslTest(t, xslFile, inputXml, outputXml)
}

// Tests the second full example presented in the XSLT 1.0 spec
func TestXsltRECexample2(t *testing.T) {
	inputXml := "testdata/REC2/data.xml"

	runXslTest(t, "testdata/REC2/html.xsl", inputXml, "testdata/REC2/html.xml")
	runXslTest(t, "testdata/REC2/svg.xsl", inputXml, "testdata/REC2/svg.xml")
	runXslTest(t, "testdata/REC2/vrml.xsl", inputXml, "testdata/REC2/vrml.xml")
}

//convenience function to fix up the paths before running a test
func runGeneralXslTest(t *testing.T, xslFile string) bool {
	//we change into the general directory to duplicate env of libxslt test run
	// unparsed-entity-uri() in particular returns a result relative to the
	// current working directory.
	pwd, _ := os.Getwd()
	defer os.Chdir(pwd)
	_ = os.Chdir("testdata/general")
	xslf := fmt.Sprintf("%v.xsl", xslFile)
	ii := fmt.Sprintf("../docs/%v.xml", xslFile)
	oo := fmt.Sprintf("%v.out", xslFile)
	return runXslTest(t, xslf, ii, oo)
}

// Runs the general suite of libxslt regression tests
func TestXsltGeneral(t *testing.T) {
	//runGeneralXslTest(t, "items") //doesn't match pattern - how do we run test?

	runGeneralXslTest(t, "array") // document('')
	runGeneralXslTest(t, "character")
	//runGeneralXslTest(t, "date_add") // EXSL date functions
	runGeneralXslTest(t, "bug-1-")
	runGeneralXslTest(t, "bug-2-")
	runGeneralXslTest(t, "bug-3-")
	runGeneralXslTest(t, "bug-4-")
	//runGeneralXslTest(t, "bug-5-") // need to implement current()
	runGeneralXslTest(t, "bug-6-")
	runGeneralXslTest(t, "bug-7-")
	runGeneralXslTest(t, "bug-8-") //issue resolving namespaces in imported stylesheet
	runGeneralXslTest(t, "bug-9-")
	runGeneralXslTest(t, "bug-10-")
	runGeneralXslTest(t, "bug-11-")
	runGeneralXslTest(t, "bug-12-")
	runGeneralXslTest(t, "bug-13-")
	runGeneralXslTest(t, "bug-14-")
	runGeneralXslTest(t, "bug-15-")
	runGeneralXslTest(t, "bug-16-")
	runGeneralXslTest(t, "bug-17-")
	runGeneralXslTest(t, "bug-18-")
	runGeneralXslTest(t, "bug-19-")
	runGeneralXslTest(t, "bug-20-")
	runGeneralXslTest(t, "bug-21-") // unparsed-entity-uri()
	runGeneralXslTest(t, "bug-22-")
	runGeneralXslTest(t, "bug-23-")
	runGeneralXslTest(t, "bug-24-")
	//runGeneralXslTest(t, "bug-25-") // encoding attr when UTF-8 explictly specified in doc
	runGeneralXslTest(t, "bug-26-")
	runGeneralXslTest(t, "bug-27-")
	runGeneralXslTest(t, "bug-28-")
	runGeneralXslTest(t, "bug-29-") // document('href'); need to resolve to new source document
	runGeneralXslTest(t, "bug-30-")
	runGeneralXslTest(t, "bug-31-")
	runGeneralXslTest(t, "bug-32-")
	runGeneralXslTest(t, "bug-33-")
	runGeneralXslTest(t, "bug-35-")
	runGeneralXslTest(t, "bug-36-") //xsl:include
	runGeneralXslTest(t, "bug-37-") //xsl:include
	//runGeneralXslTest(t, "bug-38-") // handle copy-of() for namespace nodes
	runGeneralXslTest(t, "bug-39-")
	runGeneralXslTest(t, "bug-40-") //variable scope is global when call-template is invoked
	runGeneralXslTest(t, "bug-41-") //also avoid overwriting global variable using with-param
	runGeneralXslTest(t, "bug-42-") //as 40 but for apply-templates
	runGeneralXslTest(t, "bug-43-") //as 41 but for apply-templates
	runGeneralXslTest(t, "bug-44-") // with-param
	//runGeneralXslTest(t, "bug-45-") // ensure params/variables resolve in correct order
	runGeneralXslTest(t, "bug-46-")
	runGeneralXslTest(t, "bug-47-")
	runGeneralXslTest(t, "bug-48-")
	runGeneralXslTest(t, "bug-49-") // global variable defined in terms of inner variable
	runGeneralXslTest(t, "bug-50-")
	runGeneralXslTest(t, "bug-52") //unparsed-entity-uri with nodeset argument
	runGeneralXslTest(t, "bug-53") // depends on DTD processing of ATTLIST with default attribute
	runGeneralXslTest(t, "bug-54")
	runGeneralXslTest(t, "bug-55")
	//runGeneralXslTest(t, "bug-56") // unsure what's going on here
	runGeneralXslTest(t, "bug-57")
	runGeneralXslTest(t, "bug-59")
	runGeneralXslTest(t, "bug-60") // fallback for unknown XSL element
	//runGeneralXslTest(t, "bug-61") // format-number outputs NaN correctly
	runGeneralXslTest(t, "bug-62")
	//runGeneralXslTest(t, "bug-63") //resolve namespace nodes and relative paths
	runGeneralXslTest(t, "bug-64")
	//runGeneralXslTest(t, "bug-65") // libxslt:node-set, can't fix until document('href') works as expected
	//runGeneralXslTest(t, "bug-66") //current()
	runGeneralXslTest(t, "bug-68")
	runGeneralXslTest(t, "bug-69") // stylesheet and input in iso-8859-1
	//runGeneralXslTest(t, "bug-70") // key() - nodeset as arg 2
	//runGeneralXslTest(t, "bug-71") //only fails due to order of NS declarations; need to review spec on that
	runGeneralXslTest(t, "bug-72") //variables declared in RVT
	runGeneralXslTest(t, "bug-73")
	runGeneralXslTest(t, "bug-74")
	//runGeneralXslTest(t, "bug-75") //format-number()
	//runGeneralXslTest(t, "bug-76") //issue with count? or variable resolution?
	runGeneralXslTest(t, "bug-77") //handle spaces around OR
	runGeneralXslTest(t, "bug-78")
	runGeneralXslTest(t, "bug-79")
	//runGeneralXslTest(t, "bug-80") //fails due to ordering of attributes; review, possibly edit test
	//runGeneralXslTest(t, "bug-81") //rounding error in XPath calculation; might be caused by CGO conversion
	//runGeneralXslTest(t, "bug-82") //whitespace interactions; possibly not honoring global preserve-space
	runGeneralXslTest(t, "bug-83")
	runGeneralXslTest(t, "bug-84")
	//runGeneralXslTest(t, "bug-86") //getting some unnecessary duplication of namespaces declarations using copy-of
	//runGeneralXslTest(t, "bug-87") //matching on namespace node
	runGeneralXslTest(t, "bug-88")
	runGeneralXslTest(t, "bug-89") //fails with stricter parser
	//runGeneralXslTest(t, "bug-90")  // CDATA handling
	//runGeneralXslTest(t, "bug-91") // disable-output-escaping attribute
	//runGeneralXslTest(t, "bug-92") //libxml2 doesn't output the xs namespace here; why not?
	//runGeneralXslTest(t, "bug-93") // presence of xsl:output in imported stylesheets should cause effective merge
	//runGeneralXslTest(t, "bug-94") //variable/param confusion
	//runGeneralXslTest(t, "bug-95") //format-number
	//runGeneralXslTest(t, "bug-96") //cdata-section-elements
	runGeneralXslTest(t, "bug-97")
	//runGeneralXslTest(t, "bug-98")
	//runGeneralXslTest(t, "bug-99")
	//runGeneralXslTest(t, "bug-100") // libxslt:test extension element
	runGeneralXslTest(t, "bug-101") // xsl:element with default namespace
	//runGeneralXslTest(t, "bug-102") // imported xsl:attribute-set
	//runGeneralXslTest(t, "bug-103") //copy-of needs to explicitly set empty namespace when needed
	//runGeneralXslTest(t, "bug-104") //copy-of should preserve attr prefix if plausible
	runGeneralXslTest(t, "bug-105")
	runGeneralXslTest(t, "bug-106") //copy-of
	runGeneralXslTest(t, "bug-107")
	runGeneralXslTest(t, "bug-108")
	runGeneralXslTest(t, "bug-109") // disable-output-escaping
	runGeneralXslTest(t, "bug-110") //generate-id()
	//runGeneralXslTest(t, "bug-111") //exsl:node-set()
	//runGeneralXslTest(t, "bug-112") //exsl:node-set()
	runGeneralXslTest(t, "bug-113") // stylesheet and parser in ISO-8859-1
	runGeneralXslTest(t, "bug-114")
	//runGeneralXslTest(t, "bug-115") //exsl:node-set()
	runGeneralXslTest(t, "bug-116")
	//runGeneralXslTest(t, "bug-117")
	//runGeneralXslTest(t, "bug-118") //copy-of
	runGeneralXslTest(t, "bug-119")
	//runGeneralXslTest(t, "bug-120") //xsl:sort with data-type; interaction with copy-of?
	runGeneralXslTest(t, "bug-121")
	//runGeneralXslTest(t, "bug-122") //namespace nodes
	runGeneralXslTest(t, "bug-123")
	//runGeneralXslTest(t, "bug-124") //namespace declared with multiple prefixes; really a bug?
	//runGeneralXslTest(t, "bug-125") //unclear; needs further investigation
	//runGeneralXslTest(t, "bug-126") //tests for bugs in AVT parsing
	runGeneralXslTest(t, "bug-127")
	//runGeneralXslTest(t, "bug-128") //multiple keys with the same name; need to look at spec
	//runGeneralXslTest(t, "bug-129") //cdata-section-elements
	//runGeneralXslTest(t, "bug-130") //document('href') and import
	//runGeneralXslTest(t, "bug-131") // attribute-set combine import defs
	runGeneralXslTest(t, "bug-132")
	//runGeneralXslTest(t, "bug-133") // interaction between key, generate-id?
	//runGeneralXslTest(t, "bug-134") //key with literal string in use
	//runGeneralXslTest(t, "bug-135") // same as 134
	runGeneralXslTest(t, "bug-136")
	//runGeneralXslTest(t, "bug-137") // EXSLT func
	runGeneralXslTest(t, "bug-138")
	//runGeneralXslTest(t, "bug-139") //extra output of entity definitions (why?)
	runGeneralXslTest(t, "bug-140") // failed due to standalone
	runGeneralXslTest(t, "bug-141")
	//runGeneralXslTest(t, "bug-142") //lang() function
	runGeneralXslTest(t, "bug-143")
	runGeneralXslTest(t, "bug-144")
	runGeneralXslTest(t, "bug-145") //should result in no output (calling template that doesn't exist)
	//runGeneralXslTest(t, "bug-146") // funny looking key definition plus encoding issue
	//runGeneralXslTest(t, "bug-147") //looks like import precedence related
	runGeneralXslTest(t, "bug-148")
	runGeneralXslTest(t, "bug-149")
	//runGeneralXslTest(t, "bug-150") //scoping of namespace definitions on literal result elements
	runGeneralXslTest(t, "bug-151") // outputs just the declaration; should be nothing
	//runGeneralXslTest(t, "bug-152") //libxml2 inserts a content-type meta tag; unsure why
	runGeneralXslTest(t, "bug-153") //document('href') and current()
	runGeneralXslTest(t, "bug-154") //should have no output
	runGeneralXslTest(t, "bug-155")
	runGeneralXslTest(t, "bug-156")
	runGeneralXslTest(t, "bug-157")
	runGeneralXslTest(t, "bug-158")
	//runGeneralXslTest(t, "bug-159") //escape entities appropriately if encoding=ascii
	//runGeneralXslTest(t, "bug-160") // match criteria seems to be the issue here
	runGeneralXslTest(t, "bug-161")
	runGeneralXslTest(t, "bug-163")
	runGeneralXslTest(t, "bug-164")
	//runGeneralXslTest(t, "bug-165") //xpath with unresolvable variable in predicate; differs in number of blank lines produced
	//runGeneralXslTest(t, "bug-166") //need to look closer; slow and much output!
	runGeneralXslTest(t, "bug-167")
	//runGeneralXslTest(t, "bug-168") //looks like AVT torture test
	//runGeneralXslTest(t, "bug-169") // not selecting correct output encoding?
	runGeneralXslTest(t, "bug-170")
	runGeneralXslTest(t, "bug-171")
	//runGeneralXslTest(t, "bug-172") //seems to be bug in xsl:choose (matches when test but no output)
	//runGeneralXslTest(t, "bug-173") //extra newline on output?
	//runGeneralXslTest(t, "bug-174") //exslt:func
	//runGeneralXslTest(t, "bug-175") //wrong output encoding/doctype for html output
	runGeneralXslTest(t, "bug-176")
	runGeneralXslTest(t, "bug-177") //should not create namespace declaration for built-in xml namespace
	//runGeneralXslTest(t, "bug-178") //exslt:func
	//runGeneralXslTest(t, "bug-179") // xsl:element/@namespace don't need to explicitly create namespace already in scope
	//runGeneralXslTest(t, "bug-180") //expects no output
	//runGeneralXslTest(t, "bug-181") //xsl:text missing from output
	//runGeneralXslTest(t, "bug-182") //text()[2] should match something
}

// sample usage for parse stylesheet
func ExampleParseStylesheet() {
	//parse the stylesheet
	style, _ := xml.ReadFile("testdata/test.xsl", xml.StrictParseOption)
	stylesheet, _ := ParseStylesheet(style, "testdata/test.xsl")

	//process the input
	input, _ := xml.ReadFile("testdata/test.xml", xml.StrictParseOption)
	output, _ := stylesheet.Process(input, StylesheetOptions{false, nil})
	fmt.Println(output)
}
