package xslt

import (
	"fmt"
	"math"
	"unsafe"

	"github.com/jbowtie/gokogiri/xml"
	"github.com/jbowtie/gokogiri/xpath"
)

func (style *Stylesheet) RegisterXsltFunctions() {
	// id and lang are built into libxml2, don't need to be registered
	style.Functions["{}document"] = XsltDocumentFn
	style.Functions["{}generate-id"] = XsltGenerateId
	style.Functions["{}key"] = XsltKey
	style.Functions["{}system-property"] = XsltSystemProperty
	style.Functions["{}unparsed-entity-uri"] = XsltUnparsedEntityUri
	style.Functions["{}current"] = XsltCurrent
	style.Functions["{}element-available"] = XsltElementAvailable
	style.Functions["{}function-available"] = XsltFunctionAvailable
	//format-number - requires handling decimal-format

	style.Functions["{http://xmlsoft.org/XSLT/namespace}node-set"] = EXSLTnodeset
	style.Functions["{http://exslt.org/common}node-set"] = EXSLTnodeset
	style.Functions["{http://exslt.org/math}constant"] = EXSLTmathconstant
	style.Functions["{http://exslt.org/math}sin"] = EXSLTmathsin
	style.Functions["{http://exslt.org/math}cos"] = EXSLTmathcos
	style.Functions["{http://exslt.org/math}abs"] = EXSLTmathabs
}

type Key struct {
	nodes map[string]xml.Nodeset
	use   string
	match string
}

// Implementation of key() from XSLT spec
func XsltKey(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) < 2 {
		return nil
	}
	// always convert to string
	name := argValToString(args[0])

	// convert to string
	val := argValToString(args[1])
	/*
		val := ""
		switch v := args[1].(type) {
		case string:
			val = v
		case []unsafe.Pointer:
			// nodeset; see xsl:key spec for how to handle this
		}*/
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
			nodeset := xml.Nodeset{c.Style.Doc}
			return nodeset.ToPointers()
		}
		input := c.FetchInputDocument(doc, false)
		if input != nil {
			nodeset := xml.Nodeset{input}
			return nodeset.ToPointers()
		}
		return nil
	case []unsafe.Pointer:
		n := xml.NewNode(doc[0], nil)
		location := n.Content()
		input := c.FetchInputDocument(location, true)
		if input != nil {
			nodeset := xml.Nodeset{input}
			return nodeset.ToPointers()
		}
		fmt.Println("DOCUMENT", location)
	}
	return nil
}

// Implementation of generate-id() from XSLT spec
func XsltGenerateId(context xpath.VariableScope, args []interface{}) interface{} {
	// should be 0 or 1 argument
	if len(args) > 1 {
		return nil
	}

	//c := context.(*ExecutionContext)
	if len(args) < 1 {
		fmt.Println("GENERATE-ID for current")
		return "N" //id of context node
	}

	switch v := args[0].(type) {
	case []unsafe.Pointer:
		if len(v) == 0 {
			return nil
		}
		out := fmt.Sprintf("N%v", uintptr(v[0]))
		return out
	default:
		return nil
	}
}

// Implementation of unparsed-entity-uri() from XSLT spec
func XsltUnparsedEntityUri(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) < 1 {
		return nil
	}
	c := context.(*ExecutionContext)
	name := argValToString(args[0])
	val := c.Source.UnparsedEntityURI(name)
	return val
}

// Implementation of current() from XSLT spec
func XsltCurrent(context xpath.VariableScope, args []interface{}) interface{} {
	c := context.(*ExecutionContext)
	n := xml.Nodeset{c.Current}
	return n.ToPointers()
}

// Implementation of function-available() from XSLT spec
func XsltFunctionAvailable(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) < 1 {
		return nil
	}
	c := context.(*ExecutionContext)
	qname := args[0].(string)
	//TODO: resolve namespace of argument
	return c.IsFunctionRegistered("", qname)
}

// Implementation of element-available() from XSLT spec
func XsltElementAvailable(context xpath.VariableScope, args []interface{}) interface{} {
	return false
}

// util function because we can't assume we're actually getting a string
func argValToString(val interface{}) (out string) {
	if val == nil {
		return
	}
	switch v := val.(type) {
	case string:
		return v
	case []unsafe.Pointer:
		if len(v) == 0 {
			return
		}
		n := xml.NewNode(v[0], nil)
		out = n.Content()
	default:
		out = fmt.Sprintf("%v", v)
	}
	return
}

func EXSLTnodeset(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) < 1 {
		return nil
	}
	c := context.(*ExecutionContext)
	nodes := args[0]
	switch v := nodes.(type) {
	case []unsafe.Pointer:
		if len(v) == 0 {
			return nil
		}
		fauxroot := c.Output.CreateElementNode("VARIABLE")
		for _, node := range v {
			n := xml.NewNode(node, nil)
			fauxroot.AddChild(n)
		}
		out := xml.Nodeset{fauxroot}
		return out.ToPointers()
	default:
		out := fmt.Sprintf("%v", v)
		fmt.Println("invalid argument to exslt:nodeset", out)
	}

	return nodes
}

func EXSLTmathconstant(context xpath.VariableScope, args []interface{}) interface{} {

	if len(args) != 2 {
		return 0
	}

	name := args[0].(string)
	precision := int(args[1].(float64))

	switch name {
	case "PI":
		return fmt.Sprintf("%.*f", precision, 3.1415926535897932384626433832795028841971693993751)
	case "E":
		return fmt.Sprintf("%.*f", precision, 2.71828182845904523536028747135266249775724709369996)
	case "SQRRT2":
		return fmt.Sprintf("%.*f", precision, 1.41421356237309504880168872420969807856967187537694)
	case "LN2":
		return fmt.Sprintf("%.*f", precision, 0.69314718055994530941723212145817656807550013436025)
	case "LN10":
		return fmt.Sprintf("%.*f", precision, 2.30258509299404568402)
	case "LOG2E":
		return fmt.Sprintf("%.*f", precision, 1.4426950408889634074)
	case "SQRT1_2":
		return fmt.Sprintf("%.*f", precision, 0.70710678118654752440)
	default:
		out := fmt.Sprintf("%v", name)
		fmt.Println("unsupported constant in math:constant", out)
	}

	return 0
}

func EXSLTmathsin(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) != 1 {
		return nil
	}

	return math.Sin(args[0].(float64))
}

func EXSLTmathcos(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) != 1 {
		return nil
	}

	return math.Cos(args[0].(float64))
}

func EXSLTmathabs(context xpath.VariableScope, args []interface{}) interface{} {
	if len(args) != 1 {
		return nil
	}

	return math.Abs(args[0].(float64))
}
