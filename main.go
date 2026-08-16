package main

import (
	"flag"
	"fmt"
	"net"
	"os"
	"path/filepath"

	"pake-gui/internal/appenv"
	"pake-gui/internal/applog"
	"pake-gui/internal/server"
	"pake-gui/internal/singleinstance"
)

func main() {
	port := flag.Int("port", 0, "HTTP port (0 = auto)")
	browser := flag.Bool("browser", false, "Open in system browser instead of desktop window")
	flag.Parse()

	appenv.EnrichPath()

	root, err := appRoot()
	if err != nil {
		fatalf("无法定位程序目录: %v", err)
	}

	buildsDir := filepath.Join(root, "builds")
	dataDir := filepath.Join(root, "data")
	_ = os.MkdirAll(buildsDir, 0o755)
	_ = os.MkdirAll(dataDir, 0o755)

	if _, err := applog.Init(dataDir); err != nil {
		fatalf("无法初始化日志: %v", err)
	}
	defer applog.Close()

	ok, err := singleinstance.Acquire()
	if err != nil {
		applog.Error("single instance: %v", err)
	}
	if !ok {
		applog.Info("another instance is running")
		if info, e := singleinstance.ReadRuntime(dataDir); e == nil && info.URL != "" {
			if !singleinstance.ActivateExisting() {
				openBrowser(info.URL)
			}
			messageBoxInfo("Pake GUI", "Pake GUI 已在运行。\n已尝试切换到现有窗口。")
		} else {
			messageBoxInfo("Pake GUI", "Pake GUI 已在运行。")
			_ = singleinstance.ActivateExisting()
		}
		return
	}
	defer singleinstance.Release()
	defer singleinstance.ClearRuntime(dataDir)

	addr := fmt.Sprintf("127.0.0.1:%d", *port)
	ln, err := net.Listen("tcp", addr)
	if err != nil {
		fatalf("无法监听端口: %v", err)
	}

	token := server.NewToken()
	url := "http://" + ln.Addr().String() + "/?token=" + token
	_ = singleinstance.WriteRuntime(dataDir, url)

	srv := server.New(server.Config{
		Root:      root,
		BuildsDir: buildsDir,
		DataDir:   dataDir,
		Token:     token,
	})

	applog.Info("listening on %s builds=%s", ln.Addr().String(), buildsDir)

	go func() {
		if err := srv.Serve(ln); err != nil {
			applog.Error("server stopped: %v", err)
		}
	}()

	if *browser {
		openBrowser(url)
		if err := runBrowserMode(url, dataDir); err != nil {
			fatalf("%v", err)
		}
		return
	}

	if err := openDesktop(url, dataDir); err != nil {
		fatalf("%v", err)
	}
}

func appRoot() (string, error) {
	exe, err := os.Executable()
	if err != nil {
		return os.Getwd()
	}
	exe, err = filepath.EvalSymlinks(exe)
	if err != nil {
		return filepath.Dir(os.Args[0]), nil
	}
	return filepath.Dir(exe), nil
}
