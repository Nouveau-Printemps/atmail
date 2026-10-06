package filter

import "strings"

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
	content []rune
	current uint
}

type Lexem struct {
	Kind  Kind
	Value string
}

func (lm *Lexem) IsHardSep() bool {
	return lm.Kind == separator && strings.ContainsRune(lm.Value, '\n')
}

func (l *Lexer) Next() *Lexem {
	ln := uint(len(l.content))
	beg := l.current
	var kind *Kind
	for ; l.current < ln; l.current++ {
		var next *rune
		if l.current+1 < ln {
			next = &l.content[l.current+1]
		}
		ckind := kindOf(kind, l.content[l.current], next)
		if kind != nil && *kind != ckind {
			break
		}
		kind = &ckind
	}
	if kind == nil {
		return nil
	}
	content := string(l.content[beg:l.current])
	if *kind == identifier {
		switch content {
		case "do":
			*kind = block_beg
		case "end":
			*kind = block_end
		case "filter":
			*kind = filter
		}
	}
	return &Lexem{Kind: *kind, Value: content}
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
	if (nilOr(before, identifier) || nilOr(before, generic)) &&
		current >= 'A' && current <= 'z' {
		return identifier
	}
	return generic
}
