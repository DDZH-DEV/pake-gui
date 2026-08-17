//go:build windows

package server

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"unsafe"
)

func openPath(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("empty path")
	}
	if looksLikeURL(path) {
		return shellOpen(path)
	}
	// DMG/files aren't useful on Windows; open the containing folder instead.
	if st, err := os.Stat(path); err == nil && !st.IsDir() {
		path = filepath.Dir(path)
	}
	return shellOpen(path)
}

func looksLikeURL(s string) bool {
	ls := strings.ToLower(s)
	return strings.HasPrefix(ls, "http://") || strings.HasPrefix(ls, "https://")
}

func openURL(rawURL string) error {
	return shellOpen(rawURL)
}

func shellOpen(target string) error {
	target = strings.TrimSpace(target)
	if target == "" {
		return fmt.Errorf("empty path")
	}
	// ShellExecuteW opens folders/URLs without a hidden console, unlike explorer.exe + CREATE_NO_WINDOW.
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(target)
	r, _, err := proc.Call(
		0,
		uintptr(unsafe.Pointer(verb)),
		uintptr(unsafe.Pointer(file)),
		0,
		0,
		1, // SW_SHOWNORMAL
	)
	// Per MSDN, return value > 32 means success.
	if r <= 32 {
		if err != nil {
			return fmt.Errorf("ShellExecute failed (%d): %v", r, err)
		}
		return fmt.Errorf("ShellExecute failed (%d)", r)
	}
	return nil
}
