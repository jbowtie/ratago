package xslt

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/jbowtie/gokogiri/xml"
	"github.com/jbowtie/gokogiri/xpath"
	"path/filepath"
	"strings"
	"unsafe"
)

// ExecutionContext is passed to XSLT instructions during processing.
type ExecutionContext struct {
	Style          *Stylesheet                 // The master stylesheet
	Output         xml.Document                // The output document
	Source         xml.Document                // The source input document
	OutputNode     xml.Node                    // The current output node
	Current        xml.Node                    // The node that will be returned for "current()"
	XPathContext   *xpath.XPath                //the XPath context
	Mode           string                      //The current template mode
	Stack          list.List                   //stack used for scoping local variables
	InputDocuments map[string]*xml.XmlDocument //additional input documents via document()
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
		case xpath.XPATH_NODESET, xpath.XPATH_XSLT_TREE:
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
			result, err = xpathCtx.ResultAsNumber()
		case xpath.XPATH_BOOLEAN:
			result, err = xpathCtx.ResultAsBoolean()
		default:
			result, err = xpathCtx.ResultAsString()
		}
	default:
		err = errors.New("Strange type passed to ExecutionContext.EvalXPath")
	}
	return
}

// Register the namespaces in scope with libxml2 so that XPaths with namespaces
// are resolved correctly.

// libxml2 probably already makes this info available
func (context *ExecutionContext) RegisterXPathNamespaces(node xml.Node) (err error) {
	seen := make(map[string]bool)
	for n := node; n != nil; n = n.Parent() {
		for _, decl := range n.DeclaredNamespaces() {
			alreadySeen, _ := seen[decl.Prefix]
			if !alreadySeen {
				context.XPathContext.RegisterNamespace(decl.Prefix, decl.Uri)
				seen[decl.Prefix] = true
			}
		}
	}
	return
}

// Attempt to map a prefix to a URI.
func (context *ExecutionContext) LookupNamespace(prefix string, node xml.Node) (uri string) {
	//if given a context node, see if the prefix is in scope
	if node != nil {
		for n := node; n != nil; n = n.Parent() {
			for _, decl := range n.DeclaredNamespaces() {
				if decl.Prefix == prefix {
					return decl.Uri
				}
			}
		}
		return
	}

	//if no context node, simply check the stylesheet map
	for href, pre := range context.Style.NamespaceMapping {
		if pre == prefix {
			return href
		}
	}
	return
}

func (context *ExecutionContext) EvalXPathAsNodeset(xmlNode xml.Node, data interface{}) (result xml.Nodeset, err error) {
	_, err = context.EvalXPath(xmlNode, data)
	if err != nil {
		return nil, err
	}
	nodePtrs, err := context.XPathContext.ResultAsNodeset()
	if err != nil {
		return nil, err
	}
	var output xml.Nodeset
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

func (context *ExecutionContext) EvalXPathAsString(xmlNode xml.Node, data interface{}) (result string, err error) {
	_, err = context.EvalXPath(xmlNode, data)
	if err != nil {
		return
	}
	result, err = context.XPathContext.ResultAsString()
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
			uri, name := context.ResolveQName(pat)
			if uri == ns {
				if name == elem || name == "*" {
					return true
				}
			}
		}
	}
	//do we have a match in preserve-space?
	//resolve conflicts by priority (QName, ns:*, *)
	//return a value
	return false
}

func (context *ExecutionContext) ResolveQName(qname string) (ns, name string) {
	if !strings.Contains(qname, ":") {
		//TODO: lookup default namespace
		return "", name
	}
	parts := strings.Split(qname, ":")
	for uri, prefix := range context.Style.NamespaceMapping {
		if prefix == parts[0] {
			return uri, parts[1]
		}
	}
	return
}

func (context *ExecutionContext) UseCDataSection(node xml.Node) bool {
	if node.NodeType() != xml.XML_ELEMENT_NODE {
		return false
	}
	name := node.Name()
	ns := node.Namespace()
	for _, el := range context.Style.CDataElements {
		if el == name {
			return true
		}
		uri, elname := context.ResolveQName(el)
		if uri == ns && name == elname {
			return true
		}
	}
	return false
}

func (context *ExecutionContext) ResolveVariable(name, ns string) (ret interface{}) {
	v := context.FindVariable(name, ns)

	if v == nil {
		return
	}

	switch val := v.Value.(type) {
	case xml.Nodeset:
		return unsafe.Pointer(val.ToXPathNodeset())
	case []xml.Node:
		nodeset := xml.Nodeset(val)
		return unsafe.Pointer(nodeset.ToXPathNodeset())
	default:
		return val
	}
}

func (context *ExecutionContext) FindVariable(name, ns string) (ret *Variable) {
	//consult local vars
	//consult local params
	v := context.LookupLocalVariable(name, ns)
	if v != nil {
		return v
	}
	//consult global vars (ss)
	//consult global params (ss)
	v, ok := context.Style.Variables[name]
	if ok {
		return v
	}
	return nil
}

func (context *ExecutionContext) DeclareLocalVariable(name, ns string, v *Variable) error {
	if context.Stack.Len() == 0 {
		return errors.New("Attempting to declare a local variable without a stack frame")
	}
	e := context.Stack.Front()
	scope := e.Value.(map[string]*Variable)
	scope[name] = v
	//fmt.Println("DECLARE", name, v)
	return nil
}

func (context *ExecutionContext) LookupLocalVariable(name, ns string) (ret *Variable) {
	for e := context.Stack.Front(); e != nil; e = e.Next() {
		scope := e.Value.(map[string]*Variable)
		v, ok := scope[name]
		if ok {
			//fmt.Println("FOUND", name, v)
			return v
		}
	}
	return
}

// create a local scope for variable resolution
func (context *ExecutionContext) PushStack() {
	scope := make(map[string]*Variable)
	context.Stack.PushFront(scope)
}

// leave the variable scope
func (context *ExecutionContext) PopStack() {
	if context.Stack.Len() == 0 {
		return
	}
	context.Stack.Remove(context.Stack.Front())
}

func (context *ExecutionContext) IsFunctionRegistered(name, ns string) bool {
	qname := fmt.Sprintf("{%s}%s", ns, name)
	_, ok := context.Style.Functions[qname]
	return ok
}

func (context *ExecutionContext) ResolveFunction(name, ns string) xpath.XPathFunction {
	qname := fmt.Sprintf("{%s}%s", ns, name)
	f, ok := context.Style.Functions[qname]
	if ok {
		return f
	}
	return nil
}

// Determine the default namespace currently defined in scope
func (context *ExecutionContext) DefaultNamespace(node xml.Node) string {
	//get the list of in-scope namespaces
	// any with a null prefix? return that
	decl := node.DeclaredNamespaces()
	for _, d := range decl {
		if d.Prefix == "" {
			return d.Uri
		}
	}
	return ""
}

// Propogate namespaces to the root of the output document
func (context *ExecutionContext) DeclareStylesheetNamespacesIfRoot(node xml.Node) {
	if context.OutputNode.NodeType() != xml.XML_DOCUMENT_NODE {
		return
	}
	//add all namespace declarations to r
	for uri, prefix := range context.Style.NamespaceMapping {
		if uri != XSLT_NAMESPACE {
			//these don't actually change if there is no alias
			_, uri = ResolveAlias(context.Style, prefix, uri)
			if !context.Style.IsExcluded(prefix) {
				node.DeclareNamespace(prefix, uri)
			}
		}
	}
}

func (context *ExecutionContext) FetchInputDocument(loc string, relativeToSource bool) (doc *xml.XmlDocument) {
	//create the map if needed
	if context.InputDocuments == nil {
		context.InputDocuments = make(map[string]*xml.XmlDocument)
	}

	// rely on caller to tell us how to resolve relative paths
	base := ""
	if relativeToSource {
		base, _ = filepath.Abs(filepath.Dir(context.Source.Uri()))
	} else {
		base, _ = filepath.Abs(filepath.Dir(context.Style.Doc.Uri()))
	}
	resolvedLoc := filepath.Join(base, loc)

	//if abspath in map return existing document
	doc, ok := context.InputDocuments[resolvedLoc]
	if ok {
		return
	}

	//else load the document and add to map
	doc, e := xml.ReadFile(resolvedLoc, xml.StrictParseOption)
	if e != nil {
		fmt.Println(e)
		return
	}
	context.InputDocuments[resolvedLoc] = doc
	return
}
