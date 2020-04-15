package util

import (
	"os"
	"strconv"

	"golang.org/x/crypto/ssh/terminal"
)

func GetWidth() uint {

	manwidth := os.Getenv("MANWIDTH")
	if manwidth != "" {
		v, err := strconv.ParseInt(manwidth, 10, 64)
		if err == nil {
			return uint(v)
		}
	}
	width, _, _ := terminal.GetSize(0)
	if width < 0 {
		return 80
	}
	return uint(width)
}
