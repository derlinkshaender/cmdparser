package cmdparser

import "text/scanner"

const (
	Sequence = iota
	Choice   = iota
)

const (
	CharExpr     = iota
	StringExpr   = iota
	ClassExpr    = iota
	SymbolExpr   = iota
	DataTypeExpr = iota
)

const (
	CardinalityZeroOrMore = iota
	CardinalityZeroOrOne  = iota
	CardinalityOneOrMore  = iota
	CardinalityOne        = iota
)

const (
	OptionDebug = 1 << iota
)

const COMMENTCHAR = '#'
const CHOICESTRING = "|"

type CmdToken struct {
	Type     rune
	Text     string
	Position scanner.Position
}

type RuleItem struct {
	Cardinality int
	ExprType    int
	ExprString  string
	ParseValue  string
	ParseType   rune
	Seen        bool
}

type RuleStruct struct {
	Name  string
	Type  int
	Items []RuleItem
	seen  bool
}

type CommandParser struct {
	IsMatch    bool
	options    uint64
	inputLine  string
	tokenList  []*CmdToken
	tokenIndex int
	currIndex  int
	rules      map[string]*RuleStruct
	grammar    map[string]string
}
