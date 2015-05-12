package xpath2

import (
	"fmt"
	. "github.com/smartystreets/goconvey/convey"
	"testing"
)

func Scan(input string) (tokens []*XPathToken) {
	// parse the expression
	l := &XPathLexer{Input: input, Tokens: make(chan *XPathToken)}
	go l.Run()

	for step := range l.Tokens {
		tokens = append(tokens, step)
	}
	return tokens
}

func compareToken(t *testing.T, value string, typ XPathTokenType, actual *XPathToken) {
	Convey("Comparing token", t, func() {
		eval := fmt.Sprint("Expect value: \"", value, "\"")
		etype := fmt.Sprint("Expect token type: ", typ)
		Convey(eval, func() {
			So(actual.Value, ShouldEqual, value)
		})
		Convey(etype, func() {
			So(actual.Token_Type, ShouldEqual, typ)
		})
	})
}

func TestIntLiteral(t *testing.T) {
	tokens := Scan("12 34 -001")
	compareToken(t, "12", TT_INT, tokens[0])
	compareToken(t, "34", TT_INT, tokens[1])
	compareToken(t, "-001", TT_INT, tokens[2])
}

func TestDecimalLiteral(t *testing.T) {
	tokens := Scan("0.5 .5 1.")
	compareToken(t, "0.5", TT_DECIMAL, tokens[0])
	compareToken(t, ".5", TT_DECIMAL, tokens[1])
	compareToken(t, "1.", TT_DECIMAL, tokens[2])
}

func TestDoubleLiteral(t *testing.T) {
	tokens := Scan("1.2e3 .5e1 2.e0")
	compareToken(t, "1.2e3", TT_DOUBLE, tokens[0])
	compareToken(t, ".5e1", TT_DOUBLE, tokens[1])
	compareToken(t, "2.e0", TT_DOUBLE, tokens[2])
}

func TestCommentSyntax(t *testing.T) {
	tokens := Scan("(:comment:)12(:(:nestedcomment:):)0.5")
	compareToken(t, "12", TT_INT, tokens[0])
	compareToken(t, "0.5", TT_DECIMAL, tokens[1])

	//_ = Scan("12(:xml:test and (:nested comment:):)0.5, a,b,c/xy//z,d 1.2e3 xml:test[1]")
	//_ = Scan("x \"this is a \"\"test\"\"\" return 'ye old a''pos' z")
	//_ = Scan(" a<b c<<d e<=f a>b c>>d e>=f")
}

func TestAngleBracketOperators(t *testing.T) {
	tokens := Scan("a<b c<<d")
	compareToken(t, "a", TT_NCNAME, tokens[0])
	compareToken(t, "<", TT_TERMINAL, tokens[1])
	compareToken(t, "b", TT_NCNAME, tokens[2])
	compareToken(t, "c", TT_NCNAME, tokens[3])
	compareToken(t, "<<", TT_TERMINAL, tokens[4])
	compareToken(t, "d", TT_NCNAME, tokens[5])
}

func TestDashSyntax(t *testing.T) {
	//trailing dash is part of ncname
	tokens := Scan("foo- bar")
	compareToken(t, "foo-", TT_NCNAME, tokens[0])
	compareToken(t, "bar", TT_NCNAME, tokens[1])
	//dash in middle is part of ncname
	tokens = Scan("foo-foo")
	compareToken(t, "foo-foo", TT_NCNAME, tokens[0])
	//delimited by spaces, treat as terminal
	tokens = Scan("a - b")
	compareToken(t, "a", TT_NCNAME, tokens[0])
	compareToken(t, "-", TT_TERMINAL, tokens[1])
	compareToken(t, "b", TT_NCNAME, tokens[2])
	//is delimiter, so become terminal before ncname
	tokens = Scan("foo -bar")
	compareToken(t, "foo", TT_NCNAME, tokens[0])
	compareToken(t, "-", TT_TERMINAL, tokens[1])
	compareToken(t, "bar", TT_NCNAME, tokens[2])
	//tokens = Scan("-.6 + ../foo - ./3")
}
