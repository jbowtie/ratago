package xslt

import (
	"github.com/ThomsonReutersEikon/gokogiri/xml"
	"sort"
)

func (i *XsltInstruction) Sort(nodes []xml.Node, context *ExecutionContext) {
	ns := &NodeSorter{nodes, i.sorting, context}
	sort.Sort(ns)
}

type NodeSorter struct {
	nodes   []xml.Node
	crit    []*sortCriteria
	context *ExecutionContext
}

func (s *NodeSorter) Len() int {
	return len(s.nodes)
}

func (s *NodeSorter) Swap(i, j int) {
	s.nodes[i], s.nodes[j] = s.nodes[j], s.nodes[i]
}

func (s *NodeSorter) Less(i, j int) bool {
	//make this loop through the by array
	// return as soon as non-equal result
	return execSortFunction(s.nodes[i], s.nodes[j], s.crit, s.context)
}

type sortCriteria struct {
	sel     string
	reverse bool
	numeric bool
}

func compileSortFunction(i *XsltInstruction) (s *sortCriteria) {
	s = new(sortCriteria)
	s.sel = i.Node.Attr("select")
	if s.sel == "" {
		s.sel = "string(.)"
	}
	order := i.Node.Attr("order")
	if order == "descending" {
		s.reverse = true
	}
	datatype := i.Node.Attr("data-type")
	if datatype == "number" {
		s.numeric = true
	}
	return s
}

func execSortFunction(n1, n2 xml.Node, crits []*sortCriteria, context *ExecutionContext) bool {

	for _, crit := range crits {
		if crit.numeric {
			if execNumericSortFunction(n1, n2, crit, context) {
				return true
			}
			continue
		}

		s1, _ := context.EvalXPath(n1, crit.sel)
		s1, _ = context.XPathContext.ResultAsString()
		s2, _ := context.EvalXPath(n2, crit.sel)
		s2, _ = context.XPathContext.ResultAsString()

		if s1.(string) == s2.(string) {
			continue
		}

		if !crit.reverse {
			return s1.(string) < s2.(string)
		} else {
			return s1.(string) > s2.(string)
		}
	}
	return false
	//case-order
	//lang
	//data-type (text)
}
func execNumericSortFunction(n1, n2 xml.Node, crit *sortCriteria, context *ExecutionContext) bool {

	s1, _ := context.EvalXPath(n1, crit.sel)
	s1, _ = context.XPathContext.ResultAsNumber()
	s2, _ := context.EvalXPath(n2, crit.sel)
	s2, _ = context.XPathContext.ResultAsNumber()

	if !crit.reverse {
		if s1.(float64) < s2.(float64) {
			return true
		}
	} else {
		if s1.(float64) > s2.(float64) {
			return true
		}
	}
	return false
}

/*
package collate_test

import (
        "fmt"
        "testing"

        "code.google.com/p/go.text/collate"
        "code.google.com/p/go.text/language"
)

func ExampleCollator_Strings() {
        c := collate.New(language.Und)
        strings := []string{
                "ad",
                "ab",
                "äb",
                "ac",
        }
        c.SortStrings(strings)
        fmt.Println(strings)
        // Output: [ab äb ac ad]
    }

    type sorter []string

    func (s sorter) Len() int {
        return len(s)
    }

    func (s sorter) Swap(i, j int) {
        s[j], s[i] = s[i], s[j]
    }

    func (s sorter) Bytes(i int) []byte {
        return []byte(s[i])
    }

    func TestSort(t *testing.T) {
        c := collate.New(language.En)
        strings := []string{
            "bcd",
            "abc",
            "ddd",
        }
        c.Sort(sorter(strings))
        res := fmt.Sprint(strings)
        want := "[abc bcd ddd]"
        if res != want {
                t.Errorf("found %s; want %s", res, want)
        }
}
*/
