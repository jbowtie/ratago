package xpath2

import (
	//"fmt"
	"strings"
	"unicode/utf8"
)

// add XPathLexer interface
// lex comments
// lex qname/ncname
// lex number
// lex terminal?

// lex terminal - default state
//   -- if quote, lex string
//   -- if (:, lex comment
//   -- recognize delimiting operators as token
//   -- skip whitespace
// lex identifier
//   -- accept NCName characters
//   -- if next char is :, switch to lex QName
//   -- if token is keyword, emit keyword token else emit NCName
//   -- switch to lex terminal
// lex QName
//  -- accept NCName characters
//  -- switch to lex terminal
type XPathTokenType int

const (
	TT_END      XPathTokenType = iota
	TT_INT                     //integer literal
	TT_DECIMAL                 //decimal literal
	TT_DOUBLE                  //double literal
	TT_STRING                  //string literal (may contain escaped quotes)
	TT_NCNAME                  // NCName (valid XML name) see http://www.w3.org/TR/REC-xml-names/#NT-NCName
	TT_QNAME                   // QName (namespace-qualified name) see http://www.w3.org/TR/REC-xml-names/#NT-QName
	TT_KEYWORD                 // as defined in http://www.w3.org/TR/xpath20/#id-terminal-delimitation
	TT_TERMINAL                // other terminal symbol (mostly punctuation)
	//TT_COMMENT comments are ignored
)

type XPathToken struct {
	Token_Type XPathTokenType
	Value      string
}

func (t *XPathToken) AsValue() string {
	return t.Value
}

func (t *XPathToken) TokenType() int {
	return int(t.Token_Type)
}

type stateFn func(*XPathLexer) stateFn

type XPathLexer struct {
	Input  string
	start  int
	pos    int
	width  int
	Tokens chan *XPathToken
}

func (l *XPathLexer) Run() {
	for state := lexTerminal; state != nil; {
		state = state(l)
	}
	close(l.Tokens)
}

const eof = -1

// emit passes an item back to the client.
func (l *XPathLexer) emit(t XPathTokenType) {
	l.Tokens <- &XPathToken{t, l.Input[l.start:l.pos]}
	l.start = l.pos
}

func (l *XPathLexer) next() (r rune) {
	if l.pos >= len(l.Input) {
		l.width = 0
		return eof
	}
	r, l.width = utf8.DecodeRuneInString(l.Input[l.pos:])
	l.pos += l.width
	return r
}

// ignore skips over the pending input before this point.
func (l *XPathLexer) ignore() {
	l.start = l.pos
}

// backup Tokens back one rune.
// Can be called only once per call of next.
func (l *XPathLexer) backup() {
	l.pos -= l.width
}

// peek returns but does not consume
// the next rune in the input.
func (l *XPathLexer) peek() rune {
	r := l.next()
	l.backup()
	return r
}

// accept consumes the next rune
// if it's from the valid set.
func (l *XPathLexer) accept(valid string) bool {
	if strings.IndexRune(valid, l.next()) >= 0 {
		return true
	}
	l.backup()
	return false
}

// acceptRun consumes a run of runes from the valid set.
func (l *XPathLexer) acceptRun(valid string) {
	for strings.IndexRune(valid, l.next()) >= 0 {
	}
	l.backup()
}

func lexTerminal(l *XPathLexer) stateFn {
	// if input starts with delimiting terminal
	for {
		switch r := l.next(); {
		case r == eof:
			if l.pos > l.start {
				l.emit(TT_TERMINAL)
			}
			l.emit(TT_END)
			return nil
		case r == '(':
			if l.peek() == ':' {
				l.next()
				return lexComment
			}
			l.emit(TT_TERMINAL)
			return lexTerminal
		case r == '+' || r == '-' || ('0' <= r && r <= '9'):
			l.backup()
			return lexNumber
		case r == ' ' || r == '\t' || r == '\r':
			l.ignore()
		case r == '\'' || r == '"':
			l.backup()
			return lexStringLiteral
		case r == '<':
			if l.peek() == '<' {
				l.next()
			}
			if l.peek() == '=' {
				l.next()
			}
			l.emit(TT_TERMINAL)
			return lexTerminal
		case r == '>':
			if l.peek() == '>' {
				l.next()
			}
			if l.peek() == '=' {
				l.next()
			}
			l.emit(TT_TERMINAL)
			return lexTerminal
		case r == '.':
			w := l.width
			r2 := l.peek()
			if r2 == '.' {
				l.next()
				l.emit(TT_TERMINAL)
				return lexTerminal
			}
			if '0' <= r2 && r2 <= '9' {
				l.pos -= w
				return lexNumber
			}
			l.emit(TT_TERMINAL)
			return lexTerminal
		case r == '/':
			if l.peek() == '/' {
				l.next()
			}
			l.emit(TT_TERMINAL)
			return lexTerminal
		case r == ':':
			if l.peek() == ':' {
				l.next()
			}
			l.emit(TT_TERMINAL)
			return lexTerminal
		case isNameStartChar(r):
			l.backup()
			return lexQName
		case isDelimitingOperator(r):
			if l.pos-l.start > l.width {
				l.backup()
			}
			l.emit(TT_TERMINAL)
			return lexTerminal
		default:
		}
	}
	if l.pos > l.start {
		l.emit(TT_TERMINAL)
	}
	l.emit(TT_END)
	return nil
}

func lexComment(l *XPathLexer) stateFn {
	depth := 1
	for {
		switch r := l.next(); {
		case r == '(':
			if l.peek() == ':' {
				depth += 1
			}
		case r == ':':
			if l.peek() == ')' {
				l.next()
				depth -= 1
			}
			if depth == 0 {
				l.ignore()
				return lexTerminal
			}
		default:
		}
	}
	//TODO: error to reach here
	return nil
}

func lexStringLiteral(l *XPathLexer) stateFn {
	q := l.next()
	for {
		switch r := l.next(); {
		case r == q:
			if l.peek() == q {
				l.next()
				continue
			}
			l.emit(TT_STRING)
			return lexTerminal
		default:
		}
	}
	//TODO: error to reach here
	return nil
}

func lexNumber(l *XPathLexer) stateFn {
	l.accept("+-")
	digits := "1234567890"
	l.acceptRun(digits)
	tt := TT_INT
	if l.accept(".") {
		l.acceptRun(digits)
		tt = TT_DECIMAL
	}
	if l.accept("eE") {
		l.accept("+-")
		l.acceptRun("0123456789")
		tt = TT_DOUBLE
	}
	//if we've just got '-' or '+' emit as terminal
	str := l.Input[l.start:l.pos]
	if str == "-" || str == "+" {
		tt = TT_TERMINAL
	}
	l.emit(tt)
	return lexTerminal
}

func lexQName(l *XPathLexer) stateFn {
	for isNameChar(l.next()) {
	}
	l.backup()
	//Assume we have an NCName
	tt := TT_NCNAME
	//check for QName
	if l.peek() == ':' {
		//remember position
		pos := l.pos
		//consume colon
		l.next()
		if isNameStartChar(l.peek()) {
			tt = TT_QNAME
			//handle second NCName
			for isNameChar(l.next()) {
			}
			l.backup()
		} else {
			//restore to before colon
			l.pos = pos
		}
	}
	//check to see if we have a keyword
	word := l.Input[l.start:l.pos]
	if keywords[word] {
		tt = TT_KEYWORD
	}
	l.emit(tt)
	return lexTerminal
}

//handles delimiters not otherwise handled by lexTerminal
func isDelimitingOperator(r rune) bool {
	termChars := "$()*+,-?@[]|="
	if strings.IndexRune(termChars, r) >= 0 {
		return true
	}
	return false
}

//TODO: consider XML 1.1 defintiion of NCName
func isNameStartChar(r rune) bool {
	//[A-Z] | "_" | [a-z] | [#xC0-#xD6] | [#xD8-#xF6] | [#xF8-#x2FF] | [#x370-#x37D] | [#x37F-#x1FFF] | [#x200C-#x200D] |
	//[#x2070-#x218F] | [#x2C00-#x2FEF] | [#x3001-#xD7FF] | [#xF900-#xFDCF] | [#xFDF0-#xFFFD] | [#x10000-#xEFFFF]
	return r == '_' ||
		r >= 'A' && r <= 'Z' ||
		r >= 'a' && r <= 'z' ||
		r >= 0xC0 && r <= 0xD6 ||
		r >= 0xD8 && r <= 0xF6 ||
		r >= 0xF8 && r <= 0x2FF ||
		r >= 0x370 && r <= 0x37D ||
		r >= 0x37F && r <= 0x1FFF ||
		r >= 0x200C && r <= 0x200D ||
		r >= 0x2070 && r <= 0x218F ||
		r >= 0x2C00 && r <= 0x2FEF ||
		r >= 0x3001 && r <= 0xD7FF ||
		r >= 0xF900 && r <= 0xFDCF ||
		r >= 0xFDF0 && r <= 0xFFFD ||
		r >= 0x10000 && r <= 0xEFFFF
}

func isNameChar(r rune) bool {
	// "-" | "." | [0-9] | #xB7 | [#x0300-#x036F] | [#x203F-#x2040]
	return isNameStartChar(r) ||
		r == '-' || r == '.' ||
		r >= '0' && r <= '9' ||
		r == 0xB7 || r >= 0x300 && r <= 0x36F ||
		r >= 0x203F && r <= 0x2040
}

var keywords = map[string]bool{"ancestor": true, "ancestor-or-self": true, "and": true, "as": true, "attribute": true, "cast": true, "castable": true, "child": true, "comment": true, "descendant": true, "descendant-or-self": true, "div": true, "document-node": true, "element": true, "else": true, "empty-sequence": true, "eq": true, "every": true, "except": true, "external": true, "following": true, "following-sibling": true, "for": true, "ge": true, "gt": true, "idiv": true, "if": true, "in": true, "instance": true, "intersect": true, "is": true, "item": true, "le": true, "lt": true, "mod": true, "namespace": true, "ne": true, "node": true, "of": true, "or": true, "parent": true, "preceding": true, "preceding-sibling": true, "processing-instruction": true, "return": true, "satisfies": true, "schema-attribute": true, "schema-element": true, "self": true, "some": true, "text": true, "then": true, "to": true, "treat": true, "union": true}
