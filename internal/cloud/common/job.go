// Package common defines cloud build jobs shared by macOS, Android, etc.
package common

import "time"

// Platform identifies the target artifact family.
type Platform string

const (
	PlatformMacOS   Platform = "macos"
	PlatformAndroid Platform = "android"
	PlatformWindows Platform = "windows"
)

// State is the lifecycle of a cloud job on disk under data/cloud-jobs/{id}/.
type State string

const (
	StateQueued   State = "queued"
	StateRunning  State = "running"
	StateSuccess  State = "success"
	StateFailed   State = "failed"
	StateCanceled State = "canceled"
)

// Request is persisted as request.json.
type Request struct {
	URL          string   `json:"url"`
	Name         string   `json:"name"`
	Icon         string   `json:"icon,omitempty"`
	Width        int      `json:"width,omitempty"`
	Height       int      `json:"height,omitempty"`
	AppVersion   string   `json:"appVersion,omitempty"`
	Identifier   string   `json:"identifier,omitempty"`
	HideTitleBar bool     `json:"hideTitleBar,omitempty"`
	MultiArch    bool     `json:"multiArch,omitempty"`
	NewWindow    bool     `json:"newWindow,omitempty"`
	Targets      string   `json:"targets,omitempty"`
	Platform     Platform `json:"platform"`
}

// Remote is persisted as remote.json.
type Remote struct {
	RunID          int64  `json:"runId,omitempty"`
	Workflow       string `json:"workflow,omitempty"`
	Ref            string `json:"ref,omitempty"`
	IconRemotePath string `json:"iconRemotePath,omitempty"`
	IconURL        string `json:"iconUrl,omitempty"`
	HTMLURL        string `json:"htmlUrl,omitempty"`
	ArtifactName   string `json:"artifactName,omitempty"`
}

// Status is persisted as status.json.
type Status struct {
	State     State     `json:"state"`
	Message   string    `json:"message,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"`
	LocalOut  string    `json:"localOut,omitempty"`
}

// Job is the aggregate view returned to the UI.
type Job struct {
	ID        string    `json:"id"`
	CreatedAt time.Time `json:"createdAt"`
	Request   Request   `json:"request"`
	Remote    Remote    `json:"remote"`
	Status    Status    `json:"status"`
	Dir       string    `json:"dir,omitempty"`
}
