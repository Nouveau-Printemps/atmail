package filter

import (
	"bufio"
	"errors"
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
	string_del
	number
	operator_low
	operator_high
	super_operator
	block_beg
	block_end
	filter
	separator
	generic
)

type Lexer struct {
	content *bufio.Reader
	current *Lexem
}

var validOps = [6]string{"=", "!=", ">=", "<=", ">", "<"}

func NewLexer(r io.Reader) *Lexer {
	return &Lexer{content: bufio.NewReader(r)}
}

type Lexem struct {
	Kind  Kind
	Value string
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
		ckind := kindOf(kind, current, next)
		if kind != nil && *kind != ckind {
			break
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
		}
	case operator_high:
		if !slices.Contains(validOps[:], sb.String()) {
			*kind = generic
		}
	}
	return &Lexem{Kind: *kind, Value: sb.String()}
}

func (l *Lexer) Peek() *Lexem {
	if l.current == nil {
		l.current = l.Next()
	}
	return l.current
}

func (l *Lexer) PeekOrErr() (Lexem, error) {
	next := l.Peek()
	if next == nil {
		return Lexem{}, ErrInvalidExpression
	}
	return *next, nil
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

func kindOf(before *Kind, current rune, next *rune) Kind {
	switch current {
	case '=', '!', '>', '<':
		return operator_low
	case ':':
		return identifier_execute
	case '(':
		return identifier_param_beg
	case ')':
		return identifier_param_end
	case ' ', '\n', '\r', '\t':
		return separator
	case '"':
		return string_del
	default:
	}
	if current >= '0' && current <= '9' && nilOr(before, number) {
		return number
	}
	if current == '.' && next != nil {
		if *next >= '0' && *next <= '9' {
			return number
		}
		return identifier_compose
	}
	if (nilOr(before, identifier) && (current >= 'A' && current <= 'z')) ||
		(before != nil && (current == '_' || current == '-' || (current >= '0' && current <= '9'))) {
		return identifier
	}
	return generic
}

func (l *Lexer) NextOrErr() (Lexem, error) {
	lm := l.Next()
	if lm == nil {
		return Lexem{}, ErrInvalidExpression
	}
	return *lm, nil
}
