package display

import (
	"path"
	"regexp"
	"strings"

	"nouveauprintemps.org/atmail/storage"
)

const (
	percentPattern  = "[^" + string(storage.MailboxSeparator) + "]*"
	wildcardPattern = ".*"
)

func parsePattern(base, pattern string) (*regexp.Regexp, error) {
	if !strings.HasPrefix(pattern, string(storage.MailboxSeparator)) {
		pattern = path.Join(base, pattern)
	}
	var b strings.Builder
	b.Grow(len(pattern) + 1)
	b.WriteRune('^')
	for i, r := range pattern {
		switch r {
		case '%':
			if i != len(pattern)-1 {
				b.WriteString(percentPattern)
				continue
			}
			fallthrough
		case '*':
			b.WriteString(wildcardPattern)
		default:
			b.WriteRune(r)
		}
	}
	return regexp.Compile(b.String())
}
