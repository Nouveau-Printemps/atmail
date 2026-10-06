package filter

import (
	"bufio"
	"errors"
	"io"
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
	operator
	super_operator
	block_beg
	block_end
	filter
	separator
	generic
)

type Lexer struct {
	content *bufio.Reader
}

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
	if *kind == identifier {
		switch sb.String() {
		case "do":
			*kind = block_beg
		case "end":
			*kind = block_end
		case "filter":
			*kind = filter
		}
	}
	return &Lexem{Kind: *kind, Value: sb.String()}
}

func nilOr[T comparable](val *T, or T) bool {
	return val == nil || *val == or
}

func kindOf(before *Kind, current rune, next *rune) Kind {
	switch current {
	case '=', '!', '>', '<':
		return operator
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
