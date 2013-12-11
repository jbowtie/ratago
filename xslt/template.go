package xslt

import (
	"container/list"
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
	Node     xml.Node
}

// Literal result elements are any elements in a template
// that are not in the xsl namespace.
// They are copied to the output document.
type LiteralResultElement struct {
	Node     xml.Node
	Children []CompiledStep
}

// Stylesheet text nodes
type TextOutput struct {
	Content string
}

// Used to represent an xsl:variable or xsl:param
type Variable struct {
	Name     string
	Node     xml.Node
	Children []CompiledStep
	Value    interface{}
}

// Compile the variable.
//
// TODO: compile the XPath expression and determine if it is a constant
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

// Applying a variable node is calculating its value.
func (i *Variable) Apply(node xml.Node, context *ExecutionContext) {
	scope := i.Node.Attr("select")
	// if @select
	if scope != "" {
		e := xpath.Compile(scope)
		var err error
		context.RegisterXPathNamespaces(i.Node)
		i.Value, err = context.EvalXPath(node, e)
		if err != nil {
			fmt.Println("Error evaluating variable", i.Name, err)
		}
		//fmt.Println("VARIABLE SELECT", i.Name, i.Value)
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
	context.PushStack()
	for _, c := range i.Children {
		c.Apply(node, context)
		switch v := c.(type) {
		case *Variable:
			_ = context.DeclareLocalVariable(v.Name, "", v)
		}
	}
	context.PopStack()
	i.Value = nil
	var outNodes xml.Nodeset
	for cur := context.OutputNode.FirstChild(); cur != nil; cur = cur.NextSibling() {
		outNodes = append(outNodes, cur)
	}
	i.Value = outNodes
	context.OutputNode = curOutput
	//fmt.Println("VARIABLE NODES", name, i.Value)
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
		switch v := c.(type) {
		case *Variable:
			_ = context.DeclareLocalVariable(v.Name, "", v)
		}
	}
	context.OutputNode = old
}

// Evaluate an attribute value template
func evalAVT(input string, node xml.Node, context *ExecutionContext) (out string) {
	var start, pos int
	var inSQlit, inDQlit bool
	for pos < len(input) {
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
			case "param", "with-param":
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

func selectParamValue(param *Variable, withParams []*Variable) (out *Variable) {
	for _, p := range withParams {
		if param.Name == p.Name {
			return p
		}
	}
	return param
}

func (template *Template) Apply(node xml.Node, context *ExecutionContext, params []*Variable) {
	//init local scope
	oldStack := context.Stack
	context.Stack = *new(list.List)
	context.PushStack()

	//for each node in compiled template body
	// if xsl:message
	// if forwards-compatible
	//   apply fallback
	for _, c := range template.Children {
		c.Apply(node, context)
		switch v := c.(type) {
		case *Variable:
			//populate any params (including those passed via with-params)
			if IsXsltName(v.Node, "param") {
				v = selectParamValue(v, params)
			}
			_ = context.DeclareLocalVariable(v.Name, "", v)
		}
	}
	// break out of loop if terminated by xsl:message
	//restore any existing stack
	context.PopStack()
	context.Stack = oldStack
}
