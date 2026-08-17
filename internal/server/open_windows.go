//go:build windows

package server

import (
	"fmt"
	"os/exec"
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
		return openURL(path)
	}
	// Explorer for files/folders; hide the helper console.
	cmd := exec.Command("explorer", path)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000,
	}
	return cmd.Start()
}

func looksLikeURL(s string) bool {
	ls := strings.ToLower(s)
	return strings.HasPrefix(ls, "http://") || strings.HasPrefix(ls, "https://")
}

func openURL(rawURL string) error {
	rawURL = strings.TrimSpace(rawURL)
	if rawURL == "" {
		return fmt.Errorf("empty url")
	}
	// ShellExecuteW is the reliable way to open the default browser from a GUI app.
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(rawURL)
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
