package cmdparser

import (
	"strconv"
	"text/scanner"
)

// define the various type of a grammar line
const (
	Sequence = iota // all items of this sequence must match
	Choice          // any one match of these items is sufficient
)

// GrammarItemType is the type for the type list of the grammar item
type GrammarItemType int

// the possible expression types for a grammar line
const (
	CharExpr GrammarItemType = iota
	IdentifierExpr
	ClassExpr
	SymbolExpr
	DataTypeExpr
)

// GrammarItemCardinality defines how often a token can occur
type GrammarItemCardinality int

// Cardinality definitions for grammar clauses
const (
	CardinalityZeroOrMore GrammarItemCardinality = iota
	CardinalityZeroOrOne
	CardinalityOneOrMore
	CardinalityOne
)

// OptionDebug activates verbose debug output
// OptionIgnorecase is planned to be used to case-insensitive parsing
const (
	OptionDebug = 1 << iota
	OptionIgnoreCase
)

// COMMENTCHAR starts a comment to the end of the input line
const COMMENTCHAR = '#'

// CHOICESTRING is used to mark a choice clause in the grammar
const CHOICESTRING = "|"

// TokenType for the cmdparser tokens
type TokenType int

// the token types available
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

// PreToken is the struct that is the result from the internal Go scanner
type PreToken struct {
	Type     rune
	Text     string
	Position scanner.Position
}

// CmdToken is the struct for a CmdParser token
type CmdToken struct {
	Type     TokenType
	Text     string
	Value    interface{}
	Position scanner.Position
}

// ParseError ist the structure plannes for more verbose parser messages
type ParseError struct {
	Column  int
	Message string
}

// String to implement Stringer interface for the CmdToken
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

// RuleItem ist the struct that holds a single grammar rule item
type RuleItem struct {
	ParentRule  *RuleStruct
	Cardinality GrammarItemCardinality
	ExprType    GrammarItemType
	ExprString  string
	TokenPtr    *CmdToken
	Seen        bool
}

// String to implement Stringer interface for the RuleItem
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

// RuleStruct holds the information for a complete grammar rule
type RuleStruct struct {
	Name  string
	Type  GrammarItemType
	Items []*RuleItem
	seen  bool
}

// CommandParser is the main container for run-time information of the parser
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
