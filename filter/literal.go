package filter

import (
	"strconv"
	"strings"
)

func parseString(lx *Lexer) (string, error) {
	lx.Next()
	var sb strings.Builder
	for cur, err := lx.NextOrErr(); cur.Kind != string_del; cur, err = lx.NextOrErr() {
		if err != nil {
			return "", err
		}
		sb.WriteString(cur.Value)
	}
	return sb.String(), nil
}

func parseNumber(lx *Lexer) (float64, error) {
	return strconv.ParseFloat(lx.Next().Value, 64)
}
