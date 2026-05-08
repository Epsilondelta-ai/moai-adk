package kernel

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

func (k *Kernel) executeDesign(req CommandRequest) (*CommandResult, error) {
	dir := filepath.Join(req.CWD, ".moai", "design", "pi")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	path := filepath.Join(dir, "design-session.md")
	content := fmt.Sprintf(`# Pi Design Session

## Request

%s

## Runtime

- host: pi
- image_input: detected by Pi UI state when available
- created_at: %s
`, req.Args, time.Now().UTC().Format(time.RFC3339))
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return nil, err
	}
	return &CommandResult{
		Command:   "design",
		OK:        true,
		Messages:  []Message{{Level: "info", Text: "Prepared Pi design workflow"}},
		Artifacts: []Artifact{{Type: "design", Path: path, Name: "design-session"}},
		UIState:   map[string]any{"phase": "design", "qualityStatus": "pending"},
		Data:      map[string]any{"designSession": path},
	}, nil
}
