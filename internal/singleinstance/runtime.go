package singleinstance

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type RuntimeInfo struct {
	PID       int       `json:"pid"`
	URL       string     `json:"url"`
	StartedAt time.Time `json:"startedAt"`
}

func runtimePath(dataDir string) string {
	return filepath.Join(dataDir, "runtime.json")
}

func WriteRuntime(dataDir, url string) error {
	_ = os.MkdirAll(dataDir, 0o755)
	info := RuntimeInfo{
		PID:       os.Getpid(),
		URL:       url,
		StartedAt: time.Now(),
	}
	b, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(runtimePath(dataDir), b, 0o644)
}

func ReadRuntime(dataDir string) (*RuntimeInfo, error) {
	b, err := os.ReadFile(runtimePath(dataDir))
	if err != nil {
		return nil, err
	}
	var info RuntimeInfo
	if err := json.Unmarshal(b, &info); err != nil {
		return nil, err
	}
	return &info, nil
}

func ClearRuntime(dataDir string) {
	_ = os.Remove(runtimePath(dataDir))
}
