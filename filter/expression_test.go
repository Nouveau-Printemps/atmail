package filter

import (
	"bytes"
	"reflect"
	"testing"
)

func testExpression(t *testing.T, input string) Expression {
	expr, err := parseExpression(NewLexer(bytes.NewBufferString(input)))
	if err != nil {
		t.Fatal(err)
	}
	return expr
}

func testLiteral[T comparable](t *testing.T, input string, expected T) {
	expr := testExpression(t, input)
	lit, ok := expr.(Literal[T])
	if !ok {
		t.Fatalf("invalid type: %T wanted %T", expr, Literal[T]{})
	}
	if lit.Value != expected {
		t.Errorf("invalid res: %#v wanted %#v", lit.Value, expected)
	}
}

func testOperator(t *testing.T, input string, op string, a, b Expression) {
	expr := testExpression(t, input)
	opExpr, ok := expr.(*OperatorExpression)
	if !ok {
		t.Fatalf("invalid type: %T wanted OperatorExpression", expr)
	}
	if opExpr.Operator != op {
		t.Errorf("invalid operator: %#v wanted %#v", opExpr.Operator, op)
	}
	if !reflect.DeepEqual(opExpr.A, a) {
		t.Errorf("invalid operand A: %#v wanted %#v", opExpr.A, a)
	}
	if !reflect.DeepEqual(opExpr.B, b) {
		t.Errorf("invalid operand B: %#v wanted %#v", opExpr.B, b)
	}
}

func TestAST_Expression(t *testing.T) {
	testLiteral(t, "0", float64(0))
	testLiteral(t, "0.5", float64(0.5))
	testLiteral(t, `""`, "")
	testLiteral(t, `"hello world"`, "hello world")

	op0 := testExpression(t, "0")
	op1 := testExpression(t, "1")
	for _, op := range validOps {
		testOperator(t, "0 "+op+" 1", op, op0, op1)
		testOperator(t, "0"+op+"1", op, op0, op1)
		testOperator(t, "0"+op+"1", op, op0, op1)
		testOperator(t, "0"+op+"1", op, op0, op1)
	}

	testOperator(t, "0 = 1 or 1 = 0", "or", testExpression(t, "0 = 1"), testExpression(t, "1 = 0"))

	testLiteral(t, "(0)", float64(0))
	testOperator(t, "(2 or 1) = (2 >= 1)", "=", testExpression(t, "2 or 1"), testExpression(t, "2 >= 1"))

	expr := testExpression(t, "var")
	v, ok := expr.(*Variable)
	if !ok {
		t.Fatalf("invalid type: %T wanted *Variable", expr)
	}
	if v.Name != "var" {
		t.Error("invalid name:", v.Name, "wanted 'var'")
	}

	expr = testExpression(t, "var.field")
	f, ok := expr.(*Field)
	if !ok {
		t.Fatalf("invalid type: %T wanted *Field", expr)
	}
	if f.Name != "field" {
		t.Error("invalid name:", v.Name, "wanted 'field'")
	}
	if !reflect.DeepEqual(f.Variable, v) {
		t.Errorf("invalid variable: %#v wanted %#v", f.Variable, v)
	}

	expr = testExpression(t, "var.field:method()")
	m, ok := expr.(*Method)
	if !ok {
		t.Fatalf("invalid type: %T wanted *Field", expr)
	}
	if m.Name != "method" {
		t.Error("invalid name:", v.Name, "wanted 'field'")
	}
	if !reflect.DeepEqual(m.Variable, f) {
		t.Errorf("invalid variable: %#v wanted %#v", m.Variable, f)
	}
	if len(m.Params) != 0 {
		t.Errorf("invalid params: %#v wanted nothing", m.Params)
	}

	expr = testExpression(t, "var.field:method(1)")
	m = expr.(*Method)
	if len(m.Params) != 1 || !reflect.DeepEqual(m.Params[0], op1) {
		t.Fatalf("invalid params: %#v wanted %#v", m.Params, op1)
	}

	expr = testExpression(t, `var.field:method(1, var)`)
	m = expr.(*Method)
	if len(m.Params) != 2 || !reflect.DeepEqual(m.Params[0], op1) || !reflect.DeepEqual(m.Params[1], v) {
		t.Fatalf("invalid params: %#v wanted %#v and %#v", m.Params, op1, v)
	}
}
