package cmdparser

import (
	"fmt"
	"golang.org/x/exp/utf8string"
	"regexp"
	"strconv"
	"strings"
	"text/scanner"
	"unicode/utf8"
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
	if theParser.options&OptionDebug != 0 {
		for i, v := range theParser.rules["START"].Items {
			fmt.Println(i, ":", v)
		}
	}
}

func (theParser *CommandParser) SetInputString(inputLine string) {
	theParser.inputLine = inputLine
	theParser.TokenizeCommandLine()
}

func (theParser *CommandParser) GolangTokenizer(line string) []*PreToken {
	var theScanner scanner.Scanner
	var result []*PreToken = []*PreToken{}
	theScanner.Init(strings.NewReader(line))
	theScanner.Mode = scanner.ScanFloats | scanner.ScanIdents | scanner.ScanInts | scanner.ScanStrings
	tok := theScanner.Scan()
	for tok != scanner.EOF && tok != COMMENTCHAR {
		s := theScanner.TokenText()
		theToken := &PreToken{
			Type:     tok,
			Text:     s,
			Position: theScanner.Position,
		}
		result = append(result, theToken)
		tok = theScanner.Scan()
	}
	return result
}

func (theParser *CommandParser) TokenizeCommandLine() {
	var preTokens []*PreToken = theParser.GolangTokenizer(theParser.inputLine)
	var postTokens []*CmdToken = []*CmdToken{}
	var haveError bool = false
	index := 0
	for {
		tok := preTokens[index]
		postTok := &CmdToken{}
		postTok.Position = tok.Position
		switch tok.Type {
		case scanner.Ident:
			low := strings.ToLower(tok.Text)
			// process booleans
			if low == "true" || low == "yes" || low == "false" || low == "no" {
				postTok.Text = low
				postTok.Type = TokenBool
				postTok.Value = (low == "yes" || low == "true")
			} else {
				postTok.Text = tok.Text
				postTok.Type = TokenIdent
				postTok.Value = tok.Text
			}
			postTokens = append(postTokens, postTok)
		case scanner.Int:
			postTok.Text = tok.Text
			postTok.Type = TokenInt
			val, err := strconv.Atoi(tok.Text)
			if err == nil {
				postTok.Value = val
			} else {
				postTok.Value = nil
				postTok.Type = TokenERR
				haveError = true
			}
			postTokens = append(postTokens, postTok)
		case scanner.Float:
			postTok.Text = tok.Text
			postTok.Type = TokenFloat
			val, err := strconv.ParseFloat(tok.Text, 64)
			if err == nil {
				postTok.Value = val
			} else {
				postTok.Value = nil
				postTok.Type = TokenERR
				haveError = true
			}
			postTokens = append(postTokens, postTok)
		case '\'':
			if tok.Text == "'" {
				tmpStr := ""
				startIndex := index
				for {
					index++
					if index >= len(preTokens) || preTokens[index].Type == '\'' {
						break
					}
					tok = preTokens[index]
					tmpStr += tok.Text
				}
				postTok.Text = tmpStr
				postTok.Type = TokenExpr
				postTok.Value = preTokens[startIndex:index]
			} else {
				postTok.Value = nil
				postTok.Type = TokenERR
				haveError = true
			}
			postTokens = append(postTokens, postTok)
		case scanner.String:
			postTok.Text = preTokens[index].Text
			postTok.Type = TokenString
			val, err := strconv.Unquote(preTokens[index].Text)
			if err == nil {
				postTok.Value = val
			} else {
				postTok.Value = nil
				postTok.Type = TokenERR
				haveError = true
			}
			postTokens = append(postTokens, postTok)
		default:
			postTok.Type = TokenChar
			postTok.Text = preTokens[index].Text
			r, _ := utf8.DecodeRuneInString(preTokens[index].Text)
			postTok.Value = r
			postTokens = append(postTokens, postTok)
		}
		if haveError {
			postTokens = nil
			theParser.TokenizerError = true
		} else {
			theParser.TokenizerError = false
		}
		index++
		if index >= len(preTokens) {
			break
		}
	}
	theParser.tokenList = postTokens
}

func (theParser *CommandParser) dump() {
	for i, v := range theParser.tokenList {
		fmt.Printf("%2d: ", i)
		fmt.Print(v.Type, "> ")
		fmt.Println(v.String())
	}
}

func copyToken(tok *CmdToken) *CmdToken {
	var result CmdToken = *tok
	return &result
}

// peek a token without advancing the input token list
func (theParser *CommandParser) peek() *CmdToken {
	var result *CmdToken = nil
	if len(theParser.tokenList) > 0 {
		result = copyToken(theParser.tokenList[0])
	}
	return result
}

// consume a token from the input token list
func (theParser *CommandParser) read() *CmdToken {
	var result *CmdToken = nil
	if len(theParser.tokenList) > 0 {
		result = theParser.tokenList[0]
		theParser.tokenList = append([]*CmdToken{}, theParser.tokenList[1:]...)
	}
	return result
}

// un-read a token back into the token list
func (theParser *CommandParser) unread(tok *CmdToken) {
	theParser.tokenList = append([]*CmdToken{tok}, theParser.tokenList...)
}

func (theParser *CommandParser) splitRule(ruleString string) ([]string, GrammarItemType) {
	var temp []string
	var result []string = []string{}
	var resultType GrammarItemType

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

func (theParser *CommandParser) expressionType(expr string) GrammarItemType {
	var result GrammarItemType
	switch expr[0] {
	case '"':
		result = IdentifierExpr
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

func (theParser *CommandParser) expressionCardinality(expr string) GrammarItemCardinality {
	var result GrammarItemCardinality
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
	items := []*RuleItem{}
	list, typ := theParser.splitRule(expression)
	for _, li := range list {
		item := &RuleItem{
			Cardinality: theParser.expressionCardinality(li),
			ExprType:    theParser.expressionType(li),
			ExprString:  li,
			ParseValue:  "",
			ParseType:   scanner.EOF,
		}

		if item.Cardinality != CardinalityOne {
			// strip out cardinality char
			item.ExprString = item.ExprString[0 : len(item.ExprString)-1]
		}

		switch item.ExprType {
		case IdentifierExpr, CharExpr:
			item.ExprString = item.ExprString[1 : len(item.ExprString)-1]
		case DataTypeExpr:
			item.ExprString = item.ExprString[1:]
		}

		switch item.ExprType {
		case CharExpr:
			s := utf8string.NewString(item.ExprString)
			r := s.At(1)
			if r == '\'' {
				item.ExprString = s.String()
			} else {
				panic("CharExpr does not contain rune literal!")
			}
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

func matchClassExpr(theClass string, tokptr *CmdToken) bool {
	itemMatch, err := regexp.MatchString(theClass, tokptr.Text)
	if err != nil {
		itemMatch = false
	} else {
		itemMatch = tokptr.Type == TokenString
	}
	return itemMatch
}

func matchDataTypeExpr(theDataType TokenType, tokptr *CmdToken) bool {
	return tokptr.Type == theDataType
}

func matchEvaluatorExpr(theExpression string, tokptr *CmdToken) bool {
	return tokptr.Type == TokenExpr
}

func getCardinality(ruleItemPtr *RuleItem) (minxOccur, maxOccur int) {
	min := 0
	max := 0
	switch ruleItemPtr.Cardinality {
	case CardinalityOne:
		min = 1
		max = 1
	case CardinalityOneOrMore:
		min = 1
		max = 1 << 32
	case CardinalityZeroOrOne:
		min = 0
		max = 1
	case CardinalityZeroOrMore:
		min = 0
		max = 1 << 32
	}
	return min, max
}

func (theParser *CommandParser) matchRuleItem(ruleItemPtr *RuleItem, tokptr *CmdToken) bool {
	var result bool = false
	if theParser.options&OptionDebug != 0 {
		fmt.Println("ruleItem:", ruleItemPtr.String())
		if tokptr != nil {
			fmt.Println("tokenPtr:", tokptr.String())
		} else {
			fmt.Println("tokenPtr is NIL!")
		}
	}
	switch ruleItemPtr.ExprType {
	case CharExpr:
		result = tokptr.Type == TokenChar && ruleItemPtr.ExprString == tokptr.Text
	case IdentifierExpr:
		result = tokptr.Type == TokenIdent && ruleItemPtr.ExprString == tokptr.Value
	case ClassExpr:
		result = matchClassExpr(ruleItemPtr.ExprString, tokptr)
	case SymbolExpr:
		theParser.unread(tokptr)
		result = theParser.matchRule(theParser.rules[ruleItemPtr.ExprString])
	case DataTypeExpr:
		dtStr := strings.ToLower(ruleItemPtr.ExprString)
		var reqType TokenType
		switch dtStr {
		case "string":
			reqType = TokenString
		case "int":
			reqType = TokenInt
		case "bool":
			reqType = TokenBool
		case "float":
			reqType = TokenFloat
		case "char":
			reqType = TokenChar
		}
		result = matchDataTypeExpr(reqType, tokptr)
	case EvaluatorExpr:
		result = tokptr.Type == TokenExpr
	}

	return result
}

func (theParser *CommandParser) matchItemWithToken(ruleItemPtr *RuleItem) bool {
	var result bool = false
	var matchCount int = 0
	minOccur, maxOccur := getCardinality(ruleItemPtr)

	for {
		match := theParser.matchRuleItem(ruleItemPtr, theParser.tokptr)

		// no annotation means ONE
		if minOccur == 1 && maxOccur == 1 {
			result = match
			break
		}
		if match {
			matchCount++
			result = match
			// match a matching Expr+ or a Expr*
			if maxOccur >= 1<<32 || (minOccur == 0 && maxOccur == 1) {
				theParser.tokptr = theParser.read()
				// break out, if token is nil => matching read until end of input
				if theParser.tokptr == nil {
					result = match
					break
				}
			}
			// too many matches?
			if maxOccur == 1 && matchCount > 1 {
				result = false
				break
			}
		} else {
			// match a Expr?
			if minOccur == 0 || matchCount >= minOccur {
				result = true
				theParser.unread(theParser.tokptr)
				break
			}
			if matchCount < minOccur {
				result = false
				break
			}
		}
	}
	return result
}

func (theParser *CommandParser) matchRule(rule *RuleStruct) bool {
	if theParser.options&OptionDebug != 0 {
		fmt.Println("Trying to match ", rule.Name)
	}
	var match bool = false

	if rule.Type == Sequence {
		// all must match
		for _, item := range rule.Items {
			theParser.tokptr = theParser.read()
			if theParser.options&OptionDebug != 0 {
				fmt.Println("Using Sequence Item:", item.String())
			}
			match = theParser.matchItemWithToken(item)
			if theParser.options&OptionDebug != 0 {
				fmt.Println("      Result:", match)
			}
			if !match {
				break
			}
		}
	} else if rule.Type == Choice {
		// check if any of them matched
		match = false
		theParser.tokptr = theParser.read()
		for _, item := range rule.Items {
			if theParser.options&OptionDebug != 0 {
				fmt.Println("Using Choice Item:", item.String())
			}
			im := theParser.matchItemWithToken(item)
			if theParser.options&OptionDebug != 0 {
				fmt.Println("      Result:", im)
			}
			if im {
				match = true
				break
			}
		}
	} else {
		fmt.Println("You should not be here ...")
		panic(fmt.Errorf("Invalid rule type %v", rule.Type))
	}

	theParser.rules[rule.Name].seen = true
	if theParser.options&OptionDebug != 0 {
		fmt.Println("Done rule ", rule.Name, "  match=", match)
	}
	return match
}

func (theParser *CommandParser) AtEnd() bool {
	return len(theParser.tokenList) == 0
}

func (theParser *CommandParser) Parse() bool {
	rule := theParser.rules["START"]
	match := theParser.matchRule(rule)
	if !theParser.AtEnd() {
		// if there still is stuff to parse, it's not a match ...
		if theParser.options&OptionDebug != 0 {
			fmt.Println("Not at end => no match")
			fmt.Println(theParser.tokenList)
		}
		match = false
	}
	theParser.IsMatch = match
	return match
}

func (theParser *CommandParser) DumpRules() {
	for _, rule := range theParser.rules {
		fmt.Println(rule.Name)
		for i, ri := range rule.Items {
			fmt.Println("  ", i, ri.Seen, " >", ri.ParseType, ": ", ri.ParseValue)
		}
	}
}
