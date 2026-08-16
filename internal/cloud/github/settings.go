package github

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const settingsFile = "github.json"

// Settings is stored in data/github.json (gitignored via data/).
type Settings struct {
	Owner    string `json:"owner"`
	Repo     string `json:"repo"`
	Token    string `json:"token"`
	Workflow string `json:"workflow"` // default: build-macos.yml
	Ref      string `json:"ref"`      // branch; empty = repo default
}

func settingsPath(dataDir string) string {
	return filepath.Join(dataDir, settingsFile)
}

func LoadSettings(dataDir string) (Settings, error) {
	b, err := os.ReadFile(settingsPath(dataDir))
	if err != nil {
		if os.IsNotExist(err) {
			return Settings{Workflow: "build-macos.yml"}, nil
		}
		return Settings{}, err
	}
	var s Settings
	if err := json.Unmarshal(b, &s); err != nil {
		return Settings{}, err
	}
	if strings.TrimSpace(s.Workflow) == "" {
		s.Workflow = "build-macos.yml"
	}
	return s, nil
}

func SaveSettings(dataDir string, s Settings) error {
	s.Owner = strings.TrimSpace(s.Owner)
	s.Repo = strings.TrimSpace(s.Repo)
	s.Token = strings.TrimSpace(s.Token)
	s.Workflow = strings.TrimSpace(s.Workflow)
	s.Ref = strings.TrimSpace(s.Ref)
	if s.Workflow == "" {
		s.Workflow = "build-macos.yml"
	}
	if s.Owner == "" || s.Repo == "" {
		return fmt.Errorf("owner 和 repo 不能为空")
	}
	if s.Token == "" {
		return fmt.Errorf("token 不能为空")
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
		"owner":       s.Owner,
		"repo":        s.Repo,
		"workflow":    s.Workflow,
		"ref":         s.Ref,
		"tokenMasked": masked,
		"configured":  s.Owner != "" && s.Repo != "" && s.Token != "",
	}
}
