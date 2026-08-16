//go:build !windows

package main

import (
	"fmt"
	"os/exec"
	"runtime"
)

func openDesktop(url, dataDir string) error {
	_ = dataDir
	openBrowser(url)
	fmt.Printf("当前平台暂用浏览器窗口模式: %s\n", url)
	select {}
}

func runBrowserMode(url, dataDir string) error {
	_ = dataDir
	fmt.Printf("Pake GUI 已启动: %s\n按 Ctrl+C 退出\n", url)
	select {}
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	_ = cmd.Start()
}
