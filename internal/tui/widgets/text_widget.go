package widgets

import (
	"github.com/charmbracelet/x/ansi"
	"strings"
	"unicode"
)

func Clean(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsControl(r) {
			return -1
		}
		return r
	}, ansi.Strip(s))
}
