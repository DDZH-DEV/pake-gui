package server

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"pake-gui/internal/applog"
)

type Config struct {
	Root      string
	BuildsDir string
	DataDir   string
	Token     string
}

func NewToken() string {
	b := make([]byte, 24)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return hex.EncodeToString(b)
}

func (s *Server) withSecurity(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if rec := recover(); rec != nil {
				applog.Error("panic: %v", rec)
				http.Error(w, "internal error", http.StatusInternalServerError)
			}
		}()

		// Always localhost-only listener; still reject non-loopback remote addrs.
		host, _, err := net.SplitHostPort(r.RemoteAddr)
		if err == nil && host != "127.0.0.1" && host != "::1" && host != "localhost" {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}

		if strings.HasPrefix(r.URL.Path, "/api/") {
			if !s.originOK(r) {
				http.Error(w, "invalid origin", http.StatusForbidden)
				return
			}
			if !s.tokenOK(r) {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
			if r.Body != nil && r.Method != http.MethodGet {
				limit := int64(1 << 20) // 1MB
				if r.URL.Path == "/api/upload-icon" {
					limit = 8 << 20 // 8MB for icons
				}
				if r.URL.Path == "/api/inject/upload" {
					limit = 16 << 20
				}
				r.Body = http.MaxBytesReader(w, r.Body, limit)
			}
		}

		next.ServeHTTP(w, r)
	})
}

func (s *Server) tokenOK(r *http.Request) bool {
	if s.cfg.Token == "" {
		return true
	}
	t := r.Header.Get("X-Pake-Token")
	if t == "" {
		t = r.URL.Query().Get("token")
	}
	return t == s.cfg.Token
}

func (s *Server) originOK(r *http.Request) bool {
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true // same-origin / non-browser / WebView
	}
	return isLocalOrigin(origin)
}

func isLocalOrigin(origin string) bool {
	o := strings.ToLower(origin)
	return strings.HasPrefix(o, "http://127.0.0.1") ||
		strings.HasPrefix(o, "http://localhost") ||
		strings.HasPrefix(o, "http://[::1]")
}

// defaultLocalOutDir is the platform folder under builds/ for local packaging.
func (s *Server) defaultLocalOutDir() string {
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(s.cfg.BuildsDir, "windows")
	case "darwin":
		return filepath.Join(s.cfg.BuildsDir, "macos")
	default:
		return s.cfg.BuildsDir
	}
}

// resolveOutDir confines output to buildsDir unless allowExternal is true.
func (s *Server) resolveOutDir(requested string, allowExternal bool) (string, error) {
	base, err := filepath.Abs(s.cfg.BuildsDir)
	if err != nil {
		return "", err
	}
	req := strings.TrimSpace(requested)
	if req == "" {
		return filepath.Abs(s.defaultLocalOutDir())
	}

	var abs string
	if filepath.IsAbs(req) {
		abs, err = filepath.Abs(req)
	} else {
		abs, err = filepath.Abs(filepath.Join(base, req))
	}
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)

	if isSubPath(base, abs) {
		return abs, nil
	}
	if !allowExternal {
		return "", fmt.Errorf("输出目录必须在 %s 内；若需自定义目录请确认后重试", base)
	}
	return abs, nil
}

func (s *Server) resolveOpenPath(requested string) (string, error) {
	base, err := filepath.Abs(s.cfg.BuildsDir)
	if err != nil {
		return "", err
	}
	req := strings.TrimSpace(requested)
	if req == "" {
		return filepath.Abs(s.defaultLocalOutDir())
	}
	var abs string
	if filepath.IsAbs(req) {
		abs, err = filepath.Abs(req)
	} else {
		abs, err = filepath.Abs(filepath.Join(base, req))
	}
	if err != nil {
		return "", err
	}
	abs = filepath.Clean(abs)
	if !isSubPath(base, abs) {
		return "", fmt.Errorf("只能打开输出目录内的路径")
	}
	return abs, nil
}

func isSubPath(base, target string) bool {
	base = filepath.Clean(base)
	target = filepath.Clean(target)
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return !strings.HasPrefix(rel, "..")
}
