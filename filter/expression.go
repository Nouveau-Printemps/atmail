package filter

type Expression interface {
	Eval() Expression
}

type Literal[T comparable] struct {
	Value T
}

func (l Literal[T]) Eval() Expression {
	return l
}

func parseExpression(lx *Lexer) (Expression, error) {
	cur := lx.Peek()
	if cur == nil {
		return nil, ErrExpectedStatement
	}
	switch cur.Kind {
	case identifier:
		return parseEval(lx)
	case number:
		f, err := parseNumber(lx)
		return Literal[float64]{Value: f}, err
	case string_del:
		s, err := parseString(lx)
		return Literal[string]{Value: s}, err
	default:
		return nil, ErrInvalidExpression
	}
}

type Variable struct {
	Name string
}

type Field struct {
	Name     string
	Variable Expression
}

type Method struct {
	Name     string
	Variable Expression
	Params   []Expression
}

func parseEval(lx *Lexer) (Expression, error) {
	for cur := lx.Next(); cur != nil && !cur.IsHardSep(); cur = lx.Next() {
	}
}
