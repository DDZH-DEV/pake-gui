package github

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

const (
	deviceCodeURL  = "https://github.com/login/device/code"
	accessTokenURL = "https://github.com/login/oauth/access_token"
	defaultScope   = "repo workflow read:user"
)

// DeviceStart is returned to the UI when Device Flow begins.
type DeviceStart struct {
	UserCode                string `json:"userCode"`
	VerificationURI         string `json:"verificationUri"`
	VerificationURIComplete string `json:"verificationUriComplete"`
	ExpiresIn               int    `json:"expiresIn"`
	Interval                int    `json:"interval"`
}

type deviceCodeResponse struct {
	DeviceCode              string `json:"device_code"`
	UserCode                string `json:"user_code"`
	VerificationURI         string `json:"verification_uri"`
	VerificationURIComplete string `json:"verification_uri_complete"`
	ExpiresIn               int    `json:"expires_in"`
	Interval                int    `json:"interval"`
	Error                   string `json:"error"`
	ErrorDescription        string `json:"error_description"`
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	Scope            string `json:"scope"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
	Interval         int    `json:"interval"`
}

// AuthSession tracks an in-progress Device Flow on the local server.
type AuthSession struct {
	mu          sync.Mutex
	Pending     bool
	UserCode    string
	VerifyURL   string
	VerifyFull  string
	ExpiresAt   time.Time
	Done        bool
	OK          bool
	Error       string
	Login       string
	cancel      context.CancelFunc
	deviceCode  string
	clientID    string
	interval    time.Duration
}

var globalAuth = &AuthSession{}

func CurrentAuthSession() *AuthSession { return globalAuth }

func (a *AuthSession) Snapshot() map[string]any {
	a.mu.Lock()
	defer a.mu.Unlock()
	return map[string]any{
		"pending":                   a.Pending,
		"done":                      a.Done,
		"ok":                        a.OK,
		"error":                     a.Error,
		"userCode":                  a.UserCode,
		"verificationUri":           a.VerifyURL,
		"verificationUriComplete":   a.VerifyFull,
		"expiresAt":                 a.ExpiresAt.Format(time.RFC3339),
		"login":                     a.Login,
	}
}

func (a *AuthSession) Cancel() {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
	a.Pending = false
}

// StartDeviceFlow begins GitHub Device Authorization Grant.
func StartDeviceFlow(ctx context.Context, dataDir, clientID string) (*DeviceStart, error) {
	clientID = strings.TrimSpace(clientID)
	if clientID == "" {
		return nil, fmt.Errorf("请先填写 GitHub OAuth App 的 Client ID（开发者设置里启用 Device Flow）")
	}

	form := url.Values{}
	form.Set("client_id", clientID)
	form.Set("scope", defaultScope)

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, deviceCodeURL, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("User-Agent", "pake-gui")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	var dc deviceCodeResponse
	if err := json.Unmarshal(body, &dc); err != nil {
		return nil, fmt.Errorf("解析 device code 失败: %w (%s)", err, strings.TrimSpace(string(body)))
	}
	if dc.Error != "" {
		return nil, fmt.Errorf("%s: %s", dc.Error, dc.ErrorDescription)
	}
	if dc.DeviceCode == "" || dc.UserCode == "" {
		return nil, fmt.Errorf("GitHub 未返回 device_code（请确认 OAuth App 已启用 Device Flow）: %s", strings.TrimSpace(string(body)))
	}
	if dc.Interval <= 0 {
		dc.Interval = 5
	}
	if dc.VerificationURI == "" {
		dc.VerificationURI = "https://github.com/login/device"
	}

	sess := globalAuth
	sess.Cancel()

	pollCtx, cancel := context.WithCancel(context.Background())
	sess.mu.Lock()
	sess.Pending = true
	sess.Done = false
	sess.OK = false
	sess.Error = ""
	sess.Login = ""
	sess.UserCode = dc.UserCode
	sess.VerifyURL = dc.VerificationURI
	sess.VerifyFull = dc.VerificationURIComplete
	sess.ExpiresAt = time.Now().Add(time.Duration(dc.ExpiresIn) * time.Second)
	sess.cancel = cancel
	sess.deviceCode = dc.DeviceCode
	sess.clientID = clientID
	sess.interval = time.Duration(dc.Interval) * time.Second
	sess.mu.Unlock()

	go pollDeviceToken(pollCtx, dataDir, clientID, dc.DeviceCode, time.Duration(dc.Interval)*time.Second, time.Duration(dc.ExpiresIn)*time.Second)

	return &DeviceStart{
		UserCode:                dc.UserCode,
		VerificationURI:         dc.VerificationURI,
		VerificationURIComplete: dc.VerificationURIComplete,
		ExpiresIn:               dc.ExpiresIn,
		Interval:                dc.Interval,
	}, nil
}

func pollDeviceToken(ctx context.Context, dataDir, clientID, deviceCode string, interval, expires time.Duration) {
	deadline := time.Now().Add(expires)
	if interval < 5*time.Second {
		interval = 5 * time.Second
	}
	for {
		if time.Now().After(deadline) {
			finishAuth(false, "", "授权已超时，请重试")
			return
		}
		select {
		case <-ctx.Done():
			finishAuth(false, "", "授权已取消")
			return
		case <-time.After(interval):
		}

		form := url.Values{}
		form.Set("client_id", clientID)
		form.Set("device_code", deviceCode)
		form.Set("grant_type", "urn:ietf:params:oauth:grant-type:device_code")

		req, err := http.NewRequestWithContext(ctx, http.MethodPost, accessTokenURL, strings.NewReader(form.Encode()))
		if err != nil {
			finishAuth(false, "", err.Error())
			return
		}
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		req.Header.Set("Accept", "application/json")
		req.Header.Set("User-Agent", "pake-gui")

		res, err := http.DefaultClient.Do(req)
		if err != nil {
			continue
		}
		body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
		res.Body.Close()

		var tr tokenResponse
		if err := json.Unmarshal(body, &tr); err != nil {
			continue
		}
		switch tr.Error {
		case "":
			if tr.AccessToken == "" {
				continue
			}
			login, err := fetchLogin(ctx, tr.AccessToken)
			if err != nil {
				login = ""
			}
			if err := applyOAuthToken(dataDir, tr.AccessToken, login); err != nil {
				finishAuth(false, login, err.Error())
				return
			}
			finishAuth(true, login, "")
			return
		case "authorization_pending":
			continue
		case "slow_down":
			if tr.Interval > 0 {
				interval = time.Duration(tr.Interval) * time.Second
			} else {
				interval += 5 * time.Second
			}
			continue
		case "expired_token", "access_denied":
			msg := tr.Error
			if tr.ErrorDescription != "" {
				msg = tr.ErrorDescription
			}
			finishAuth(false, "", msg)
			return
		default:
			msg := tr.Error
			if tr.ErrorDescription != "" {
				msg = tr.Error + ": " + tr.ErrorDescription
			}
			finishAuth(false, "", msg)
			return
		}
	}
}

func finishAuth(ok bool, login, errMsg string) {
	a := globalAuth
	a.mu.Lock()
	defer a.mu.Unlock()
	a.Pending = false
	a.Done = true
	a.OK = ok
	a.Login = login
	a.Error = errMsg
	if a.cancel != nil {
		a.cancel()
		a.cancel = nil
	}
}

func fetchLogin(ctx context.Context, token string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user", nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "pake-gui")
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("user api: %s", strings.TrimSpace(string(body)))
	}
	var u struct {
		Login string `json:"login"`
	}
	if err := json.Unmarshal(body, &u); err != nil {
		return "", err
	}
	return u.Login, nil
}

func applyOAuthToken(dataDir, token, login string) error {
	st, _ := LoadSettings(dataDir)
	st.Token = token
	st.Login = login
	if strings.TrimSpace(st.Owner) == "" {
		st.Owner = DefaultOwner
	}
	if strings.TrimSpace(st.Repo) == "" {
		st.Repo = DefaultRepo
	}
	if strings.TrimSpace(st.Workflow) == "" {
		st.Workflow = "build-macos.yml"
	}
	return SaveSettings(dataDir, st)
}

// Logout clears the stored access token.
func Logout(dataDir string) error {
	globalAuth.Cancel()
	st, err := LoadSettings(dataDir)
	if err != nil {
		return err
	}
	st.Token = ""
	st.Login = ""
	return SaveSettingsAllowEmptyToken(dataDir, st)
}
