package filter

import "errors"

type Filter struct {
	Name    string
	Content []Statement
}

var (
	ErrInvalidFilterStatement = errors.New("invalid filter statement")
	ErrBlockNotEnded          = errors.New("block not ended")
)

func ParseFilter(lx *Lexer) (*Filter, error) {
	lm, err := lx.NextOrErr()
	if err != nil {
		return nil, err
	}
	var name string
	switch lm.Kind {
	case block_beg:
	case string_del:
		//TODO: parse string
	default:
		return nil, ErrInvalidFilterStatement
	}
	stmts, err := ParseStatements(lx)
	if err == nil {
		return nil, ErrBlockNotEnded
	}
	if !errors.Is(err, ErrBlockEnded) {
		return nil, err
	}
	return &Filter{Name: name, Content: stmts}, nil
}

func (f *Filter) Eval() bool {
	return false
}
