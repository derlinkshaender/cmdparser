package cmdparser

import (
	"strconv"
	"text/scanner"
)

const (
	Sequence = iota
	Choice
)

type GrammarItemType int

const (
	CharExpr GrammarItemType = iota
	IdentifierExpr
	ClassExpr
	SymbolExpr
	DataTypeExpr
)

type GrammarItemCardinality int

const (
	CardinalityZeroOrMore GrammarItemCardinality = iota
	CardinalityZeroOrOne
	CardinalityOneOrMore
	CardinalityOne
)

const (
	OptionDebug = 1 << iota
	OptionIgnoreCase
)

const COMMENTCHAR = '#'
const CHOICESTRING = "|"

type TokenType int

const (
	TokenEOF TokenType = iota
	TokenIdent
	TokenChar
	TokenString
	TokenInt
	TokenFloat
	TokenBool
	TokenExpr
	TokenERR
)

type PreToken struct {
	Type     rune
	Text     string
	Position scanner.Position
}

type CmdToken struct {
	Type     TokenType
	Text     string
	Value    interface{}
	Position scanner.Position
}

type ParseError struct {
	Column  int
	Message string
}

func (tok CmdToken) String() string {
	s := ""
	switch tok.Type {
	case TokenBool:
		s += "Bool "
	case TokenChar:
		s += "Char "
	case TokenERR:
		s += "ERR "
	case TokenExpr:
		s += "Expression "
	case TokenFloat:
		s += "Float "
	case TokenIdent:
		s += "Ident "
	case TokenInt:
		s += "Int "
	case TokenString:
		s += "String "
	}
	s += " [" + tok.Text + "]"
	s += " at col " + strconv.Itoa(tok.Position.Column)
	return s
}

type RuleItem struct {
	ParentRule  *RuleStruct
	Cardinality GrammarItemCardinality
	ExprType    GrammarItemType
	ExprString  string
	TokenPtr    *CmdToken
	Seen        bool
}

func (item RuleItem) String() string {
	s := "RuleItem "
	switch item.ExprType {
	case CharExpr:
		s += "CharExpr"
	case ClassExpr:
		s += "ClassExpr"
	case SymbolExpr:
		s += "SymbolExpr"
	case DataTypeExpr:
		s += "DataTypeExpr"
	case IdentifierExpr:
		s += "IdentifierExpr"
	}
	s += " Expression: [" + item.ExprString + "]"
	switch item.Cardinality {
	case CardinalityOne:
		s += " ONE"
	case CardinalityOneOrMore:
		s += " ONE OR MORE"
	case CardinalityZeroOrOne:
		s += "ZERO OR ONE"
	case CardinalityZeroOrMore:
		s += "ZERO OR MORE"
	}
	return s
}

type RuleStruct struct {
	Name  string
	Type  GrammarItemType
	Items []*RuleItem
	seen  bool
}

type CommandParser struct {
	IsMatch        bool
	TokenizerError bool
	tokptr         *CmdToken
	options        uint64
	inputLine      string
	tokenList      []*CmdToken
	errorList      []*ParseError
	rules          map[string]*RuleStruct
	grammar        map[string]string
	ParseResult    map[string]CmdToken
}
