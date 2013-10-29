package xslt

import (
	"errors"
	"github.com/moovweb/gokogiri/xml"
	"github.com/moovweb/gokogiri/xpath"
	"strings"
	"unsafe"
)

// ExecutionContext is passed to XSLT instructions during processing.
type ExecutionContext struct {
	Style        *Stylesheet  // The master stylesheet
	Output       xml.Document // The output document
	OutputNode   xml.Node     // The current output node
	Current      xml.Node     // The current input node
	XPathContext *xpath.XPath //the XPath context
	Mode         string       //The current template mode
}

func (context *ExecutionContext) EvalXPath(xmlNode xml.Node, data interface{}) (result interface{}, err error) {
	switch data := data.(type) {
	case string:
		if xpathExpr := xpath.Compile(data); xpathExpr != nil {
			defer xpathExpr.Free()
			result, err = context.EvalXPath(xmlNode, xpathExpr)
		} else {
			err = errors.New("cannot compile xpath: " + data)
		}
	case []byte:
		result, err = context.EvalXPath(xmlNode, string(data))
	case *xpath.Expression:
		xpathCtx := context.XPathContext
		xpathCtx.SetResolver(context)
		err := xpathCtx.Evaluate(xmlNode.NodePtr(), data)
		if err != nil {
			return nil, err
		}
		rt := xpathCtx.ReturnType()
		switch rt {
		case xpath.XPATH_NODESET:
			nodePtrs, err := xpathCtx.ResultAsNodeset()
			if err != nil {
				return nil, err
			}
			var output []xml.Node
			for _, nodePtr := range nodePtrs {
				output = append(output, xml.NewNode(nodePtr, xmlNode.MyDocument()))
			}
			result = output
		case xpath.XPATH_NUMBER:
			result, _ = xpathCtx.ResultAsNumber()
		case xpath.XPATH_BOOLEAN:
			result, _ = xpathCtx.ResultAsBoolean()
		default:
			result, _ = xpathCtx.ResultAsString()
		}
	default:
		err = errors.New("Strange type passed to ExecutionContext.EvalXPath")
	}
	return
}

func (context *ExecutionContext) EvalXPathAsNodeset(xmlNode xml.Node, data interface{}) (result []xml.Node, err error) {
	_, err = context.EvalXPath(xmlNode, data)
	if err != nil {
		return nil, err
	}
	nodePtrs, err := context.XPathContext.ResultAsNodeset()
	if err != nil {
		return nil, err
	}
	var output []xml.Node
	for _, nodePtr := range nodePtrs {
		output = append(output, xml.NewNode(nodePtr, xmlNode.MyDocument()))
	}
	result = output
	return
}

func (context *ExecutionContext) EvalXPathAsBoolean(xmlNode xml.Node, data interface{}) (result bool) {
	_, err := context.EvalXPath(xmlNode, data)
	if err != nil {
		return false
	}
	result, _ = context.XPathContext.ResultAsBoolean()
	return
}

// ChildrenOf returns the node children, ignoring any whitespace-only text nodes that
// are stripped by strip-space or xml:space
func (context *ExecutionContext) ChildrenOf(node xml.Node) (children []xml.Node) {

	for cur := node.FirstChild(); cur != nil; cur = cur.NextSibling() {
		//don't count stripped nodes
		if context.ShouldStrip(cur) {
			continue
		}
		children = append(children, cur)
	}
	return
}

// ShouldStrip evaluates the strip-space, preserve-space, and xml:space rules
// and returns true if a node is a whitespace-only text node that should
// be stripped.
func (context *ExecutionContext) ShouldStrip(xmlNode xml.Node) bool {
	if xmlNode.NodeType() != xml.XML_TEXT_NODE {
		return false
	}
	if !IsBlank(xmlNode) {
		return false
	}
	//do we have a match in strip-space?
	elem := xmlNode.Parent().Name()
	ns := xmlNode.Parent().Namespace()
	for _, pat := range context.Style.StripSpace {
		if pat == elem {
			return true
		}
		if pat == "*" {
			return true
		}
		if strings.Contains(pat, ":") {
			parts := strings.Split(pat, ":")
			for uri, prefix := range context.Style.NamespaceMapping {
				if uri == ns && prefix == parts[0] {
					if parts[1] == elem || parts[1] == "*" {
						return true
					}
				}
			}
		}
	}
	//do we have a match in preserve-space?
	//resolve conflicts by priority (QName, ns:*, *)
	//return a value
	return false
}

func (context *ExecutionContext) UseCDataSection(node xml.Node) bool {
	if node.NodeType() != xml.XML_ELEMENT_NODE {
		return false
	}
	name := node.Name()
	for _, el := range context.Style.CDataElements {
		if name == el {
			return true
		}
	}
	return false
}

func (context *ExecutionContext) ResolveVariable(name, ns string) (ret interface{}) {
	//consult local vars
	//consult local params
	//consult global vars (ss)
	//consult global params (ss)
	v, ok := context.Style.Variables[name]
	if !ok {
		return
	}
	if v == nil {
		return
	}

	switch val := v.Value.(type) {
	case []xml.Node:
		var res []unsafe.Pointer
		for _, n := range val {
			res = append(res, n.NodePtr())
		}
		return res
	default:
		return val
	}
}

func (context *ExecutionContext) IsFunctionRegistered(name, ns string) bool {
	_, ok := context.Style.Functions[name]
	return ok
}

func (context *ExecutionContext) ResolveFunction(name, ns string) xpath.XPathFunction {
	f, ok := context.Style.Functions[name]
	if ok {
		return f
	}
	return nil
}
