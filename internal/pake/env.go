package pake

import (
	"os/exec"
	"runtime"
	"strings"
)

type ToolStatus struct {
	Name    string `json:"name"`
	OK      bool   `json:"ok"`
	Version string `json:"version"`
	Detail  string `json:"detail"`
}

type EnvStatus struct {
	OS     string       `json:"os"`
	Arch   string       `json:"arch"`
	Tools  []ToolStatus `json:"tools"`
	Ready  bool         `json:"ready"`
	PakeOK bool         `json:"pakeOk"`
}

func CheckEnv() EnvStatus {
	st := EnvStatus{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	node := checkCmd("node", "-v")
	npm := checkCmd("npm", "-v")
	rustc := checkCmd("rustc", "--version")
	cargo := checkCmd("cargo", "--version")

	pake := ToolStatus{Name: "pake-cli"}
	if _, _, err := ResolveRunner(); err == nil {
		pake.OK = true
		if out, e := runHidden("pake", "--version"); e == nil {
			pake.Version = strings.TrimSpace(string(out))
		} else if out, e := runHidden("npx", "--yes", "pake-cli", "--version"); e == nil {
			pake.Version = strings.TrimSpace(string(out))
			pake.Detail = "via npx"
		} else {
			pake.Detail = "available via npx"
		}
	} else {
		pake.Detail = err.Error()
	}

	st.Tools = []ToolStatus{node, npm, rustc, cargo, pake}
	st.PakeOK = pake.OK
	st.Ready = node.OK && npm.OK && pake.OK
	return st
}

func checkCmd(name string, args ...string) ToolStatus {
	st := ToolStatus{Name: name}
	path, err := exec.LookPath(name)
	if err != nil {
		st.Detail = "not found"
		return st
	}
	out, err := runHidden(name, args...)
	ver := strings.TrimSpace(string(out))
	if err != nil && ver == "" {
		st.Detail = path
		st.OK = true
		return st
	}
	st.OK = true
	st.Version = ver
	return st
}

func InstallPakeCLI() (string, error) {
	out, err := runHidden("npm", "install", "-g", "pake-cli")
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, err
	}
	return text, nil
}

func runHidden(name string, args ...string) ([]byte, error) {
	cmd := exec.Command(name, args...)
	prepareCmd(cmd)
	return cmd.CombinedOutput()
}
