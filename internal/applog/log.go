package applog

import (
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	mu     sync.Mutex
	logger *log.Logger
	file   *os.File
	path   string
)

func Init(dataDir string) (string, error) {
	mu.Lock()
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		mu.Unlock()
		return "", err
	}
	path = filepath.Join(dataDir, "app.log")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		mu.Unlock()
		return "", err
	}
	if file != nil {
		_ = file.Close()
	}
	file = f
	logger = log.New(io.MultiWriter(f), "", 0)
	p := path
	mu.Unlock()

	Info("—— log started %s ——", time.Now().Format(time.RFC3339))
	return p, nil
}

func Path() string {
	mu.Lock()
	defer mu.Unlock()
	return path
}

func Info(format string, args ...any) {
	write("INFO", format, args...)
}

func Error(format string, args ...any) {
	write("ERROR", format, args...)
}

func write(level, format string, args ...any) {
	mu.Lock()
	defer mu.Unlock()
	line := fmt.Sprintf("%s [%s] %s", time.Now().Format("2006-01-02 15:04:05"), level, fmt.Sprintf(format, args...))
	if logger != nil {
		logger.Println(line)
	}
}

func Close() {
	mu.Lock()
	defer mu.Unlock()
	if file != nil {
		_ = file.Close()
		file = nil
	}
}
