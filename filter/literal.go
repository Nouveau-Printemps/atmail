package filter

import (
	"strconv"
	"strings"
)

func parseString(lx *Lexer) (string, error) {
	lx.Next()
	var sb strings.Builder
	cur, err := lx.NextOrErr()
	for ; err == nil && cur.Kind != string_del; cur, err = lx.NextOrErr() {
		sb.WriteString(cur.Value)
	}
	return sb.String(), err
}

func parseNumber(lx *Lexer) (float64, error) {
	return strconv.ParseFloat(lx.Next().Value, 64)
}
