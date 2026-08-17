package server

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/fs"
	"net"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"pake-gui/internal/applog"
	"pake-gui/internal/pake"
)

type Server struct {
	cfg          Config
	mux          *http.ServeMux
	mu           sync.Mutex
	cancel       context.CancelFunc
	cloudCancels map[string]context.CancelFunc
	webFS        fs.FS
}

type projectRecord struct {
	ID        string       `json:"id"`
	CreatedAt time.Time    `json:"createdAt"`
	Options   pake.Options `json:"options"`
	Result    *pake.Result `json:"result,omitempty"`
}

func New(cfg Config) *Server {
	if cfg.Token == "" {
		cfg.Token = NewToken()
	}
	s := &Server{
		cfg:   cfg,
		mux:   http.NewServeMux(),
		webFS: mustSub(webContent, "web"),
	}
	s.routes()
	return s
}

func (s *Server) Token() string { return s.cfg.Token }

func (s *Server) Serve(ln net.Listener) error {
	return http.Serve(ln, s.withSecurity(s.mux))
}

func (s *Server) routes() {
	s.mux.HandleFunc("/api/env", s.handleEnv)
	s.mux.HandleFunc("/api/install-pake", s.handleInstallPake)
	s.mux.HandleFunc("/api/build", s.handleBuild)
	s.mux.HandleFunc("/api/cancel", s.handleCancel)
	s.mux.HandleFunc("/api/history", s.handleHistory)
	s.mux.HandleFunc("/api/open-output", s.handleOpenOutput)
	s.mux.HandleFunc("/api/open-log", s.handleOpenLog)
	s.mux.HandleFunc("/api/preview-cmd", s.handlePreviewCmd)
	s.mux.HandleFunc("/api/upload-icon", s.handleUploadIcon)
	s.mux.HandleFunc("/api/icon-file", s.handleIconFile)
	s.mux.HandleFunc("/api/inject/list", s.handleInjectList)
	s.mux.HandleFunc("/api/inject/upload", s.handleInjectUpload)
	s.mux.HandleFunc("/api/cloud/github/settings", s.handleCloudGitHubSettings)
	s.mux.HandleFunc("/api/cloud/github/test", s.handleCloudGitHubTest)
	s.mux.HandleFunc("/api/cloud/github/oauth/start", s.handleCloudGitHubOAuthStart)
	s.mux.HandleFunc("/api/cloud/github/oauth/status", s.handleCloudGitHubOAuthStatus)
	s.mux.HandleFunc("/api/cloud/github/oauth/logout", s.handleCloudGitHubOAuthLogout)
	s.mux.HandleFunc("/api/cloud/jobs", s.handleCloudJobs)
	s.mux.HandleFunc("/api/cloud/jobs/", s.handleCloudJobByID)
	s.mux.Handle("/", s.spa())
}

func (s *Server) spa() http.Handler {
	fileServer := http.FileServer(http.FS(s.webFS))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if path == "/" || path == "/index.html" {
			s.serveIndex(w, r)
			return
		}
		name := strings.TrimPrefix(path, "/")
		if _, err := fs.Stat(s.webFS, name); err != nil {
			s.serveIndex(w, r)
			return
		}
		fileServer.ServeHTTP(w, r)
	})
}

func (s *Server) serveIndex(w http.ResponseWriter, r *http.Request) {
	b, err := fs.ReadFile(s.webFS, "index.html")
	if err != nil {
		http.Error(w, "index missing", http.StatusInternalServerError)
		return
	}
	html := string(b)
	inject := fmt.Sprintf(`<script>window.__PAKE_TOKEN__=%q;window.__PAKE_BUILDS__=%q;</script>`, s.cfg.Token, s.cfg.BuildsDir)
	if strings.Contains(html, "</head>") {
		html = strings.Replace(html, "</head>", inject+"\n</head>", 1)
	} else {
		html = inject + html
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	_, _ = w.Write([]byte(html))
}

func (s *Server) handleEnv(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st := pake.CheckEnv()
	writeJSON(w, map[string]any{
		"os":        st.OS,
		"arch":      st.Arch,
		"tools":     st.Tools,
		"ready":     st.Ready,
		"pakeOk":    st.PakeOK,
		"builds":    s.cfg.BuildsDir,
		"injectDir": s.injectDir(),
		"logPath":   applog.Path(),
	})
}

func (s *Server) handleInstallPake(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	applog.Info("install pake-cli requested")
	out, err := pake.InstallPakeCLI()
	if err != nil {
		applog.Error("install pake-cli failed: %v", err)
		writeJSON(w, map[string]any{"ok": false, "output": out, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "output": out})
}

func (s *Server) handlePreviewCmd(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var opts pake.Options
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	outDir, err := s.resolveOutDir(opts.OutDir, opts.AllowExternalOutDir)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	opts.OutDir = outDir
	args, err := pake.BuildArgs(opts)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	bin, prefix, err := pake.ResolveRunner()
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	full := append(append([]string{}, prefix...), args...)
	cmd := bin
	for _, a := range full {
		cmd += " " + quote(a)
	}
	writeJSON(w, map[string]any{"ok": true, "command": cmd, "outDir": outDir})
}

func (s *Server) handleBuild(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	var opts pake.Options
	if err := json.NewDecoder(r.Body).Decode(&opts); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	outDir, err := s.resolveOutDir(opts.OutDir, opts.AllowExternalOutDir)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	opts.OutDir = outDir

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")

	ctx, cancel := context.WithCancel(r.Context())
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
	}
	s.cancel = cancel
	s.mu.Unlock()
	defer cancel()

	send := func(event string, payload any) {
		defer func() { _ = recover() }()
		b, _ := json.Marshal(payload)
		fmt.Fprintf(w, "event: %s\ndata: %s\n\n", event, b)
		flusher.Flush()
	}

	applog.Info("build start name=%s url=%s out=%s", opts.Name, opts.URL, outDir)
	send("log", map[string]string{"line": "starting build…"})

	result := pake.Run(ctx, opts, func(line string) {
		send("log", map[string]string{"line": line})
	})

	if result.OK {
		applog.Info("build ok: %s", outDir)
	} else {
		applog.Error("build failed: %s", result.Message)
	}

	rec := projectRecord{
		ID:        fmt.Sprintf("%d", time.Now().UnixMilli()),
		CreatedAt: time.Now(),
		Options:   opts,
		Result:    &result,
	}
	_ = s.appendHistory(rec)
	send("done", result)
}

func (s *Server) handleCancel(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	s.mu.Lock()
	if s.cancel != nil {
		s.cancel()
		s.cancel = nil
	}
	s.mu.Unlock()
	applog.Info("build cancel requested")
	writeJSON(w, map[string]any{"ok": true})
}

func (s *Server) handleHistory(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	items, err := s.loadHistory()
	if err != nil {
		writeJSON(w, []projectRecord{})
		return
	}
	writeJSON(w, items)
}

func (s *Server) handleUploadIcon(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	if err := r.ParseMultipartForm(8 << 20); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "文件过大或无效（最大 8MB）"})
		return
	}
	file, hdr, err := r.FormFile("icon")
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "请选择图标文件"})
		return
	}
	defer file.Close()

	ext := strings.ToLower(filepath.Ext(hdr.Filename))
	allowed := map[string]bool{".png": true, ".ico": true, ".icns": true, ".jpg": true, ".jpeg": true, ".webp": true}
	if !allowed[ext] {
		writeJSON(w, map[string]any{"ok": false, "error": "仅支持 png / ico / icns / jpg / webp"})
		return
	}

	data, err := io.ReadAll(io.LimitReader(file, 8<<20))
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "读取文件失败"})
		return
	}
	if len(data) == 0 || !looksLikeImage(data, ext) {
		writeJSON(w, map[string]any{"ok": false, "error": "文件内容不是有效图片"})
		return
	}

	dir := filepath.Join(s.cfg.DataDir, "icons")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	name := fmt.Sprintf("icon-%d%s", time.Now().UnixNano(), ext)
	dest := filepath.Join(dir, name)
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}

	applog.Info("icon uploaded: %s", dest)
	writeJSON(w, map[string]any{
		"ok":       true,
		"path":     dest,
		"filename": hdr.Filename,
		"preview":  "/api/icon-file?name=" + url.QueryEscape(name) + "&token=" + url.QueryEscape(s.cfg.Token),
	})
}

func looksLikeImage(head []byte, ext string) bool {
	if len(head) < 4 {
		return false
	}
	switch {
	case bytes.HasPrefix(head, []byte{0x89, 0x50, 0x4E, 0x47}):
		return true
	case bytes.HasPrefix(head, []byte{0x00, 0x00, 0x01, 0x00}):
		return true
	case bytes.HasPrefix(head, []byte{0xFF, 0xD8, 0xFF}):
		return true
	case bytes.HasPrefix(head, []byte("RIFF")) && len(head) >= 12 && string(head[8:12]) == "WEBP":
		return true
	case bytes.HasPrefix(head, []byte("icns")):
		return true
	case ext == ".ico" || ext == ".icns":
		return true
	default:
		return false
	}
}

func (s *Server) handleIconFile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	name := filepath.Base(r.URL.Query().Get("name"))
	if name == "" || name == "." || name == ".." {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	path := filepath.Join(s.cfg.DataDir, "icons", name)
	if !isSubPath(filepath.Join(s.cfg.DataDir, "icons"), path) {
		http.Error(w, "forbidden", http.StatusForbidden)
		return
	}
	http.ServeFile(w, r, path)
}

func (s *Server) handleOpenOutput(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var body struct {
		Path string `json:"path"`
	}
	_ = json.NewDecoder(r.Body).Decode(&body)
	path, err := s.resolveOpenPath(body.Path)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	_ = os.MkdirAll(path, 0o755)
	if err := openPath(path); err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, map[string]any{"ok": true, "path": path})
}

func (s *Server) handleOpenLog(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	path := applog.Path()
	if path == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "log not ready"})
		return
	}
	if err := openPath(path); err != nil {
		// fallback: open containing folder
		_ = openPath(filepath.Dir(path))
	}
	writeJSON(w, map[string]any{"ok": true, "path": path})
}

func (s *Server) historyPath() string {
	return filepath.Join(s.cfg.DataDir, "history.json")
}

func (s *Server) loadHistory() ([]projectRecord, error) {
	b, err := os.ReadFile(s.historyPath())
	if err != nil {
		return nil, err
	}
	var items []projectRecord
	if err := json.Unmarshal(b, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func (s *Server) appendHistory(rec projectRecord) error {
	items, _ := s.loadHistory()
	items = append([]projectRecord{rec}, items...)
	if len(items) > 50 {
		items = items[:50]
	}
	b, err := json.MarshalIndent(items, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.historyPath(), b, 0o644)
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	_ = enc.Encode(v)
}

func quote(s string) string {
	if s == "" {
		return `""`
	}
	needs := false
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '"' {
			needs = true
			break
		}
	}
	if !needs {
		return s
	}
	return `"` + s + `"`
}
