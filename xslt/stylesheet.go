// Package xslt provides an XSLT 1.0 processor.
package xslt

import (
	"container/list"
	"errors"
	"fmt"
	"github.com/moovweb/gokogiri/xml"
	"github.com/moovweb/gokogiri/xpath"
	"path"
	"strconv"
	"strings"
	"unsafe"
)

const XSLT_NAMESPACE = "http://www.w3.org/1999/XSL/Transform"

// Stylesheet is an XSLT 1.0 processor.
type Stylesheet struct {
	Doc                *xml.XmlDocument
	Parent             *Stylesheet //xsl:import
	NamedTemplates     map[string]*Template
	NamespaceMapping   map[string]string
	NamespaceAlias     map[string]string
	ElementMatches     map[string]*list.List //matches on element name
	AttrMatches        map[string]*list.List //matches on attr name
	NodeMatches        *list.List            //matches on node()
	TextMatches        *list.List            //matches on text()
	PIMatches          *list.List            //matches on processing-instruction()
	IdKeyMatches       *list.List            //matches on id() or key()
	Imports            *list.List
	Variables          map[string]*Variable
	Functions          map[string]xpath.XPathFunction
	AttributeSets      map[string]CompiledStep
	ExcludePrefixes    []string
	ExtensionPrefixes  []string
	StripSpace         []string
	PreserveSpace      []string
	CDataElements      []string
	includes           map[string]bool
	Keys               map[string]*Key
	OutputMethod       string //html, xml, text
	OmitXmlDeclaration bool   //defaults to false
}

// ExecutionContext is passed to XSLT instructions during processing.
type ExecutionContext struct {
	Style        *Stylesheet  // The master stylesheet
	Output       xml.Document // The output document
	OutputNode   xml.Node     // The current output node
	Current      xml.Node     // The current input node
	XPathContext *xpath.XPath //the XPath context
	Mode         string       //The current template mode
}

// Returns true if the node is in the XSLT namespace
func IsXsltName(xmlnode xml.Node, name string) bool {
	if xmlnode.Name() == name && xmlnode.Namespace() == XSLT_NAMESPACE {
		return true
	}
	return false
}

// Returns true if the node is a whitespace-only text node
func IsBlank(xmlnode xml.Node) bool {
	if xmlnode.NodeType() == xml.XML_TEXT_NODE || xmlnode.NodeType() == xml.XML_CDATA_SECTION_NODE {
		content := xmlnode.Content()
		if content == "" || strings.TrimSpace(content) == "" {
			return true
		}
	}
	return false
}

// ParseStylesheet compiles the stylesheet's XML representation
// and returns a Stylesheet instance.
func ParseStylesheet(doc *xml.XmlDocument, fileuri string) (style *Stylesheet) {
	style = &Stylesheet{Doc: doc,
		NamespaceMapping: make(map[string]string),
		NamespaceAlias:   make(map[string]string),
		ElementMatches:   make(map[string]*list.List),
		AttrMatches:      make(map[string]*list.List),
		PIMatches:        list.New(),
		IdKeyMatches:     list.New(),
		NodeMatches:      list.New(),
		TextMatches:      list.New(),
		Imports:          list.New(),
		NamedTemplates:   make(map[string]*Template),
		AttributeSets:    make(map[string]CompiledStep),
		includes:         make(map[string]bool),
		Keys:             make(map[string]*Key),
		Functions:        make(map[string]xpath.XPathFunction),
		Variables:        make(map[string]*Variable)}

	//set parent (importing stylesheet, if any)
	//creates a namespace hash, should be able to eval in context
	// will look at during compilation
	//XsltGatherNamespaces(style)
	//we need to create a compilation context for the main stylesheet
	//push and pop from the compilation stack as we handle imported stylesheets

	// register the built-in XSLT functions
	style.RegisterXsltFunctions()

	//XsltParseStylesheetProcess
	cur := xml.Node(doc.Root())

	// get all the namespace mappings
	for _, ns := range cur.DeclaredNamespaces() {
		style.NamespaceMapping[ns.Uri] = ns.Prefix
	}

	//get xsl:version, should be 1.0 or 2.0
	version := cur.Attr("version")
	if version != "1.0" {
		fmt.Println("VERSION 1.0 expected")
	}

	//record excluded prefixes
	excl := cur.Attr("exclude-result-prefixes")
	if excl != "" {
		style.ExcludePrefixes = strings.Fields(excl)
	}
	//record extension prefixes
	ext := cur.Attr("extension-element-prefixes")
	if ext != "" {
		style.ExtensionPrefixes = strings.Fields(ext)
	}

	//if the root is an LRE, this is an simplified stylesheet
	if !IsXsltName(cur, "stylesheet") && !IsXsltName(cur, "transform") {
		template := &Template{Match: "/", Priority: 0}
		template.CompileContent(doc)
		style.compilePattern(template, "")
		return
	}

	//optionally optimize by removing blank nodes, combining adjacent text nodes, etc

	//iterate through children
	for cur = cur.FirstChild(); cur != nil; cur = cur.NextSibling() {
		//skip blank nodes
		if IsBlank(cur) {
			continue
		}
		//handle templates
		if IsXsltName(cur, "template") {
			style.ParseTemplate(cur)
			continue
		}

		if IsXsltName(cur, "variable") {
			style.RegisterGlobalVariable(cur)
			continue
		}

		if IsXsltName(cur, "key") {
			name := cur.Attr("name")
			use := cur.Attr("use")
			match := cur.Attr("match")
			k := &Key{make(map[string]Nodeset), use, match}
			style.Keys[name] = k
			continue
		}

		//TODO: this is cheating. Also note global params can have their
		// value overwritten
		if IsXsltName(cur, "param") {
			style.RegisterGlobalVariable(cur)
			continue
		}

		if IsXsltName(cur, "attribute-set") {
			style.RegisterAttributeSet(cur)
			continue
		}

		if IsXsltName(cur, "include") {
			//check for recursion, multiple includes
			loc := cur.Attr("href")
			fmt.Println("INCLUDE", loc)
			_, already := style.includes[loc]
			if already {
				panic("Multiple include detected of " + loc)
			}
			style.includes[loc] = true

			//load the stylesheet
			//update the including stylesheet
			continue
		}

		if IsXsltName(cur, "import") {
			//check for recursion, multiple includes
			loc := cur.Attr("href")
			base := path.Dir(fileuri)
			loc = path.Join(base, loc)
			_, already := style.includes[loc]
			if already {
				panic("Multiple include detected of " + loc)
			}
			style.includes[loc] = true
			//increment import; new style context
			doc, _ := xmlReadFile(loc)
			_import := ParseStylesheet(doc, loc)
			style.Imports.PushFront(_import)
			continue
		}

		if IsXsltName(cur, "output") {
			cdata := cur.Attr("cdata-section-elements")
			if cdata != "" {
				style.CDataElements = strings.Fields(cdata)
			}
			style.OutputMethod = cur.Attr("method")
			omit := cur.Attr("omit-xml-declaration")
			if omit == "yes" {
				style.OmitXmlDeclaration = true
			}
			continue
		}

		if IsXsltName(cur, "strip-space") {
			el := cur.Attr("elements")
			if el != "" {
				style.StripSpace = strings.Fields(el)
			}
			continue
		}

		if IsXsltName(cur, "preserve-space") {
			el := cur.Attr("elements")
			if el != "" {
				style.PreserveSpace = strings.Fields(el)
			}
			continue
		}

		if IsXsltName(cur, "namespace-alias") {
			stylens := cur.Attr("stylesheet-prefix")
			resns := cur.Attr("result-prefix")
			style.NamespaceAlias[stylens] = resns
			continue
		}
		//key
		//decimal-format
		fmt.Println("GLOBAL SS TODO ", cur.Name())
	}
	//xsl:import (must be first)
	//flag non-empty text nodes, non XSL-namespaced nodes
	//  actually registered extension namspaces are good!
	//all other types
	//  include, output,key,decimal-format
	//warn unknown XSLT element (forwards-compatible mode)

	return
}

func (style *Stylesheet) IsExcluded(prefix string) bool {
	for _, p := range style.ExcludePrefixes {
		if p == prefix {
			return true
		}
	}
	return false
}

// Process takes an input document and returns the output produced
// by executing the stylesheet.

// The output is not guaranteed to be well-formed XML, so the
// serialized string is returned. Consideration is being given
// to returning a slice of bytes.
func (style *Stylesheet) Process(doc *xml.XmlDocument) (out string) {
	// lookup output method, doctypes, encoding
	// create output document with appropriate values
	output := xml.CreateEmptyDocument(doc.InputEncoding(), doc.OutputEncoding())
	// init context node/document
	context := &ExecutionContext{Output: output.Me, OutputNode: output, Style: style}
	context.Current = doc
	context.XPathContext = doc.DocXPathCtx()
	start := doc
	style.populateKeys(start, context)
	// eval global params
	// eval global variables
	for _, val := range style.Variables {
		val.Apply(doc, context)
	}
	// set xpath context
	// process nodes
	style.processNode(start, context)

	// construct DTD, xml declarations depending on xsl:output settings

	//if not explicitly set, spec requires us to check for html
	outputType := style.OutputMethod
	if outputType == "" {
		outputType = "xml"
		root := output.Root()
		if root != nil && root.Name() == "html" && root.Namespace() == "" {
			outputType = "html"
		}
	}

	if outputType == "xml" {
		if !style.OmitXmlDeclaration {
			out = "<?xml version=\"1.0\"?>\n"
		}
		// we get slightly incorrect output if we call out.ToUnformattedXml directly
		// this seems to be a libxml bug; we work around it the same way libxslt does
		for cur := output.FirstChild(); cur != nil; cur = cur.NextSibling() {
			out = out + cur.ToUnformattedXml()
		}
		out = out + "\n"
	}
	if outputType == "html" {
		b, size := output.ToHtml(nil, nil)
		out = out + string(b[:size])
	}
	// reset anything required for re-use
	return
}

// Determine which template, if any, matches the current node.

// If there is no matching template, nil is returned.
func (style *Stylesheet) LookupTemplate(node xml.Node, mode string, context *ExecutionContext) (template *Template) {
	name := node.Name()
	if node.NodeType() == xml.XML_DOCUMENT_NODE {
		name = "/"
	}
	l := style.ElementMatches[name]
	if l != nil {
		for i := l.Front(); i != nil; i = i.Next() {
			c := i.Value.(*CompiledMatch)
			if c.EvalMatch(node, mode, context) {
				return c.Template
			}
		}
	}
	l = style.ElementMatches["*"]
	if l != nil {
		for i := l.Front(); i != nil; i = i.Next() {
			c := i.Value.(*CompiledMatch)
			if c.EvalMatch(node, mode, context) {
				return c.Template
			}
		}
	}
	l = style.AttrMatches[name]
	if l != nil {
		for i := l.Front(); i != nil; i = i.Next() {
			c := i.Value.(*CompiledMatch)
			if c.EvalMatch(node, mode, context) {
				return c.Template
			}
		}
	}
	l = style.AttrMatches["*"]
	if l != nil {
		for i := l.Front(); i != nil; i = i.Next() {
			c := i.Value.(*CompiledMatch)
			if c.EvalMatch(node, mode, context) {
				return c.Template
			}
		}
	}
	//TODO: review order in which we consult generic matches
	for i := style.IdKeyMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			return c.Template
		}
	}
	for i := style.NodeMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			return c.Template
		}
	}
	for i := style.TextMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			return c.Template
		}
	}
	for i := style.PIMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			return c.Template
		}
	}

	//consult the imported stylesheets
	for i := style.Imports.Front(); i != nil; i = i.Next() {
		s := i.Value.(*Stylesheet)
		t := s.LookupTemplate(node, mode, context)
		if t != nil {
			return t
		}
	}
	return
}

func (style *Stylesheet) RegisterAttributeSet(node xml.Node) {
	name := node.Attr("name")
	res := CompileSingleNode(node)
	res.Compile(node)
	style.AttributeSets[name] = res
}

func (style *Stylesheet) RegisterGlobalVariable(node xml.Node) {
	name := node.Attr("name")
	_var := CompileSingleNode(node).(*Variable)
	_var.Compile(node)
	style.Variables[name] = _var
}

func (style *Stylesheet) processDefaultRule(node xml.Node, context *ExecutionContext) {
	//default for DOCUMENT, ELEMENT
	children := context.ChildrenOf(node)
	total := len(children)
	for i, cur := range children {
		context.XPathContext.SetContextPosition(i+1, total)
		style.processNode(cur, context)
	}
	//default for CDATA, TEXT, ATTR is copy as text
	if node.NodeType() == xml.XML_TEXT_NODE {
		if context.ShouldStrip(node) {
			return
		}
		if context.UseCDataSection(context.OutputNode) {
			r := context.Output.CreateCDataNode(node.Content())
			context.OutputNode.AddChild(r)
		} else {
			r := context.Output.CreateTextNode(node.Content())
			context.OutputNode.AddChild(r)
		}
	}
	//default for namespace declaration is copy to output document
}

func (style *Stylesheet) processNode(node xml.Node, context *ExecutionContext) {
	//get template
	template := style.LookupTemplate(node, context.Mode, context)
	//  for each import scope
	//    get the list of applicable templates  for this mode
	//    (assume compilation ordered appropriately)
	//    eval each one until we get a match
	//    eval generic templates that might apply until we get a match
	//apply default rule if null template
	if template == nil {
		style.processDefaultRule(node, context)
		return
	}
	//apply template to current node
	template.Apply(node, context)
}

func (style *Stylesheet) populateKeys(node xml.Node, context *ExecutionContext) {
	for _, key := range style.Keys {
		//see if the current node matches
		matches := CompileMatch(key.match, nil)
		hasMatch := false
		for _, m := range matches {
			if m.EvalMatch(node, "", nil) {
				hasMatch = true
				break
			}
		}
		if !hasMatch {
			continue
		}
		lookupkey, _ := node.EvalXPath(key.use, context)
		lookup := ""
		switch lk := lookupkey.(type) {
		case []xml.Node:
			if len(lk) == 0 {
				continue
			}
			lookup = lk[0].String()
		case string:
			lookup = lk
		default:
			lookup = fmt.Sprintf("%v", lk)
		}
		key.nodes[lookup] = append(key.nodes[lookup], node)
	}
	children := context.ChildrenOf(node)
	for _, cur := range children {
		style.populateKeys(cur, context)
	}
}

// ParseTemplate parses and compiles the xsl:template elements.
func (style *Stylesheet) ParseTemplate(node xml.Node) {
	//add to template list of stylesheet
	//parse mode, match, name, priority
	mode := node.Attr("mode")
	name := node.Attr("name")
	match := node.Attr("match")
	priority := node.Attr("priority")
	p := 0.0
	if priority != "" {
		p, _ = strconv.ParseFloat(priority, 64)
	}

	// TODO: validate the name (duplicate should raise error)
	template := &Template{Match: match, Mode: mode, Name: name, Priority: p}

	template.CompileContent(node)

	//  compile pattern
	style.compilePattern(template, priority)
}

func (style *Stylesheet) compilePattern(template *Template, priority string) {
	if template.Name != "" {
		style.NamedTemplates[template.Name] = template
	}

	if template.Match == "" {
		return
	}

	matches := CompileMatch(template.Match, template)
	for _, c := range matches {
		//  calculate priority if not explicitly set
		if priority == "" {
			template.Priority = c.DefaultPriority()
		}
		// insert into 'best' collection
		if c.IsElement() {
			hash := c.Hash()
			l := style.ElementMatches[hash]
			if l == nil {
				l = list.New()
				style.ElementMatches[hash] = l
			}
			insertByPriority(l, c)
		}
		if c.IsAttr() {
			hash := c.Hash()
			l := style.AttrMatches[hash]
			if l == nil {
				l = list.New()
				style.AttrMatches[hash] = l
			}
			insertByPriority(l, c)
		}
		if c.IsIdKey() {
			insertByPriority(style.IdKeyMatches, c)
		}
		if c.IsText() {
			insertByPriority(style.TextMatches, c)
		}
		if c.IsPI() {
			insertByPriority(style.PIMatches, c)
		}
		if c.IsNode() {
			insertByPriority(style.NodeMatches, c)
		}
	}
}

func insertByPriority(l *list.List, match *CompiledMatch) {
	for i := l.Front(); i != nil; i = i.Next() {
		cur := i.Value.(*CompiledMatch)
		if cur.Template.Priority <= match.Template.Priority {
			l.InsertBefore(match, i)
			return
		}
	}
	//either list is empty, or we're lowest priority template
	l.PushBack(match)
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
