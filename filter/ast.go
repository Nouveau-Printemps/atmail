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
	ErrInvalidStatement  = errors.New("invalid statement")
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
			stmt, err = parseFilter(lx)
		case identifier, number, string_del:
			stmt, err = parseExprStatement(lx)
		case block_beg:
			lx.Next()
			stmt, err = parseActionStatement(lx)
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
	return stmts, nil
}

type ExpressionStatement struct {
	Expr Expression
}

func (expr ExpressionStatement) Eval() bool {
	res, err := expr.Expr.Eval()
	if err != nil {
		return false
	}
	cv, ok := res.(Literal[bool])
	if !ok {
		return false
	}
	return cv.Value
}

func parseExprStatement(lx *Lexer) (Statement, error) {
	expr, err := parseExpression(lx)
	if err != nil {
		return nil, err
	}
	next := lx.Next()
	if next != nil && !next.IsHardSep() {
		return nil, ErrInvalidExpression
	}
	return ExpressionStatement{Expr: expr}, nil
}

type ActionStatement struct {
	Name   string
	Params []Expression
}

func (a *ActionStatement) Eval() bool {
	return true
}

func parseActionStatement(lx *Lexer) (Statement, error) {
	lx.SkipSep()
	name, err := lx.NextOrErr()
	if err != nil {
		return nil, err
	}
	if name.Kind != identifier {
		return nil, ErrInvalidStatement
	}
	next, err := lx.NextOrErr()
	if err != nil {
		return nil, err
	}
	if next.Kind != identifier_param_beg {
		return nil, ErrInvalidStatement
	}
	params, err := parseParams(lx)
	if err != nil {
		return nil, err
	}
	return &ActionStatement{Name: name.Value, Params: params}, nil
}
