//go:build windows

package singleinstance

import (
	"fmt"
	"syscall"
	"unsafe"
)

const mutexName = "Local\\PakeGUI_SingleInstance_v1"

var heldMutex syscall.Handle

// Acquire returns false if another instance already holds the lock.
func Acquire() (ok bool, err error) {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	createMutex := kernel32.NewProc("CreateMutexW")

	name, _ := syscall.UTF16PtrFromString(mutexName)
	// bInitialOwner = FALSE; existence is enough for single-instance.
	h, _, callErr := createMutex.Call(0, 0, uintptr(unsafe.Pointer(name)))
	if h == 0 {
		return false, fmt.Errorf("CreateMutex: %v", callErr)
	}
	heldMutex = syscall.Handle(h)

	const errorAlreadyExists = syscall.Errno(183)
	if errno, ok := callErr.(syscall.Errno); ok && errno == errorAlreadyExists {
		Release()
		return false, nil
	}
	return true, nil
}

func Release() {
	if heldMutex == 0 {
		return
	}
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	closeHandle := kernel32.NewProc("CloseHandle")
	closeHandle.Call(uintptr(heldMutex))
	heldMutex = 0
}

// ActivateExisting tries to bring the existing Pake GUI window to the front.
func ActivateExisting() bool {
	user32 := syscall.NewLazyDLL("user32.dll")
	findWindow := user32.NewProc("FindWindowW")
	setForeground := user32.NewProc("SetForegroundWindow")
	showWindow := user32.NewProc("ShowWindow")
	isIconic := user32.NewProc("IsIconic")

	title, _ := syscall.UTF16PtrFromString("Pake GUI")
	hwnd, _, _ := findWindow.Call(0, uintptr(unsafe.Pointer(title)))
	if hwnd == 0 {
		return false
	}
	iconic, _, _ := isIconic.Call(hwnd)
	if iconic != 0 {
		showWindow.Call(hwnd, 9) // SW_RESTORE
	}
	setForeground.Call(hwnd)
	return true
}
