package filter

import (
	"bytes"
	"reflect"
	"testing"
)

func testStatements(t *testing.T, input string) []Statement {
	expr, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Fatal(err)
	}
	return expr.Statements
}

func testExpressionStatement(t *testing.T, stmt Statement, input Expression) {
	expr, ok := stmt.(ExpressionStatement)
	if !ok {
		t.Fatalf("invalid statement: %T wanted ExpressionStatement", stmt)
	}
	if !reflect.DeepEqual(expr.Expr, input) {
		t.Errorf("invalid value: %#v wanted %#v", expr.Expr, input)
	}
}

func TestAST_ExpressionStatement(t *testing.T) {
	stmts := testStatements(t, `foo = bar`)
	res := testExpression(t, "foo = bar")
	if len(stmts) != 1 {
		t.Fatalf("invalid statement: %#v wanted %#v", stmts, res)
	}
	testExpressionStatement(t, stmts[0], res)

	stmts = testStatements(t, `foo = bar
bar = baz`)
	res1 := testExpression(t, "foo = bar")
	res2 := testExpression(t, "bar = baz")
	if len(stmts) != 2 {
		t.Fatalf("invalid statement: %#v wanted %#v and %#v", stmts, res1, res2)
	}
	testExpressionStatement(t, stmts[0], res1)
	testExpressionStatement(t, stmts[1], res2)
}

func TestAST_ActionStatement(t *testing.T) {
	stmts := testStatements(t, "do hello()")
	if len(stmts) != 1 {
		t.Errorf("invalid statements: %#v wanted only one value", stmts)
	}
	action, ok := stmts[0].(*ActionStatement)
	if !ok {
		t.Fatalf("invalid statement: %T wanted *ActionStatement", stmts[0])
	}
	if action.Name != "hello" {
		t.Error("invalid action name:", action.Name, "wanted hello")
	}
	if len(action.Params) != 0 {
		t.Error("invalid action params:", action.Params, "wanted nothing")
	}
}
