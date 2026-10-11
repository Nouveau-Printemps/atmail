package filter

import (
	"errors"
	"fmt"
	"io"
)

type Statement interface {
	// Eval returns true if the evaluation can continue
	Eval(*EvaluationContext) bool
}

type Tree struct {
	Statements []Statement
}

func (t *Tree) Eval(ctx *EvaluationContext) (*Result, bool) {
	for _, stmt := range t.Statements {
		if !stmt.Eval(ctx) {
			return nil, false
		}
		if ctx.Result.Action != ActionContinue {
			break
		}
	}
	return &ctx.Result, true
}

var (
	ErrExpectedStatement = errors.New("expected statement")
	ErrInvalidExpression = errors.New("invalid expression")
	ErrInvalidStatement  = errors.New("invalid statement")
	ErrBlockEnded        = errors.New("block ended")
)

type ParsingError struct {
	Line uint
	Err  error
}

func (e ParsingError) Error() string {
	return fmt.Sprintf("parsing file: %s (line %d)", e.Err, e.Line)
}

func (e ParsingError) Unwrap() error {
	return e.Err
}

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
		case identifier, number, string_del:
			stmt, err = parseExprStatement(lx)
		case filter:
			lx.Next()
			stmt, err = parseFilter(lx)
		case block_beg:
			lx.Next()
			stmt, err = parseActionStatement(lx)
		case let:
			lx.Next()
			stmt, err = parseLetStatement(lx)
		case block_end:
			lx.Next()
			return stmts, ParsingError{lx.line, ErrBlockEnded}
		default:
			return nil, ParsingError{lx.line, ErrExpectedStatement}
		}
		if err != nil {
			return nil, err
		}
		stmts = append(stmts, stmt)
		lx.SkipSep()
	}
	return stmts, nil
}

type ExpressionStatement struct {
	Expr Expression
}

func (expr ExpressionStatement) Eval(ctx *EvaluationContext) bool {
	res, err := expr.Expr.Eval(ctx)
	if err != nil {
		if ctx.Verbose {
			ctx.Logger().Info("evaluation of expression failed", "error", err.Error())
		}
		return false
	}
	if res.Type != TypeBool {
		if ctx.Verbose {
			ctx.Logger().Warn("expression doesn't return a bool")
		}
		return false
	}
	val := res.Value.(bool)
	if ctx.Verbose && !val {
		ctx.Logger().Info("expression returned false")
	}
	return val
}

func parseExprStatement(lx *Lexer) (stmt Statement, err error) {
	defer func() {
		if err != nil {
			err = ParsingError{lx.line, err}
		}
	}()
	expr, err := parseExpression(lx)
	if err != nil {
		return
	}
	next := lx.Next()
	if next != nil && !next.IsHardSep() {
		err = ErrInvalidStatement
		return
	}
	return ExpressionStatement{Expr: expr}, nil
}

type ActionStatement struct {
	Name   string
	Params []Expression
}

func (a *ActionStatement) Eval(ctx *EvaluationContext) bool {
	action, ok := ctx.Actions[a.Name]
	if !ok {
		if ctx.Verbose {
			ctx.Logger().Warn("action not found", "name", a.Name)
		}
		return false
	}
	acc := make([]*EvaluationVariable, 0, len(a.Params))
	for _, p := range a.Params {
		pa, err := p.Eval(ctx)
		if err != nil {
			if ctx.Verbose {
				ctx.Logger().Info("evaluation of expression failed", "error", err.Error())
			}
			return false
		}
		acc = append(acc, pa)
	}
	err := action(ctx, acc)
	if err != nil {
		if ctx.Verbose {
			ctx.Logger().Info("evaluation of action failed", "error", err.Error())
		}
		return false
	}
	return true
}

func parseActionStatement(lx *Lexer) (stmt Statement, err error) {
	defer func() {
		if err != nil {
			err = ParsingError{lx.line, err}
		}
	}()
	lx.SkipSep()
	name, err := lx.NextOrErr()
	if err != nil {
		return
	}
	if name.Kind != identifier {
		err = ErrInvalidStatement
		return
	}
	next, err := lx.NextOrErr()
	if err != nil {
		return
	}
	if next.Kind != identifier_param_beg {
		err = ErrInvalidStatement
		return
	}
	params, err := parseParams(lx)
	if err != nil {
		return
	}
	return &ActionStatement{Name: name.Value, Params: params}, nil
}

type LetStatement struct {
	Name  string
	Value Expression
}

func (l *LetStatement) Eval(ctx *EvaluationContext) bool {
	cv, err := l.Value.Eval(ctx)
	if err != nil {
		if ctx.Verbose {
			ctx.Logger().Info("evaluation of expression failed", "error", err.Error())
		}
		return false
	}
	ctx.Variables[l.Name] = cv
	return true
}

func parseLetStatement(lx *Lexer) (stmt Statement, err error) {
	defer func() {
		if err != nil {
			err = ParsingError{lx.line, err}
		}
	}()
	lx.SkipSep()
	name, err := lx.NextOrErr()
	if err != nil {
		return nil, err
	}
	if name.Kind != identifier {
		err = ErrInvalidStatement
		return
	}
	lx.SkipSep()
	next, err := lx.NextOrErr()
	if err != nil {
		return
	}
	if next.Kind != operator_low || next.Value != "=" {
		err = ErrInvalidStatement
		return
	}
	lx.SkipSep()
	expr, err := parseExpression(lx)
	if err != nil {
		return
	}
	return &LetStatement{Name: name.Value, Value: expr}, nil
}
