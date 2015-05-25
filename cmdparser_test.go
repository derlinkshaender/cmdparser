package cmdparser

import (
	"testing"
)

func Assert(t *testing.T, expr bool, msg string) {
	if !expr {
		t.Error(msg)
	}
}

func TestTokenizer(t *testing.T) {
	p := NewParser()
	p.SetInputString(`SET "key" = 4.56 FOR ' var * 3 < 15' `)
	Assert(t, len(p.tokenList) == 6, "Expected 6 tokens!")
}

type grammarTestStruct struct {
	Rule  string
	Input string
	Match bool
}

type grammarTest []grammarTestStruct

func TestGrammarRules(t *testing.T) {
	data := grammarTest{
		grammarTestStruct{Rule: `"show"+`, Input: "show show show ", Match: true},
		grammarTestStruct{Rule: `"show"+`, Input: "show show blar ", Match: false},
		grammarTestStruct{Rule: `"show"*`, Input: "show show show ", Match: true},
		grammarTestStruct{Rule: `"show"*`, Input: "blar ", Match: false},
		grammarTestStruct{Rule: `"show"`, Input: "show ", Match: true},
		grammarTestStruct{Rule: `"show"?`, Input: "show ", Match: true},
		grammarTestStruct{Rule: `"show"? `, Input: "blar ", Match: false},
		grammarTestStruct{Rule: `"show"+ "blar"`, Input: "show show blar ", Match: true},
		grammarTestStruct{Rule: `"foo" "bar"? "baz"`, Input: "foo bar baz ", Match: true},
		grammarTestStruct{Rule: `"foo" "bar"? "baz"`, Input: "foo  baz ", Match: true},
		grammarTestStruct{Rule: `"show" !string `, Input: ` show "/tmp/test.csv" `, Match: true},
		grammarTestStruct{Rule: `"show" !string `, Input: ` show 42 `, Match: false},
		grammarTestStruct{Rule: `"show" !int `, Input: ` show 42 `, Match: true},
		grammarTestStruct{Rule: `"foo" | "bar" | "baz" `, Input: ` foo `, Match: true},
		grammarTestStruct{Rule: `"foo" | "bar" | "baz" `, Input: ` show `, Match: false},
	}

	for _, entry := range data {
		p := NewParser()
		//p.SetOptions(OptionDebug)
		Grammar := map[string]string{
			"START": entry.Rule,
		}
		p.SetCommandGrammar(Grammar)
		p.SetInputString(entry.Input)
		match := p.Parse()
		if match != entry.Match {
			t.Error("Entry for rule " + entry.Rule + " failed!")
		}
	}
}

func TestCommandGrammar(t *testing.T) {
	Grammar := map[string]string{
		"START":         `"show"  FeatureClause     Options  ToClause? `,
		"ToClause":      `"to"  !string `,
		"FeatureClause": `"feature"  !string? `,
		"Options":       `TranClause | DefClause | ValueClause `,
		"TranClause":    `"translation"  LangList? `,
		"LangList":      `"lang"  !string `,
		"ValueClause":   `"unique"?  "values" `,
		"DefClause":     `"definition" `,
	}

	inputString := `show feature translation lang "xx,de,it" to "/tmp/test.csv" `
	p := NewParser()
	//p.SetOptions(OptionDebug)
	p.SetCommandGrammar(Grammar)
	p.SetInputString(inputString)
	match := p.Parse()
	Assert(t, match == true, "Should match input string, but does not!")

}

func TestExpressionGrammar(t *testing.T) {
	Grammar := map[string]string{
		"START":       ` "list" Item WhereClause?`,
		"Item":        ` "card" | "board" | "list" `,
		"WhereClause": ` "where" !expression `,
	}

	inputString := `list board where '4*(5+6) < 17' `
	p := NewParser()
	//p.SetOptions(OptionDebug)
	p.SetCommandGrammar(Grammar)
	p.SetInputString(inputString)
	match := p.Parse()
	Assert(t, match == true, "Should match input string, but does not!")
}
