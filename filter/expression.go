package filter

type Expression interface {
	Eval() (Expression, error)
}

type Literal[T comparable] struct {
	Value T
}

func (l Literal[T]) Eval() (Expression, error) {
	return l, nil
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

func (v *Variable) Eval() (Expression, error) {
	return nil, nil
}

type Field struct {
	Name     string
	Variable Expression
}

func (f *Field) Eval() (Expression, error) {
	return f.Variable, nil
}

type Method struct {
	Name     string
	Variable Expression
	Params   []Expression
}

func (m *Method) Eval() (Expression, error) {
	return m.Variable, nil
}

func parseEval(lx *Lexer) (Expression, error) {
	var before Expression
	before = &Variable{Name: lx.Next().Value}
	for cur := lx.Next(); cur != nil && !cur.IsHardSep(); cur = lx.Next() {
		switch cur.Kind {
		case identifier_compose:
			cur, err := lx.NextOrErr()
			if err != nil {
				return nil, err
			}
			if cur.Kind != identifier {
				return nil, ErrInvalidExpression
			}
			before = &Field{Name: cur.Value, Variable: before}
		case identifier_execute:
			cur, err := lx.NextOrErr()
			if err != nil {
				return nil, err
			}
			if cur.Kind != identifier {
				return nil, ErrInvalidExpression
			}
			name := cur.Value
			cur, err = lx.NextOrErr()
			if err != nil {
				return nil, err
			}
			if cur.Kind != identifier_param_beg {
				return nil, ErrInvalidExpression
			}
			var params []Expression
			for ; err == nil && cur.Kind != identifier_param_end; cur, err = lx.PeekOrErr() {
				expr, err := parseExpression(lx)
				if err != nil {
					return nil, err
				}
				next := lx.Peek()
				if next == nil || (next.Kind != identifier_param_end && next.Kind != separator) {
					return nil, ErrInvalidExpression
				}
				params = append(params, expr)
			}
			if err != nil {
				return nil, err
			}
			lx.Next()
			before = &Method{Name: name, Variable: before, Params: params}
		default:
			return nil, ErrInvalidExpression
		}
	}
	return before, nil
}
