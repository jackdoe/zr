package util

import (
	"os/user"
	"path"
)

func GetDefaultRoot() string {
	u, err := user.Current()
	if err != nil {
		return ""
	}

	return path.Join(u.HomeDir, ".zr-data")
}
