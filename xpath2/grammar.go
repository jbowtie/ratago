package xpath2

import "github.com/jbowtie/kowhai"

func XPathGrammar() (g *kowhai.Grammar) {
	g = kowhai.CreateGrammar()

	// Common symbols
	LPAREN := g.Symbol("(")
	RPAREN := g.Symbol(")")
	DCOLON := g.Symbol("::")
	NCNAME := g.Type(int(TT_NCNAME))

	// QNAME matches both a QNAME and an NCNAME
	g.UnionSingleTerms("QNAME", g.Type(int(TT_QNAME)), NCNAME)

	//[1] XPath ::= Expr
	g.CreateRule("XPath", g.Lookup("Expr"))
	//[2] Expr ::= ExprSingle ("," ExprSingle)*
	g.CreateRule("Expr", g.Lookup("ExprSingle"), g.Star(g.Symbol(","), g.Lookup("ExprSingle")))
	//[3] ExprSingle ::= ForExpr | QuantifiedExpr | IfExpr | OrExpr
	g.CreateRule("ExprSingle", g.Lookup("ForExpr"))
	g.CreateRule("ExprSingle", g.Lookup("QuantifiedExpr"))
	g.CreateRule("ExprSingle", g.Lookup("IfExpr"))
	g.CreateRule("ExprSingle", g.Lookup("OrExpr"))
	//[4] ForExpr 	   ::=    	SimpleForClause "return" ExprSingle
	g.CreateRule("ForExpr", g.Lookup("SimpleForClause"), g.Symbol("return"), g.Lookup("ExprSingle"))
	//[5] SimpleForClause 	   ::=    	"for" "$" VarName "in" ExprSingle ("," "$" VarName "in" ExprSingle)*
	g.CreateRule("SimpleForClause", g.Symbol("for"), g.Symbol("$"), g.Lookup("VarName"), g.Symbol("in"), g.Lookup("ExprSingle"),
		g.Star(g.Symbol(","), g.Symbol("$"), g.Lookup("VarName"), g.Symbol("in"), g.Lookup("ExprSingle")))
	//[6] QuantifiedExpr 	   ::=    	("some" | "every") "$" VarName "in" ExprSingle ("," "$" VarName "in" ExprSingle)* "satisfies" ExprSingle
	g.CreateRule("QuantifiedExpr", g.Or(g.Symbol("some"), g.Symbol("every")), g.Symbol("$"), g.Lookup("VarName"), g.Symbol("in"), g.Lookup("ExprSingle"),
		g.Star(g.Symbol(","), g.Symbol("$"), g.Lookup("VarName"), g.Symbol("in"), g.Lookup("ExprSingle")), g.Symbol("satisfies"), g.Lookup("ExprSingle"))
	//[7] IfExpr ::= "if" "(" Expr ")" "then" ExprSingle "else" ExprSingle
	g.CreateRule("IfExpr", g.Symbol("if"), LPAREN, g.Lookup("Expr"), RPAREN, g.Symbol("then"), g.Lookup("ExprSingle"), g.Symbol("else"), g.Lookup("ExprSingle"))
	//[8] OrExpr 	   ::=    	AndExpr ( "or" AndExpr )*
	g.CreateRule("OrExpr", g.Lookup("AndExpr"), g.Star(g.Symbol("or"), g.Lookup("AndExpr")))
	//[9] AndExpr 	   ::=    	ComparisonExpr ( "and" ComparisonExpr )*
	g.CreateRule("AndExpr", g.Lookup("ComparisonExpr"), g.Star(g.Symbol("and"), g.Lookup("ComparisonExpr")))
	//[10] ComparisonExpr 	   ::=    	RangeExpr ( (ValueComp | GeneralComp | NodeComp) RangeExpr )?
	g.CreateRule("ComparisonExpr", g.Lookup("RangeExpr"), g.Optional(g.Or(g.Lookup("ValueComp"), g.Lookup("GeneralComp"), g.Lookup("NodeComp")), g.Lookup("RangeExpr")))
	//[11] RangeExpr 	   ::=    	AdditiveExpr ( "to" AdditiveExpr )?
	g.CreateRule("RangeExpr", g.Lookup("AdditiveExpr"), g.Optional(g.Symbol("to"), g.Lookup("AdditiveExpr")))
	//[12] AdditiveExpr 	   ::=    	MultiplicativeExpr ( ("+" | "-") MultiplicativeExpr )*
	g.CreateRule("AdditiveExpr", g.Lookup("MultiplicativeExpr"), g.Star(g.Or(g.Symbol("+"), g.Symbol("-")), g.Lookup("MultiplicativeExpr")))
	//[13] MultiplicativeExpr 	   ::=    	UnionExpr ( ("*" | "div" | "idiv" | "mod") UnionExpr )*
	g.CreateRule("MultiplicativeExpr", g.Lookup("UnionExpr"), g.Star(g.Or(g.Symbol("*"), g.Symbol("div"), g.Symbol("idiv"), g.Symbol("mod")), g.Lookup("UnionExpr")))
	//[14] UnionExpr 	   ::=    	IntersectExceptExpr ( ("union" | "|") IntersectExceptExpr )*
	g.CreateRule("UnionExpr", g.Lookup("IntersectExceptExpr"), g.Star(g.Or(g.Symbol("union"), g.Symbol("|")), g.Lookup("IntersectExceptExpr")))
	//[15] IntersectExceptExpr 	   ::=    	InstanceofExpr ( ("intersect" | "except") InstanceofExpr )*
	g.CreateRule("IntersectExceptExpr", g.Lookup("InstanceofExpr"), g.Star(g.Or(g.Symbol("intersect"), g.Symbol("except")), g.Lookup("InstanceofExpr")))
	//[16] InstanceofExpr 	   ::=    	TreatExpr ( "instance" "of" SequenceType )?
	g.CreateRule("InstanceofExpr", g.Lookup("TreatExpr"), g.Optional(g.Symbol("instance"), g.Symbol("of"), g.Lookup("SequenceType")))
	//[17] TreatExpr 	   ::=    	CastableExpr ( "treat" "as" SequenceType )?
	g.CreateRule("TreatExpr", g.Lookup("CastableExpr"), g.Optional(g.Symbol("treat"), g.Symbol("as"), g.Lookup("SequenceType")))
	//[18] CastableExpr 	   ::=    	CastExpr ( "castable" "as" SingleType )?
	g.CreateRule("CastableExpr", g.Lookup("CastExpr"), g.Optional(g.Symbol("castable"), g.Symbol("as"), g.Lookup("SingleType")))
	//[19] CastExpr 	   ::=    	UnaryExpr ( "cast" "as" SingleType )?
	g.CreateRule("CastExpr", g.Lookup("UnaryExpr"), g.Optional(g.Symbol("cast"), g.Symbol("as"), g.Lookup("SingleType")))
	//[20] UnaryExpr 	   ::=    	("-" | "+")* ValueExpr
	g.CreateRule("UnaryExpr", g.Star(g.Or(g.Symbol("-"), g.Symbol("+"))), g.Lookup("ValueExpr"))
	//[21] ValueExpr 	   ::=    	PathExpr
	g.CreateRule("ValueExpr", g.Lookup("PathExpr"))
	//[22] GeneralComp 	   ::=    	"=" | "!=" | "<" | "<=" | ">" | ">="
	g.UnionSingleTerms("GeneralComp", g.Symbol("="), g.Symbol("!="), g.Symbol("<"), g.Symbol("<="), g.Symbol(">"), g.Symbol(">="))
	//[23] ValueComp 	   ::=    	"eq" | "ne" | "lt" | "le" | "gt" | "ge"
	g.UnionSingleTerms("ValueComp", g.Symbol("eq"), g.Symbol("ne"), g.Symbol("lt"), g.Symbol("le"), g.Symbol("gt"), g.Symbol("ge"))
	//[24] NodeComp 	   ::=    	"is" | "<<" | ">>"
	g.UnionSingleTerms("NodeComp", g.Symbol("is"), g.Symbol("<<"), g.Symbol(">>"))
	//[25] PathExpr 	   ::=    	("/" RelativePathExpr?) | ("//" RelativePathExpr) | RelativePathExpr 	// xgs: leading-lone-slash
	g.CreateRule("PathExpr", g.Symbol("/"), g.Optional(g.Lookup("RelativePathExpr")))
	g.CreateRule("PathExpr", g.Symbol("//"), g.Lookup("RelativePathExpr"))
	g.CreateRule("PathExpr", g.Lookup("RelativePathExpr"))
	//[26] RelativePathExpr 	   ::=    	StepExpr (("/" | "//") StepExpr)*
	g.CreateRule("RelativePathExpr", g.Lookup("StepExpr"), g.Star(g.Or(g.Symbol("/"), g.Symbol("//")), g.Lookup("StepExpr")))
	//[27] StepExpr 	   ::=    	FilterExpr | AxisStep
	g.CreateRule("StepExpr", g.Lookup("FilterExpr"))
	g.CreateRule("StepExpr", g.Lookup("AxisStep"))
	//[28] AxisStep 	   ::=    	(ReverseStep | ForwardStep) PredicateList
	g.CreateRule("AxisStep", g.Or(g.Lookup("ReverseStep"), g.Lookup("ForwardStep")), g.Lookup("PredicateList"))
	//[29] ForwardStep 	   ::=    	(ForwardAxis NodeTest) | AbbrevForwardStep
	g.CreateRule("ForwardStep", g.Lookup("AbbrevForwardStep"))
	g.CreateRule("ForwardStep", g.Lookup("ForwardAxis"), g.Lookup("NodeTest"))
	//[30] ForwardAxis 	   ::=    	("child" "::") | ("descendant" "::") | ("attribute" "::") | ("self" "::") | ("descendant-or-self" "::") | ("following-sibling" "::") | ("following" "::") | ("namespace" "::")
	g.CreateRule("ForwardAxis", g.Symbol("child"), DCOLON)
	g.CreateRule("ForwardAxis", g.Symbol("descendant"), DCOLON)
	g.CreateRule("ForwardAxis", g.Symbol("attribute"), DCOLON)
	g.CreateRule("ForwardAxis", g.Symbol("self"), DCOLON)
	g.CreateRule("ForwardAxis", g.Symbol("descendant-or-self"), DCOLON)
	g.CreateRule("ForwardAxis", g.Symbol("following-sibling"), DCOLON)
	g.CreateRule("ForwardAxis", g.Symbol("following"), DCOLON)
	g.CreateRule("ForwardAxis", g.Symbol("namespace"), DCOLON)
	//[31] AbbrevForwardStep 	   ::=    	"@"? NodeTest
	g.CreateRule("AbbrevForwardStep", g.Optional(g.Symbol("@")), g.Lookup("NodeTest"))
	//[32] ReverseStep 	   ::=    	(ReverseAxis NodeTest) | AbbrevReverseStep
	g.CreateRule("ReverseStep", g.Lookup("AbbrevReverseStep"))
	g.CreateRule("ReverseStep", g.Lookup("ReverseAxis"), g.Lookup("NodeTest"))
	//[33] ReverseAxis 	   ::=    	("parent" "::") | ("ancestor" "::") | ("preceding-sibling" "::") | ("preceding" "::") | ("ancestor-or-self" "::")
	g.CreateRule("ReverseAxis", g.Symbol("parent"), DCOLON)
	g.CreateRule("ReverseAxis", g.Symbol("ancestor"), DCOLON)
	g.CreateRule("ReverseAxis", g.Symbol("preceding-sibling"), DCOLON)
	g.CreateRule("ReverseAxis", g.Symbol("preceding"), DCOLON)
	g.CreateRule("ReverseAxis", g.Symbol("ancestor-or-self"), DCOLON)
	//[34] AbbrevReverseStep 	   ::=    	".."
	g.CreateRule("AbbrevReverseStep", g.Symbol(".."))
	//[35] NodeTest 	   ::=    	KindTest | NameTest
	g.UnionSingleTerms("NodeTest", g.Lookup("KindTest"), g.Lookup("NameTest"))
	//[36] NameTest 	   ::=    	QName | Wildcard
	g.UnionSingleTerms("NameTest", g.Lookup("QNAME"), g.Lookup("Wildcard"))
	//[37] Wildcard 	   ::=    	"*" | (NCName ":" "*") | ("*" ":" NCName) 	// ws: explicit
	g.CreateRule("Wildcard", g.Symbol("*"))
	g.CreateRule("Wildcard", NCNAME, g.Symbol(":"), g.Symbol("*"))
	g.CreateRule("Wildcard", g.Symbol("*"), g.Symbol(":"), NCNAME)
	//[38] FilterExpr 	   ::=    	PrimaryExpr PredicateList
	g.CreateRule("FilterExpr", g.Lookup("PrimaryExpr"), g.Lookup("PredicateList"))
	//[39] PredicateList 	   ::=    	Predicate*
	g.CreateRule("PredicateList", g.Star(g.Lookup("Predicate")))
	//[40] Predicate 	   ::=    	"[" Expr "]"
	g.CreateRule("Predicate", g.Symbol("["), g.Lookup("Expr"), g.Symbol("]"))
	//[41] PrimaryExpr ::= Literal | VarRef | ParenthesizedExpr | ContextItemExpr | FunctionCall
	g.UnionSingleTerms("PrimaryExpr", g.Lookup("Literal"), g.Lookup("VarRef"),
		g.Lookup("ParenthesizedExpr"), g.Lookup("ContextItemExpr"), g.Lookup("FunctionCall"))
	//[42] Literal ::= NumericLiteral | StringLiteral
	g.UnionSingleTerms("Literal", g.Lookup("NumericLiteral"), g.Type(int(TT_STRING)))
	//[43] NumericLiteral 	::= IntegerLiteral | DecimalLiteral | DoubleLiteral
	g.UnionSingleTerms("NumericLiteral", g.Type(int(TT_INT)), g.Type(int(TT_DECIMAL)), g.Type(int(TT_DOUBLE)))
	//[44] VarRef ::= "$" VarName
	g.CreateRule("VarRef", g.Symbol("$"), g.Lookup("VarName"))
	//[45] VarName         ::=    	QName
	g.CreateRule("VarName", g.Lookup("QNAME"))
	//[46] ParenthesizedExpr ::= "(" Expr? ")"
	g.CreateRule("ParenthesizedExpr", LPAREN, g.Optional(g.Lookup("Expr")), RPAREN)
	//[47] ContextItemExpr ::= "."
	g.CreateRule("ContextItemExpr", g.Symbol("."))
	//[48] FunctionCall 	   ::=    	QName "(" (ExprSingle ("," ExprSingle)*)? ")" 	// xgs: reserved-function-names // gn: parens
	g.CreateRule("FunctionCall", g.Lookup("QNAME"), LPAREN, g.Optional(g.Lookup("ExprSingle"), g.Star(g.Symbol(","), g.Lookup("ExprSingle"))), RPAREN)
	//[49] SingleType 	   ::=    	AtomicType "?"?
	g.CreateRule("SingleType", g.Lookup("AtomicType"), g.Optional(g.Symbol("?")))
	//[50] SequenceType 	   ::=    	("empty-sequence" "(" ")") | (ItemType OccurrenceIndicator?)
	g.CreateRule("SequenceType", g.Symbol("empty-sequence"), LPAREN, RPAREN)
	g.CreateRule("SequenceType", g.Lookup("ItemType"), g.Optional(g.Lookup("OccurrenceIndicator")))
	//[51] OccurrenceIndicator 	   ::=    	"?" | "*" | "+" 	// xgs: occurrence-indicators
	g.UnionSingleTerms("OccurrenceIndicator", g.Symbol("?"), g.Symbol("*"), g.Symbol("+"))
	//[52] ItemType 	   ::=    	KindTest | ("item" "(" ")") | AtomicType
	g.CreateRule("ItemType", g.Lookup("KindTest"))
	g.CreateRule("ItemType", g.Symbol("item"), LPAREN, RPAREN)
	g.CreateRule("ItemType", g.Lookup("AtomicType"))
	//[53] AtomicType 	   ::=    	QName
	g.CreateRule("AtomicType", g.Lookup("QNAME"))
	//[54] KindTest 	   ::=    	DocumentTest | ElementTest | AttributeTest | SchemaElementTest | SchemaAttributeTest | PITest | CommentTest | TextTest | AnyKindTest
	g.UnionSingleTerms("KindTest", g.Lookup("DocumentTest"), g.Lookup("ElementTest"), g.Lookup("AttributeTest"),
		g.Lookup("SchemaElementTest"), g.Lookup("SchemaAttributeTest"), g.Lookup("PITest"), g.Lookup("CommentTest"), g.Lookup("TextTest"), g.Lookup("AnyKindTest"))
	//[55] AnyKindTest 	   ::=    	"node" "(" ")"
	g.CreateRule("AnyKindTest", g.Symbol("node"), LPAREN, RPAREN)
	//[56] DocumentTest 	   ::=    	"document-node" "(" (ElementTest | SchemaElementTest)? ")"
	g.CreateRule("DocumentTest", g.Symbol("document-node"), LPAREN, g.Optional(g.Or(g.Lookup("ElementTest"), g.Lookup("SchemaElementTest"))), RPAREN)
	//[57] TextTest 	   ::=    	"text" "(" ")"
	g.CreateRule("TextTest", g.Symbol("text"), LPAREN, RPAREN)
	//[58] CommentTest 	   ::=    	"comment" "(" ")"
	g.CreateRule("CommentTest", g.Symbol("comment"), LPAREN, RPAREN)
	//[59] PITest 	   ::=    	"processing-instruction" "(" (NCName | StringLiteral)? ")"
	g.CreateRule("PITest", g.Symbol("processing-instruction"), LPAREN, g.Optional(g.Or(NCNAME, g.Type(int(TT_STRING)))), RPAREN)
	//[60] AttributeTest 	   ::=    	"attribute" "(" (AttribNameOrWildcard ("," TypeName)?)? ")"
	g.CreateRule("AttributeTest", g.Symbol("attribute"), LPAREN, g.Optional(g.Lookup("AttribNameOrWildcard"), g.Optional(g.Symbol(","), g.Lookup("TypeName"))), RPAREN)
	//[61] AttribNameOrWildcard 	   ::=    	AttributeName | "*"
	g.UnionSingleTerms("AttribNameOrWildcard", g.Lookup("AttributeName"), g.Symbol("*"))
	//[62] SchemaAttributeTest 	   ::=    	"schema-attribute" "(" AttributeDeclaration ")"
	g.CreateRule("SchemaAttributeTest", g.Symbol("schema-attribute"), LPAREN, g.Lookup("AttributeDeclaration"), RPAREN)
	//[63] AttributeDeclaration 	   ::=    	AttributeName
	g.CreateRule("AttributeDeclaration", g.Lookup("AttributeName"))
	//[64] ElementTest 	   ::=    	"element" "(" (ElementNameOrWildcard ("," TypeName "?"?)?)? ")"
	g.CreateRule("ElementTest", g.Symbol("element"), LPAREN, g.Optional(g.Lookup("ElementNameOrWildcard"), g.Optional(g.Symbol(","), g.Lookup("TypeName"), g.Optional(g.Symbol("?")))), RPAREN)
	//[65] ElementNameOrWildcard 	   ::=    	ElementName | "*"
	g.UnionSingleTerms("ElementNameOrWildcard", g.Lookup("ElementName"), g.Symbol("*"))
	//[66] SchemaElementTest 	   ::=    	"schema-element" "(" ElementDeclaration ")"
	g.CreateRule("SchemaElementTest", g.Symbol("schema-element"), LPAREN, g.Lookup("ElementDeclaration"), RPAREN)
	//[67] ElementDeclaration 	   ::=    	ElementName
	g.CreateRule("ElementDeclaration", g.Lookup("ElementName"))
	//[68] AttributeName   ::=    	QName
	g.CreateRule("AttributeName", g.Lookup("QNAME"))
	//[69] ElementName 	   ::=    	QName
	g.CreateRule("ElementName", g.Lookup("QNAME"))
	//[70] TypeName 	   ::=    	QName
	g.CreateRule("TypeName", g.Lookup("QNAME"))

	g.SetStart("XPath")
	return g
}
