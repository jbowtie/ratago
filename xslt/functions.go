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
	//element-available
	//function-available
	//document
	//id
	//current
	//lang
	//generate-id
	//unparsed-entity-uri
	//format-number
}

type Nodeset []xml.Node

type Key struct {
	nodes map[string]Nodeset
	use   string
	match string
}

/*
func (key *Key) Evaluate() {
    c := CompileMatch()
    for n in doc.Nodes {
        if c.Matches(n) {
            Nodes = append(Nodes, n)
        }
    }
}
*/

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
	val := args[1].(string)
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
		return "http://github.com/jbowtie"
	default:
		fmt.Println("EXEC system-property", args[0])
	}
	return nil
}
