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
		if out, e := exec.Command("pake", "--version").CombinedOutput(); e == nil {
			pake.Version = strings.TrimSpace(string(out))
		} else if out, e := exec.Command("npx", "--yes", "pake-cli", "--version").CombinedOutput(); e == nil {
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
	out, err := exec.Command(name, args...).CombinedOutput()
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
	cmd := exec.Command("npm", "install", "-g", "pake-cli")
	out, err := cmd.CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		if text == "" {
			text = err.Error()
		}
		return text, err
	}
	return text, nil
}
