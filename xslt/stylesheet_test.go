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
				fmt.Println("Cannot test", xslname)
			}
		}
	}
	//for _, p := range passed {
	//	fmt.Println("PASSED", p)
	//}
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
	//runGeneralXslTest(t, "bug-8-")
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
	//runGeneralXslTest(t, "bug-100") // libxslt:test extension element
	runGeneralXslTest(t, "bug-101") // xsl:element with default namespace
	//runGeneralXslTest(t, "bug-102") // imported xsl:attribute-set
	//runGeneralXslTest(t, "bug-103") //copy-of needs to explcitly set empty namespace when needed
	//runGeneralXslTest(t, "bug-104") //copy-of should preserve attr prefix if plausible
	runGeneralXslTest(t, "bug-105")
	runGeneralXslTest(t, "bug-106") //copy-of
}
