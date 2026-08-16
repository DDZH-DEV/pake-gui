//go:build windows

package main

import (
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"unsafe"
)

func openBrowser(url string) {
	cmd := exec.Command("cmd", "/c", "start", "", url)
	cmd.SysProcAttr = &syscall.SysProcAttr{
		HideWindow:    true,
		CreationFlags: 0x08000000, // CREATE_NO_WINDOW
	}
	_ = cmd.Start()
}

func fatalf(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	messageBox("Pake GUI", msg)
	os.Exit(1)
}

func messageBox(title, text string) {
	messageBoxEx(title, text, 0x10) // MB_ICONERROR
}

func messageBoxInfo(title, text string) {
	messageBoxEx(title, text, 0x40) // MB_ICONINFORMATION
}

func messageBoxYesNo(title, text string) bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(text)
	r, _, _ := proc.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), 0x24) // MB_YESNO|MB_ICONQUESTION
	return r == 6 // IDYES
}

func messageBoxEx(title, text string, flags uintptr) {
	user32 := syscall.NewLazyDLL("user32.dll")
	proc := user32.NewProc("MessageBoxW")
	t, _ := syscall.UTF16PtrFromString(title)
	m, _ := syscall.UTF16PtrFromString(text)
	proc.Call(0, uintptr(unsafe.Pointer(m)), uintptr(unsafe.Pointer(t)), flags)
}
