//go:build windows

package main

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"syscall"
	"time"
	"unsafe"
)

const webview2BootstrapURL = "https://go.microsoft.com/fwlink/p/?LinkId=2124703"

// Evergreen WebView2 Runtime client GUID used by EdgeUpdate.
var webview2ClientGUIDs = []string{
	`SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-A99D-07A98BA79449}`,
	`SOFTWARE\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-A99D-07A98BA79449}`,
	`SOFTWARE\WOW6432Node\Microsoft\EdgeUpdate\Clients\{F3017226-FE2A-4295-A99D-07A98BA79449}`,
}

func hasWebView2() bool {
	if ver := readWebView2Version(); ver != "" {
		return true
	}
	// Fallback: msedgewebview2.exe near Edge installs
	localApp := os.Getenv("LOCALAPPDATA")
	programFiles := os.Getenv("ProgramFiles")
	programFilesX86 := os.Getenv("ProgramFiles(x86)")
	candidates := []string{
		filepath.Join(programFilesX86, "Microsoft", "EdgeWebView", "Application"),
		filepath.Join(programFiles, "Microsoft", "EdgeWebView", "Application"),
		filepath.Join(localApp, "Microsoft", "EdgeWebView", "Application"),
	}
	for _, dir := range candidates {
		if st, err := os.Stat(dir); err == nil && st.IsDir() {
			return true
		}
	}
	return false
}

func readWebView2Version() string {
	advapi := syscall.NewLazyDLL("advapi32.dll")
	regOpen := advapi.NewProc("RegOpenKeyExW")
	regQuery := advapi.NewProc("RegQueryValueExW")
	regClose := advapi.NewProc("RegCloseKey")

	const hkeyLocalMachine = 0x80000002
	const keyRead = 0x20019

	for _, path := range webview2ClientGUIDs {
		var hKey syscall.Handle
		p, _ := syscall.UTF16PtrFromString(path)
		r, _, _ := regOpen.Call(hkeyLocalMachine, uintptr(unsafe.Pointer(p)), 0, keyRead, uintptr(unsafe.Pointer(&hKey)))
		if r != 0 {
			continue
		}
		name, _ := syscall.UTF16PtrFromString("pv")
		var typ uint32
		buf := make([]uint16, 64)
		bufLen := uint32(len(buf) * 2)
		r, _, _ = regQuery.Call(
			uintptr(hKey),
			uintptr(unsafe.Pointer(name)),
			0,
			uintptr(unsafe.Pointer(&typ)),
			uintptr(unsafe.Pointer(&buf[0])),
			uintptr(unsafe.Pointer(&bufLen)),
		)
		regClose.Call(uintptr(hKey))
		if r != 0 {
			continue
		}
		ver := syscall.UTF16ToString(buf)
		if ver != "" && ver != "0.0.0.0" {
			return ver
		}
	}
	return ""
}

func installWebView2Runtime(dataDir string) error {
	dir := filepath.Join(dataDir, "webview2-setup")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	setup := filepath.Join(dir, "MicrosoftEdgeWebview2Setup.exe")

	messageBoxInfo("Pake GUI", "正在下载 WebView2 Runtime 安装包，请稍候…")

	if err := downloadFile(webview2BootstrapURL, setup); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}

	cmd := exec.Command(setup, "/install")
	cmd.SysProcAttr = &syscall.SysProcAttr{HideWindow: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("启动安装程序失败: %w", err)
	}
	_ = cmd.Wait()

	// Wait until runtime becomes visible in registry (installer may return early).
	deadline := time.Now().Add(3 * time.Minute)
	for time.Now().Before(deadline) {
		if hasWebView2() {
			return nil
		}
		time.Sleep(2 * time.Second)
	}
	if hasWebView2() {
		return nil
	}
	return fmt.Errorf("安装超时，请手动安装: https://developer.microsoft.com/microsoft-edge/webview2/")
}

func downloadFile(url, dest string) error {
	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = io.Copy(f, resp.Body)
	return err
}
