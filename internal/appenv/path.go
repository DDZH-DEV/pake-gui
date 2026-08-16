package appenv

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// EnrichPath prepends common Node/npm/Rust locations so Explorer-launched
// desktop builds can still find tooling that only exists in a shell PATH.
func EnrichPath() {
	home, _ := os.UserHomeDir()
	localApp := os.Getenv("LOCALAPPDATA")
	appData := os.Getenv("APPDATA")
	programFiles := os.Getenv("ProgramFiles")
	userProfile := os.Getenv("USERPROFILE")

	candidates := []string{}
	if runtime.GOOS == "windows" {
		candidates = append(candidates,
			filepath.Join(programFiles, "nodejs"),
			filepath.Join(localApp, "Programs", "nodejs"),
			filepath.Join(localApp, "nvs", "default"),
			filepath.Join(userProfile, ".nvs", "default"),
			filepath.Join(appData, "npm"),
			filepath.Join(userProfile, ".cargo", "bin"),
			filepath.Join(userProfile, ".rustup", "toolchains"),
		)
		// nvs default may be a junction to a versioned node
		if localApp != "" {
			nvs := filepath.Join(localApp, "nvs")
			entries, _ := os.ReadDir(nvs)
			for _, e := range entries {
				if !e.IsDir() {
					continue
				}
				p := filepath.Join(nvs, e.Name())
				if exists(filepath.Join(p, "node.exe")) {
					candidates = append(candidates, p)
				}
			}
		}
	} else {
		candidates = append(candidates,
			"/usr/local/bin",
			filepath.Join(home, ".nvm", "current", "bin"),
			filepath.Join(home, ".local", "bin"),
			filepath.Join(home, ".cargo", "bin"),
		)
	}

	path := os.Getenv("PATH")
	parts := splitPath(path)
	seen := map[string]bool{}
	for _, p := range parts {
		seen[strings.ToLower(p)] = true
	}

	var prepend []string
	for _, c := range candidates {
		if c == "" || !exists(c) {
			continue
		}
		key := strings.ToLower(c)
		if seen[key] {
			continue
		}
		seen[key] = true
		prepend = append(prepend, c)
	}

	if len(prepend) == 0 {
		return
	}
	os.Setenv("PATH", strings.Join(append(prepend, parts...), string(os.PathListSeparator)))
}

func splitPath(path string) []string {
	if path == "" {
		return nil
	}
	return strings.Split(path, string(os.PathListSeparator))
}

func exists(p string) bool {
	_, err := os.Stat(p)
	return err == nil
}
