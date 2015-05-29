package cmdparser

import (
	"errors"
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
		options:     0,
		inputLine:   "",
		grammar:     map[string]string{},
		rules:       map[string]*RuleStruct{},
		ParseResult: map[string]CmdToken{},
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

func (theParser *CommandParser) golangTokenizer(line string) []*PreToken {
	var theScanner scanner.Scanner
	result := []*PreToken{}
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

func tokenFromIdentifier(preToken *PreToken) (*CmdToken, error) {
	low := strings.ToLower(preToken.Text)
	token := &CmdToken{
		Position: preToken.Position,
	}
	var err error
	// process booleans
	if low == "true" || low == "yes" || low == "false" || low == "no" {
		token.Text = low
		token.Type = TokenBool
		token.Value = (low == "yes" || low == "true")
	} else {
		token.Text = preToken.Text
		token.Type = TokenIdent
		token.Value = preToken.Text
	}
	return token, err
}

func tokenFromInt(preToken *PreToken) (*CmdToken, error) {
	var err error
	token := &CmdToken{
		Text:     preToken.Text,
		Type:     TokenInt,
		Position: preToken.Position,
	}
	val, err := strconv.Atoi(preToken.Text)
	if err == nil {
		token.Value = val
	} else {
		token.Value = nil
		token.Type = TokenERR
		err = errors.New("NOT_AN_INT")
	}
	return token, err
}

func tokenFromFloat(preToken *PreToken) (*CmdToken, error) {
	var err error
	token := &CmdToken{
		Text:     preToken.Text,
		Type:     TokenFloat,
		Position: preToken.Position,
	}
	val, err := strconv.ParseFloat(preToken.Text, 64)
	if err == nil {
		token.Value = val
	} else {
		token.Value = nil
		token.Type = TokenERR
		err = errors.New("NOT_A_FLOAT")
	}
	return token, err
}

func tokenFromString(preToken *PreToken) (*CmdToken, error) {
	var err error
	token := &CmdToken{
		Text:     preToken.Text,
		Type:     TokenString,
		Position: preToken.Position,
	}
	val, err := strconv.Unquote(preToken.Text)
	if err == nil {
		token.Value = val
	} else {
		token.Value = nil
		token.Type = TokenERR
		err = errors.New("COULD_NOT_UNQUOTE")
	}
	return token, err
}

func tokenFromChar(preToken *PreToken) (*CmdToken, error) {
	var err error
	token := &CmdToken{
		Text:     preToken.Text,
		Type:     TokenChar,
		Position: preToken.Position,
	}
	r, _ := utf8.DecodeRuneInString(preToken.Text)
	token.Value = r
	return token, err
}

// watch out, different signature from the rest, because we must combine more than
// one pretoken to form the expression string
func tokenFromExpression(preTokens []*PreToken, startIndex int) (*CmdToken, int, error) {
	var err error
	currIndex := startIndex
	token := &CmdToken{
		Text:     "",
		Type:     TokenExpr,
		Position: preTokens[currIndex].Position,
	}
	if preTokens[currIndex].Text == "'" {
		tmpStr := ""
		for {
			currIndex++
			// are we done with the expression?
			if currIndex >= len(preTokens) || preTokens[currIndex].Type == '\'' {
				break
			}
			tmpStr += preTokens[currIndex].Text
		}
		token.Text = tmpStr
		token.Type = TokenExpr
		token.Value = preTokens[startIndex:currIndex]
	} else {
		token.Value = nil
		token.Type = TokenERR
		err = errors.New("MISSING_SINGLE_QUOTE")
	}
	return token, currIndex, err
}

/*
  tokenize the commandline into CmdParser tokens. to make things easier, we first
  use the internal scanner from GO, then post-process the tokens from the scanner.
*/
func (theParser *CommandParser) TokenizeCommandLine() {
	preTokens := theParser.golangTokenizer(theParser.inputLine)
	postTokens := []*CmdToken{}
	var err error
	var postTok *CmdToken
	index := 0
	for {
		tok := preTokens[index]
		switch tok.Type {
		case scanner.Ident:
			postTok, err = tokenFromIdentifier(tok)
			if err == nil {
				postTokens = append(postTokens, postTok)
			}
		case scanner.Int:
			postTok, err = tokenFromInt(tok)
			if err == nil {
				postTokens = append(postTokens, postTok)
			}
		case scanner.Float:
			postTok, err = tokenFromFloat(tok)
			if err == nil {
				postTokens = append(postTokens, postTok)
			}
		case scanner.String:
			postTok, err = tokenFromString(tok)
			if err == nil {
				postTokens = append(postTokens, postTok)
			}
		case '\'':
			postTok, index, err = tokenFromExpression(preTokens, index)
			if err == nil {
				postTokens = append(postTokens, postTok)
			}
		default:
			postTok, err = tokenFromChar(tok)
			if err == nil {
				postTokens = append(postTokens, postTok)
			}
		}
		if err != nil {
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

// convenience function to dump the token list of the parser
func (theParser *CommandParser) dump() {
	for i, v := range theParser.tokenList {
		fmt.Printf("%2d: ", i)
		fmt.Print(v.Type, "> ")
		fmt.Println(v.String())
	}
}

// copy a token struct
func copyToken(tok *CmdToken) *CmdToken {
	result := *tok
	return &result
}

// peek a token without advancing the input token list
func (theParser *CommandParser) peek() *CmdToken {
	var result *CmdToken
	if len(theParser.tokenList) > 0 {
		result = copyToken(theParser.tokenList[0])
	}
	return result
}

// consume a token from the input token list
func (theParser *CommandParser) read() *CmdToken {
	var result *CmdToken
	if len(theParser.tokenList) > 0 {
		result = theParser.tokenList[0]
		theParser.tokenList = append([]*CmdToken{}, theParser.tokenList[1:]...)
	}
	return result
}

// un-read a token back into the token list
func (theParser *CommandParser) unread(tok *CmdToken) {
	if tok != nil {
		theParser.tokenList = append([]*CmdToken{tok}, theParser.tokenList...)
	}
}

func (theParser *CommandParser) splitRule(ruleString string) ([]string, GrammarItemType) {
	var temp []string
	result := []string{}
	var resultType GrammarItemType

	if strings.Index(ruleString, CHOICESTRING) > 0 {
		resultType = Choice
		temp = strings.Split(ruleString, CHOICESTRING)
	} else {
		resultType = Sequence
		temp = strings.Split(ruleString, " ")
	}

	for i := range temp {
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
	ruleItems, ruleType := theParser.splitRule(expression)
	rs := &RuleStruct{
		Name:  name,
		Items: items,
		Type:  ruleType,
	}
	for _, ruleItem := range ruleItems {
		item := &RuleItem{
			Cardinality: theParser.expressionCardinality(ruleItem),
			ExprType:    theParser.expressionType(ruleItem),
			ExprString:  ruleItem,
			ParentRule:  rs,
			Seen:        false,
		}
		if item.Cardinality != CardinalityOne {
			// strip out cardinality char
			item.ExprString = item.ExprString[0 : len(item.ExprString)-1]
		}
		switch item.ExprType {
		case IdentifierExpr, CharExpr:
			// remove quotes
			item.ExprString = item.ExprString[1 : len(item.ExprString)-1]
		case DataTypeExpr:
			// remove exclamation mark
			item.ExprString = item.ExprString[1:]
		}
		// TODO: find a better way to extract runes from a string
		if item.ExprType == CharExpr {
			s := utf8string.NewString(item.ExprString)
			r := s.At(1)
			if r == '\'' {
				item.ExprString = s.String()
			} else {
				panic("CharExpr does not contain rune literal!")
			}
		}
		rs.Items = append(rs.Items, item)
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
	isMatch := false
	ruleItemPtr.Seen = true

	if theParser.options&OptionDebug != 0 {
		fmt.Println("ruleItem:", ruleItemPtr.String())
		if tokptr != nil {
			fmt.Println("tokenPtr:", tokptr.String())
		} else {
			fmt.Println("tokenPtr is NIL!")
		}
	}

	if tokptr == nil {
		return false
	}

	switch ruleItemPtr.ExprType {
	case CharExpr:
		isMatch = tokptr.Type == TokenChar && ruleItemPtr.ExprString == tokptr.Text
	case IdentifierExpr:
		isMatch = tokptr.Type == TokenIdent && ruleItemPtr.ExprString == tokptr.Value
	case ClassExpr:
		isMatch = matchClassExpr(ruleItemPtr.ExprString, tokptr)
	case SymbolExpr:
		if tokptr != nil {
			theParser.unread(tokptr)
		}
		isMatch = theParser.matchRule(theParser.rules[ruleItemPtr.ExprString])
	case DataTypeExpr:
		dtStr := strings.ToLower(ruleItemPtr.ExprString)
		var reqType TokenType
		switch dtStr {
		case "expression":
			reqType = TokenExpr
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
		isMatch = matchDataTypeExpr(reqType, tokptr)
	}

	if isMatch {
		ruleItemPtr.TokenPtr = tokptr
	}

	if !isMatch {
		if !ruleItemPtr.Seen {
			theParser.errorList = append(theParser.errorList, &ParseError{Column: theParser.tokptr.Position.Column, Message: "Rule " + ruleItemPtr.ParentRule.Name + ",  Expected " + ruleItemPtr.String()})
		}
	}

	return isMatch
}

func (theParser *CommandParser) matchItemWithToken(ruleItemPtr *RuleItem) bool {
	result := false
	var matchCount int
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
				if theParser.tokptr != nil {
					theParser.unread(theParser.tokptr)
				}
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
	match := false

	if rule.Type == Sequence {
		// all must match
		for _, item := range rule.Items {
			theParser.tokptr = theParser.read()
			if theParser.options&OptionDebug != 0 {
				fmt.Println("Using Sequence Item:", item.String())
			}
			match = theParser.matchItemWithToken(item)
			if theParser.options&OptionDebug != 0 {
				fmt.Println(">>      Result:", match)
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

/*
  detect if the parser has processed the input stream to the end
*/
func (theParser *CommandParser) AtEnd() bool {
	return len(theParser.tokenList) == 0
}

func (theParser *CommandParser) buildParseResults() {
	for _, rule := range theParser.rules {
		for _, v := range rule.Items {
			key := strings.ToLower(rule.Name + "_" + v.ExprString)
			if v.TokenPtr != nil {
				theParser.ParseResult[key] = *v.TokenPtr
			}
		}
	}
}

/*
  Parse() ist the function you call to start the parsing process.
*/
func (theParser *CommandParser) Parse() bool {
	rule := theParser.rules["START"]
	match := theParser.matchRule(rule)
	if !theParser.AtEnd() {
		// if there still is stuff to parse, it's not a match ...
		match = false
		if theParser.options&OptionDebug != 0 {
			fmt.Println("Not at end => no match")
			fmt.Println("Token list =>", theParser.tokenList)
		}
	}
	theParser.buildParseResults()
	theParser.IsMatch = match
	return match
}

/*
  Convenience function to dump a rule set
*/
func (theParser *CommandParser) DumpRules() {
	for _, rule := range theParser.rules {
		fmt.Println(rule.Name)
		for i, ri := range rule.Items {
			fmt.Println("  ", i, ri.Seen, " >", ri.TokenPtr.String())
		}
	}
}
