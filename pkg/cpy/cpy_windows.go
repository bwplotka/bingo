//go:build windows

package cpy

import (
	"strings"
)

func ensureExe(f string) string {
	if strings.HasSuffix(f, ".exe") {
		return f
	}
	return f + ".exe"
}
