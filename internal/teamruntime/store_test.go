package teamruntime

import "testing"

func TestStoreCreateStatusMessageDelete(t *testing.T) {
	store := NewStore(t.TempDir())
	state, err := store.Create([]Task{{Role: "implementer", Agent: "expert-backend", Task: "build", OwnedPatterns: []string{"internal/**"}}}, DefaultProfiles())
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if state.ID == "" || state.Members[0].Isolation != "worktree" || len(state.Events) < 2 {
		t.Fatalf("unexpected state: %#v", state)
	}
	loaded, err := store.Get(state.ID)
	if err != nil || loaded.ID != state.ID {
		t.Fatalf("Get() = %#v, %v", loaded, err)
	}
	loaded.Messages = append(loaded.Messages, TeamMessage{From: "orchestrator", Text: "hello", At: "now"})
	if err := store.Save(loaded); err != nil {
		t.Fatal(err)
	}
	if err := store.ReviewCompletion(loaded, "implementer", true, "ok"); err != nil {
		t.Fatal(err)
	}
	latest, err := store.Latest()
	if err != nil || len(latest.Messages) != 1 || latest.Members[0].Status != "accepted" {
		t.Fatalf("Latest() = %#v, %v", latest, err)
	}
	if err := store.Delete(state.ID); err != nil {
		t.Fatal(err)
	}
}

func TestStoreCreateRejectsOwnershipConflicts(t *testing.T) {
	_, err := NewStore(t.TempDir()).Create([]Task{
		{Role: "implementer", Agent: "expert-backend", Task: "a", OwnedPatterns: []string{"src/**"}},
		{Role: "tester", Agent: "expert-testing", Task: "b", OwnedPatterns: []string{"src/**"}},
	}, DefaultProfiles())
	if err == nil {
		t.Fatal("expected ownership conflict")
	}
}
