package xslt

import (
	"container/list"
	"fmt"
	"github.com/jbowtie/gokogiri/xml"
	"github.com/jbowtie/gokogiri/xpath"
	"log"
	"path"
	"strconv"
	"strings"
)

const XSLT_NAMESPACE = "http://www.w3.org/1999/XSL/Transform"
const XML_NAMESPACE = "http://www.w3.org/XML/1998/namespace"

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
	CommentMatches     *list.List            //matches on comment()
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
	GlobalParameters   []string
	includes           map[string]bool
	Keys               map[string]*Key
	OutputMethod       string //html, xml, text
	DesiredEncoding    string //encoding specified by xsl:output
	OmitXmlDeclaration bool   //defaults to false
	IndentOutput       bool   //defaults to false
	Standalone         bool   //defaults to false
	doctypeSystem      string
	doctypePublic      string
}

// StylesheetOptions to control processing. Parameters values are passed into
// the stylesheet via this structure.
type StylesheetOptions struct {
	IndentOutput bool                   //force the output to be indented
	Parameters   map[string]interface{} //supply values for stylesheet parameters
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
//
// The fileuri argument is used to resolve relative paths for xsl:import and xsl:include
// instructions and should generally be the filename of the stylesheet. If you pass
// an empty string, the working directory will be used for path resolution.
func ParseStylesheet(doc *xml.XmlDocument, fileuri string) (style *Stylesheet, err error) {
	style = &Stylesheet{Doc: doc,
		NamespaceMapping: make(map[string]string),
		NamespaceAlias:   make(map[string]string),
		ElementMatches:   make(map[string]*list.List),
		AttrMatches:      make(map[string]*list.List),
		PIMatches:        list.New(),
		CommentMatches:   list.New(),
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
		log.Println("VERSION 1.0 expected")
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
	err = style.parseChildren(cur, fileuri)

	//xsl:import (must be first)
	//flag non-empty text nodes, non XSL-namespaced nodes
	//  actually registered extension namspaces are good!
	//warn unknown XSLT element (forwards-compatible mode)

	return
}

// Here we iterate through the children; this has been moved to its own function
// to facilitate the implementation of xsl:include (where we want the children to
// be treated as if they were part of the calling stylesheet)
func (style *Stylesheet) parseChildren(root xml.Node, fileuri string) (err error) {
	//iterate through children
	for cur := root.FirstChild(); cur != nil; cur = cur.NextSibling() {
		//skip blank nodes
		if IsBlank(cur) {
			continue
		}

		//skip comment nodes
		if cur.NodeType() == xml.XML_COMMENT_NODE {
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
			k := &Key{make(map[string]xml.Nodeset), use, match}
			style.Keys[name] = k
			continue
		}

		if IsXsltName(cur, "param") {
			name := cur.Attr("name")
			// record that it's a global parameter - we'll check supplied options against this list
			style.GlobalParameters = append(style.GlobalParameters, name)
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
			base := path.Dir(fileuri)
			loc = path.Join(base, loc)
			_, already := style.includes[loc]
			if already {
				panic("Multiple include detected of " + loc)
			}
			style.includes[loc] = true

			//load the stylesheet
			doc, e := xml.ReadFile(loc, xml.StrictParseOption)
			if e != nil {
				fmt.Println(e)
				err = e
				return
			}
			//_, _ = ParseStylesheet(doc, loc)
			//update the including stylesheet
			e = style.parseChildren(doc.Root(), loc)
			if e != nil {
				fmt.Println(e)
				err = e
				return
			}
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
			_import, _ := ParseStylesheet(doc, loc)
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
			indent := cur.Attr("indent")
			if indent == "yes" {
				style.IndentOutput = true
			}
			standalone := cur.Attr("standalone")
			if standalone == "yes" {
				style.Standalone = true
			}
			encoding := cur.Attr("encoding")
			if encoding != "" && encoding != "utf-8" {
				//TODO: emit a warning if we do not support the encoding
				// if unsupported, leave blank to output default UTF-8
				style.DesiredEncoding = encoding
			}
			style.doctypeSystem = cur.Attr("doctype-system")
			style.doctypePublic = cur.Attr("doctype-public")
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

		if IsXsltName(cur, "decimal-format") {
			fmt.Println("GLOBAL TODO ", cur.Name())
			continue
		}
	}
	return
}

func (style *Stylesheet) IsExcluded(prefix string) bool {
	for _, p := range style.ExcludePrefixes {
		if p == prefix {
			return true
		}
	}
	for _, p := range style.ExtensionPrefixes {
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
// to returning a slice of bytes and encoding information.
func (style *Stylesheet) Process(doc *xml.XmlDocument, options StylesheetOptions) (out string, err error) {
	// lookup output method, doctypes, encoding
	// create output document with appropriate values
	output := xml.CreateEmptyDocument(doc.InputEncoding(), doc.OutputEncoding())
	// init context node/document
	context := &ExecutionContext{Output: output.Me, OutputNode: output, Style: style, Source: doc}
	context.Current = doc
	context.XPathContext = doc.DocXPathCtx()
	// when evaluating keys/global vars position is always 1
	context.XPathContext.SetContextPosition(1, 1)
	start := doc
	style.populateKeys(start, context)
	// eval global params
	// eval global variables
	for _, val := range style.Variables {
		val.Apply(doc, context)
	}

	// for each global parameter
	for _, param := range style.GlobalParameters {
		// was a parameter passed with this name?
		gp_value, gp_ok := options.Parameters[param]
		if gp_ok {
			gp_var := style.Variables[param]
			// replace value of style.Variables[key]
			gp_var.Value = gp_value
			// fmt.Println("Existing", param, "set to", gp_value)
		}
	}

	// process nodes
	style.processNode(start, context, nil)

	out, err = style.constructOutput(output, options)
	// reset anything required for re-use
	return
}

func (style *Stylesheet) constructXmlDeclaration() (out string) {
	out = "<?xml version=\"1.0\""
	if style.DesiredEncoding != "" {
		out = out + fmt.Sprintf(" encoding=\"%s\"", style.DesiredEncoding)
	}
	if style.Standalone {
		out = out + " standalone=\"yes\""
	}
	out = out + "?>\n"
	return
}

// actually produce (and possibly write) the final output
func (style *Stylesheet) constructOutput(output *xml.XmlDocument, options StylesheetOptions) (out string, err error) {
	//if not explicitly set, spec requires us to check for html
	outputType := style.OutputMethod
	if outputType == "" {
		outputType = "xml"
		root := output.Root()
		if root != nil && root.Name() == "html" && root.Namespace() == "" {
			outputType = "html"
		}
	}

	// construct DTD declaration depending on xsl:output settings
	docType := ""
	if style.doctypeSystem != "" {
		docType = "<!DOCTYPE "
		docType = docType + output.Root().Name()
		if style.doctypePublic != "" {
			docType = docType + fmt.Sprintf(" PUBLIC \"%s\"", style.doctypePublic)
		} else {
			docType = docType + " SYSTEM"
		}
		docType = docType + fmt.Sprintf(" \"%s\"", style.doctypeSystem)
		docType = docType + ">\n"
	}

	// create the XML declaration depending on xsl:output settings
	decl := ""
	if outputType == "xml" {
		if !style.OmitXmlDeclaration {
			decl = style.constructXmlDeclaration()
		}
		format := xml.XML_SAVE_NO_DECL | xml.XML_SAVE_AS_XML
		if options.IndentOutput || style.IndentOutput {
			format = format | xml.XML_SAVE_FORMAT
		}
		// we get slightly incorrect output if we call out.SerializeWithFormat directly
		// this seems to be a libxml bug; we work around it the same way libxslt does

		//TODO: honor desired encoding
		//  this involves decisions about supported encodings, strings vs byte slices
		//  we can sidestep a little if we enable option to write directly to file
		for cur := output.FirstChild(); cur != nil; cur = cur.NextSibling() {
			b, size := cur.SerializeWithFormat(format, nil, nil)
			if b != nil {
				out = out + string(b[:size])
			}
		}
		if out != "" {
			out = decl + docType + out + "\n"
		}
	}
	if outputType == "html" {
		out = docType
		b, size := output.ToHtml(nil, nil)
		out = out + string(b[:size])
	}
	if outputType == "text" {
		format := xml.XML_SAVE_NO_DECL
		for cur := output.FirstChild(); cur != nil; cur = cur.NextSibling() {
			b, size := cur.SerializeWithFormat(format, nil, nil)
			if b != nil {
				out = out + string(b[:size])
			}
		}
	}
	return
}

// Determine which template, if any, matches the current node.

// If there is no matching template, nil is returned.
func (style *Stylesheet) LookupTemplate(node xml.Node, mode string, context *ExecutionContext) (template *Template) {
	name := node.Name()
	if node.NodeType() == xml.XML_DOCUMENT_NODE {
		name = "/"
	}
	found := new(list.List)
	l := style.ElementMatches[name]
	if l != nil {
		for i := l.Front(); i != nil; i = i.Next() {
			c := i.Value.(*CompiledMatch)
			if c.EvalMatch(node, mode, context) {
				insertByPriority(found, c)
				break
			}
		}
	}
	l = style.ElementMatches["*"]
	if l != nil {
		for i := l.Front(); i != nil; i = i.Next() {
			c := i.Value.(*CompiledMatch)
			if c.EvalMatch(node, mode, context) {
				insertByPriority(found, c)
				break
			}
		}
	}
	l = style.AttrMatches[name]
	if l != nil {
		for i := l.Front(); i != nil; i = i.Next() {
			c := i.Value.(*CompiledMatch)
			if c.EvalMatch(node, mode, context) {
				insertByPriority(found, c)
				break
			}
		}
	}
	l = style.AttrMatches["*"]
	if l != nil {
		for i := l.Front(); i != nil; i = i.Next() {
			c := i.Value.(*CompiledMatch)
			if c.EvalMatch(node, mode, context) {
				insertByPriority(found, c)
				break
			}
		}
	}
	//TODO: review order in which we consult generic matches
	for i := style.IdKeyMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			insertByPriority(found, c)
			break
		}
	}
	for i := style.NodeMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			insertByPriority(found, c)
			break
		}
	}
	for i := style.TextMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			insertByPriority(found, c)
			break
		}
	}
	for i := style.PIMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			insertByPriority(found, c)
			break
		}
	}
	for i := style.CommentMatches.Front(); i != nil; i = i.Next() {
		c := i.Value.(*CompiledMatch)
		if c.EvalMatch(node, mode, context) {
			insertByPriority(found, c)
			break
		}
	}

	// if there's a match at this import precedence, return
	// the one with the highest priority
	f := found.Front()
	if f != nil {
		template = f.Value.(*CompiledMatch).Template
		return
	}

	// no match at this import precedence,
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
		style.processNode(cur, context, nil)
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

func (style *Stylesheet) processNode(node xml.Node, context *ExecutionContext, params []*Variable) {
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
	template.Apply(node, context, params)
}

func (style *Stylesheet) populateKeys(node xml.Node, context *ExecutionContext) {
	for _, key := range style.Keys {
		//see if the current node matches
		matches := CompileMatch(key.match, nil)
		hasMatch := false
		for _, m := range matches {
			if m.EvalMatch(node, "", context) {
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
	template := &Template{Match: match, Mode: mode, Name: name, Priority: p, Node: node}

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
			//fmt.Println("COMPILED", template.Match, c.Steps[0].Value, c.Steps[0].Op, template.Priority)
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
		if c.IsComment() {
			insertByPriority(style.CommentMatches, c)
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

// Locate an attribute set by name
func (style *Stylesheet) LookupAttributeSet(name string) CompiledStep {
	attset, ok := style.AttributeSets[name]
	if ok {
		return attset
	}
	for i := style.Imports.Front(); i != nil; i = i.Next() {
		s := i.Value.(*Stylesheet)
		t := s.LookupAttributeSet(name)
		if t != nil {
			return t
		}
	}
	return nil
}
