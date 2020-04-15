package util

import (
	"fmt"
	"strings"
)

func BannerLeft(topDashLen int, prefix string, s []string) string {
	out := "┌"
	for i := 0; i < topDashLen; i++ {
		out += "-"
	}
	out += "\n"

	for _, l := range s {
		out += prefix
		out += " "

		out += l
		out += "\n"
	}
	out += "└--"
	out += "\n"
	return out
}
func Center(s string, around rune) string {
	width := GetWidth()
	r := (int(width) - len(s) - 2) / 2
	symbol := "█"
	if r < 0 {
		r = 10
		// good chance if we didnt get the width its some funky tty
		symbol = "*"
	}
	side := strings.Repeat(symbol, r)
	return fmt.Sprintf("%s %s %s", side, s, side)
}
