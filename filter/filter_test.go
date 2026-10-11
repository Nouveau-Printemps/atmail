package filter

import (
	"bytes"
	"testing"

	"github.com/emersion/go-message"
)

func testFilter(t *testing.T, input string) *Tree {
	tree, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Fatal(err)
	}
	return tree
}

func testContext() *EvaluationContext {
	ctx := InitEvalulationContext("filter", Rcpt{}, Rcpt{}, message.Header{}, "")
	ctx.Verbose = true
	return ctx
}

func TestFilter_Simple(t *testing.T) {
	tree := testFilter(t, `filter do
	"a" = "a"
end`)
	if len(tree.Statements) != 1 {
		t.Fatalf("invalid statements: %#v wanted one filter", tree)
	}
	filter, ok := tree.Statements[0].(*Filter)
	if !ok {
		t.Fatalf("invalid statement: %T wanted *Filter", tree.Statements[0])
	}
	if !filter.Eval(testContext()) {
		t.Error("invalid filter")
	}
}
func TestFilter_Composed(t *testing.T) {
	tree := testFilter(t, `filter "1.1" do
	false
end

filter "1.2" do
	true
	do accept()
end`)
	res, ok := tree.Eval(testContext())
	if !ok {
		t.Fatal("invalid filter")
	}
	if res.Action != ActionAccept {
		t.Error("invalid result:", res.Action, "wanted", ActionAccept)
	}

	tree = testFilter(t, `filter "2.1" do
	do accept()
end

filter "2.2" do
	do reject()
end`)
	res, ok = tree.Eval(testContext())
	if !ok {
		t.Fatal("invalid filter")
	}
	if res.Action != ActionAccept {
		t.Error("invalid result:", res.Action, "wanted", ActionAccept)
	}

	tree = testFilter(t, `filter "3.1" do
	false
end

filter "3.2" do
	false
end

do quarantine()`)
	res, ok = tree.Eval(testContext())
	if !ok {
		t.Fatal("invalid filter")
	}
	if res.Action != ActionQuarantine {
		t.Error("invalid result:", res.Action, "wanted", ActionQuarantine)
	}
}
