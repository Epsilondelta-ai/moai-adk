package bridge

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestHandleToolSurfaceSpecMemoryContextAndMX(t *testing.T) {
	cwd := t.TempDir()
	h := NewHandler()
	created := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_spec_create", "payload": map[string]any{"description": "build tool surface"}}})
	if !created.OK {
		t.Fatalf("spec create failed: %#v", created)
	}
	specID := created.Data["specId"].(string)
	listed := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_spec_list"}})
	if !listed.OK {
		t.Fatalf("spec list failed: %#v", listed)
	}
	updated := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_spec_update", "payload": map[string]any{"specId": specID, "file": "notes.md", "content": "hello context"}}})
	if !updated.OK {
		t.Fatalf("spec update failed: %#v", updated)
	}
	if _, err := os.Stat(filepath.Join(cwd, ".moai", "specs", specID, "history.log")); err != nil {
		t.Fatalf("spec history missing: %v", err)
	}
	memWrite := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_memory_write", "payload": map[string]any{"scope": "session", "name": "project.md", "content": "remember this"}}})
	if !memWrite.OK {
		t.Fatalf("memory write failed: %#v", memWrite)
	}
	memRead := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_memory_read", "payload": map[string]any{"scope": "session", "name": "project.md"}}})
	if !memRead.OK || memRead.Data["content"].(string) != "remember this" {
		t.Fatalf("memory read failed: %#v", memRead)
	}
	search := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_context_search", "payload": map[string]any{"query": "context"}}})
	if !search.OK || len(search.Data["matches"].([]contextMatch)) == 0 {
		t.Fatalf("context search failed: %#v", search)
	}
	mxFile := filepath.Join(cwd, "src.go")
	if err := os.WriteFile(mxFile, []byte("package main\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mx := h.Handle(context.Background(), Request{Kind: "tool", CWD: cwd, Payload: map[string]any{"tool": "moai_mx_update", "payload": map[string]any{"path": "src.go", "annotation": "// @MX:NOTE generated", "mode": "insert", "line": float64(1)}}})
	if !mx.OK {
		t.Fatalf("mx update failed: %#v", mx)
	}
	if _, err := os.Stat(mxFile); err != nil {
		t.Fatalf("mx file missing: %v", err)
	}
}

func TestHandleConfigSetProtectsUserConfig(t *testing.T) {
	res := NewHandler().Handle(context.Background(), Request{Kind: "tool", CWD: t.TempDir(), Payload: map[string]any{"tool": "moai_config_set", "payload": map[string]any{"section": "user", "content": "secret: true\n"}}})
	if res.OK || res.Error.Code != "protected_path" {
		t.Fatalf("expected protected path failure: %#v", res)
	}
}
