package filter

import (
	"bytes"
	"testing"

	"github.com/emersion/go-message"
)

func testResult(t *testing.T, input string) *Result {
	tree, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Fatal(err)
	}
	ctx := InitEvalulationContext(Rcpt{}, Rcpt{}, message.Header{}, "")
	ctx.Verbose = true
	res, ok := tree.Eval(ctx)
	if !ok {
		t.Fatal("invalid result")
	}
	return res
}

func TestResult_Action(t *testing.T) {
	res := testResult(t, `do accept()`)
	if res.Action != ActionAccept {
		t.Error("invalid action:", res.Action, "wanted", ActionAccept)
	}
	res = testResult(t, `do reject()`)
	if res.Action != ActionReject {
		t.Error("invalid action:", res.Action, "wanted", ActionReject)
	}
	res = testResult(t, `do temp_reject()`)
	if res.Action != ActionTempFail {
		t.Error("invalid action:", res.Action, "wanted", ActionTempFail)
	}
	res = testResult(t, `do discard()`)
	if res.Action != ActionDiscard {
		t.Error("invalid action:", res.Action, "wanted", ActionDiscard)
	}
	res = testResult(t, `do quarantine()`)
	if res.Action != ActionQuarantine {
		t.Error("invalid action:", res.Action, "wanted", ActionQuarantine)
	}
}

func TestResult_Forward(t *testing.T) {
	res := testResult(t, `do forward("send@here")`)
	if len(res.AddRecipient) != 1 || res.AddRecipient[0] != "send@here" {
		t.Errorf("invalid forward: %#v wanted send@here", res.AddRecipient)
	}
	if res.Action != ActionContinue {
		t.Error("invalid action:", res.Action, "wanted", ActionContinue)

	}
	res = testResult(t, `do forward("send@here")
do accept()`)
	if len(res.AddRecipient) != 1 || res.AddRecipient[0] != "send@here" {
		t.Errorf("invalid forward: %#v wanted send@here", res.AddRecipient)
	}
	if res.Action != ActionAccept {
		t.Error("invalid action:", res.Action, "wanted", ActionAccept)
	}
}

func TestResult_Mailbox(t *testing.T) {
	res := testResult(t, `do forward("send@here")`)
	if res.Mailbox != nil {
		t.Error("invalid mailbox:", *res.Mailbox, "wanted nil")
	}

	res = testResult(t, `do mailbox("hehe")`)
	if res.Mailbox == nil {
		t.Fatal("expected mailbox")
	}
	if *res.Mailbox != "hehe" {
		t.Errorf("invalid mailbox: %#v wanted hehe", *res.Mailbox)
	}
}

func TestResult_Header(t *testing.T) {
	res := testResult(t, `do forward("send@here")`)
	if len(res.ModifyHeader) != 0 {
		t.Errorf("invalid header: %#v wanted empty", res.ModifyHeader)
	}

	res = testResult(t, `do set_header("foo", "bar")`)
	h, ok := res.ModifyHeader["foo"]
	if !ok {
		t.Fatal("invalid header: foo not set")
	}
	if h != "bar" {
		t.Errorf("invalid header foo: %#v wanted bar", h)
	}

	res = testResult(t, `do set_header("foo", "bar")
do set_header("baz", "aaa")`)
	h, ok = res.ModifyHeader["foo"]
	if !ok {
		t.Fatal("invalid header: foo not set")
	}
	if h != "bar" {
		t.Errorf("invalid header foo: %#v wanted bar", h)
	}
	h, ok = res.ModifyHeader["baz"]
	if !ok {
		t.Fatal("invalid header: baz not set")
	}
	if h != "aaa" {
		t.Errorf("invalid header baz: %#v wanted aaa", h)
	}
	res = testResult(t, `do set_header("foo", "bar")
do set_header("baz", "aaa")
do set_header("foo", "uwu")`)
	h, ok = res.ModifyHeader["foo"]
	if !ok {
		t.Fatal("invalid header: foo not set")
	}
	if h != "uwu" {
		t.Errorf("invalid header foo: %#v wanted uwu", h)
	}
	h, ok = res.ModifyHeader["baz"]
	if !ok {
		t.Fatal("invalid header: baz not set")
	}
	if h != "aaa" {
		t.Errorf("invalid header baz: %#v wanted aaa", h)
	}
}
