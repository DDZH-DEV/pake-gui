package server

import (
	"context"
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"pake-gui/internal/applog"
	"pake-gui/internal/cloud/android"
	"pake-gui/internal/cloud/common"
	"pake-gui/internal/cloud/github"
)

func (s *Server) cloudStore() *common.Store {
	return common.NewStore(s.cfg.DataDir)
}

func (s *Server) platformBuildsDir(p common.Platform) string {
	switch p {
	case common.PlatformMacOS:
		return filepath.Join(s.cfg.BuildsDir, "macos")
	case common.PlatformAndroid:
		return filepath.Join(s.cfg.BuildsDir, "android")
	case common.PlatformWindows:
		return filepath.Join(s.cfg.BuildsDir, "windows")
	default:
		return s.cfg.BuildsDir
	}
}

func (s *Server) handleCloudGitHubSettings(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		st, err := github.LoadSettings(s.cfg.DataDir)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "settings": st.PublicView()})
	case http.MethodPost:
		var body struct {
			Owner    string `json:"owner"`
			Repo     string `json:"repo"`
			Token    string `json:"token"`
			Workflow string `json:"workflow"`
			Ref      string `json:"ref"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		existing, _ := github.LoadSettings(s.cfg.DataDir)
		token := strings.TrimSpace(body.Token)
		if token == "" {
			token = existing.Token // allow update without re-entering token
		}
		st := github.Settings{
			Owner:    body.Owner,
			Repo:     body.Repo,
			Token:    token,
			Workflow: body.Workflow,
			Ref:      body.Ref,
		}
		if err := github.SaveSettings(s.cfg.DataDir, st); err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		applog.Info("github settings saved owner=%s repo=%s", st.Owner, st.Repo)
		writeJSON(w, map[string]any{"ok": true, "settings": st.PublicView()})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCloudGitHubTest(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	st, err := github.LoadSettings(s.cfg.DataDir)
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": "请先保存 GitHub 设置"})
		return
	}
	if strings.TrimSpace(st.Token) == "" || strings.TrimSpace(st.Owner) == "" || strings.TrimSpace(st.Repo) == "" {
		writeJSON(w, map[string]any{"ok": false, "error": "请先保存 GitHub 设置"})
		return
	}
	client := github.NewClient(st)
	info, err := client.Test(r.Context())
	if err != nil {
		writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
		return
	}
	writeJSON(w, info)
}

func (s *Server) handleCloudJobs(w http.ResponseWriter, r *http.Request) {
	store := s.cloudStore()
	switch r.Method {
	case http.MethodGet:
		items, err := store.List()
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "jobs": items})
	case http.MethodPost:
		var body struct {
			Platform     string `json:"platform"`
			URL          string `json:"url"`
			Name         string `json:"name"`
			Icon         string `json:"icon"`
			Width        int    `json:"width"`
			Height       int    `json:"height"`
			AppVersion   string `json:"appVersion"`
			Identifier   string `json:"identifier"`
			HideTitleBar bool   `json:"hideTitleBar"`
			MultiArch    bool   `json:"multiArch"`
			NewWindow    bool   `json:"newWindow"`
			Targets      string `json:"targets"`
		}
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		platform := common.Platform(strings.TrimSpace(body.Platform))
		if platform == "" {
			platform = common.PlatformMacOS
		}
		if strings.TrimSpace(body.URL) == "" || strings.TrimSpace(body.Name) == "" {
			writeJSON(w, map[string]any{"ok": false, "error": "url 和 name 必填"})
			return
		}
		if platform == common.PlatformAndroid {
			writeJSON(w, map[string]any{"ok": false, "error": android.ErrNotImplemented.Error()})
			return
		}
		if platform != common.PlatformMacOS {
			writeJSON(w, map[string]any{"ok": false, "error": "暂不支持的平台: " + string(platform)})
			return
		}

		st, err := github.LoadSettings(s.cfg.DataDir)
		if err != nil || strings.TrimSpace(st.Token) == "" {
			writeJSON(w, map[string]any{"ok": false, "error": "请先在「云端设置」保存 GitHub Token 与仓库"})
			return
		}

		req := common.Request{
			Platform:     platform,
			URL:          strings.TrimSpace(body.URL),
			Name:         strings.TrimSpace(body.Name),
			Icon:         strings.TrimSpace(body.Icon),
			Width:        body.Width,
			Height:       body.Height,
			AppVersion:   strings.TrimSpace(body.AppVersion),
			Identifier:   strings.TrimSpace(body.Identifier),
			HideTitleBar: body.HideTitleBar,
			MultiArch:    body.MultiArch,
			NewWindow:    body.NewWindow,
			Targets:      strings.TrimSpace(body.Targets),
		}
		job, err := store.Create(req)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}

		outDir := s.platformBuildsDir(platform)
		_ = os.MkdirAll(outDir, 0o755)

		go s.runCloudJob(job.ID, st, outDir)

		applog.Info("cloud job created id=%s platform=%s name=%s", job.ID, platform, req.Name)
		writeJSON(w, map[string]any{"ok": true, "job": job})
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) handleCloudJobByID(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/cloud/jobs/")
	id = strings.Trim(id, "/")
	if id == "" || strings.Contains(id, "/") {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}
	store := s.cloudStore()

	switch r.Method {
	case http.MethodGet:
		job, err := store.Get(id)
		if err != nil {
			writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
			return
		}
		writeJSON(w, map[string]any{"ok": true, "job": job})
	case http.MethodPost:
		// /api/cloud/jobs/{id}/open or cancel via query?action=
		action := r.URL.Query().Get("action")
		if action == "" {
			var body struct {
				Action string `json:"action"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			action = body.Action
		}
		switch action {
		case "open":
			job, err := store.Get(id)
			if err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			path := job.Status.LocalOut
			if path == "" {
				path = s.platformBuildsDir(job.Request.Platform)
			}
			_ = os.MkdirAll(path, 0o755)
			if err := openPath(path); err != nil {
				writeJSON(w, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			writeJSON(w, map[string]any{"ok": true, "path": path})
		case "cancel":
			s.mu.Lock()
			if c, ok := s.cloudCancels[id]; ok && c != nil {
				c()
				delete(s.cloudCancels, id)
			}
			s.mu.Unlock()
			_ = store.SaveStatus(id, common.Status{State: common.StateCanceled, Message: "cancel requested"})
			writeJSON(w, map[string]any{"ok": true})
		default:
			writeJSON(w, map[string]any{"ok": false, "error": "unknown action"})
		}
	default:
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *Server) runCloudJob(jobID string, st github.Settings, buildsDir string) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	if s.cloudCancels == nil {
		s.cloudCancels = map[string]context.CancelFunc{}
	}
	if prev, ok := s.cloudCancels[jobID]; ok && prev != nil {
		prev()
	}
	s.cloudCancels[jobID] = cancel
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.cloudCancels, jobID)
		s.mu.Unlock()
		cancel()
	}()

	store := s.cloudStore()
	err := github.RunMacOSJob(ctx, github.MacOSSubmitOptions{
		DataDir:   s.cfg.DataDir,
		BuildsDir: buildsDir,
		Store:     store,
		Settings:  st,
		JobID:     jobID,
		Log: func(line string) {
			applog.Info("cloud[%s] %s", jobID, line)
		},
	})
	if err != nil && ctx.Err() == nil {
		applog.Error("cloud job %s failed: %v", jobID, err)
	}
}
