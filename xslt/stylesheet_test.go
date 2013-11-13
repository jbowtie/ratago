package xslt

import (
	"fmt"
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

func runXslTest(t *testing.T, xslFile, inputXmlFile, outputXmlFile string) bool {
	style, _ := xmlReadFile(xslFile)
	input, _ := xmlReadFile(inputXmlFile)
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

func visit(path string, f os.FileInfo, err error) error {
	fmt.Printf("Visited: %s\n", path)
	return nil
}

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
	xslf := fmt.Sprintf("testdata/general/%v.xsl", xslFile)
	ii := fmt.Sprintf("testdata/docs/%v.xml", xslFile)
	oo := fmt.Sprintf("testdata/general/%v.out", xslFile)
	return runXslTest(t, xslf, ii, oo)
}

// Runs the general suite of libxslt regression tests
func TestXsltGeneral(t *testing.T) {
	//runGeneralXslTest(t, "array") // document('') needs to tweak XPath context somehow
	runGeneralXslTest(t, "bug-1-")
	runGeneralXslTest(t, "bug-2-")
	runGeneralXslTest(t, "bug-3-")
	runGeneralXslTest(t, "bug-4-")
	//runGeneralXslTest(t, "bug-5-") //seems to be an issue with for-each select=$ACTIONgrid
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
	//runGeneralXslTest(t, "bug-21-") // unparsed-entity-uri()
	runGeneralXslTest(t, "bug-22-")
	runGeneralXslTest(t, "bug-23-")
	runGeneralXslTest(t, "bug-24-")
	//runGeneralXslTest(t, "bug-25-") // encoding attr when UTF-8 explictly specified in doc
	runGeneralXslTest(t, "bug-26-")
	runGeneralXslTest(t, "bug-27-")
	runGeneralXslTest(t, "bug-28-")
	// runGeneralXslTest(t, "bug-29-") // document('href'); need to resolve to new source document
	runGeneralXslTest(t, "bug-30-")
	runGeneralXslTest(t, "bug-31-")
	runGeneralXslTest(t, "bug-32-")
	runGeneralXslTest(t, "bug-33-")
	//runGeneralXslTest(t, "bug-34-")
	runGeneralXslTest(t, "bug-35-")
	//runGeneralXslTest(t, "bug-36-") //xsl:include
	//runGeneralXslTest(t, "bug-37-") //xsl:include
	//runGeneralXslTest(t, "bug-38-") // document('')
	runGeneralXslTest(t, "bug-39-")
	//runGeneralXslTest(t, "bug-40-") //variable scope is globals when call-template is invoked
	//runGeneralXslTest(t, "bug-41-") //also avoid overwriting global variable using with-param
	//runGeneralXslTest(t, "bug-42-") //as 40 but for apply-templates
	//runGeneralXslTest(t, "bug-43-") //as 41 but for apply-templates
	//runGeneralXslTest(t, "bug-44-") // with-param
	//runGeneralXslTest(t, "bug-45-") // with-param
	runGeneralXslTest(t, "bug-46-")
	runGeneralXslTest(t, "bug-47-")
	runGeneralXslTest(t, "bug-48-")
	//runGeneralXslTest(t, "bug-49-") // global variable defined in terms of inner variable
	runGeneralXslTest(t, "bug-50-")
	//runGeneralXslTest(t, "bug-52") //unparsed-entity-uri
	//runGeneralXslTest(t, "bug-53") // depends on DTD processing of ATTLIST with default attribute
	runGeneralXslTest(t, "bug-54")
	runGeneralXslTest(t, "bug-55")
	//runGeneralXslTest(t, "bug-56") // unsure what's going on here
	runGeneralXslTest(t, "bug-57")
	runGeneralXslTest(t, "bug-59")
	//runGeneralXslTest(t, "bug-60")
	//runGeneralXslTest(t, "bug-61")
	runGeneralXslTest(t, "bug-62")
	//runGeneralXslTest(t, "bug-63")
	runGeneralXslTest(t, "bug-64")
	//runGeneralXslTest(t, "bug-65")
	//runGeneralXslTest(t, "bug-66")
	runGeneralXslTest(t, "bug-68")
	runGeneralXslTest(t, "bug-69")
	//runGeneralXslTest(t, "bug-100") // libxslt:test extension element
	runGeneralXslTest(t, "bug-101") // xsl:element with default namespace
	//runGeneralXslTest(t, "bug-102") // imported xsl:attribute-set
	//runGeneralXslTest(t, "bug-103") //copy-of needs to explicitly set empty namespace when needed
	//runGeneralXslTest(t, "bug-104") //copy-of should preserve attr prefix if plausible
	runGeneralXslTest(t, "bug-105")
	runGeneralXslTest(t, "bug-106") //copy-of
}
