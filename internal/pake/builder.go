package pake

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
)

// Options mirrors common pake-cli flags used by the GUI.
type Options struct {
	URL                     string   `json:"url"`
	Name                    string   `json:"name"`
	Icon                    string   `json:"icon"`
	Width                   int      `json:"width"`
	Height                  int      `json:"height"`
	MinWidth                int      `json:"minWidth"`
	MinHeight               int      `json:"minHeight"`
	Zoom                    int      `json:"zoom"`
	AppVersion              string   `json:"appVersion"`
	Title                    string   `json:"title"`
	Identifier              string   `json:"identifier"`
	UserAgent               string   `json:"userAgent"`
	ActivationShortcut      string   `json:"activationShortcut"`
	Targets                 string   `json:"targets"`
	Inject                  []string `json:"inject"`
	SafeDomain              string   `json:"safeDomain"`
	InternalURLRegex        string   `json:"internalUrlRegex"`
	ProxyURL                string   `json:"proxyUrl"`
	SystemTrayIcon          string   `json:"systemTrayIcon"`
	InstallerLanguage       string   `json:"installerLanguage"`
	HideTitleBar            bool     `json:"hideTitleBar"`
	HideWindowDecorations   bool     `json:"hideWindowDecorations"`
	Fullscreen              bool     `json:"fullscreen"`
	Maximize                bool     `json:"maximize"`
	AlwaysOnTop             bool     `json:"alwaysOnTop"`
	DarkMode                bool     `json:"darkMode"`
	Debug                   bool     `json:"debug"`
	ShowSystemTray          bool     `json:"showSystemTray"`
	HideOnClose             *bool    `json:"hideOnClose"`
	StartToTray             bool     `json:"startToTray"`
	Incognito               bool     `json:"incognito"`
	Wasm                    bool     `json:"wasm"`
	EnableDragDrop          bool     `json:"enableDragDrop"`
	KeepBinary              bool     `json:"keepBinary"`
	MultiInstance           bool     `json:"multiInstance"`
	MultiWindow             bool     `json:"multiWindow"`
	ForceInternalNavigation bool     `json:"forceInternalNavigation"`
	EnableFind              bool     `json:"enableFind"`
	IgnoreCertificateErrors bool     `json:"ignoreCertificateErrors"`
	DisabledWebShortcuts    bool     `json:"disabledWebShortcuts"`
	UseLocalFile            bool     `json:"useLocalFile"`
	MultiArch               bool     `json:"multiArch"`
	IterativeBuild          bool     `json:"iterativeBuild"`
	OutDir                  string   `json:"outDir"`
	AllowExternalOutDir     bool     `json:"allowExternalOutDir"`
}

type LogFn func(line string)

type Result struct {
	OK      bool   `json:"ok"`
	Message string `json:"message"`
	OutDir  string `json:"outDir"`
	Command string `json:"command"`
}

func BuildArgs(o Options) ([]string, error) {
	url := strings.TrimSpace(o.URL)
	name := strings.TrimSpace(o.Name)
	if url == "" {
		return nil, fmt.Errorf("url is required")
	}
	if name == "" {
		return nil, fmt.Errorf("name is required")
	}

	args := []string{url, "--name", name, "--json"}

	if v := strings.TrimSpace(o.Icon); v != "" {
		args = append(args, "--icon", v)
	}
	if o.Width > 0 {
		args = append(args, "--width", strconv.Itoa(o.Width))
	}
	if o.Height > 0 {
		args = append(args, "--height", strconv.Itoa(o.Height))
	}
	if o.MinWidth > 0 {
		args = append(args, "--min-width", strconv.Itoa(o.MinWidth))
	}
	if o.MinHeight > 0 {
		args = append(args, "--min-height", strconv.Itoa(o.MinHeight))
	}
	if o.Zoom > 0 {
		args = append(args, "--zoom", strconv.Itoa(o.Zoom))
	}
	if v := strings.TrimSpace(o.AppVersion); v != "" {
		args = append(args, "--app-version", v)
	}
	if v := strings.TrimSpace(o.Title); v != "" {
		args = append(args, "--title", v)
	}
	if v := strings.TrimSpace(o.Identifier); v != "" {
		args = append(args, "--identifier", v)
	}
	if v := strings.TrimSpace(o.UserAgent); v != "" {
		args = append(args, "--user-agent", v)
	}
	if v := strings.TrimSpace(o.ActivationShortcut); v != "" {
		args = append(args, "--activation-shortcut", v)
	}
	if v := strings.TrimSpace(o.Targets); v != "" {
		args = append(args, "--targets", v)
	}
	if v := strings.TrimSpace(o.SafeDomain); v != "" {
		args = append(args, "--safe-domain", v)
	}
	if v := strings.TrimSpace(o.InternalURLRegex); v != "" {
		args = append(args, "--internal-url-regex", v)
	}
	if v := strings.TrimSpace(o.ProxyURL); v != "" {
		args = append(args, "--proxy-url", v)
	}
	if v := strings.TrimSpace(o.SystemTrayIcon); v != "" {
		args = append(args, "--system-tray-icon", v)
	}
	if v := strings.TrimSpace(o.InstallerLanguage); v != "" {
		args = append(args, "--installer-language", v)
	}
	for _, inj := range o.Inject {
		inj = strings.TrimSpace(inj)
		if inj != "" {
			args = append(args, "--inject", inj)
		}
	}

	addFlag := func(on bool, flag string) {
		if on {
			args = append(args, flag)
		}
	}
	addFlag(o.HideTitleBar, "--hide-title-bar")
	addFlag(o.HideWindowDecorations, "--hide-window-decorations")
	addFlag(o.Fullscreen, "--fullscreen")
	addFlag(o.Maximize, "--maximize")
	addFlag(o.AlwaysOnTop, "--always-on-top")
	addFlag(o.DarkMode, "--dark-mode")
	addFlag(o.Debug, "--debug")
	addFlag(o.ShowSystemTray, "--show-system-tray")
	addFlag(o.StartToTray, "--start-to-tray")
	addFlag(o.Incognito, "--incognito")
	addFlag(o.Wasm, "--wasm")
	addFlag(o.EnableDragDrop, "--enable-drag-drop")
	addFlag(o.KeepBinary, "--keep-binary")
	addFlag(o.MultiInstance, "--multi-instance")
	addFlag(o.MultiWindow, "--multi-window")
	addFlag(o.ForceInternalNavigation, "--force-internal-navigation")
	addFlag(o.EnableFind, "--enable-find")
	addFlag(o.IgnoreCertificateErrors, "--ignore-certificate-errors")
	addFlag(o.DisabledWebShortcuts, "--disabled-web-shortcuts")
	addFlag(o.UseLocalFile, "--use-local-file")
	addFlag(o.MultiArch, "--multi-arch")
	addFlag(o.IterativeBuild, "--iterative-build")

	if o.HideOnClose != nil {
		args = append(args, "--hide-on-close", strconv.FormatBool(*o.HideOnClose))
	}

	return args, nil
}

func ResolveRunner() (bin string, prefix []string, err error) {
	if path, e := exec.LookPath("pake"); e == nil {
		return path, nil, nil
	}
	if path, e := exec.LookPath("pake-cli"); e == nil {
		return path, nil, nil
	}
	if path, e := exec.LookPath("npx"); e == nil {
		return path, []string{"--yes", "pake-cli"}, nil
	}
	return "", nil, fmt.Errorf("pake-cli not found; install with: npm i -g pake-cli")
}

func Run(ctx context.Context, o Options, log LogFn) Result {
	if note := NormalizePackageIdentity(&o); note != "" && log != nil {
		log(note)
	}

	args, err := BuildArgs(o)
	if err != nil {
		return Result{OK: false, Message: err.Error()}
	}

	bin, prefix, err := ResolveRunner()
	if err != nil {
		return Result{OK: false, Message: err.Error()}
	}

	fullArgs := append(append([]string{}, prefix...), args...)
	cmdLine := shellQuote(bin) + " " + strings.Join(quoteEach(fullArgs), " ")

	outDir := strings.TrimSpace(o.OutDir)
	if outDir == "" {
		cwd, _ := os.Getwd()
		switch runtime.GOOS {
		case "windows":
			outDir = filepath.Join(cwd, "builds", "windows")
		case "darwin":
			outDir = filepath.Join(cwd, "builds", "macos")
		default:
			outDir = filepath.Join(cwd, "builds")
		}
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return Result{OK: false, Message: err.Error(), Command: cmdLine}
	}

	log(fmt.Sprintf("$ %s", cmdLine))
	log(fmt.Sprintf("cwd: %s", outDir))

	cmd := exec.Command(bin, fullArgs...)
	cmd.Dir = outDir
	cmd.Env = os.Environ()
	prepareCmd(cmd)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return Result{OK: false, Message: err.Error(), Command: cmdLine, OutDir: outDir}
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return Result{OK: false, Message: err.Error(), Command: cmdLine, OutDir: outDir}
	}

	if err := cmd.Start(); err != nil {
		return Result{OK: false, Message: err.Error(), Command: cmdLine, OutDir: outDir}
	}

	doneKill := make(chan struct{})
	go func() {
		select {
		case <-ctx.Done():
			if cmd.Process != nil {
				killProcessTree(cmd.Process.Pid)
			}
		case <-doneKill:
		}
	}()

	var wg sync.WaitGroup
	pipe := func(r io.Reader) {
		defer wg.Done()
		sc := bufio.NewScanner(r)
		sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		for sc.Scan() {
			log(sc.Text())
		}
	}
	wg.Add(2)
	go pipe(stdout)
	go pipe(stderr)
	wg.Wait()

	err = cmd.Wait()
	close(doneKill)
	if err != nil {
		msg := err.Error()
		if ctx.Err() != nil {
			msg = "build cancelled"
		}
		return Result{OK: false, Message: msg, Command: cmdLine, OutDir: outDir}
	}

	return Result{OK: true, Message: "build finished", Command: cmdLine, OutDir: outDir}
}

func quoteEach(args []string) []string {
	out := make([]string, len(args))
	for i, a := range args {
		out[i] = shellQuote(a)
	}
	return out
}

func shellQuote(s string) string {
	if s == "" {
		return `""`
	}
	if !strings.ContainsAny(s, " \t\"'\\&|;<>()$`") {
		return s
	}
	if runtime.GOOS == "windows" {
		return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"`
	}
	return `'` + strings.ReplaceAll(s, `'`, `'\''`) + `'`
}
