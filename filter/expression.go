package filter

import (
	"errors"
	"fmt"
)

type Expression interface {
	Eval(*EvaluationContext) (*EvaluationVariable, error)
}

type Literal[T comparable] struct {
	Value T
}

func (l Literal[T]) Eval(ctx *EvaluationContext) (*EvaluationVariable, error) {
	var val any = l.Value
	switch cv := val.(type) {
	case bool:
		return newBool(cv), nil
	case float64:
		return newNumber(cv), nil
	case string:
		return newString(cv), nil
	default:
		panic("internal error: unsupported type")
	}
}

type OperatorExpression struct {
	Operator string
	A, B     Expression
}

var (
	ErrNotNumber = errors.New("not a number")
	ErrNotBool   = errors.New("not a bool")
)

func (op *OperatorExpression) Eval(ctx *EvaluationContext) (*EvaluationVariable, error) {
	a, err := op.A.Eval(ctx)
	if err != nil {
		return nil, err
	}
	b, err := op.B.Eval(ctx)
	if err != nil {
		return nil, err
	}
	if a.Type != b.Type {
		return nil, errors.New("incompatible type")
	}
	switch op.Operator {
	case "=":
		return newBool(a.Value == b.Value), nil
	case "!=":
		return newBool(a.Value != b.Value), nil
	case ">=":
		if a.Type != TypeNumber {
			return nil, ErrNotNumber
		}
		return newBool(a.Value.(float64) >= b.Value.(float64)), nil
	case "<=":
		if a.Type != TypeNumber {
			return nil, ErrNotNumber
		}
		return newBool(a.Value.(float64) <= b.Value.(float64)), nil
	case ">":
		if a.Type != TypeNumber {
			return nil, ErrNotNumber
		}
		return newBool(a.Value.(float64) > b.Value.(float64)), nil
	case "<":
		if a.Type != TypeNumber {
			return nil, ErrNotNumber
		}
		return newBool(a.Value.(float64) < b.Value.(float64)), nil
	case "or":
		if a.Type != TypeBool {
			return nil, ErrNotBool
		}
		return newBool(a.Value.(bool) || b.Value.(bool)), nil
	default:
		panic("internal error: unsupported operator")
	}
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
			right, err := nextOp(lx)
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

func (v *Variable) Eval(ctx *EvaluationContext) (*EvaluationVariable, error) {
	val, ok := ctx.Variables[v.Name]
	if !ok {
		return nil, fmt.Errorf("variable %s not found", v.Name)
	}
	return val, nil
}

type Field struct {
	Name     string
	Variable Expression
}

func (f *Field) Eval(ctx *EvaluationContext) (*EvaluationVariable, error) {
	val, err := f.Variable.Eval(ctx)
	if err != nil {
		return nil, err
	}
	fl, ok := val.Fields[f.Name]
	if !ok {
		return nil, fmt.Errorf("field %s not found", f.Name)
	}
	return fl, nil
}

type Method struct {
	Name     string
	Variable Expression
	Params   []Expression
}

func (m *Method) Eval(ctx *EvaluationContext) (*EvaluationVariable, error) {
	val, err := m.Variable.Eval(ctx)
	if err != nil {
		return nil, err
	}
	cv, ok := val.Methods[m.Name]
	if !ok {
		return nil, fmt.Errorf("method %s not found", m.Name)
	}
	acc := make([]*EvaluationVariable, 0, len(m.Params))
	for _, p := range m.Params {
		pa, err := p.Eval(ctx)
		if err != nil {
			return nil, err
		}
		acc = append(acc, pa)
	}
	return cv.Eval(ctx, val, acc)
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
			params, err := parseParams(lx)
			if err != nil {
				return nil, err
			}
			before = &Method{Name: name, Variable: before, Params: params}
		default:
			return before, nil
		}
	}
	return before, nil
}

func parseParams(lx *Lexer) ([]Expression, error) {
	var params []Expression
	lx.SkipSep()
	cur, err := lx.PeekOrErr()
	for ; err == nil && cur.Kind != identifier_param_end; cur, err = lx.PeekOrErr() {
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
	return params, nil
}
