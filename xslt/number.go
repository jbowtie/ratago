package xslt

import (
	"fmt"
	"github.com/jbowtie/gokogiri/xml"
	"strconv"
	"strings"
	"unicode"
)

var numLetter = "- a b c d e f g h i j k l m n o p q r s t u v w x y z"
var aLetter = strings.Split(numLetter, " ")

var units = []string{"", "one", "two", "three", "four", "five",
	"six", "seven", "eight", "nine"}
var teens = []string{"", "eleven", "twelve", "thirteen", "fourteen",
	"fifteen", "sixteen", "seventeen", "eighteen", "nineteen"}
var tens = []string{"", "ten", "twenty", "thirty", "forty",
	"fifty", "sixty", "seventy", "eighty", "ninety"}
var thousands = []string{"", "thousand", "million", "billion", "trillion",
	"quadrillion", "quintillion", "sextillion", "septillion", "octillion",
	"nonillion", "decillion", "undecillion", "duodecillion", "tredecillion",
	"quattuordecillion", "sexdecillion", "septendecillion", "octodecillion",
	"novemdecillion", "vigintillion "}

type romanEntry struct {
	letter string
	number int
}

var romanNumeralMap = []romanEntry{
	romanEntry{"M", 1000},
	romanEntry{"CM", 900},
	romanEntry{"D", 500},
	romanEntry{"CD", 400},
	romanEntry{"C", 100},
	romanEntry{"XC", 90},
	romanEntry{"L", 50},
	romanEntry{"XL", 40},
	romanEntry{"X", 10},
	romanEntry{"IX", 9},
	romanEntry{"V", 5},
	romanEntry{"IV", 4},
	romanEntry{"I", 1},
}

type RomanNumber int

func (n RomanNumber) String() (out string) {
	w := int(n)
	for _, entry := range romanNumeralMap {
		for w >= entry.number {
			out = out + entry.letter
			w = w - entry.number
		}
	}
	return out
}

//TODO: breaks after 701
func toAlphaIndex(n int) (out string) {
	w := n
	for w > 26 {
		i := w / 26
		w = w - (i * 26)
		out = out + aLetter[i]
	}
	out = out + aLetter[w]
	return
}

func toWords(num int) (out string) {
	if num == 0 {
		return "zero"
	}
	var words []string
	numStr := fmt.Sprintf("%d", num)
	numStrLen := len(numStr)
	groups := (numStrLen + 2) / 3
	if numStrLen < groups*3 {
		numStr = strings.Repeat("0", groups*3-numStrLen) + numStr
	}
	for i := 0; i < len(numStr); i = i + 3 {
		h, _ := strconv.Atoi(numStr[i : i+1])
		t, _ := strconv.Atoi(numStr[i+1 : i+2])
		u, _ := strconv.Atoi(numStr[i+2 : i+3])
		g := groups - (i/3 + 1)
		if h >= 1 {
			words = append(words, units[h])
			words = append(words, "hundred")
		}
		if t > 1 {
			words = append(words, tens[t])
			if u >= 1 {
				words = append(words, units[u])
			}
		} else {
			if t == 1 {
				if u >= 1 {
					words = append(words, teens[u])
				} else {
					words = append(words, tens[t])
				}
			} else {
				if u >= 1 {
					words = append(words, units[u])
				}
			}
			if g >= 1 && (h+t+u) > 0 {
				words = append(words, thousands[g])
			}

		}
	}
	out = strings.Join(words, " ")
	return
}

func formatNumber(n int, format string) (out string) {
	if format == "Ww" {
		out = strings.Title(toWords(n))
		return
	}
	switch format[0] {
	case '1':
		out = fmt.Sprintf("%d", n)
	case '0':
		z := fmt.Sprintf("%%0%dd", len(format))
		out = fmt.Sprintf(z, n)
	case 'I':
		out = RomanNumber(n).String()
	case 'i':
		out = strings.ToLower(RomanNumber(n).String())
	case 'a':
		out = toAlphaIndex(n)
	case 'A':
		out = strings.ToUpper(toAlphaIndex(n))
	case 'w':
		out = toWords(n)
	case 'W':
		out = strings.ToUpper(toWords(n))
	default:
		out = fmt.Sprintf("%d", n)
	}
	return
}

type fmtToken struct {
	s        string
	isNumber bool
}

func formatNumbers(numbers []int, format string) (out string) {
	//TODO: special cases: no numbers, punctuation-only format token
	if format == "" {
		format = "1"
	}
	tokens := parseFormatString(format)

	// capture the last numeric format token
	lastNum := 0
	if tokens[len(tokens)-1].isNumber {
		lastNum = len(tokens) - 1
	} else {
		if len(tokens) > 2 && tokens[len(tokens)-2].isNumber {
			lastNum = len(tokens) - 2
		}
	}

	var t fmtToken
	ti := 0
	for i, n := range numbers {
		//if there are more numbers than formatting tokens
		// each extra number is output as "." + last numeric format token
		if i > 0 && ti >= lastNum {
			out = out + "."
			t = tokens[lastNum]
		} else {
			t = tokens[ti]
		}
		if !t.isNumber {
			out = out + t.s
			ti = ti + 1
			if ti < len(tokens) {
				t = tokens[ti]
			} else {
				t = tokens[lastNum]
			}
		}
		if t.isNumber {
			out = out + formatNumber(n, t.s)
		} else {
			out = out + t.s
		}
		ti = ti + 1
	}
	// if there's a punctuation token at the end of the format string, append it
	suffix := tokens[len(tokens)-1]
	if !suffix.isNumber {
		out = out + suffix.s
	}
	return
}

func parseFormatString(format string) (tokens []fmtToken) {
	s := strings.NewReader(format)
	punct := true
	start := 0
	pos := 0
	for s.Len() > 0 {
		r, w, _ := s.ReadRune()
		if unicode.IsDigit(r) || unicode.IsLetter(r) {
			pos = pos + w
			if !punct {
				continue
			}
			punct = false
			s.UnreadRune()
			pos = pos - w
			punctok := format[start:pos]
			if punctok != "" {
				tokens = append(tokens, fmtToken{punctok, false})
			}
			start = pos
		} else {
			pos = pos + w
			if punct {
				continue
			}
			punct = true
			s.UnreadRune()
			pos = pos - w
			punctok := format[start:pos]
			if punctok != "" {
				tokens = append(tokens, fmtToken{punctok, true})
			}
			start = pos
		}
	}
	if pos > start {
		tokens = append(tokens, fmtToken{format[start:], !punct})
	}
	return
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

/*
func main() {
	formatNumbers([]int{5}, "1.")
	formatNumbers([]int{5}, "(1)")
	formatNumbers([]int{5}, "1.A.i>")
	formatNumbers([]int{5, 2}, "1.A.i>")
	formatNumbers([]int{5, 2}, "1.")
	formatNumbers([]int{5, 2}, "1")
	formatNumbers([]int{1, 2, 3}, "1+1")
}
*/
/*
public string AddOrdinal(int num)
{
    if( num <= 0 ) return num.ToString();

    switch(num % 100)
    {
        case 11:
        case 12:
        case 13:
            return num + "th";
    }

    switch(num % 10)
    {
        case 1:
            return num + "st";
        case 2:
            return num + "nd";
        case 3:
            return num + "rd";
        default:
            return num + "th";
    }

}
*/
