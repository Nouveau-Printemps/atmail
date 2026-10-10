package filter

import (
	"bytes"
	"errors"
	"testing"
)

func verifyLexem(t *testing.T, lexem *Lexem, kind Kind, val string, line uint) {
	if lexem == nil {
		t.Fatal("nil lexem")
	}
	t.Log(lexem)
	if lexem.Kind != kind {
		t.Error("invalid kind:", lexem.Kind, "wanted", kind)
	}
	if lexem.Value != val {
		t.Errorf("invalid value: %#v wanted %#v", lexem.Value, val)
	}
	if lexem.Line != line {
		t.Error("invalid line:", lexem.Line, "wanted", line)
	}
}

func TestLexer_Next(t *testing.T) {
	lx := NewLexer(bytes.NewBufferString(`from = "hello@example.org"
to = "me@example.com"

filter "spam" do
    header.x-spam-score >= 5.3

    do spam()
end

filter "uwu" do
    body:contains("uwu")

    do mailbox("owo")
    do forward("here@example.com")
    do finish()
end`))
	verifyLexem(t, lx.Next(), identifier, "from", 1)
	verifyLexem(t, lx.Next(), separator, " ", 1)
	verifyLexem(t, lx.Next(), operator_low, "=", 1)
	verifyLexem(t, lx.Next(), separator, " ", 1)
	verifyLexem(t, lx.Next(), string_del, "\"", 1)
	verifyLexem(t, lx.Next(), identifier, "hello", 1)
	verifyLexem(t, lx.Next(), generic, "@example", 1)
	verifyLexem(t, lx.Next(), identifier_compose, ".", 1)
	verifyLexem(t, lx.Next(), identifier, "org", 1)
	verifyLexem(t, lx.Next(), string_del, "\"", 1)
	verifyLexem(t, lx.Next(), separator, "\n", 1)
	verifyLexem(t, lx.Next(), identifier, "to", 2)
	verifyLexem(t, lx.Next(), separator, " ", 2)
	verifyLexem(t, lx.Next(), operator_low, "=", 2)
	verifyLexem(t, lx.Next(), separator, " ", 2)
	verifyLexem(t, lx.Next(), string_del, "\"", 2)
	verifyLexem(t, lx.Next(), identifier, "me", 2)
	verifyLexem(t, lx.Next(), generic, "@example", 2)
	verifyLexem(t, lx.Next(), identifier_compose, ".", 2)
	verifyLexem(t, lx.Next(), identifier, "com", 2)
	verifyLexem(t, lx.Next(), string_del, "\"", 2)
	verifyLexem(t, lx.Next(), separator, "\n\n", 2)
	verifyLexem(t, lx.Next(), filter, "filter", 4)
	verifyLexem(t, lx.Next(), separator, " ", 4)
	verifyLexem(t, lx.Next(), string_del, "\"", 4)
	verifyLexem(t, lx.Next(), identifier, "spam", 4)
	verifyLexem(t, lx.Next(), string_del, "\"", 4)
	verifyLexem(t, lx.Next(), separator, " ", 4)
	verifyLexem(t, lx.Next(), block_beg, "do", 4)
	verifyLexem(t, lx.Next(), separator, "\n    ", 4)
	verifyLexem(t, lx.Next(), identifier, "header", 5)
	verifyLexem(t, lx.Next(), identifier_compose, ".", 5)
	verifyLexem(t, lx.Next(), identifier, "x-spam-score", 5)
	verifyLexem(t, lx.Next(), separator, " ", 5)
	verifyLexem(t, lx.Next(), operator_low, ">=", 5)
	verifyLexem(t, lx.Next(), separator, " ", 5)
	verifyLexem(t, lx.Next(), number, "5.3", 5)
	verifyLexem(t, lx.Next(), separator, "\n\n    ", 5)
	verifyLexem(t, lx.Next(), block_beg, "do", 7)
	verifyLexem(t, lx.Next(), separator, " ", 7)
	verifyLexem(t, lx.Next(), identifier, "spam", 7)
	verifyLexem(t, lx.Next(), identifier_param_beg, "(", 7)
	verifyLexem(t, lx.Next(), identifier_param_end, ")", 7)
	verifyLexem(t, lx.Next(), separator, "\n", 7)
	verifyLexem(t, lx.Next(), block_end, "end", 8)
	verifyLexem(t, lx.Next(), separator, "\n\n", 8)
	verifyLexem(t, lx.Next(), filter, "filter", 10)
	verifyLexem(t, lx.Next(), separator, " ", 10)
	verifyLexem(t, lx.Next(), string_del, "\"", 10)
	verifyLexem(t, lx.Next(), identifier, "uwu", 10)
	verifyLexem(t, lx.Next(), string_del, "\"", 10)
	verifyLexem(t, lx.Next(), separator, " ", 10)
	verifyLexem(t, lx.Next(), block_beg, "do", 10)
	verifyLexem(t, lx.Next(), separator, "\n    ", 10)
	verifyLexem(t, lx.Next(), identifier, "body", 11)
	verifyLexem(t, lx.Next(), identifier_execute, ":", 11)
	verifyLexem(t, lx.Next(), identifier, "contains", 11)
	verifyLexem(t, lx.Next(), identifier_param_beg, "(", 11)
	verifyLexem(t, lx.Next(), string_del, "\"", 11)
	verifyLexem(t, lx.Next(), identifier, "uwu", 11)
	verifyLexem(t, lx.Next(), string_del, "\"", 11)
	verifyLexem(t, lx.Next(), identifier_param_end, ")", 11)
	verifyLexem(t, lx.Next(), separator, "\n\n    ", 11)
	verifyLexem(t, lx.Next(), block_beg, "do", 13)
	verifyLexem(t, lx.Next(), separator, " ", 13)
	verifyLexem(t, lx.Next(), identifier, "mailbox", 13)
	verifyLexem(t, lx.Next(), identifier_param_beg, "(", 13)
	verifyLexem(t, lx.Next(), string_del, "\"", 13)
	verifyLexem(t, lx.Next(), identifier, "owo", 13)
	verifyLexem(t, lx.Next(), string_del, "\"", 13)
	verifyLexem(t, lx.Next(), identifier_param_end, ")", 13)
	verifyLexem(t, lx.Next(), separator, "\n    ", 13)
	verifyLexem(t, lx.Next(), block_beg, "do", 14)
	verifyLexem(t, lx.Next(), separator, " ", 14)
	verifyLexem(t, lx.Next(), identifier, "forward", 14)
	verifyLexem(t, lx.Next(), identifier_param_beg, "(", 14)
	verifyLexem(t, lx.Next(), string_del, "\"", 14)
	verifyLexem(t, lx.Next(), identifier, "here", 14)
	verifyLexem(t, lx.Next(), generic, "@example", 14)
	verifyLexem(t, lx.Next(), identifier_compose, ".", 14)
	verifyLexem(t, lx.Next(), identifier, "com", 14)
	verifyLexem(t, lx.Next(), string_del, "\"", 14)
	verifyLexem(t, lx.Next(), identifier_param_end, ")", 14)
	verifyLexem(t, lx.Next(), separator, "\n    ", 14)
	verifyLexem(t, lx.Next(), block_beg, "do", 15)
	verifyLexem(t, lx.Next(), separator, " ", 15)
	verifyLexem(t, lx.Next(), identifier, "finish", 15)
	verifyLexem(t, lx.Next(), identifier_param_beg, "(", 15)
	verifyLexem(t, lx.Next(), identifier_param_end, ")", 15)
	verifyLexem(t, lx.Next(), separator, "\n", 15)
	verifyLexem(t, lx.Next(), block_end, "end", 16)
	if lx.Next() != nil {
		t.Error("lexing not finished")
	}
	_, err := lx.NextOrErr()
	if err == nil {
		t.Fatal("expecting error")
	}
	e, ok := errors.AsType[ExpectingTokenError](err)
	if !ok {
		t.Fatalf("invalid error: %T wanted ErrExpectingToken", err)
	}
	if e.Char != 3 {
		t.Error("invalid error char:", e.Char, "wanted 3")
	}
}
