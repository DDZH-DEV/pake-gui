package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const settingsFile = "github.json"

// Defaults for this project (https://github.com/DDZH-DEV/pake-gui).
const (
	DefaultOwner    = "DDZH-DEV"
	DefaultRepo     = "pake-gui"
	DefaultWorkflow = "build-macos.yml"
)

// Settings is stored in data/github.json (gitignored via data/).
type Settings struct {
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	Token    string `json:"token"`
	Login    string `json:"login,omitempty"`    // GitHub username after OAuth
	ClientID string `json:"clientId,omitempty"` // OAuth App client id (public)
	Workflow string `json:"workflow"`           // default: build-macos.yml
	Ref      string `json:"ref"`                // branch; empty = repo default
}

func settingsPath(dataDir string) string {
	return filepath.Join(dataDir, settingsFile)
}

func LoadSettings(dataDir string) (Settings, error) {
	b, err := os.ReadFile(settingsPath(dataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{
				Owner:    DefaultOwner,
				Repo:     DefaultRepo,
				Workflow: DefaultWorkflow,
			}, nil
		}
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(b, &s); err != nil {
		return Settings{}, err
	}
	if strings.TrimSpace(s.Workflow) == "" {
		s.Workflow = DefaultWorkflow
	}
	if strings.TrimSpace(s.Owner) == "" {
		s.Owner = DefaultOwner
	}
	if strings.TrimSpace(s.Repo) == "" {
		s.Repo = DefaultRepo
	}
	// Merge public client id from configs if local empty.
	if strings.TrimSpace(s.ClientID) == "" {
		if id := loadBundledClientID(dataDir); id != "" {
			s.ClientID = id
		}
	}
	return s, nil
}

func loadBundledClientID(dataDir string) string {
	// Prefer repo configs next to dataDir's parent (app root).
	root := filepath.Dir(dataDir)
	candidates := []string{
		filepath.Join(root, "configs", "github-oauth.json"),
		filepath.Join("configs", "github-oauth.json"),
	}
	for _, p := range candidates {
		b, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		var cfg struct {
			ClientID string `json:"clientId"`
		}
		if json.Unmarshal(b, &cfg) == nil && strings.TrimSpace(cfg.ClientID) != "" {
			return strings.TrimSpace(cfg.ClientID)
		}
	}
	return ""
}

func SaveSettings(dataDir string, s Settings) error {
	return saveSettings(dataDir, s, true)
}

// SaveSettingsAllowEmptyToken persists owner/repo/clientId without requiring a token.
func SaveSettingsAllowEmptyToken(dataDir string, s Settings) error {
	return saveSettings(dataDir, s, false)
}

func saveSettings(dataDir string, s Settings, requireToken bool) error {
	s.Owner = strings.TrimSpace(s.Owner)
	s.Repo = strings.TrimSpace(s.Repo)
	s.Token = strings.TrimSpace(s.Token)
	s.Login = strings.TrimSpace(s.Login)
	s.ClientID = strings.TrimSpace(s.ClientID)
	s.Workflow = strings.TrimSpace(s.Workflow)
	s.Ref = strings.TrimSpace(s.Ref)
	if s.Workflow == "" {
		s.Workflow = DefaultWorkflow
	}
	if s.Owner == "" {
		s.Owner = DefaultOwner
	}
	if s.Repo == "" {
		s.Repo = DefaultRepo
	}
	if requireToken && s.Token == "" {
		return fmt.Errorf("尚未登录：请先使用 GitHub 授权，或在高级选项中填写 Token")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		return err
	}
	b, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(settingsPath(dataDir), b, 0o600)
}

// PublicView hides the raw token for API responses.
func (s Settings) PublicView() map[string]any {
	masked := ""
	tok := s.Token
	if len(tok) > 8 {
		masked = tok[:4] + "…" + tok[len(tok)-4:]
	} else if tok != "" {
		masked = "****"
	}
	return map[string]any{
		"owner":         s.Owner,
		"repo":          s.Repo,
		"workflow":      s.Workflow,
		"ref":           s.Ref,
		"clientId":      s.ClientID,
		"login":         s.Login,
		"tokenMasked":   masked,
		"configured":    s.Owner != "" && s.Repo != "" && s.Token != "",
		"defaultOwner":  DefaultOwner,
		"defaultRepo":   DefaultRepo,
	}
}
