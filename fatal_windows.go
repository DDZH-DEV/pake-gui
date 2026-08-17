//go:build windows

package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

func openBrowser(url string) {
	_ = shellOpen(url)
}

func shellOpen(rawURL string) error {
	shell32 := syscall.NewLazyDLL("shell32.dll")
	proc := shell32.NewProc("ShellExecuteW")
	verb, _ := syscall.UTF16PtrFromString("open")
	file, _ := syscall.UTF16PtrFromString(rawURL)
	r, _, err := proc.Call(0, uintptr(unsafe.Pointer(verb)), uintptr(unsafe.Pointer(file)), 0, 0, 1)
	if r <= 32 {
		if err != nil {
			return err
		}
		return fmt.Errorf("ShellExecute failed: %d", r)
	}
	return nil
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
