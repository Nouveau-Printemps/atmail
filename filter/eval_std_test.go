package filter

import (
	"bytes"
	"testing"

	"github.com/emersion/go-message"
)

func testStd(t *testing.T, input string) {
	tree, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Fatal(err)
	}
	if !tree.Eval(InitEvalulationContext(Rcpt{}, Rcpt{}, message.Header{}, "")) {
		t.Errorf("invalid filter:\n%v", input)
	}
}

func TestStd_String(t *testing.T) {
	testStd(t, `let hello = "hello"
hello:len() = 5
hello:contains("llo")
hello:get(0) = "h"
hello:sub(0, 3) = "hel"`)
}

func TestStd_Collection(t *testing.T) {
	testStd(t, `let val = "part1+part2"
let sp = val:split("+")
sp:get(0) = "part1"
sp:get(1) = "part2"

let up = sp:set(0, "updated")
up:get(0) = "updated"

let sp = sp:set(2, "new")
sp:get(2) = "new"
sp:get(1) = "part2"
sp:get(0) = "part1"`)

	testStd(t, `let val = "/usr/local/bin/uwu"
let sp = val:split("/")
sp:get(0):len() = 0
sp:len() = 5

sp:has_key(4)
sp:has_key(sp:len()):not()

sp:has_value("uwu")
sp:has_value("nothing"):not()`)
}

func TestStd_Bool(t *testing.T) {
	testStd(t, `let true = 0 = 0
true
true:not():not()`)

	testStd(t, `let false = 0 != 0
false:not()`)
}
