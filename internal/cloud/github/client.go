package github

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path"
	"strconv"
	"strings"
	"time"
)

const apiBase = "https://api.github.com"

// Client talks to the GitHub REST API.
type Client struct {
	Owner  string
	Repo   string
	Token  string
	HTTP   *http.Client
	APIURL string // override for tests; empty = api.github.com
}

func NewClient(s Settings) *Client {
	return &Client{
		Owner: s.Owner,
		Repo:  s.Repo,
		Token: s.Token,
		HTTP:  &http.Client{Timeout: 60 * time.Second},
	}
}

func (c *Client) base() string {
	if c.APIURL != "" {
		return strings.TrimRight(c.APIURL, "/")
	}
	return apiBase
}

func (c *Client) do(ctx context.Context, method, p string, body any) (*http.Response, []byte, error) {
	var rdr io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, nil, err
		}
		rdr = bytes.NewReader(b)
	}
	req, err := http.NewRequestWithContext(ctx, method, c.base()+p, rdr)
	if err != nil {
		return nil, nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "pake-gui")
	if c.Token != "" {
		req.Header.Set("Authorization", "Bearer "+c.Token)
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer res.Body.Close()
	data, err := io.ReadAll(io.LimitReader(res.Body, 32<<20))
	if err != nil {
		return res, nil, err
	}
	if res.StatusCode >= 400 {
		msg := strings.TrimSpace(string(data))
		var errBody struct {
			Message string `json:"message"`
		}
		if json.Unmarshal(data, &errBody) == nil && errBody.Message != "" {
			msg = errBody.Message
		}
		return res, data, fmt.Errorf("GitHub API %s %s: %s (%d)", method, p, msg, res.StatusCode)
	}
	return res, data, nil
}

type repoInfo struct {
	DefaultBranch string `json:"default_branch"`
	Permissions   struct {
		Push bool `json:"push"`
	} `json:"permissions"`
}

// Test verifies token can read the repo and Actions are reachable.
func (c *Client) Test(ctx context.Context) (map[string]any, error) {
	_, data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s", c.Owner, c.Repo), nil)
	if err != nil {
		return nil, err
	}
	var info repoInfo
	_ = json.Unmarshal(data, &info)

	_, _, err = c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/actions/workflows", c.Owner, c.Repo), nil)
	if err != nil {
		return nil, fmt.Errorf("无法访问 Actions（检查 workflow 权限）: %w", err)
	}

	out := map[string]any{
		"ok":            true,
		"defaultBranch": info.DefaultBranch,
		"canPush":       info.Permissions.Push,
		"owner":         c.Owner,
		"repo":          c.Repo,
	}

	// Prefer checking the configured workflow file (404 = not registered yet).
	var missing []string
	for _, workflow := range []string{DefaultWorkflow, "build-windows.yml"} {
		if _, _, e := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s/actions/workflows/%s",
			c.Owner, c.Repo, url.PathEscape(workflow)), nil); e != nil {
			missing = append(missing, workflow)
		}
	}
	if len(missing) == 0 {
		out["workflowRegistered"] = true
	} else {
		out["workflowRegistered"] = false
		out["workflowHint"] = fmt.Sprintf(
			"%s 尚未被 GitHub Actions 注册。请 push 对应 workflow 文件到默认分支后再试。",
			strings.Join(missing, ", "))
	}

	return out, nil
}

func (c *Client) DefaultBranch(ctx context.Context) (string, error) {
	_, data, err := c.do(ctx, http.MethodGet, fmt.Sprintf("/repos/%s/%s", c.Owner, c.Repo), nil)
	if err != nil {
		return "", err
	}
	var info repoInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return "", err
	}
	if info.DefaultBranch == "" {
		return "main", nil
	}
	return info.DefaultBranch, nil
}

type contentMeta struct {
	SHA string `json:"sha"`
}

// UploadContent creates or updates a file; returns raw.githubusercontent.com URL.
func (c *Client) UploadContent(ctx context.Context, ref, remotePath string, content []byte, message string) (rawURL string, err error) {
	remotePath = strings.TrimPrefix(remotePath, "/")
	apiPath := fmt.Sprintf("/repos/%s/%s/contents/%s", c.Owner, c.Repo, encodeContentPath(remotePath))

	var sha string
	q := apiPath + "?ref=" + url.QueryEscape(ref)
	if _, data, e := c.do(ctx, http.MethodGet, q, nil); e == nil {
		var meta contentMeta
		if json.Unmarshal(data, &meta) == nil {
			sha = meta.SHA
		}
	}

	body := map[string]any{
		"message": message,
		"content": base64.StdEncoding.EncodeToString(content),
		"branch":  ref,
	}
	if sha != "" {
		body["sha"] = sha
	}
	_, _, err = c.do(ctx, http.MethodPut, apiPath, body)
	if err != nil {
		return "", err
	}
	rawURL = fmt.Sprintf("https://raw.githubusercontent.com/%s/%s/%s/%s",
		c.Owner, c.Repo, ref, remotePath)
	return rawURL, nil
}

func encodeContentPath(p string) string {
	parts := strings.Split(p, "/")
	for i, part := range parts {
		parts[i] = url.PathEscape(part)
	}
	return strings.Join(parts, "/")
}

// DispatchWorkflow triggers workflow_dispatch.
func (c *Client) DispatchWorkflow(ctx context.Context, workflowFile, ref string, inputs map[string]string) error {
	body := map[string]any{
		"ref":    ref,
		"inputs": inputs,
	}
	p := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/dispatches",
		c.Owner, c.Repo, url.PathEscape(workflowFile))
	_, _, err := c.do(ctx, http.MethodPost, p, body)
	if err != nil && strings.Contains(err.Error(), "(404)") {
		return fmt.Errorf("%w — 仓库里虽有该文件，但 GitHub 尚未将其注册为可调度 workflow（Actions 列表里看不到）。请确保已 push 到默认分支，并至少因 push/手动运行注册过一次；本仓库可 push `.github/workflows/%s` 触发自注册", err, workflowFile)
	}
	return err
}

type workflowRun struct {
	ID         int64     `json:"id"`
	Status     string    `json:"status"`
	Conclusion string    `json:"conclusion"`
	HTMLURL    string    `json:"html_url"`
	CreatedAt  time.Time `json:"created_at"`
	Name       string    `json:"name"`
	DisplayTitle string  `json:"display_title"`
	Path       string    `json:"path"`
}

type runsResponse struct {
	WorkflowRuns []workflowRun `json:"workflow_runs"`
}

// FindRunAfter finds the newest run of a workflow created at/after since.
func (c *Client) FindRunAfter(ctx context.Context, workflowFile string, since time.Time) (*workflowRun, error) {
	p := fmt.Sprintf("/repos/%s/%s/actions/workflows/%s/runs?event=workflow_dispatch&per_page=10",
		c.Owner, c.Repo, url.PathEscape(workflowFile))
	_, data, err := c.do(ctx, http.MethodGet, p, nil)
	if err != nil {
		return nil, err
	}
	var rr runsResponse
	if err := json.Unmarshal(data, &rr); err != nil {
		return nil, err
	}
	since = since.Add(-15 * time.Second) // clock skew
	for i := range rr.WorkflowRuns {
		run := &rr.WorkflowRuns[i]
		if run.CreatedAt.After(since) || run.CreatedAt.Equal(since) {
			return run, nil
		}
	}
	if len(rr.WorkflowRuns) > 0 {
		// fallback: newest run
		return &rr.WorkflowRuns[0], nil
	}
	return nil, fmt.Errorf("尚未找到 workflow run，请稍后重试")
}

func (c *Client) GetRun(ctx context.Context, runID int64) (*workflowRun, error) {
	p := fmt.Sprintf("/repos/%s/%s/actions/runs/%d", c.Owner, c.Repo, runID)
	_, data, err := c.do(ctx, http.MethodGet, p, nil)
	if err != nil {
		return nil, err
	}
	var run workflowRun
	if err := json.Unmarshal(data, &run); err != nil {
		return nil, err
	}
	return &run, nil
}

type artifact struct {
	ID                 int64  `json:"id"`
	Name               string `json:"name"`
	ArchiveDownloadURL string `json:"archive_download_url"`
	Expired            bool   `json:"expired"`
}

type artifactsResponse struct {
	Artifacts []artifact `json:"artifacts"`
}

func (c *Client) ListArtifacts(ctx context.Context, runID int64) ([]artifact, error) {
	p := fmt.Sprintf("/repos/%s/%s/actions/runs/%d/artifacts", c.Owner, c.Repo, runID)
	_, data, err := c.do(ctx, http.MethodGet, p, nil)
	if err != nil {
		return nil, err
	}
	var ar artifactsResponse
	if err := json.Unmarshal(data, &ar); err != nil {
		return nil, err
	}
	return ar.Artifacts, nil
}

// DownloadArtifactZIP downloads the artifact archive bytes.
func (c *Client) DownloadArtifactZIP(ctx context.Context, artifactID int64) ([]byte, error) {
	p := fmt.Sprintf("/repos/%s/%s/actions/artifacts/%d/zip", c.Owner, c.Repo, artifactID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.base()+p, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	req.Header.Set("User-Agent", "pake-gui")

	res, err := c.HTTP.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		b, _ := io.ReadAll(io.LimitReader(res.Body, 4096))
		return nil, fmt.Errorf("下载 artifact 失败: %s (%d)", strings.TrimSpace(string(b)), res.StatusCode)
	}
	return io.ReadAll(io.LimitReader(res.Body, 200<<20))
}

func FormatBool(v bool) string {
	return strconv.FormatBool(v)
}

func BasenameURL(u string) string {
	return path.Base(u)
}
