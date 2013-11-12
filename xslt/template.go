package xslt

import (
	"fmt"
	"github.com/moovweb/gokogiri/xml"
	"github.com/moovweb/gokogiri/xpath"
	"strings"
	"unicode/utf8"
)

type CompiledStep interface {
	Compile(node xml.Node)
	Apply(node xml.Node, context *ExecutionContext)
}

type Template struct {
	Name     string
	Mode     string
	Match    string
	Priority float64
	Children []CompiledStep
}

type XsltInstruction struct {
	Node     xml.Node
	Name     string
	Children []CompiledStep
	sorting  []*sortCriteria
}

type LiteralResultElement struct {
	Node     xml.Node
	Children []CompiledStep
}

type TextOutput struct {
	Content string
}

type Variable struct {
	Name     string
	Node     xml.Node
	Children []CompiledStep
	Value    interface{}
}

func (i *XsltInstruction) Compile(node xml.Node) {
	for cur := node.FirstChild(); cur != nil; cur = cur.NextSibling() {
		res := CompileSingleNode(cur)
		if cur.Name() == "sort" && cur.Namespace() == XSLT_NAMESPACE {
			i.sorting = append(i.sorting, compileSortFunction(res.(*XsltInstruction)))
			continue
		}
		if res != nil {
			res.Compile(cur)
			i.Children = append(i.Children, res)
		}
	}
}

func (i *Variable) Compile(node xml.Node) {
	i.Name = i.Node.Attr("name")
	for cur := node.FirstChild(); cur != nil; cur = cur.NextSibling() {
		res := CompileSingleNode(cur)
		if res != nil {
			res.Compile(cur)
			i.Children = append(i.Children, res)
		}
	}
}

func (i *Variable) Apply(node xml.Node, context *ExecutionContext) {
	scope := i.Node.Attr("select")
	// if @select
	if scope != "" {
		e := xpath.Compile(scope)
		i.Value, _ = context.EvalXPath(node, e)
		//fmt.Println("VARIABLE SELECT", name, i.Value)
		return
	}

	if len(i.Children) == 0 {
		//fmt.Println("VARIABLE NIL", name, i.Value)
		i.Value = nil
		return
	}

	// if multiple children, return nodeset
	curOutput := context.OutputNode
	context.OutputNode = context.Output.CreateElementNode("RVT")
	for _, c := range i.Children {
		c.Apply(node, context)
	}
	i.Value = nil
	var outNodes xml.Nodeset
	for cur := context.OutputNode.FirstChild(); cur != nil; cur = cur.NextSibling() {
		outNodes = append(outNodes, cur)
	}
	i.Value = outNodes
	context.OutputNode = curOutput
	//fmt.Println("VARIABLE NODES", name, i.Value)
}

// Some instructions (such as xsl:attribute) require the template body
// to be instantiated as a string.

// In those cases, it is an error if any non-text nodes are generated in the
// course of evaluation.
func (i *XsltInstruction) evalChildrenAsText(node xml.Node, context *ExecutionContext) (out string, err error) {
	curOutput := context.OutputNode
	context.OutputNode = context.Output.CreateElementNode("RVT")
	for _, c := range i.Children {
		c.Apply(node, context)
	}
	for cur := context.OutputNode.FirstChild(); cur != nil; cur = cur.NextSibling() {
		//TODO: generate error if cur is not a text node
		out = out + cur.Content()
	}
	context.OutputNode = curOutput
	return
}

// Evaluate an instruction and generate output nodes
func (i *XsltInstruction) Apply(node xml.Node, context *ExecutionContext) {
	//push context if children to apply!
	switch i.Name {
	case "apply-templates":
		scope := i.Node.Attr("select")
		mode := i.Node.Attr("mode")
		// #current is a 2.0 keyword
		if mode != context.Mode && mode != "#current" {
			context.Mode = mode
		}
		if scope == "" {
			children := context.ChildrenOf(node)
			if i.sorting != nil {
				i.Sort(children, context)
			}
			total := len(children)
			oldpos, oldtotal := context.XPathContext.GetContextPosition()
			for i, cur := range children {
				context.XPathContext.SetContextPosition(i+1, total)
				context.Style.processNode(cur, context)
			}
			context.XPathContext.SetContextPosition(oldpos, oldtotal)
			return
		}
		e := xpath.Compile(scope)
		// TODO: ensure we apply strip-space if required
		nodes, _ := context.EvalXPathAsNodeset(node, e)
		if i.sorting != nil {
			i.Sort(nodes, context)
		}
		total := len(nodes)
		for i, cur := range nodes {
			context.XPathContext.SetContextPosition(i+1, total)
			context.Style.processNode(cur, context)
		}
	case "number":
		i.numbering(node, context)

	case "text":
		r := context.Output.CreateTextNode(i.Node.Content())
		context.OutputNode.AddChild(r)

	case "call-template":
		name := i.Node.Attr("name")
		t, ok := context.Style.NamedTemplates[name]
		if ok && t != nil {
			t.Apply(node, context)
		}

	case "element":
		ename := i.Node.Attr("name")
		if strings.ContainsRune(ename, '{') {
			ename = evalAVT(ename, node, context)
		}
		r := context.Output.CreateElementNode(ename)
		ns := i.Node.Attr("namespace")
		if strings.ContainsRune(ns, '{') {
			ns = evalAVT(ns, node, context)
		}
		if ns != "" {
			//TODO: search through namespaces in-scope
			// not just top-level stylesheet mappings
			prefix, _ := context.Style.NamespaceMapping[ns]
			r.SetNamespace(prefix, ns)
		} else {
			// if no namespace specified, use the default namespace
			// in scope at this point in the stylesheet
			defaultNS := context.DefaultNamespace(i.Node)
			if defaultNS != "" {
				r.SetNamespace("", defaultNS)
			}
		}
		context.OutputNode.AddChild(r)
		context.DeclareStylesheetNamespacesIfRoot(r)
		old := context.OutputNode
		context.OutputNode = r

		attsets := i.Node.Attr("use-attribute-sets")
		if attsets != "" {
			asets := strings.Fields(attsets)
			for _, attsetname := range asets {
				a, _ := context.Style.AttributeSets[attsetname]
				if a != nil {
					a.Apply(node, context)
				}
			}
		}
		for _, c := range i.Children {
			c.Apply(node, context)
		}
		context.OutputNode = old

	case "comment":
		val, _ := i.evalChildrenAsText(node, context)
		r := context.Output.CreateCommentNode(val)
		context.OutputNode.AddChild(r)

	case "processing-instruction":
		name := i.Node.Attr("name")
		val, _ := i.evalChildrenAsText(node, context)
		//TODO: it is an error if val contains "?>"
		r := context.Output.CreatePINode(name, val)
		context.OutputNode.AddChild(r)

	case "attribute":
		aname := i.Node.Attr("name")
		if strings.ContainsRune(aname, '{') {
			aname = evalAVT(aname, node, context)
		}
		ahref := i.Node.Attr("namespace")
		if strings.ContainsRune(ahref, '{') {
			ahref = evalAVT(ahref, node, context)
		}
		val, _ := i.evalChildrenAsText(node, context)
		if ahref == "" {
			context.OutputNode.SetAttr(aname, val)
		} else {
			decl := context.OutputNode.DeclaredNamespaces()
			dfound := false
			for _, d := range decl {
				if ahref == d.Uri {
					dfound = true
					break
				}
			}
			if !dfound {
				//TODO: increment val of generated prefix
				context.OutputNode.DeclareNamespace("ns_1", ahref)
			}
			//if a QName, we ignore the prefix when setting namespace
			if strings.Contains(aname, ":") {
				aname = aname[strings.Index(aname, ":")+1:]
			}
			context.OutputNode.SetNsAttr(ahref, aname, val)
		}
		//context.OutputNode.AddChild(a)

	case "value-of":
		e := xpath.Compile(i.Node.Attr("select"))
		context.RegisterXPathNamespaces(i.Node.MyDocument().Root())
		o, _ := context.EvalXPath(node, e)
		switch output := o.(type) {
		case []xml.Node:
			if len(output) > 0 {
				content := output[0].Content()
				//don't bother creating a text node for an empty string
				if content != "" {
					r := context.Output.CreateTextNode(content)
					context.OutputNode.AddChild(r)
				}
			}
		case float64:
			r := context.Output.CreateTextNode(fmt.Sprintf("%v", output))
			context.OutputNode.AddChild(r)
		case string:
			r := context.Output.CreateTextNode(output)
			context.OutputNode.AddChild(r)
		}
	case "when":
	case "if":
		e := xpath.Compile(i.Node.Attr("test"))
		if context.EvalXPathAsBoolean(node, e) {
			for _, c := range i.Children {
				c.Apply(node, context)
			}
		}
	case "attribute-set":
		othersets := i.Node.Attr("use-attribute-sets")
		if othersets != "" {
			asets := strings.Fields(othersets)
			for _, attsetname := range asets {
				a := context.Style.LookupAttributeSet(attsetname)
				if a != nil {
					a.Apply(node, context)
				}
			}
		}
		for _, c := range i.Children {
			c.Apply(node, context)
		}
	case "fallback":
		for _, c := range i.Children {
			c.Apply(node, context)
		}
	case "otherwise":
		for _, c := range i.Children {
			c.Apply(node, context)
		}

	case "choose":
		for _, c := range i.Children {
			if c.(*XsltInstruction).Node.Name() == "when" {
				xp := xpath.Compile(c.(*XsltInstruction).Node.Attr("test"))
				if context.EvalXPathAsBoolean(node, xp) {
					c.Apply(node, context)
					break
				}
			} else {
				c.Apply(node, context)
			}
		}
	case "copy":
		switch node.NodeType() {
		case xml.XML_TEXT_NODE:
			r := context.Output.CreateTextNode(node.Content())
			context.OutputNode.AddChild(r)
		case xml.XML_ATTRIBUTE_NODE:
			aname := node.Name()
			ahref := node.Namespace()
			val := node.Content()
			if ahref == "" {
				context.OutputNode.SetAttr(aname, val)
			} else {
				context.OutputNode.SetNsAttr(ahref, aname, val)
			}
		case xml.XML_COMMENT_NODE:
			r := context.Output.CreateCommentNode(node.Content())
			context.OutputNode.AddChild(r)
		case xml.XML_PI_NODE:
			name := node.Name()
			r := context.Output.CreatePINode(name, node.Content())
			context.OutputNode.AddChild(r)
		case xml.XML_ELEMENT_NODE:
			aname := node.Name()
			r := context.Output.CreateElementNode(aname)
			ns := node.Namespace()
			if ns != "" {
				//TODO: search through namespaces in-scope
				prefix, _ := context.Style.NamespaceMapping[ns]
				r.SetNamespace(prefix, ns)
			}
			context.OutputNode.AddChild(r)

			//copy namespace declarations
			for _, decl := range node.DeclaredNamespaces() {
				r.DeclareNamespace(decl.Prefix, decl.Uri)
			}

			old := context.OutputNode
			context.OutputNode = r

			attsets := i.Node.Attr("use-attribute-sets")
			if attsets != "" {
				asets := strings.Fields(attsets)
				for _, attsetname := range asets {
					a := context.Style.LookupAttributeSet(attsetname)
					if a != nil {
						a.Apply(node, context)
					}
				}
			}
			for _, c := range i.Children {
				c.Apply(node, context)
			}
			context.OutputNode = old
		}
	case "for-each":
		scope := i.Node.Attr("select")
		e := xpath.Compile(scope)
		nodes, _ := context.EvalXPathAsNodeset(node, e)
		if i.sorting != nil {
			i.Sort(nodes, context)
		}
		total := len(nodes)
		for j, cur := range nodes {
			context.PushStack()
			context.XPathContext.SetContextPosition(j+1, total)
			for _, c := range i.Children {
				c.Apply(cur, context)
				switch v := c.(type) {
				case *Variable:
					_ = context.DeclareLocalVariable(v.Name, "", v)
				}
			}
			context.PopStack()
		}
	case "copy-of":
		scope := i.Node.Attr("select")
		e := xpath.Compile(scope)
		nodes, _ := context.EvalXPathAsNodeset(node, e)
		total := len(nodes)
		for j, cur := range nodes {
			context.XPathContext.SetContextPosition(j+1, total)
			i.copyToOutput(cur, context, true)
		}

	case "apply-imports":
	case "message":
	case "with-param":
		fmt.Println("TODO instruction ", i.Name)
	default:
		fmt.Println("UNKNOWN instruction ", i.Name)
	}
}

func (i *XsltInstruction) numbering(node xml.Node, context *ExecutionContext) {
	//level
	level := i.Node.Attr("level")
	if level == "" {
		level = "single"
	}
	//count
	count := i.Node.Attr("count")
	if count == "" {
		//TODO: qname (should match NS as well
		count = node.Name()
	}
	//from
	from := i.Node.Attr("from")
	//value
	valattr := i.Node.Attr("value")
	//format
	format := i.Node.Attr("format")
	if format == "" {
		format = "1"
	}
	//lang
	//letter-value
	//grouping-seperator
	//grouping-size

	var numbers []int
	//if value, just use that!
	if valattr != "" {
		v, _ := node.EvalXPath(valattr, context)
		if v == nil {
			numbers = append(numbers, 0)
		} else {
			numbers = append(numbers, int(v.(float64)))
		}
	} else {

		target := findTarget(node, count)
		v := countNodes(level, target, count, from)
		numbers = append(numbers, v)

		if level == "multiple" {
			for cur := target.Parent(); cur != nil; cur = cur.Parent() {
				v = countNodes(level, cur, count, from)
				if v > 0 {
					numbers = append(numbers, v)
				}
			}
			if len(numbers) > 1 {
				for i, j := 0, len(numbers)-1; i < j; i, j = i+1, j-1 {
					numbers[i], numbers[j] = numbers[j], numbers[i]
				}
			}
		}
	}

	// level = multiple
	// count preceding siblings AT EACH LEVEL

	// format using the format string
	outtxt := formatNumbers(numbers, format)
	r := context.Output.CreateTextNode(outtxt)
	context.OutputNode.AddChild(r)
}

func matchesOne(node xml.Node, patterns []*CompiledMatch) bool {
	for _, m := range patterns {
		if m.EvalMatch(node, "", nil) {
			return true
		}
	}
	return false
}

func findTarget(node xml.Node, count string) (target xml.Node) {
	countExpr := CompileMatch(count, nil)
	for cur := node; cur != nil; cur = cur.Parent() {
		if matchesOne(cur, countExpr) {
			return cur
		}
	}
	return
}

func countNodes(level string, node xml.Node, count string, from string) (num int) {
	//compile count, from matches
	countExpr := CompileMatch(count, nil)
	fromExpr := CompileMatch(from, nil)
	cur := node
	for cur != nil {
		//if matches count, num++
		if matchesOne(cur, countExpr) {
			num = num + 1
		}
		//if matches from, break
		if matchesOne(cur, fromExpr) {
			break
		}

		t := cur
		cur = cur.PreviousSibling()

		//for level = 'any' we need walk the preceding axis
		//this finds the last descendant of our previous sibling
		if cur != nil && level == "any" {
			for cur.LastChild() != nil {
				cur = cur.LastChild()
			}
		}

		// no preceding node; for level='any' go to ancestor
		if cur == nil && level == "any" {
			cur = t.Parent()
		}

		//break on document node
	}
	return
}

func (i *XsltInstruction) copyToOutput(node xml.Node, context *ExecutionContext, recursive bool) {
	switch node.NodeType() {
	case xml.XML_TEXT_NODE:
		if context.UseCDataSection(context.OutputNode) {
			r := context.Output.CreateCDataNode(node.Content())
			context.OutputNode.AddChild(r)
		} else {
			r := context.Output.CreateTextNode(node.Content())
			context.OutputNode.AddChild(r)
		}
	case xml.XML_ATTRIBUTE_NODE:
		aname := node.Name()
		ahref := node.Namespace()
		val := node.Content()
		if ahref == "" {
			context.OutputNode.SetAttr(aname, val)
		} else {
			context.OutputNode.SetNsAttr(ahref, aname, val)
		}
	case xml.XML_COMMENT_NODE:
		r := context.Output.CreateCommentNode(node.Content())
		context.OutputNode.AddChild(r)
	case xml.XML_PI_NODE:
		name := node.Attr("name")
		r := context.Output.CreatePINode(name, node.Content())
		context.OutputNode.AddChild(r)
	case xml.XML_ELEMENT_NODE:
		aname := node.Name()
		r := context.Output.CreateElementNode(aname)
		ns := node.Namespace()
		if ns != "" {
			//TODO: search through namespaces in-scope
			prefix, _ := context.Style.NamespaceMapping[ns]
			r.SetNamespace(prefix, ns)
		}
		context.OutputNode.AddChild(r)

		//copy namespace declarations
		for _, decl := range node.DeclaredNamespaces() {
			r.DeclareNamespace(decl.Prefix, decl.Uri)
		}

		old := context.OutputNode
		context.OutputNode = r
		if recursive {
			//copy attributes
			for _, attr := range node.Attributes() {
				i.copyToOutput(attr, context, recursive)
			}
			for cur := node.FirstChild(); cur != nil; cur = cur.NextSibling() {
				i.copyToOutput(cur, context, recursive)
			}
		}
		context.OutputNode = old
	}
}

func (e *LiteralResultElement) Compile(node xml.Node) {
	for cur := node.FirstChild(); cur != nil; cur = cur.NextSibling() {
		res := CompileSingleNode(cur)
		if res != nil {
			res.Compile(cur)
			e.Children = append(e.Children, res)
		}
	}
}

func ResolveAlias(style *Stylesheet, alias, auri string) (prefix, uri string) {
	k, ok := style.NamespaceAlias[alias]
	//short circuit if prefix is not aliased
	if !ok {
		return alias, auri
	}
	for uri, prefix = range style.NamespaceMapping {
		if k == prefix {
			return
		}
	}
	return "", ""
}

func (e *LiteralResultElement) IsExtension(node xml.Node, context *ExecutionContext) bool {
	ns := e.Node.Namespace()
	prefix, _ := context.Style.NamespaceMapping[ns]
	if prefix != "" {
		for _, ex := range context.Style.ExtensionPrefixes {
			if ex == prefix {
				return true
			}
		}
	}
	return false
}

func (e *LiteralResultElement) Apply(node xml.Node, context *ExecutionContext) {
	//TODO: recognize extension elements at compile time
	if e.IsExtension(node, context) {
		for _, c := range e.Children {
			inst := c.(*XsltInstruction)
			if inst != nil && inst.Name == "fallback" {
				c.Apply(node, context)
			}
		}
		return
	}

	r := context.Output.CreateElementNode(e.Node.Name())
	context.OutputNode.AddChild(r)
	context.DeclareStylesheetNamespacesIfRoot(r)
	ns := e.Node.Namespace()
	if ns != "" {
		prefix, _ := context.Style.NamespaceMapping[ns]
		_, ns = ResolveAlias(context.Style, prefix, ns)
		//TODO: handle aliases
		r.SetNamespace(prefix, ns)
	}

	attsets := ""
	for _, attr := range e.Node.Attributes() {
		//fmt.Println(attr.Namespace(), attr.Name(), attr.Content())
		txt := attr.Content()
		if strings.ContainsRune(txt, '{') {
			txt = evalAVT(txt, node, context)
		}
		if attr.Namespace() != "" {
			if attr.Namespace() == XSLT_NAMESPACE {
				if attr.Name() == "use-attribute-sets" {
					attsets = txt
				}
			} else {
				r.SetNsAttr(attr.Namespace(), attr.Name(), txt)
			}
		} else {
			r.SetAttr(attr.Name(), txt)
		}
	}

	old := context.OutputNode
	context.OutputNode = r

	if attsets != "" {
		asets := strings.Fields(attsets)
		for _, attsetname := range asets {
			a := context.Style.LookupAttributeSet(attsetname)
			if a != nil {
				a.Apply(node, context)
			}
		}
	}
	for _, c := range e.Children {
		c.Apply(node, context)
	}
	context.OutputNode = old
}

// Evaluate an attribute value template
func evalAVT(input string, node xml.Node, context *ExecutionContext) (out string) {
	var start, pos int
	var inSQlit, inDQlit bool
	for {
		if pos >= len(input) {
			break
		}
		r, width := utf8.DecodeRuneInString(input[pos:])
		pos += width
		if r == '\'' {
			inSQlit = !inSQlit
		}
		if r == '"' {
			inDQlit = !inDQlit
		}
		if r == '{' {
			// if we're not the last character
			if pos < len(input) {
				// check for doubled opening brace
				peek, w := utf8.DecodeRuneInString(input[pos:])
				if peek == '{' {
					out = out + input[start:pos]
					pos += w
					start = pos
					continue
				}
			}
			out = out + input[start:pos-width]
			start = pos
		}
		if r == '}' {
			if inSQlit || inDQlit {
				continue
			}
			// if we're not the last character
			if pos < len(input) {
				// check for doubled closing brace
				peek, w := utf8.DecodeRuneInString(input[pos:])
				if peek == '}' {
					out = out + input[start:pos]
					pos += w
					start = pos
					continue
				}
			}
			expr := input[start : pos-width]
			ret, _ := context.EvalXPath(node, expr)
			switch val := ret.(type) {
			case []xml.Node:
				for _, n := range val {
					out = out + n.Content()
				}
			case float64:
				out = out + fmt.Sprintf("%v", val)
			case string:
				out = out + val
			}
			start = pos
		}
	}
	out = out + input[start:pos]
	return
}

func (t *TextOutput) Compile(node xml.Node) {
}

func (t *TextOutput) Apply(node xml.Node, context *ExecutionContext) {
	if context.UseCDataSection(context.OutputNode) {
		r := context.Output.CreateCDataNode(t.Content)
		context.OutputNode.AddChild(r)
	} else {
		r := context.Output.CreateTextNode(t.Content)
		context.OutputNode.AddChild(r)
	}
}

func (template *Template) AddChild(child CompiledStep) {
	template.Children = append(template.Children, child)
}

func (template *Template) CompileContent(node xml.Node) {
	//parse the content and register the match pattern
	for cur := node.FirstChild(); cur != nil; cur = cur.NextSibling() {
		res := CompileSingleNode(cur)
		if res != nil {
			res.Compile(cur)
			template.AddChild(res)
		}
	}
}

func CompileSingleNode(node xml.Node) (step CompiledStep) {
	switch node.NodeType() {
	case xml.XML_ELEMENT_NODE:
		ns := node.Namespace()
		// element, extension namespace = extension
		if ns == XSLT_NAMESPACE {
			// element, XSLT namespace = instruction
			switch node.Name() {
			case "variable":
				step = &Variable{Node: node}
			case "param":
				step = &Variable{Node: node}
			default:
				step = &XsltInstruction{Name: node.Name(), Node: node}
			}
		} else {
			// element other namespace = LRE
			step = &LiteralResultElement{Node: node}
		}
	// text, CDATA node
	case xml.XML_TEXT_NODE, xml.XML_CDATA_SECTION_NODE:
		if !IsBlank(node) {
			step = &TextOutput{Content: node.Content()}
		}
	}
	return
}

func (template *Template) Apply(node xml.Node, context *ExecutionContext) {
	//init local scope
	context.PushStack()
	//populate any params (including those passed via with-params)
	for _, c := range template.Children {
		c.Apply(node, context)
		switch v := c.(type) {
		case *Variable:
			_ = context.DeclareLocalVariable(v.Name, "", v)
		}
	}
	context.PopStack()
	//apply sequence ctr
	//for each node in compiled template body
	// if xsl:message
	// if forwards-compatible
	//   apply fallback
	// break out of loop if terminated by xsl:message
}
