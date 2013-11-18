package xslt

import (
	"fmt"
	"github.com/moovweb/gokogiri/xml"
	"github.com/moovweb/gokogiri/xpath"
	"unsafe"
)

func (style *Stylesheet) RegisterXsltFunctions() {
	style.Functions["key"] = XsltKey
	style.Functions["system-property"] = XsltSystemProperty
	style.Functions["document"] = XsltDocumentFn
	//element-available
	//function-available - possibly internal to Gokogiri?
	//id - see implementation in match.go
	//current - need to set appropriately in context
	//lang
	//generate-id - just use pointer to node as string?
	//unparsed-entity-uri - requires Gokogiri to expose API
	//format-number - requires handling decimal-format
}

type Nodeset []xml.Node

type Key struct {
	nodes map[string]Nodeset
	use   string
	match string
}

func (n Nodeset) ToPointers() (pointers []unsafe.Pointer) {
	for _, node := range n {
		pointers = append(pointers, node.NodePtr())
	}
	return
}

// Implementation of key() from XSLT spec
func XsltKey(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) < 2 {
		return nil
	}
	// always convert to string
	name := args[0].(string)
	// convert to string (TODO: unless nodeset)
	val := ""
	switch v := args[1].(type) {
	case string:
		val = v
	case []unsafe.Pointer:
		// nodeset; see xsl:key spec for how to handle this
	}
	//get the execution context
	c := context.(*ExecutionContext)
	//look up the key
	k, ok := c.Style.Keys[name]
	if !ok {
		return nil
	}
	result, _ := k.nodes[val]
	//return the nodeset
	return result.ToPointers()
}

// Implementation of system-property() from XSLT spec
func XsltSystemProperty(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) < 1 {
		return nil
	}
	switch args[0].(string) {
	case "xsl:version":
		return 1.0
	case "xsl:vendor":
		return "John C Barstow"
	case "xsl:vendor-url":
		return "http://github.com/jbowtie/ratago"
	default:
		fmt.Println("EXEC system-property", args[0])
	}
	return nil
}

//Implementation of document() from XSLT spec
func XsltDocumentFn(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) < 1 {
		return nil
	}
	c := context.(*ExecutionContext)

	switch doc := args[0].(type) {
	case string:
		if doc == "" {
			return []unsafe.Pointer{c.Style.Doc.DocPtr()}
		}
		return nil
	}
	return nil
}
