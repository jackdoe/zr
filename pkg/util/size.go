package util

import (
	"os"
	"strconv"
	"syscall"
	"unsafe"
)

type winsize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func getConsoleWidth() uint {
	ws := &winsize{}
	retCode, _, _ := syscall.Syscall(syscall.SYS_IOCTL,
		uintptr(syscall.Stdin),
		uintptr(syscall.TIOCGWINSZ),
		uintptr(unsafe.Pointer(ws)))

	if int(retCode) == -1 {
		return 80
	}
	return uint(ws.Col)
}

func GetWidth() uint {
	w := getConsoleWidth()
	manwidth := os.Getenv("MANWIDTH")
	if manwidth != "" {
		v, err := strconv.ParseInt(manwidth, 10, 64)
		if err == nil {
			return uint(v)
		}
	}
	return w
}
