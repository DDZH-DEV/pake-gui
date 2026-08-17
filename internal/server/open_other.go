//go:build !windows

package server

import (
	"os/exec"
	"runtime"
	"strings"
)

func openPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return nil
	}
	if looksLikeURL(path) {
		return openURL(path)
	}
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", path).Start()
	default:
		return exec.Command("xdg-open", path).Start()
	}
}

func looksLikeURL(s string) bool {
	ls := strings.ToLower(s)
	return strings.HasPrefix(ls, "http://") || strings.HasPrefix(ls, "https://")
}

func openURL(rawURL string) error {
	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Start()
	default:
		return exec.Command("xdg-open", rawURL).Start()
	}
}
