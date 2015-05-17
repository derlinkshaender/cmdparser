package cmdparser

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"text/scanner"
)

func NewParser() *CommandParser {
	return &CommandParser{
		options:   0,
		inputLine: "",
		grammar:   map[string]string{},
		rules:     map[string]*RuleStruct{},
	}
}

func (theParser *CommandParser) SetOptions(options uint64) {
	theParser.options = options
}

func (theParser *CommandParser) SetCommandGrammar(cg map[string]string) {
	for k, v := range cg {
		theParser.grammar[k] = v
		theParser.rules[k] = theParser.prepareRule(k, v)
	}
}

func (theParser *CommandParser) SetInputString(inputLine string) {
	theParser.inputLine = inputLine
	theParser.TokenizeCommandLine(inputLine)
}

func (theParser *CommandParser) TokenizeCommandLine(line string) {
	var theScanner scanner.Scanner
	var result []*CmdToken = []*CmdToken{}
	theScanner.Init(strings.NewReader(line))
	theScanner.Mode = scanner.ScanFloats | scanner.ScanIdents | scanner.ScanInts | scanner.ScanStrings
	tok := theScanner.Scan()
	for tok != scanner.EOF && tok != COMMENTCHAR {
		s := theScanner.TokenText()
		theToken := &CmdToken{
			Type:     tok,
			Text:     s,
			Position: theScanner.Position,
		}
		result = append(result, theToken)
		tok = theScanner.Scan()
	}
	theParser.tokenList = result
}

func (theParser *CommandParser) splitRule(ruleString string) ([]string, int) {
	var temp []string
	var result []string = []string{}
	var resultType int

	if strings.Index(ruleString, CHOICESTRING) > 0 {
		resultType = Choice
		temp = strings.Split(ruleString, CHOICESTRING)
	} else {
		resultType = Sequence
		temp = strings.Split(ruleString, " ")
	}

	for i, _ := range temp {
		if strings.TrimSpace(temp[i]) != "" {
			result = append(result, strings.TrimSpace(temp[i]))
		}
	}
	return result, resultType
}

func (theParser *CommandParser) expressionType(expr string) int {
	var result int
	switch expr[0] {
	case '"':
		result = StringExpr
	case '\'':
		result = CharExpr
	case '[':
		result = ClassExpr
	case '!':
		result = DataTypeExpr
	default:
		result = SymbolExpr
	}
	return result
}

func (theParser *CommandParser) expressionCardinality(expr string) int {
	var result int
	switch expr[len(expr)-1] {
	case '*':
		result = CardinalityZeroOrMore
	case '+':
		result = CardinalityOneOrMore
	case '?':
		result = CardinalityZeroOrOne
	default:
		result = CardinalityOne
	}
	return result
}

func (theParser *CommandParser) prepareRule(name, expression string) *RuleStruct {
	items := []RuleItem{}
	list, typ := theParser.splitRule(expression)
	for _, li := range list {
		item := RuleItem{
			Cardinality: theParser.expressionCardinality(li),
			ExprType:    theParser.expressionType(li),
			ExprString:  li,
			ParseValue:  "",
			ParseType:   scanner.EOF,
		}
		if item.Cardinality != CardinalityOne {
			item.ExprString = item.ExprString[:len(item.ExprString)-1]
		}

		items = append(items, item)
	}
	rs := &RuleStruct{
		Name:  name,
		Items: items,
		Type:  typ,
	}
	return rs
}

func (theParser *CommandParser) matchItem(ruleItem *RuleItem) bool {
	var itemMatch bool
	var err error
	if theParser.options&OptionDebug != 0 {
		fmt.Print("  Trying item ", ruleItem.ExprString, " ")
		switch ruleItem.Cardinality {
		case CardinalityOne:
			fmt.Println("Exactly ONE")
		case CardinalityOneOrMore:
			fmt.Println("ONE or more")
		case CardinalityZeroOrMore:
			fmt.Println("ZERO or more")
		case CardinalityZeroOrOne:
			fmt.Println("ZERO or ONE")
		}
		if theParser.currIndex < len(theParser.tokenList) {
			fmt.Println("  Token: ", theParser.currIndex, CHOICESTRING, len(theParser.tokenList)-1, ":", theParser.tokenList[theParser.currIndex])
		} else {
			fmt.Println("  Token unavailable, parsed beyond end.")
		}
	}
	ruleItem.Seen = true
	if ruleItem.ExprType == SymbolExpr {
		if theParser.currIndex < len(theParser.tokenList) {
			s := ruleItem.ExprString
			itemMatch, theParser.currIndex = theParser.matchRule(theParser.rules[s])
		} else {
			if ruleItem.Cardinality == CardinalityZeroOrMore || ruleItem.Cardinality == CardinalityZeroOrOne {
				itemMatch = true
			}
		}
	} else {
		if theParser.currIndex < len(theParser.tokenList) {
			switch ruleItem.ExprType {
			case StringExpr:
				unq, err := strconv.Unquote(ruleItem.ExprString)
				if err != nil {
					return false
				}
				itemMatch = theParser.tokenList[theParser.currIndex].Text == unq
			case CharExpr:
				itemMatch = theParser.tokenList[theParser.currIndex].Text == ruleItem.ExprString
			case ClassExpr:
				itemMatch, err = regexp.MatchString(ruleItem.ExprString, theParser.tokenList[theParser.currIndex].Text)
				if err != nil {
					itemMatch = false
				}
			case DataTypeExpr:
				dataType := strings.ToLower(ruleItem.ExprString[1:])
				switch dataType {
				case "string":
					itemMatch = theParser.tokenList[theParser.currIndex].Type == scanner.String
				case "int":
					itemMatch = theParser.tokenList[theParser.currIndex].Type == scanner.Int
				case "float":
					itemMatch = theParser.tokenList[theParser.currIndex].Type == scanner.Float
				case "bool":
					s := strings.ToLower(theParser.tokenList[theParser.currIndex].Text)
					itemMatch = theParser.tokenList[theParser.currIndex].Type == scanner.String &&
						(s == "on" || s == "off" || s == "true" || s == "false")
				}
			}
			if itemMatch {
				ruleItem.ParseValue = theParser.tokenList[theParser.currIndex].Text
				ruleItem.ParseType = theParser.tokenList[theParser.currIndex].Type
				theParser.currIndex++
			}
		} else {
			itemMatch = false
		}
	}
	if !itemMatch && (ruleItem.Cardinality == CardinalityZeroOrMore || ruleItem.Cardinality == CardinalityZeroOrOne) {
		itemMatch = true
	}
	if theParser.options&OptionDebug != 0 {
		fmt.Println("  >", itemMatch, theParser.currIndex)
	}
	theParser.tokenIndex = theParser.currIndex
	return itemMatch
}

func (theParser *CommandParser) matchRule(rule *RuleStruct) (bool, int) {
	if theParser.options&OptionDebug != 0 {
		fmt.Println("Trying to match ", rule.Name)
	}
	var match bool
	//var newIndex int = theParser.tokenIndex
	if rule.Type == Sequence {
		// all must match
		for i, _ := range rule.Items {
			match = theParser.matchItem(&rule.Items[i])
			if !match {
				break
			}
		}
	} else if rule.Type == Choice {
		// check if any of them matched
		match = false
		for i, _ := range rule.Items {
			im := theParser.matchItem(&rule.Items[i])
			if im {
				match = true
				//newIndex = theParser.currIndex
			}
		}
		if !match { // parse error
			return false, len(theParser.tokenList)
		}
	} else {
		fmt.Println("You should not be here ...")
		panic(fmt.Errorf("Invalid rule type %v", rule.Type))
	}
	theParser.rules[rule.Name].seen = true
	if theParser.options&OptionDebug != 0 {
		fmt.Println("Done rule ", rule.Name, "  (match=", match, "  newIndex=", theParser.tokenIndex, ")")
	}
	return match, theParser.tokenIndex // newIndex
}

func (theParser *CommandParser) Parse() bool {
	theParser.tokenIndex = 0
	match := false
	rule := theParser.rules["START"]
	match, theParser.tokenIndex = theParser.matchRule(rule)
	atEnd := theParser.tokenIndex == len(theParser.tokenList)
	if !atEnd {
		// if there still is stuff to parse, it's not a match ...
		match = false
	}
	theParser.IsMatch = match
	return match
}

func (theParser *CommandParser) Dump() {
	for _, rule := range theParser.rules {
		fmt.Println(rule.Name)
		for i, ri := range rule.Items {
			fmt.Println("  ", i, ri.Seen, " >", ri.ParseType, ": ", ri.ParseValue)
		}
	}
}
