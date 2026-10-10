package filter

import (
	"bytes"
	"testing"

	"github.com/emersion/go-message"
)

func TestAST_Eval(t *testing.T) {
	tree, err := Parse(bytes.NewBufferString(`from = "hello@example.org" or
to = "world@example.org"

from.user != "world"
to.user != "hello"

from:contains("@")
to.domain:contains(".")`))
	if err != nil {
		t.Fatal(err)
	}
	ctx := InitEvalulationContext(Rcpt{
		User:    "hello",
		Domain:  "example.org",
		Address: "hello@example.org",
	}, Rcpt{
		User:    "world",
		Domain:  "example.org",
		Address: "world@example.org",
	}, message.Header{}, "")
	ctx.Verbose = true
	if !tree.Eval(ctx) {
		t.Error("expected valid eval")
	}

	testEvalError(t, `from = "no@example.org" or
to = "world@example.org"

from.user = "here"`)
}

func testEvalError(t *testing.T, input string) {
	tree, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Fatal(err)
	}
	ctx := InitEvalulationContext(Rcpt{
		User:    "hello",
		Domain:  "example.org",
		Address: "hello@example.org",
	}, Rcpt{
		User:    "world",
		Domain:  "example.org",
		Address: "world@example.org",
	}, message.Header{}, "")
	ctx.Verbose = true
	if tree.Eval(ctx) {
		t.Error("expected invalid eval for", input)
	}
}

func TestAST_EvalRuntimeError(t *testing.T) {
	testEvalError(t, `nothing = "hello@example.org"`)
	testEvalError(t, `from.nothing = "hello@example.org"`)
	testEvalError(t, `from:nothing()`)
	testEvalError(t, `from = 5`)
}
