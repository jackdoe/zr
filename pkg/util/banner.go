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
	side := strings.Repeat("█", (int(width)-len(s)-2)/2)
	return fmt.Sprintf("%s %s %s", side, s, side)
}
