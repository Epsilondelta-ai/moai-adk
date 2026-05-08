package taskstore

import (
	"path/filepath"
	"testing"
	"time"
)

func TestStoreCreateListGetUpdate(t *testing.T) {
	store := NewAtPath(filepath.Join(t.TempDir(), ".moai", "tasks", "tasks.json"))
	fixed := time.Date(2026, 5, 8, 0, 0, 0, 0, time.UTC)
	store.now = func() time.Time { return fixed }

	task, err := store.Create("Implement Pi bridge", "Create bridge foundation", map[string]any{"runtime": "pi"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.ID == "" || task.Status != TaskStatusPending || task.Version != 1 {
		t.Fatalf("unexpected task: %#v", task)
	}

	listed, err := store.List()
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || listed[0].ID != task.ID {
		t.Fatalf("List() = %#v", listed)
	}

	got, err := store.Get(task.ID)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if got.Subject != "Implement Pi bridge" {
		t.Fatalf("Get().Subject = %q", got.Subject)
	}

	status := TaskStatusInProgress
	updated, err := store.Update(task.ID, Patch{Status: &status})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Status != TaskStatusInProgress || updated.Version != 2 {
		t.Fatalf("unexpected updated task: %#v", updated)
	}
}

func TestStoreCreateRequiresSubject(t *testing.T) {
	store := NewAtPath(filepath.Join(t.TempDir(), "tasks.json"))
	if _, err := store.Create(" ", "", nil); err == nil {
		t.Fatal("Create() expected error for empty subject")
	}
}
