package filter

import (
	"bytes"
	"testing"
)

func verifyLexem(t *testing.T, lexem *Lexem, kind Kind, val string) {
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
}

func TestLexer_Next(t *testing.T) {
	lx := NewLexer(bytes.NewBufferString(`from = "hello@example.org"
to = "me@example.com"

filter "spam" do
    header.x-spam-score >= 5.3

    spam
end

filter "uwu" do
    body:contains("uwu")

    mailbox "owo"
    forward "here@example.com"
    finish
end`))
	verifyLexem(t, lx.Next(), identifier, "from")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), operator, "=")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), identifier, "hello")
	verifyLexem(t, lx.Next(), generic, "@example")
	verifyLexem(t, lx.Next(), identifier_compose, ".")
	verifyLexem(t, lx.Next(), identifier, "org")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), separator, "\n")
	verifyLexem(t, lx.Next(), identifier, "to")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), operator, "=")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), identifier, "me")
	verifyLexem(t, lx.Next(), generic, "@example")
	verifyLexem(t, lx.Next(), identifier_compose, ".")
	verifyLexem(t, lx.Next(), identifier, "com")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), separator, "\n\n")
	verifyLexem(t, lx.Next(), filter, "filter")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), identifier, "spam")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), block_beg, "do")
	verifyLexem(t, lx.Next(), separator, "\n    ")
	verifyLexem(t, lx.Next(), identifier, "header")
	verifyLexem(t, lx.Next(), identifier_compose, ".")
	verifyLexem(t, lx.Next(), identifier, "x-spam-score")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), operator, ">=")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), number, "5.3")
	verifyLexem(t, lx.Next(), separator, "\n\n    ")
	verifyLexem(t, lx.Next(), identifier, "spam")
	verifyLexem(t, lx.Next(), separator, "\n")
	verifyLexem(t, lx.Next(), block_end, "end")
	verifyLexem(t, lx.Next(), separator, "\n\n")
	verifyLexem(t, lx.Next(), filter, "filter")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), identifier, "uwu")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), block_beg, "do")
	verifyLexem(t, lx.Next(), separator, "\n    ")
	verifyLexem(t, lx.Next(), identifier, "body")
	verifyLexem(t, lx.Next(), identifier_execute, ":")
	verifyLexem(t, lx.Next(), identifier, "contains")
	verifyLexem(t, lx.Next(), identifier_param_beg, "(")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), identifier, "uwu")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), identifier_param_end, ")")
	verifyLexem(t, lx.Next(), separator, "\n\n    ")
	verifyLexem(t, lx.Next(), identifier, "mailbox")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), identifier, "owo")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), separator, "\n    ")
	verifyLexem(t, lx.Next(), identifier, "forward")
	verifyLexem(t, lx.Next(), separator, " ")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), identifier, "here")
	verifyLexem(t, lx.Next(), generic, "@example")
	verifyLexem(t, lx.Next(), identifier_compose, ".")
	verifyLexem(t, lx.Next(), identifier, "com")
	verifyLexem(t, lx.Next(), string_del, "\"")
	verifyLexem(t, lx.Next(), separator, "\n    ")
	verifyLexem(t, lx.Next(), identifier, "finish")
	verifyLexem(t, lx.Next(), separator, "\n")
	verifyLexem(t, lx.Next(), block_end, "end")
	if lx.Next() != nil {
		t.Error("lexing not finished")
	}
}
