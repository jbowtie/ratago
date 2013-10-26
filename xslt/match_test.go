package xslt

//import "github.com/moovweb/gokogiri/xml"
//import "unicode/utf8"
import "testing"

func compareStep(t *testing.T, m *MatchStep, op StepOperation, val string) {
	if m.Op != op || val != m.Value {
		t.Error("Expected", op, "=", val, "\nActual  ", m.Op, "=", m.Value)
	}
}

func TestPatternSimple(t *testing.T) {
	steps := parseMatchPattern("test")
	compareStep(t, steps[0], OP_ELEM, "test")
}

func TestPatternParent(t *testing.T) {
	steps := parseMatchPattern("foo/bar")
	compareStep(t, steps[0], OP_ELEM, "bar")
	compareStep(t, steps[1], OP_PARENT, "/")
	compareStep(t, steps[2], OP_ELEM, "foo")
}

func TestPatternRootParent(t *testing.T) {
	steps := parseMatchPattern("/bar")
	compareStep(t, steps[0], OP_ELEM, "bar")
	compareStep(t, steps[1], OP_ROOT, "/")
	compareStep(t, steps[2], OP_END, "")
}

func TestPatternAncestor(t *testing.T) {
	steps := parseMatchPattern("foo//bar")
	compareStep(t, steps[0], OP_ELEM, "bar")
	compareStep(t, steps[1], OP_ANCESTOR, "//")
	compareStep(t, steps[2], OP_ELEM, "foo")
}

func TestPatternRootAncestor(t *testing.T) {
	steps := parseMatchPattern("//bar")
	compareStep(t, steps[0], OP_ELEM, "bar")
	compareStep(t, steps[1], OP_END, "")
}

func TestPatternAttrib(t *testing.T) {
	steps := parseMatchPattern("@test")
	compareStep(t, steps[0], OP_ATTR, "test")
}

func TestPatternWildcardAttrib(t *testing.T) {
	steps := parseMatchPattern("@*")
	compareStep(t, steps[0], OP_ATTR, "*")
	compareStep(t, steps[1], OP_END, "")
}

func TestPatternAttribWithParent(t *testing.T) {
	steps := parseMatchPattern("foo/@bar")
	compareStep(t, steps[0], OP_ATTR, "bar")
	compareStep(t, steps[1], OP_PARENT, "/")
	compareStep(t, steps[2], OP_ELEM, "foo")
}

func TestPatternNodeType(t *testing.T) {
	steps := parseMatchPattern("node()")
	compareStep(t, steps[0], OP_NODE, "()")
}
func TestPatternCommentType(t *testing.T) {
	steps := parseMatchPattern("comment()")
	compareStep(t, steps[0], OP_COMMENT, "()")
}
func TestPatternTextType(t *testing.T) {
	steps := parseMatchPattern("text()")
	compareStep(t, steps[0], OP_TEXT, "()")
}
func TestPatternPiType(t *testing.T) {
	steps := parseMatchPattern("processing-instruction()")
	compareStep(t, steps[0], OP_PI, "()")
}
func TestPatternAll(t *testing.T) {
	steps := parseMatchPattern("*")
	compareStep(t, steps[0], OP_ALL, "*")
}
func TestPatternParentAll(t *testing.T) {
	steps := parseMatchPattern("foo/*")
	compareStep(t, steps[0], OP_ALL, "*")
	compareStep(t, steps[1], OP_PARENT, "/")
	compareStep(t, steps[2], OP_ELEM, "foo")
}

func TestPatternPred(t *testing.T) {
	steps := parseMatchPattern("item[position()=1]")
	compareStep(t, steps[0], OP_ELEM, "item")
	compareStep(t, steps[1], OP_PREDICATE, "position()=1")
}

func TestPatternNonTerminalPred(t *testing.T) {
	steps := parseMatchPattern("item[position()=1]/foo")
	compareStep(t, steps[0], OP_ELEM, "foo")
	compareStep(t, steps[1], OP_PARENT, "/")
	compareStep(t, steps[2], OP_ELEM, "item")
	compareStep(t, steps[3], OP_PREDICATE, "position()=1")
}

func TestPatternNestedPred(t *testing.T) {
	steps := parseMatchPattern("list[item[@foo='bar']]")
	compareStep(t, steps[0], OP_ELEM, "list")
	compareStep(t, steps[1], OP_PREDICATE, "item[@foo='bar']")
}

func TestPatternNamespace(t *testing.T) {
	steps := parseMatchPattern("foo:bar")
	compareStep(t, steps[0], OP_ELEM, "bar")
	compareStep(t, steps[1], OP_NS, "foo")
}

func TestPatternAxis(t *testing.T) {
	steps := parseMatchPattern("foo/child::bar")
	compareStep(t, steps[0], OP_ELEM, "bar")
	compareStep(t, steps[1], OP_PARENT, "/")
	compareStep(t, steps[2], OP_ELEM, "foo")
}

func TestPatternAttributeAxis(t *testing.T) {
	steps := parseMatchPattern("foo/attribute::bar")
	compareStep(t, steps[0], OP_ATTR, "bar")
	compareStep(t, steps[1], OP_PARENT, "/")
	compareStep(t, steps[2], OP_ELEM, "foo")
}

func TestPatternChildAxis(t *testing.T) {
	steps := parseMatchPattern("child::bar")
	compareStep(t, steps[0], OP_ELEM, "bar")
	compareStep(t, steps[1], OP_END, "")
}

func TestPatternOr(t *testing.T) {
	steps := parseMatchPattern("foo|bar")
	compareStep(t, steps[0], OP_ELEM, "bar")
	compareStep(t, steps[1], OP_OR, "|")
	compareStep(t, steps[2], OP_ELEM, "foo")
	compareStep(t, steps[3], OP_END, "")
}
