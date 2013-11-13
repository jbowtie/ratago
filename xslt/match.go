package xslt

import (
	"container/list"
	"github.com/moovweb/gokogiri/xml"
	"github.com/moovweb/gokogiri/xpath"
	"strconv"
	"strings"
	"unicode/utf8"
)

type StepOperation int

const (
	OP_END StepOperation = iota
	OP_ROOT
	OP_ELEM
	OP_ATTR
	OP_PARENT
	OP_ANCESTOR
	OP_ID
	OP_KEY
	OP_NS
	OP_ALL
	OP_PI
	OP_COMMENT
	OP_TEXT
	OP_NODE
	OP_PREDICATE
	OP_OR
	OP_ERROR
)

// An individual step in the pattern
type MatchStep struct {
	Op    StepOperation
	Value string
}

// The compiled match pattern
type CompiledMatch struct {
	pattern  string
	Steps    []*MatchStep
	Template *Template
}

type stateFn func(*lexer) stateFn

type lexer struct {
	input string
	start int
	pos   int
	width int //really?
	steps chan *MatchStep
}

func (l *lexer) run() {
	for state := lexNodeTest; state != nil; {
		state = state(l)
	}
	close(l.steps)

	// the | operator

	// see a ::, either set axis or emit error
	// see a :, emit op_NS?  or just modify next op?
	// inside a () consume to close, check for validity of arguments
}

const eof = -1

// emit passes an item back to the client.
func (l *lexer) emit(t StepOperation) {
	l.steps <- &MatchStep{t, l.input[l.start:l.pos]}
	l.start = l.pos
}

func (l *lexer) next() (r rune) {
	if l.pos >= len(l.input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.input[l.pos:])
	l.pos += l.width
	return r
}

// ignore skips over the pending input before this point.
func (l *lexer) ignore() {
	l.start = l.pos
}

// backup steps back one rune.
// Can be called only once per call of next.
func (l *lexer) backup() {
	l.pos -= l.width
}

// peek returns but does not consume
// the next rune in the input.
func (l *lexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

func lexNodeTest(l *lexer) stateFn {
	attr := false
	for {
		r := l.next()
		switch r {
		case '/':
			l.backup()
			if l.pos > l.start {
				if attr {
					l.emit(OP_ATTR)
				} else {
					l.emit(OP_ELEM)
				}
			}
			return lexParent
		case '(':
			l.backup()
			if attr {
				return lexAttrNodeTest
			} else {
				return lexFunctionCall
			}
		case '[':
			l.backup()
			if l.pos > l.start {
				if attr {
					l.emit(OP_ATTR)
				} else {
					l.emit(OP_ELEM)
				}
			}
			return lexPredicate
		case '@':
			l.ignore()
			attr = true
		case '*':
			if attr {
				l.emit(OP_ATTR)
			} else {
				return lexAll
			}
		case ':':
			if l.peek() == ':' {
				//axis specifier
				_ = l.next()
				axisName := l.input[l.start:l.pos]
				if axisName == "attribute::" {
					attr = true
				}
				//TODO: only child and attribute axes allowed in pattern
				l.ignore()
			} else {
				l.backup()
				l.emit(OP_NS)
				_ = l.next()
				l.ignore()
			}
		case '|':
			l.backup()
			if l.pos > l.start {
				if attr {
					l.emit(OP_ATTR)
				} else {
					l.emit(OP_ELEM)
				}
			}
			_ = l.next()
			l.emit(OP_OR)
			l.ignore()
			return lexNodeTest
		default:
		}
		//switch?
		if r == eof {
			break
		}
	}
	if l.pos > l.start {
		if attr {
			l.emit(OP_ATTR)
		} else {
			l.emit(OP_ELEM)
		}
	}
	return nil
}

func lexFunctionCall(l *lexer) stateFn {
	fnName := l.input[l.start:l.pos]
	op := OP_ERROR
	switch fnName {
	case "comment":
		op = OP_COMMENT
	case "text":
		op = OP_TEXT
	case "node":
		op = OP_NODE
	case "id":
		op = OP_ID
	case "key":
		op = OP_KEY
	case "processing-instruction":
		op = OP_PI
	}
	l.ignore()
	depth := 0
	for {
		r := l.next()
		if r == eof {
			//TODO: parse error
			break
		}
		if r == '(' {
			depth = depth + 1
		}
		if r == ')' {
			depth = depth - 1
			if depth == 0 {
				l.emit(op)
			}
		}
	}
	return lexNodeTest
}

func lexAttrNodeTest(l *lexer) stateFn {
	fnName := l.input[l.start:l.pos]
	op := OP_ERROR
	switch fnName {
	case "node":
		op = OP_ATTR
	}
	l.ignore()
	depth := 0
	for {
		r := l.next()
		if r == eof {
			//TODO: parse error
			break
		}
		if r == '(' {
			depth = depth + 1
		}
		if r == ')' {
			depth = depth - 1
			if depth == 0 {
				l.steps <- &MatchStep{op, "*"}
				l.start = l.pos
			}
		}
	}
	return lexNodeTest
}

func lexPredicate(l *lexer) stateFn {
	depth := 0
	for {
		r := l.next()
		if r == '[' {
			depth = depth + 1
		}
		if r == ']' {
			depth = depth - 1
			if depth == 0 {
				l.emit(OP_PREDICATE)
				break
			}
		}
		if r == eof {
			//TODO: parse error
			break
		}
	}
	return lexNodeTest
}

func lexParent(l *lexer) stateFn {
	_ = l.next()
	if l.peek() == '/' {
		_ = l.next()
		//we can ignore it at the root!
		if l.start == 0 {
			l.ignore()
		} else {
			l.emit(OP_ANCESTOR)
		}
		return lexNodeTest
	}
	if l.start == 0 {
		l.emit(OP_ROOT)
		return lexNodeTest
	}
	l.emit(OP_PARENT)
	return lexNodeTest
}

func lexAll(l *lexer) stateFn {
	l.emit(OP_ALL)
	return lexNodeTest
}

func parseMatchPattern(s string) (steps []*MatchStep) {
	//create a lexer
	//run the state machine
	// each state emits steps into the stream
	// when it recognizes new state returns new state
	// state returns nil when out of input
	// break out of loop and close channel
	//get the channel of steps

	//range over the steps until we have them all
	//reverse the array for fast matching?
	//assign priority/mode

	// for now shortcut the common ROOT
	if s == "/" {
		steps = []*MatchStep{&MatchStep{Op: OP_ROOT, Value: s}, &MatchStep{Op: OP_END}}
		return
	}

	ls := list.New()
	ls.PushFront(&MatchStep{Op: OP_END})

	// parse the expression
	l := &lexer{input: s, steps: make(chan *MatchStep)}
	go l.run()

	// prepend steps to avoid reversing later
	for step := range l.steps {
		//we don't want predicates at the front
		if step.Op == OP_PREDICATE {
			//TODO: fix lexer to trim outer braces
			step.Value = step.Value[1 : len(step.Value)-1]
			ls.InsertAfter(step, ls.Front())
		} else {
			ls.PushFront(step)
		}
	}

	for i := ls.Front(); i != nil; i = i.Next() {
		steps = append(steps, i.Value.(*MatchStep))
	}
	return
}

func CompileMatch(s string, t *Template) (matches []*CompiledMatch) {
	if s == "" {
		return
	}
	steps := parseMatchPattern(s)
	start := 0
	for i, step := range steps {
		if step.Op == OP_OR {
			matches = append(matches, &CompiledMatch{s, steps[start:i], t})
			start = i + 1
		}
	}
	matches = append(matches, &CompiledMatch{s, steps[start:], t})
	return
}

// Returns true if the node matches the pattern
func (m *CompiledMatch) EvalMatch(node xml.Node, mode string, context *ExecutionContext) bool {
	cur := node
	//false if wrong mode
	// #all is an XSLT 2.0 feature
	if m.Template != nil && mode != m.Template.Mode && m.Template.Mode != "#all" {
		return false
	}

	for i, step := range m.Steps {
		switch step.Op {
		case OP_END:
			return true
		case OP_ROOT:
			if cur.NodeType() != xml.XML_DOCUMENT_NODE {
				return false
			}
		case OP_ELEM:
			if cur.NodeType() != xml.XML_ELEMENT_NODE {
				return false
			}
			if step.Value != cur.Name() && step.Value != "*" {
				return false
			}
		case OP_NS:
			uri := ""
			// m.Template.Node
			if m.Template != nil {
				uri = context.LookupNamespace(step.Value, m.Template.Node)
			} else {
				uri = context.LookupNamespace(step.Value, nil)
			}
			if uri != cur.Namespace() {
				return false
			}
		case OP_ATTR:
			if cur.NodeType() != xml.XML_ATTRIBUTE_NODE {
				return false
			}
			if step.Value != cur.Name() && step.Value != "*" {
				return false
			}
		case OP_TEXT:
			if cur.NodeType() != xml.XML_TEXT_NODE && cur.NodeType() != xml.XML_CDATA_SECTION_NODE {
				return false
			}
		case OP_COMMENT:
			if cur.NodeType() != xml.XML_COMMENT_NODE {
				return false
			}
		case OP_ALL:
			if cur.NodeType() != xml.XML_ELEMENT_NODE {
				return false
			}
		case OP_PI:
			if cur.NodeType() != xml.XML_PI_NODE {
				return false
			}
		case OP_NODE:
			switch cur.NodeType() {
			case xml.XML_ELEMENT_NODE, xml.XML_CDATA_SECTION_NODE, xml.XML_TEXT_NODE, xml.XML_COMMENT_NODE, xml.XML_PI_NODE:
				// matches any of these node types
			default:
				return false
			}
		case OP_PARENT:
			cur = cur.Parent()
			if cur == nil {
				return false
			}
		case OP_ANCESTOR:
			next := m.Steps[i+1]
			if next.Op != OP_ELEM {
				return false
			}
			for {
				cur = cur.Parent()
				if cur == nil {
					return false
				}
				if next.Value == cur.Name() {
					break
				}
			}
		case OP_PREDICATE:
			// see test REC/5.2-16
			// see test REC/5.2-22
			evalFull := true
			if context != nil {

				prev := m.Steps[i-1]
				if prev.Op == OP_PREDICATE {
					prev = m.Steps[i-2]
				}
				if prev.Op == OP_ELEM || prev.Op == OP_ALL {
					parent := cur.Parent()
					sibs := context.ChildrenOf(parent)
					var clen, pos int
					for _, n := range sibs {
						if n.NodePtr() == cur.NodePtr() {
							pos = clen + 1
							clen = clen + 1
						} else {
							if n.NodeType() == xml.XML_ELEMENT_NODE {
								if n.Name() == cur.Name() || prev.Op == OP_ALL {
									clen = clen + 1
								}
							}
						}
					}
					if step.Value == "last()" {
						if pos != clen {
							return false
						}
					}
					//eval predicate should do special number handling
					postest, err := strconv.Atoi(step.Value)
					if err == nil {
						if pos != postest {
							return false
						}
					}
					opos, olen := context.XPathContext.GetContextPosition()
					context.XPathContext.SetContextPosition(pos, clen)
					result := cur.EvalXPathAsBoolean(step.Value, context)
					context.XPathContext.SetContextPosition(opos, olen)
					if result == false {
						return false
					}
					evalFull = false
				}
			}
			if evalFull {
				//if we made it this far, fall back to the more expensive option of evaluating
				// the entire pattern globally
				//TODO: cache results on first run for given document
				xp := m.pattern
				if m.pattern[0] != '/' {
					xp = "//" + m.pattern
				}
				e := xpath.Compile(xp)
				o, err := node.Search(e)
				if err != nil {
					//fmt.Println("ERROR",err)
				}
				for _, n := range o {
					if cur.NodePtr() == n.NodePtr() {
						return true
					}
				}
				return false
			}

		case OP_ID:
			//TODO: fix lexer to only put literal inside step value
			val := strings.Trim(step.Value, "()\"'")
			id := cur.MyDocument().NodeById(val)
			if id == nil || node.NodePtr() != id.NodePtr() {
				return false
			}
		case OP_KEY:
			//  TODO: make this robust
			if context != nil {
				val := strings.Trim(step.Value, "()")
				v := strings.Split(val, ",")
				keyname := strings.Trim(v[0], "\"'")
				keyval := strings.Trim(v[1], "\"'")
				key, _ := context.Style.Keys[keyname]
				if key != nil {
					o, _ := key.nodes[keyval]
					for _, n := range o {
						if cur.NodePtr() == n.NodePtr() {
							return true
						}
					}
				}
			}
			return false
		default:
			return false
		}
	}
	//in theory, OP_END means we never reach here
	// in practice, we can generate match patterns
	// that are missing OP_END due to how we handle OP_OR
	return true
}

func (m *CompiledMatch) Hash() (hash string) {
	base := m.Steps[0]
	switch base.Op {
	case OP_ATTR:
		return base.Value
	case OP_ELEM:
		return base.Value
	case OP_ALL:
		return "*"
	case OP_ROOT:
		return "/"
	}
	return
}

func (m *CompiledMatch) IsElement() bool {
	op := m.Steps[0].Op
	if op == OP_ELEM || op == OP_ROOT || op == OP_ALL {
		return true
	}
	return false
}

func (m *CompiledMatch) IsAttr() bool {
	op := m.Steps[0].Op
	return op == OP_ATTR
}

func (m *CompiledMatch) IsNode() bool {
	op := m.Steps[0].Op
	return op == OP_NODE
}

func (m *CompiledMatch) IsPI() bool {
	op := m.Steps[0].Op
	return op == OP_PI
}

func (m *CompiledMatch) IsIdKey() bool {
	op := m.Steps[0].Op
	return op == OP_ID || op == OP_KEY
}

func (m *CompiledMatch) IsText() bool {
	op := m.Steps[0].Op
	return op == OP_TEXT
}

func (m *CompiledMatch) IsComment() bool {
	op := m.Steps[0].Op
	return op == OP_COMMENT
}

func (m *CompiledMatch) endsAfter(n int) bool {
	steps := len(m.Steps)
	if n == steps {
		return true
	}
	if n+1 == steps && m.Steps[n].Op == OP_END {
		return true
	}
	return false
}

func (m *CompiledMatch) DefaultPriority() (priority float64) {
	//TODO: calculate defaults according to spec
	step := m.Steps[0]
	// *
	if step.Op == OP_ALL {
		if m.endsAfter(1) {
			return -0.5
		}
		// ns:*
		if m.endsAfter(2) && m.Steps[1].Op == OP_NS {
			return -0.25
		}
	}
	// @*
	if step.Op == OP_ATTR && step.Value == "*" {
		if m.endsAfter(1) {
			return -0.5
		}
		if m.endsAfter(2) && m.Steps[1].Op == OP_NS {
			return -0.25
		}
	}
	// text(), node(), comment()
	if step.Op == OP_TEXT || step.Op == OP_NODE || step.Op == OP_COMMENT {
		if m.endsAfter(1) {
			return -0.5
		}
	}
	// QName
	if step.Op == OP_ELEM {
		if m.endsAfter(1) {
			return 0
		}
		if m.endsAfter(2) && m.Steps[1].Op == OP_NS {
			return 0
		}
	}
	// @QName
	if step.Op == OP_ATTR && step.Value != "*" {
		if m.endsAfter(1) {
			return 0
		}
		if m.endsAfter(2) && m.Steps[1].Op == OP_NS {
			return 0
		}
	}
	return 0.5
}
