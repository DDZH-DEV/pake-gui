package common

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

// Store manages data/cloud-jobs/{id}/ folders.
type Store struct {
	Root string // typically filepath.Join(dataDir, "cloud-jobs")
}

func NewStore(dataDir string) *Store {
	return &Store{Root: filepath.Join(dataDir, "cloud-jobs")}
}

func (s *Store) EnsureRoot() error {
	return os.MkdirAll(s.Root, 0o755)
}

func NewJobID() string {
	var b [4]byte
	_, _ = rand.Read(b[:])
	return time.Now().Format("20060102-150405") + "-" + hex.EncodeToString(b[:])
}

func (s *Store) JobDir(id string) string {
	return filepath.Join(s.Root, sanitizeID(id))
}

func sanitizeID(id string) string {
	id = filepath.Base(strings.TrimSpace(id))
	id = strings.ReplaceAll(id, "..", "")
	return id
}

func (s *Store) Create(req Request) (*Job, error) {
	if err := s.EnsureRoot(); err != nil {
		return nil, err
	}
	id := NewJobID()
	dir := s.JobDir(id)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	now := time.Now()
	job := &Job{
		ID:        id,
		CreatedAt: now,
		Request:   req,
		Status: Status{
			State:     StateQueued,
			Message:   "queued",
			UpdatedAt: now,
		},
		Dir: dir,
	}
	if err := s.writeJSON(filepath.Join(dir, "request.json"), req); err != nil {
		return nil, err
	}
	if err := s.SaveStatus(id, job.Status); err != nil {
		return nil, err
	}
	if err := s.SaveRemote(id, Remote{}); err != nil {
		return nil, err
	}
	_ = os.WriteFile(filepath.Join(dir, "created_at.txt"), []byte(now.Format(time.RFC3339)), 0o644)
	return job, nil
}

func (s *Store) SaveStatus(id string, st Status) error {
	st.UpdatedAt = time.Now()
	return s.writeJSON(filepath.Join(s.JobDir(id), "status.json"), st)
}

func (s *Store) SaveRemote(id string, r Remote) error {
	return s.writeJSON(filepath.Join(s.JobDir(id), "remote.json"), r)
}

func (s *Store) AppendLog(id, line string) {
	path := filepath.Join(s.JobDir(id), "logs.txt")
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o644)
	if err != nil {
		return
	}
	defer f.Close()
	_, _ = fmt.Fprintf(f, "[%s] %s\n", time.Now().Format("15:04:05"), line)
}

func (s *Store) Get(id string) (*Job, error) {
	id = sanitizeID(id)
	dir := s.JobDir(id)
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return nil, fmt.Errorf("job not found: %s", id)
	}
	job := &Job{ID: id, Dir: dir}
	_ = s.readJSON(filepath.Join(dir, "request.json"), &job.Request)
	_ = s.readJSON(filepath.Join(dir, "remote.json"), &job.Remote)
	_ = s.readJSON(filepath.Join(dir, "status.json"), &job.Status)
	if b, err := os.ReadFile(filepath.Join(dir, "created_at.txt")); err == nil {
		if t, err := time.Parse(time.RFC3339, strings.TrimSpace(string(b))); err == nil {
			job.CreatedAt = t
		}
	}
	if job.CreatedAt.IsZero() {
		job.CreatedAt = job.Status.UpdatedAt
	}
	return job, nil
}

// Delete removes data/cloud-jobs/{id}/. Does not touch builds/ artifacts.
func (s *Store) Delete(id string) error {
	id = sanitizeID(id)
	if id == "" || id == "." {
		return fmt.Errorf("bad job id")
	}
	dir := s.JobDir(id)
	rel, err := filepath.Rel(s.Root, dir)
	if err != nil || rel == "." || strings.HasPrefix(rel, "..") {
		return fmt.Errorf("invalid job path")
	}
	if st, err := os.Stat(dir); err != nil || !st.IsDir() {
		return fmt.Errorf("job not found: %s", id)
	}
	return os.RemoveAll(dir)
}

func (s *Store) List() ([]Job, error) {
	if err := s.EnsureRoot(); err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(s.Root)
	if err != nil {
		return nil, err
	}
	var out []Job
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		j, err := s.Get(e.Name())
		if err != nil {
			continue
		}
		out = append(out, *j)
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out, nil
}

func (s *Store) writeJSON(path string, v any) error {
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

func (s *Store) readJSON(path string, v any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, v)
}
