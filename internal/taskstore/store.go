package taskstore

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var ErrTaskNotFound = errors.New("task not found")

// Store persists MoAI tasks under .moai/tasks/tasks.json.
type Store struct {
	path string
	now  func() time.Time
}

// New creates a project task store rooted at cwd.
func New(cwd string) *Store {
	return &Store{
		path: filepath.Join(cwd, ".moai", "tasks", "tasks.json"),
		now:  func() time.Time { return time.Now().UTC() },
	}
}

// NewAtPath creates a store at an explicit path. Useful for tests.
func NewAtPath(path string) *Store {
	return &Store{path: path, now: func() time.Time { return time.Now().UTC() }}
}

// Create adds a task.
func (s *Store) Create(subject, description string, metadata map[string]any) (*Task, error) {
	subject = strings.TrimSpace(subject)
	if subject == "" {
		return nil, fmt.Errorf("subject is required")
	}
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	now := s.now()
	task := &Task{
		ID:          newID(),
		Subject:     subject,
		Description: description,
		Status:      TaskStatusPending,
		Metadata:    metadata,
		CreatedAt:   now,
		UpdatedAt:   now,
		Version:     1,
	}
	data.Tasks = append(data.Tasks, task)
	if err := s.save(data); err != nil {
		return nil, err
	}
	return task, nil
}

// Update patches an existing task.
func (s *Store) Update(id string, patch Patch) (*Task, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	for _, task := range data.Tasks {
		if task.ID != id {
			continue
		}
		if patch.Subject != nil {
			task.Subject = strings.TrimSpace(*patch.Subject)
		}
		if patch.Description != nil {
			task.Description = *patch.Description
		}
		if patch.Status != nil {
			task.Status = *patch.Status
		}
		if patch.Metadata != nil {
			task.Metadata = patch.Metadata
		}
		task.UpdatedAt = s.now()
		task.Version++
		if err := s.save(data); err != nil {
			return nil, err
		}
		return task, nil
	}
	return nil, ErrTaskNotFound
}

// Get returns a task by ID.
func (s *Store) Get(id string) (*Task, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	for _, task := range data.Tasks {
		if task.ID == id {
			return task, nil
		}
	}
	return nil, ErrTaskNotFound
}

// List returns tasks sorted by creation time.
func (s *Store) List() ([]*Task, error) {
	data, err := s.load()
	if err != nil {
		return nil, err
	}
	tasks := append([]*Task(nil), data.Tasks...)
	sort.SliceStable(tasks, func(i, j int) bool { return tasks[i].CreatedAt.Before(tasks[j].CreatedAt) })
	return tasks, nil
}

func (s *Store) load() (*FileData, error) {
	bytes, err := os.ReadFile(s.path)
	if errors.Is(err, os.ErrNotExist) {
		return &FileData{Version: 1, Tasks: []*Task{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var data FileData
	if err := json.Unmarshal(bytes, &data); err != nil {
		return nil, err
	}
	if data.Version == 0 {
		data.Version = 1
	}
	if data.Tasks == nil {
		data.Tasks = []*Task{}
	}
	return &data, nil
}

func (s *Store) save(data *FileData) error {
	if err := os.MkdirAll(filepath.Dir(s.path), 0o755); err != nil {
		return err
	}
	bytes, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}
	tmp := s.path + ".tmp"
	if err := os.WriteFile(tmp, append(bytes, '\n'), 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, s.path)
}

func newID() string {
	var b [6]byte
	if _, err := rand.Read(b[:]); err == nil {
		return "TASK-" + strings.ToUpper(hex.EncodeToString(b[:]))
	}
	return fmt.Sprintf("TASK-%d", time.Now().UnixNano())
}
