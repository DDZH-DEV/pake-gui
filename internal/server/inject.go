package server

import (
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

func (s *Server) injectDir() string {
	return filepath.Join(s.cfg.DataDir, "inject")
}

func (s *Server) resolveInjectScanDir(requested string) (string, error) {
	base := s.injectDir()
	_ = os.MkdirAll(base, 0o755)
	req := strings.TrimSpace(requested)
	if req == "" {
		return filepath.Abs(base)
	}
	var abs string
	var err error
	if filepath.IsAbs(req) {
		abs, err = filepath.Abs(req)
	} else {
		abs, err = filepath.Abs(filepath.Join(base, req))
	}
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	// Allow scanning under data/inject, or any path under dataDir.
	dataAbs, _ := filepath.Abs(s.cfg.DataDir)
	if isSubPath(base, abs) || isSubPath(dataAbs, abs) {
		return abs, nil
	}
	return "", fmt.Errorf("只能扫描 data/inject 或其子目录（当前限制在程序 data 目录内）")
}

func (s *Server) handleInjectList(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet && r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	dirReq := r.URL.Query().Get("dir")
	if r.Method == http.MethodPost {
		var body struct {
			Dir string `json:"dir"`
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		if strings.TrimSpace(body.Dir) != "" {
			dirReq = body.Dir
		}
	}
	dir, err := s.resolveInjectScanDir(dirReq)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	st, err := os.Stat(dir)
	if err != nil || !st.IsDir() {
		writeJSON(w, map[string]any{"ok": false, "error": "目录不存在：" + dir, "dir": dir})
		return
	}

	var files []map[string]any
	_ = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			// Keep depth shallow: skip nested dirs beyond 2 levels under inject root.
			rel, _ := filepath.Rel(dir, path)
			if rel != "." && strings.Count(rel, string(os.PathSeparator)) >= 2 {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if ext != ".js" && ext != ".css" {
			return nil
		}
		info, e := d.Info()
		size := int64(0)
		if e == nil {
			size = info.Size()
		}
		files = append(files, map[string]any{
			"name": d.Name(),
			"path": path,
			"ext":  ext,
			"size": size,
		})
		return nil
	})
	if files == nil {
		files = []map[string]any{}
	}
	writeJSON(w, map[string]any{
		"ok":    true,
		"dir":   dir,
		"files": files,
		"count": len(files),
	})
}

func (s *Server) handleInjectUpload(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(16 << 20); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "文件过大或无效（合计最大 16MB）"})
		return
	}
	dir := s.injectDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	form := r.MultipartForm
	if form == nil || len(form.File["files"]) == 0 {
		// also accept single "file"
		if f, hdr, err := r.FormFile("file"); err == nil {
			defer f.Close()
			ext := strings.ToLower(filepath.Ext(hdr.Filename))
			if ext != ".js" && ext != ".css" {
				writeJSON(w, map[string]any{"ok": false, "error": "仅支持 .js / .css"})
				return
			}
			dest := filepath.Join(dir, filepath.Base(hdr.Filename))
			data, err := io.ReadAll(io.LimitReader(f, 4<<20))
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": "读取失败"})
				return
			}
			if err := os.WriteFile(dest, data, 0o644); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "paths": []string{dest}, "dir": dir})
			return
		}
		writeJSON(w, map[string]any{"ok": false, "error": "请选择 .js / .css 文件"})
		return
	}

	var saved []string
	for _, hdr := range form.File["files"] {
		ext := strings.ToLower(filepath.Ext(hdr.Filename))
		if ext != ".js" && ext != ".css" {
			continue
		}
		f, err := hdr.Open()
		if err != nil {
			continue
		}
		data, err := io.ReadAll(io.LimitReader(f, 4<<20))
		f.Close()
		if err != nil || len(data) == 0 {
			continue
		}
		dest := filepath.Join(dir, filepath.Base(hdr.Filename))
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			continue
		}
		saved = append(saved, dest)
	}
	if len(saved) == 0 {
		writeJSON(w, map[string]any{"ok": false, "error": "没有成功保存的 .js / .css 文件"})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "paths": saved, "dir": dir})
}
