package xslt

import (
	"fmt"
	"github.com/moovweb/gokogiri/xml"
	"github.com/moovweb/gokogiri/xpath"
	"strings"
)

// Most xsl elements are compiled to an instruction.
//
type XsltInstruction struct {
	Node     xml.Node
	Name     string
	Children []CompiledStep
	sorting  []*sortCriteria
}

// Compile the instruction.
//
// TODO: we should validate the structure during this step
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
		// TODO: determine with-params at compile time
		var params []*Variable
		for _, cur := range i.Children {
			switch p := cur.(type) {
			case *Variable:
				if IsXsltName(p.Node, "with-param") {
					p.Apply(node, context)
					params = append(params, p)
				}
			}
		}
		// By default, scope is children of current node
		if scope == "" {
			children := context.ChildrenOf(node)
			if i.sorting != nil {
				i.Sort(children, context)
			}
			total := len(children)
			oldpos, oldtotal := context.XPathContext.GetContextPosition()
			for i, cur := range children {
				context.XPathContext.SetContextPosition(i+1, total)
				context.Style.processNode(cur, context, params)
			}
			context.XPathContext.SetContextPosition(oldpos, oldtotal)
			return
		}
		context.RegisterXPathNamespaces(i.Node)
		e := xpath.Compile(scope)
		// TODO: ensure we apply strip-space if required
		nodes, err := context.EvalXPathAsNodeset(node, e)
		if err != nil {
			fmt.Println("apply-templates @select", err)
		}
		if i.sorting != nil {
			i.Sort(nodes, context)
		}
		total := len(nodes)
		for i, cur := range nodes {
			context.XPathContext.SetContextPosition(i+1, total)
			context.Style.processNode(cur, context, params)
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
			// TODO: determine with-params at compile time
			var params []*Variable
			for _, cur := range i.Children {
				switch p := cur.(type) {
				case *Variable:
					if IsXsltName(p.Node, "with-param") {
						p.Apply(node, context)
						params = append(params, p)
					}
				}
			}
			t.Apply(node, context, params)
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
			if !dfound && ahref != XML_NAMESPACE {
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
		disableEscaping := i.Node.Attr("disable-output-escaping") == "yes"

		context.RegisterXPathNamespaces(i.Node)
		o, _ := context.EvalXPath(node, e)
		switch output := o.(type) {
		case []xml.Node:
			if len(output) > 0 {
				content := output[0].Content()
				//don't bother creating a text node for an empty string
				if content != "" {
					r := context.Output.CreateTextNode(content)
					if disableEscaping {
						fmt.Println("Disable escaping")
						r.DisableOutputEscaping()
						//r.SetName("textnoenc")
					}
					context.OutputNode.AddChild(r)
				}
			}
		case float64:
			r := context.Output.CreateTextNode(fmt.Sprintf("%v", output))
			context.OutputNode.AddChild(r)
		case string:
			r := context.Output.CreateTextNode(output)
			if disableEscaping {
				r.SetName("textnoenc")
			}
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
		//i.copyToOutput(cur, context, false)
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
			context.Current = cur
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
		context.RegisterXPathNamespaces(i.Node)
		nodes, _ := context.EvalXPathAsNodeset(node, e)
		total := len(nodes)
		for j, cur := range nodes {
			context.XPathContext.SetContextPosition(j+1, total)
			i.copyToOutput(cur, context, true)
		}

	case "message":
		val, _ := i.evalChildrenAsText(node, context)
		terminate := i.Node.Attr("terminate")
		if terminate == "yes" {
			//TODO: fixup error flow to terminate more gracefully
			panic(val)
		} else {
			fmt.Println(val)
		}
	case "apply-imports":
		fmt.Println("TODO handle xsl:apply-imports instruction")
	default:
		hasFallback := false
		for _, c := range i.Children {
			switch v := c.(type) {
			case *XsltInstruction:
				if v.Name == "fallback" {
					c.Apply(node, context)
					hasFallback = true
					break
				}
			}
		}
		if !hasFallback {
			fmt.Println("UNKNOWN instruction ", i.Name)
		}
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
	case xml.XML_NAMESPACE_DECL:
		//in theory this should work
		//in practice it's a little complicated due to the fact
		//that namespace declarations don't map to the node type
		//very well
		//will need to revisit
		//context.OutputNode.DeclareNamespace(node.Name(), node.Content())
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
