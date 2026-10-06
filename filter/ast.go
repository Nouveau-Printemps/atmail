package filter

import "errors"

type Statement interface {
	// Eval returns true if the evaluation can continue
	Eval() bool
}

type Tree struct {
	Statements []Statement
}

var (
	ErrExpectedStatement = errors.New("expected statement")
	ErrInvalidExpression = errors.New("invalid expression")
	ErrBlockEnded        = errors.New("block ended")
)

func Parse(content string) (*Tree, error) {
	lx := Lexer{content: []rune(content)}
	stmts, err := ParseStatements(&lx)
	return &Tree{Statements: stmts}, err
}

func ParseStatements(lx *Lexer) ([]Statement, error) {
	lm := lx.Next()
	var stmts []Statement
	for lm != nil {
		var stmt Statement
		var err error
		switch lm.Kind {
		case filter:
			stmt, err = ParseFilter(lx)
		case identifier:
		case number:
		case string_del:
		case block_end:
			return stmts, ErrBlockEnded
		default:
			return nil, ErrExpectedStatement
		}
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
		lm = lx.Next()
	}
	return nil, nil
}
