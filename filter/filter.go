package filter

import (
	"errors"
)

type Filter struct {
	Name    string
	Content []Statement
}

var (
	ErrBlockNotEnded = errors.New("block not ended")
)

func parseFilter(lx *Lexer) (*Filter, error) {
	lx.SkipSep()
	lm, err := lx.PeekOrErr()
	if err != nil {
		return nil, err
	}
	var name string
	switch lm.Kind {
	case block_beg:
	case string_del:
		name, err = parseString(lx)
		if err != nil {
			return nil, ParsingError{lm.Line, err}
		}
		lx.SkipSep()
	default:
		err = ParsingError{lm.Line, ErrInvalidStatement}
	}
	if err != nil {
		return nil, err
	}
	lm, err = lx.NextOrErr()
	if err != nil {
		return nil, ParsingError{lm.Line, err}
	}
	if lm.Kind != block_beg {
		return nil, ParsingError{lm.Line, ErrInvalidStatement}
	}
	lx.SkipSep()
	stmts, err := ParseStatements(lx)
	if err == nil {
		return nil, ErrBlockNotEnded
	}
	if !errors.Is(err, ErrBlockEnded) {
		return nil, err
	}
	lx.SkipSep()
	return &Filter{Name: name, Content: stmts}, nil
}

func (f *Filter) Eval(parent *EvaluationContext) bool {
	ctx := parent.Clone()
	ctx.LoggerKey = parent.LoggerKey + " > " + f.Name
	before := true
	for i := 0; before && i < len(f.Content); i++ {
		before = f.Content[i].Eval(ctx)
	}
	if before {
		parent.Result = ctx.Result
	}
	return true
}
