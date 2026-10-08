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

type OperatorExpression struct {
	Operator string
	A, B     Expression
}

func (op *OperatorExpression) Eval() (Expression, error) {
	return op, nil
}

func parseExpression(lx *Lexer) (Expression, error) {
	return parseExpressionHigh(lx)
}

type parseOperatorFunc = func(*Lexer) (Expression, error)

func parseOperator(kind Kind, nextOp parseOperatorFunc) parseOperatorFunc {
	return func(lx *Lexer) (Expression, error) {
		left, err := nextOp(lx)
		if err != nil {
			return nil, err
		}
		for next := lx.Peek(); next != nil && !next.IsHardSep(); next = lx.Peek() {
			if next.Kind == separator {
				lx.Next()
			}
			op := lx.Peek()
			if op == nil || op.Kind != kind {
				break
			}
			lx.Next()
			lx.SkipSep()
			_, err = lx.PeekOrErr()
			if err != nil {
				return nil, err
			}
			right, err := parseExpression(lx)
			if err != nil {
				return nil, err
			}
			left = &OperatorExpression{Operator: op.Value, A: left, B: right}
		}
		return left, nil
	}
}

func parseExpressionHigh(lx *Lexer) (Expression, error) {
	return parseOperator(operator_high, parseExpressionLow)(lx)
}

func parseExpressionLow(lx *Lexer) (Expression, error) {
	return parseOperator(operator_low, parseExpressionLiteral)(lx)
}

func parseExpressionLiteral(lx *Lexer) (Expression, error) {
	cur := lx.Peek()
	if cur == nil {
		return nil, ErrExpectedStatement
	}
	switch cur.Kind {
	case identifier_param_beg:
		lx.Next()
		lx.SkipSep()
		expr, err := parseExpression(lx)
		if err != nil {
			return nil, err
		}
		lx.SkipSep()
		next, err := lx.NextOrErr()
		if err != nil || next.Kind != identifier_param_end {
			return nil, ErrInvalidExpression
		}
		return expr, nil
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
	for cur := lx.Peek(); cur != nil && !cur.IsHardSep(); cur = lx.Peek() {
		switch cur.Kind {
		case identifier_compose:
			lx.Next()
			cur, err := lx.NextOrErr()
			if err != nil {
				return nil, err
			}
			if cur.Kind != identifier {
				return nil, ErrInvalidExpression
			}
			before = &Field{Name: cur.Value, Variable: before}
		case identifier_execute:
			lx.Next()
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
			lx.SkipSep()
			for cur, err = lx.PeekOrErr(); err == nil && cur.Kind != identifier_param_end; cur, err = lx.PeekOrErr() {
				expr, err := parseExpression(lx)
				if err != nil {
					return nil, err
				}
				lx.SkipSep()
				next := lx.Peek()
				if next == nil || (next.Kind != identifier_param_end && next.Kind != identifier_param_sep) {
					return nil, ErrInvalidExpression
				}
				if next.Kind == identifier_param_sep {
					lx.Next()
					lx.SkipSep()
				}
				params = append(params, expr)
			}
			if err != nil {
				return nil, err
			}
			lx.Next()
			before = &Method{Name: name, Variable: before, Params: params}
		default:
			return before, nil
		}
	}
	return before, nil
}
