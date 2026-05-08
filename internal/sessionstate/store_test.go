package sessionstate

import "testing"

func TestStoreSaveLoad(t *testing.T) {
	store := New(t.TempDir())
	state := &State{SessionFile: "s.jsonl", LeafID: "leaf", LastEvent: "input", Data: map[string]any{"x": "y"}}
	if err := store.Save(state); err != nil {
		t.Fatalf("Save() error = %v", err)
	}
	loaded, err := store.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}
	if loaded.SessionFile != "s.jsonl" || loaded.Data["x"] != "y" {
		t.Fatalf("unexpected state: %#v", loaded)
	}
}
