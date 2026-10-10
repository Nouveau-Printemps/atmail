package filter

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"slices"
	"strings"
)

type Kind uint8

const (
	identifier Kind = iota
	identifier_compose
	identifier_execute
	identifier_param_beg
	identifier_param_end
	identifier_param_sep
	string_del
	number
	operator_low
	operator_high
	super_operator
	block_beg
	block_end
	filter
	separator
	let
	generic
)

type Lexer struct {
	content *bufio.Reader
	current *Lexem
	line    uint
	char    uint
}

var validOps = [6]string{"=", "!=", ">=", "<=", ">", "<"}

func NewLexer(r io.Reader) *Lexer {
	return &Lexer{content: bufio.NewReader(r), line: 1, char: 0}
}

type Lexem struct {
	Kind  Kind
	Value string
	Line  uint
}

func (lm *Lexem) IsHardSep() bool {
	return lm.Kind == separator && strings.ContainsRune(lm.Value, '\n')
}

func (l *Lexer) Next() *Lexem {
	if l.current != nil {
		lm := l.current
		l.current = nil
		return lm
	}
	begLine := l.line
	var sb strings.Builder
	var kind *Kind
	for {
		var current rune
		var next *rune
		n, err := l.content.Peek(2)
		if err != nil {
			if errors.Is(err, io.EOF) {
				// nothing was read
				if len(n) == 0 {
					break
				}
				// only current was read
			} else {
				panic(err)
			}
		} else {
			// everything was read
			next = new(rune(n[1]))
		}
		current = rune(n[0])
		ckind, unique := l.kindOf(kind, current, next)
		if kind != nil && *kind != ckind {
			break
		}
		if unique && sb.Len() > 0 {
			break
		}
		if current == '\n' {
			l.line++
			l.char = 0
		} else {
			l.char++
		}
		_, _ = l.content.ReadByte()
		sb.WriteRune(current)
		kind = &ckind
		if next == nil {
			break
		}
	}
	if kind == nil {
		return nil
	}
	switch *kind {
	case identifier:
		switch sb.String() {
		case "do":
			*kind = block_beg
		case "end":
			*kind = block_end
		case "filter":
			*kind = filter
		case "or":
			*kind = operator_high
		case "let":
			*kind = let
		}
	case operator_low:
		if !slices.Contains(validOps[:], sb.String()) {
			*kind = generic
		}
	}
	return &Lexem{Kind: *kind, Value: sb.String(), Line: begLine}
}

func (l *Lexer) Peek() *Lexem {
	if l.current == nil {
		l.current = l.Next()
	}
	return l.current
}

func (l *Lexer) SkipSep() bool {
	lm := l.Peek()
	if lm == nil || lm.Kind != separator {
		return false
	}
	l.Next()
	return true
}

func nilOr[T comparable](val *T, or T) bool {
	return val == nil || *val == or
}

func (l *Lexer) kindOf(before *Kind, current rune, next *rune) (Kind, bool) {
	switch current {
	case '=', '!', '>', '<':
		return operator_low, false
	case ' ', '\n', '\r', '\t':
		return separator, false
	case ':':
		return identifier_execute, true
	case '(':
		return identifier_param_beg, true
	case ')':
		return identifier_param_end, true
	case ',':
		return identifier_param_sep, true
	case '"':
		return string_del, true
	case '-':
		if before == nil && next != nil && isDigit(*next) {
			return number, false
		}
	default:
	}
	if isDigit(current) && nilOr(before, number) {
		return number, false
	}
	if current == '.' && next != nil {
		if isDigit(*next) {
			return number, false
		}
		return identifier_compose, false
	}
	if (nilOr(before, identifier) && (current >= 'A' && current <= 'z')) ||
		(before != nil && (current == '_' || current == '-' || isDigit(current))) {
		return identifier, false
	}
	return generic, false
}

func isDigit(r rune) bool {
	return r >= '0' && r <= '9'
}

type ExpectingTokenError struct {
	Char uint
}

func (e ExpectingTokenError) Error() string {
	return fmt.Sprintf("expecting token at character %d", e.Char)
}

func (l *Lexer) NextOrErr() (Lexem, error) {
	lm := l.Next()
	if lm == nil {
		return Lexem{}, ExpectingTokenError{l.char}
	}
	return *lm, nil
}

func (l *Lexer) PeekOrErr() (Lexem, error) {
	next := l.Peek()
	if next == nil {
		return Lexem{}, ExpectingTokenError{l.char}
	}
	return *next, nil
}
