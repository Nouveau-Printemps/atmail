package filter

import (
	"errors"
	"io"
)

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

func Parse(r io.Reader) (*Tree, error) {
	lx := NewLexer(r)
	stmts, err := ParseStatements(lx)
	return &Tree{Statements: stmts}, err
}

func ParseStatements(lx *Lexer) ([]Statement, error) {
	var stmts []Statement
	for lm := lx.Peek(); lm != nil; lm = lx.Peek() {
		var stmt Statement
		var err error
		switch lm.Kind {
		case filter:
			lx.Next()
			stmt, err = ParseFilter(lx)
		case identifier, number, string_del:
		case block_end:
			lx.Next()
			return stmts, ErrBlockEnded
		default:
			return nil, ErrExpectedStatement
		}
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
	}
	return nil, nil
}
