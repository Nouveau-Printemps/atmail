package filter

import (
	"bytes"
	"testing"

	"github.com/emersion/go-message"
)

func testStd(t *testing.T, input string) bool {
	tree, err := Parse(bytes.NewBufferString(input))
	if err != nil {
		t.Fatal(err)
	}
	ctx := InitEvalulationContext(Rcpt{}, Rcpt{}, message.Header{}, "")
	ctx.Verbose = true
	return tree.Eval(ctx)
}

func testValidStd(t *testing.T, input string) {
	if !testStd(t, input) {
		t.Errorf("invalid filter:\n%v", input)
	}
}

func testInvalidStd(t *testing.T, input string) {
	if testStd(t, input) {
		t.Errorf("valid filter:\n%v", input)
	}
}

func TestStd_String(t *testing.T) {
	testValidStd(t, `let hello = "hello"
hello:len() = 5
hello:contains("llo")
hello:get(0) = "h"
hello:sub(0, 3) = "hel"
hello:sub(hello:len(), 10) = ""`)

	testInvalidStd(t, `let hello = "hello"
hello:get(hello:len()) = ""`)
	testInvalidStd(t, `let hello = "hello"
hello:sub(-1, 1)`)
	testInvalidStd(t, `let hello = "hello"
hello:sub(2, 1)`)
}

func TestStd_Collection(t *testing.T) {
	testValidStd(t, `let val = "part1+part2"
let sp = val:split("+")
sp:get(0) = "part1"
sp:get(1) = "part2"

let up = sp:set(0, "updated")
up:get(0) = "updated"

let sp = sp:set(2, "new")
sp:get(2) = "new"
sp:get(1) = "part2"
sp:get(0) = "part1"
up:get(0) = "updated"`)

	testValidStd(t, `let val = "/usr/local/bin/uwu"
let sp = val:split("/")
sp:get(0):len() = 0
sp:len() = 5

sp:has_key(4)
sp:has_key(sp:len()):not()

sp:has_value("uwu")
sp:has_value("nothing"):not()`)

	testInvalidStd(t, `let val = "a/b"
let sp = val:split("/")
sp:get(2)`)
}

func TestStd_Bool(t *testing.T) {
	testValidStd(t, `let true = 0 = 0
true
true:not():not()`)

	testValidStd(t, `let false = 0 != 0
false:not()`)
}
