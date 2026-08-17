package github

import (
	"archive/zip"
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pake-gui/internal/cloud/common"
)

// CloudJobOptions drives a GitHub Actions build (macOS DMG, Windows exe, or Android APK).
type CloudJobOptions struct {
	DataDir   string
	BuildsDir string
	Store     *common.Store
	Settings  Settings
	JobID     string
	Log       func(string)
}

// MacOSSubmitOptions is kept as an alias for older call sites.
type MacOSSubmitOptions = CloudJobOptions

func (o CloudJobOptions) log(msg string) {
	if o.Log != nil {
		o.Log(msg)
	}
	if o.Store != nil && o.JobID != "" {
		o.Store.AppendLog(o.JobID, msg)
	}
}

// RunCloudJob uploads icon (if any), dispatches workflow, polls, downloads artifact.
func RunCloudJob(ctx context.Context, o CloudJobOptions) error {
	if o.Store == nil {
		return fmt.Errorf("store required")
	}
	job, err := o.Store.Get(o.JobID)
	if err != nil {
		return err
	}
	platform := job.Request.Platform
	if platform == "" {
		platform = common.PlatformMacOS
	}
	spec := specFor(platform)
	client := NewClient(o.Settings)

	_ = o.Store.SaveStatus(o.JobID, common.Status{State: common.StateRunning, Message: "preparing"})
	o.log("准备 GitHub 云端 " + spec.Label + " 打包…")

	ref := strings.TrimSpace(o.Settings.Ref)
	if ref == "" {
		ref, err = client.DefaultBranch(ctx)
		if err != nil {
			_ = fail(o, err)
			return err
		}
	}
	workflow := spec.Workflow

	remote := job.Remote
	remote.Workflow = workflow
	remote.Ref = ref

	iconInput := ""
	rawIcon := strings.TrimSpace(job.Request.Icon)
	if rawIcon != "" && !strings.HasPrefix(strings.ToLower(rawIcon), "http://") &&
		!strings.HasPrefix(strings.ToLower(rawIcon), "https://") {
		// local file → copy into job dir → upload to repo (available after checkout)
		localIcon := rawIcon
		data, err := os.ReadFile(localIcon)
		if err != nil {
			_ = fail(o, fmt.Errorf("读取图标失败: %w", err))
			return err
		}
		ext := filepath.Ext(localIcon)
		if ext == "" {
			ext = ".png"
		}
		jobIcon := filepath.Join(job.Dir, "icon"+ext)
		if err := os.WriteFile(jobIcon, data, 0o644); err != nil {
			_ = fail(o, err)
			return err
		}
		remotePath := fmt.Sprintf("%s/%s/icon%s", spec.IconPrefix, o.JobID, ext)
		o.log("上传图标到仓库: " + remotePath)
		raw, err := client.UploadContent(ctx, ref, remotePath, data, "ci: upload icon for "+o.JobID)
		if err != nil {
			_ = fail(o, fmt.Errorf("上传图标失败: %w", err))
			return err
		}
		// Prefer repo-relative path so private repos work after checkout.
		iconInput = remotePath
		remote.IconRemotePath = remotePath
		remote.IconURL = raw
		o.log("图标已提交，构建将使用路径: " + remotePath)
	} else if rawIcon != "" {
		iconInput = rawIcon
		remote.IconURL = rawIcon
		o.log("使用网络图标: " + rawIcon)
	}

	inputs := map[string]string{
		"url":         job.Request.URL,
		"name":        job.Request.Name,
		"icon":        iconInput,
		"app_version": strDefault(job.Request.AppVersion, "1.0.0"),
		"identifier":  job.Request.Identifier,
		"job_id":      o.JobID,
	}
	switch platform {
	case common.PlatformAndroid:
		// url / name / icon / app_version / identifier / job_id only
	case common.PlatformWindows:
		inputs["width"] = itoaDefault(job.Request.Width, 1200)
		inputs["height"] = itoaDefault(job.Request.Height, 780)
		inputs["new_window"] = FormatBool(job.Request.NewWindow)
	default:
		inputs["width"] = itoaDefault(job.Request.Width, 1200)
		inputs["height"] = itoaDefault(job.Request.Height, 780)
		inputs["new_window"] = FormatBool(job.Request.NewWindow)
		inputs["hide_title_bar"] = FormatBool(job.Request.HideTitleBar)
		inputs["multi_arch"] = FormatBool(job.Request.MultiArch)
		inputs["targets"] = strDefault(job.Request.Targets, spec.DefaultTargets)
	}

	_ = o.Store.SaveRemote(o.JobID, remote)
	_ = o.Store.SaveStatus(o.JobID, common.Status{State: common.StateRunning, Message: "dispatching workflow"})

	started := time.Now().UTC()
	o.log(fmt.Sprintf("触发 workflow %s @ %s", workflow, ref))
	if err := client.DispatchWorkflow(ctx, workflow, ref, inputs); err != nil {
		_ = fail(o, fmt.Errorf("触发 workflow 失败: %w", err))
		return err
	}

	// Wait for run to appear
	var run *workflowRun
	for i := 0; i < 30; i++ {
		select {
		case <-ctx.Done():
			_ = o.Store.SaveStatus(o.JobID, common.Status{State: common.StateCanceled, Message: "canceled"})
			return ctx.Err()
		case <-time.After(2 * time.Second):
		}
		run, err = client.FindRunAfter(ctx, workflow, started)
		if err == nil && run != nil {
			break
		}
	}
	if run == nil {
		err := fmt.Errorf("触发成功但未找到 run；请到 GitHub Actions 页面查看")
		_ = fail(o, err)
		return err
	}

	remote.RunID = run.ID
	remote.HTMLURL = run.HTMLURL
	remote.ArtifactName = job.Request.Name + spec.ArtifactSuffix
	_ = o.Store.SaveRemote(o.JobID, remote)
	o.log(fmt.Sprintf("已关联 run #%d %s", run.ID, run.HTMLURL))

	// Poll until complete
	for {
		select {
		case <-ctx.Done():
			_ = o.Store.SaveStatus(o.JobID, common.Status{State: common.StateCanceled, Message: "canceled"})
			return ctx.Err()
		case <-time.After(8 * time.Second):
		}

		run, err = client.GetRun(ctx, remote.RunID)
		if err != nil {
			o.log("查询 run 状态失败: " + err.Error())
			continue
		}
		msg := "status=" + run.Status
		if run.Conclusion != "" {
			msg += " conclusion=" + run.Conclusion
		}
		_ = o.Store.SaveStatus(o.JobID, common.Status{State: common.StateRunning, Message: msg})
		o.log("构建中: " + msg)

		if run.Status == "completed" {
			if run.Conclusion != "success" {
				err := fmt.Errorf("云端构建失败: %s（详见 %s）", run.Conclusion, run.HTMLURL)
				_ = fail(o, err)
				return err
			}
			break
		}
	}

	o.log("构建成功，下载 Artifact…")
	_ = o.Store.SaveStatus(o.JobID, common.Status{State: common.StateRunning, Message: "downloading artifact"})

	arts, err := client.ListArtifacts(ctx, remote.RunID)
	if err != nil {
		_ = fail(o, err)
		return err
	}
	var chosen *artifact
	for i := range arts {
		a := &arts[i]
		if a.Expired {
			continue
		}
		if strings.Contains(a.Name, job.Request.Name) || strings.Contains(a.Name, o.JobID) || strings.HasSuffix(a.Name, spec.ArtifactSuffix) {
			chosen = a
			break
		}
	}
	if chosen == nil && len(arts) > 0 {
		chosen = &arts[0]
	}
	if chosen == nil {
		err := fmt.Errorf("未找到 Artifact")
		_ = fail(o, err)
		return err
	}

	zipData, err := client.DownloadArtifactZIP(ctx, chosen.ID)
	if err != nil {
		_ = fail(o, err)
		return err
	}

	if err := os.MkdirAll(o.BuildsDir, 0o755); err != nil {
		_ = fail(o, err)
		return err
	}
	outDir, err := unzipArtifact(zipData, o.BuildsDir, job.Request.Name)
	if err != nil {
		_ = fail(o, err)
		return err
	}
	remote.ArtifactName = chosen.Name
	_ = o.Store.SaveRemote(o.JobID, remote)
	_ = o.Store.SaveStatus(o.JobID, common.Status{
		State:    common.StateSuccess,
		Message:  "success",
		LocalOut: outDir,
	})
	o.log("✓ 已下载到: " + outDir)
	return nil
}

func fail(o CloudJobOptions, err error) error {
	_ = o.Store.SaveStatus(o.JobID, common.Status{
		State:   common.StateFailed,
		Message: err.Error(),
	})
	o.log("✗ " + err.Error())
	return err
}

type pakePlatformSpec struct {
	Label          string
	Workflow       string
	IconPrefix     string
	ArtifactSuffix string
	DefaultTargets string
}

func specFor(p common.Platform) pakePlatformSpec {
	switch p {
	case common.PlatformWindows:
		return pakePlatformSpec{
			Label:          "Windows",
			Workflow:       "build-windows.yml",
			IconPrefix:     "ci-assets/windows",
			ArtifactSuffix: "-Windows",
			DefaultTargets: "exe",
		}
	case common.PlatformAndroid:
		return pakePlatformSpec{
			Label:          "Android",
			Workflow:       "build-android.yml",
			IconPrefix:     "ci-assets/android",
			ArtifactSuffix: "-Android",
			DefaultTargets: "apk",
		}
	default:
		return pakePlatformSpec{
			Label:          "macOS",
			Workflow:       "build-macos.yml",
			IconPrefix:     "ci-assets/macos",
			ArtifactSuffix: "-macOS",
			DefaultTargets: "dmg",
		}
	}
}

func itoaDefault(v, def int) string {
	if v <= 0 {
		v = def
	}
	return fmt.Sprintf("%d", v)
}

func strDefault(v, def string) string {
	v = strings.TrimSpace(v)
	if v == "" {
		return def
	}
	return v
}

func unzipArtifact(data []byte, destDir, nameHint string) (string, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", err
	}
	target := filepath.Join(destDir, sanitizeFile(nameHint)+"-"+time.Now().Format("20060102-150405"))
	if err := os.MkdirAll(target, 0o755); err != nil {
		return "", err
	}
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := filepath.Base(f.Name)
		if base == "" || base == "." || base == ".." {
			continue
		}
		outPath := filepath.Join(target, base)
		if err := extractZipFile(f, outPath); err != nil {
			return "", err
		}
	}
	return target, nil
}

func extractZipFile(f *zip.File, dest string) error {
	rc, err := f.Open()
	if err != nil {
		return err
	}
	defer rc.Close()
	w, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer w.Close()
	_, err = io.Copy(w, io.LimitReader(rc, 200<<20))
	return err
}

func sanitizeFile(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "app"
	}
	repl := strings.NewReplacer(`/`, "-", `\`, "-", `:`, "-", `*`, "-", `?`, "-", `"`, "-", `<`, "-", `>`, "-", `|`, "-")
	return repl.Replace(name)
}

// FetchURL is a small helper for future use.
func FetchURL(ctx context.Context, rawURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("GET %s: %d", rawURL, res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 8<<20))
}
