//go:build windows

package main

import (
	"fmt"
	"path/filepath"
	"syscall"
	"unsafe"

	"github.com/jchv/go-webview2"
)

func openDesktop(url, dataDir string) error {
	if err := tryWebView(url, dataDir); err == nil {
		return nil
	}

	if !hasWebView2() {
		if messageBoxYesNo(
			"Pake GUI",
			"当前电脑未安装 Microsoft Edge WebView2 Runtime，无法使用内嵌窗口。\n\n"+
				"是否现在自动安装？（需要联网）\n\n"+
				"选「否」将改用系统浏览器打开。",
		) {
			if err := installWebView2Runtime(dataDir); err != nil {
				messageBoxInfo("Pake GUI", "WebView2 安装失败：\n"+err.Error()+"\n\n将改用浏览器打开。")
			} else if err := tryWebView(url, dataDir); err == nil {
				return nil
			} else {
				messageBoxInfo("Pake GUI", "WebView2 已安装，但窗口仍无法创建。\n将改用浏览器打开。\n\n"+err.Error())
			}
		}
	} else {
		messageBoxInfo("Pake GUI", "WebView2 存在，但窗口创建失败。\n将改用系统浏览器打开界面。")
	}

	return runBrowserFallback(url, dataDir)
}

func tryWebView(url, dataDir string) error {
	w := webview2.NewWithOptions(webview2.WebViewOptions{
		Debug:     false,
		AutoFocus: true,
		DataPath:  filepath.Join(dataDir, "webview"),
		WindowOptions: webview2.WindowOptions{
			Title:  "Pake GUI",
			Width:  1320,
			Height: 900,
			Center: true,
		},
	})
	if w == nil {
		return fmt.Errorf("WebView2 窗口创建失败")
	}
	defer w.Destroy()
	w.SetSize(1320, 900, webview2.HintNone)
	w.Navigate(url)
	w.Run()
	return nil
}

func runBrowserFallback(url, dataDir string) error {
	openBrowser(url)
	return showFallbackHost(url, dataDir)
}

func runBrowserMode(url, dataDir string) error {
	return showFallbackHost(url, dataDir)
}

func showFallbackHost(url, dataDir string) error {
	const (
		wsCaption     = 0x00C00000
		wsSysMenu     = 0x00080000
		wsMinimizeBox = 0x00020000
		wsVisible      = 0x10000000
		wsChild       = 0x40000000
		wsTabStop     = 0x00010000
		bsPushButton  = 0x00000000
		wmDestroy     = 0x0002
		wmClose       = 0x0010
		wmCommand     = 0x0111
		wmCreate      = 0x0001
		wmSetFont     = 0x0030
		swShow        = 5
		idOpen        = 1001
		idInstall     = 1002
		idExit        = 1003
		colorBtnFace  = 15
	)

	user32 := syscall.NewLazyDLL("user32.dll")
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	gdi32 := syscall.NewLazyDLL("gdi32.dll")

	registerClass := user32.NewProc("RegisterClassExW")
	createWindowEx := user32.NewProc("CreateWindowExW")
	showWindow := user32.NewProc("ShowWindow")
	updateWindow := user32.NewProc("UpdateWindow")
	getMessage := user32.NewProc("GetMessageW")
	translateMessage := user32.NewProc("TranslateMessage")
	dispatchMessage := user32.NewProc("DispatchMessageW")
	defWindowProc := user32.NewProc("DefWindowProcW")
	postQuitMessage := user32.NewProc("PostQuitMessage")
	destroyWindow := user32.NewProc("DestroyWindow")
	loadCursor := user32.NewProc("LoadCursorW")
	getModuleHandle := kernel32.NewProc("GetModuleHandleW")
	sendMessage := user32.NewProc("SendMessageW")
	createFont := gdi32.NewProc("CreateFontW")
	getSysColorBrush := user32.NewProc("GetSysColorBrush")

	hInstance, _, _ := getModuleHandle.Call(0)
	className, _ := syscall.UTF16PtrFromString("PakeGUIFallbackHost")
	segoe, _ := syscall.UTF16PtrFromString("Segoe UI")
	btnClass, _ := syscall.UTF16PtrFromString("BUTTON")
	staticClass, _ := syscall.UTF16PtrFromString("STATIC")

	wndProc := syscall.NewCallback(func(hwnd syscall.Handle, msg uint32, wParam, lParam uintptr) uintptr {
		switch msg {
		case wmCreate:
			font, _, _ := createFont.Call(16, 0, 0, 0, 400, 0, 0, 0, 1, 0, 0, 0, 0, uintptr(unsafe.Pointer(segoe)))
			createChild := func(class *uint16, text string, style, x, y, w, h, id uintptr) {
				title, _ := syscall.UTF16PtrFromString(text)
				hCtl, _, _ := createWindowEx.Call(
					0,
					uintptr(unsafe.Pointer(class)),
					uintptr(unsafe.Pointer(title)),
					style,
					x, y, w, h,
					uintptr(hwnd),
					id,
					hInstance,
					0,
				)
				if font != 0 && hCtl != 0 {
					sendMessage.Call(hCtl, wmSetFont, font, 1)
				}
			}
			createChild(staticClass,
				"未检测到可用的 WebView2 内嵌窗口。\r\n已在系统浏览器中打开 Pake GUI。\r\n关闭本窗口将退出程序。",
				wsChild|wsVisible, 24, 20, 420, 72, 0)
			createChild(btnClass, "重新打开界面", wsChild|wsVisible|wsTabStop|bsPushButton, 24, 110, 130, 34, idOpen)
			createChild(btnClass, "安装 WebView2", wsChild|wsVisible|wsTabStop|bsPushButton, 168, 110, 140, 34, idInstall)
			createChild(btnClass, "退出", wsChild|wsVisible|wsTabStop|bsPushButton, 324, 110, 100, 34, idExit)
			return 0
		case wmCommand:
			switch wParam & 0xffff {
			case idOpen:
				openBrowser(url)
			case idInstall:
				go func() {
					if err := installWebView2Runtime(dataDir); err != nil {
						messageBoxInfo("Pake GUI", "安装失败：\n"+err.Error())
						return
					}
					messageBoxInfo("Pake GUI", "WebView2 安装完成。\n请重新打开 PakeGUI.exe 以使用内嵌窗口。")
				}()
			case idExit:
				destroyWindow.Call(uintptr(hwnd))
			}
			return 0
		case wmClose:
			destroyWindow.Call(uintptr(hwnd))
			return 0
		case wmDestroy:
			postQuitMessage.Call(0)
			return 0
		}
		ret, _, _ := defWindowProc.Call(uintptr(hwnd), uintptr(msg), wParam, lParam)
		return ret
	})

	type wndClassEx struct {
		size       uint32
		style      uint32
		wndProc    uintptr
		clsExtra   int32
		wndExtra   int32
		instance   uintptr
		icon       uintptr
		cursor     uintptr
		background uintptr
		menuName   uintptr
		className  *uint16
		iconSm     uintptr
	}

	cursor, _, _ := loadCursor.Call(0, 32512)
	bg, _, _ := getSysColorBrush.Call(colorBtnFace)
	wc := wndClassEx{
		size:       uint32(unsafe.Sizeof(wndClassEx{})),
		wndProc:    wndProc,
		instance:   hInstance,
		cursor:     cursor,
		background: bg,
		className:  className,
	}
	atom, _, _ := registerClass.Call(uintptr(unsafe.Pointer(&wc)))
	_ = atom

	title, _ := syscall.UTF16PtrFromString("Pake GUI")
	style := uintptr(wsCaption | wsSysMenu | wsMinimizeBox | wsVisible)
	hwnd, _, err := createWindowEx.Call(
		0,
		uintptr(unsafe.Pointer(className)),
		uintptr(unsafe.Pointer(title)),
		style,
		200, 200, 480, 210,
		0, 0, hInstance, 0,
	)
	if hwnd == 0 {
		messageBoxInfo("Pake GUI", "已在浏览器打开界面。\n\n点击确定退出程序。\n也可使用: PakeGUI.exe -browser")
		_ = err
		return nil
	}

	showWindow.Call(hwnd, swShow)
	updateWindow.Call(hwnd)

	type point struct{ x, y int32 }
	type msg struct {
		hwnd    syscall.Handle
		message uint32
		wParam  uintptr
		lParam  uintptr
		time    uint32
		pt      point
	}
	var m msg
	for {
		ret, _, _ := getMessage.Call(uintptr(unsafe.Pointer(&m)), 0, 0, 0)
		if int32(ret) <= 0 {
			break
		}
		translateMessage.Call(uintptr(unsafe.Pointer(&m)))
		dispatchMessage.Call(uintptr(unsafe.Pointer(&m)))
	}
	return nil
}
