package xslt

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestNaive(t *testing.T) {
	//proper way to do this:
	// foreach test
	// load input, stylesheet
	// process
	// compare to output
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
