// Package kernel is the shared MoAI Runtime Kernel used by Claude and Pi adapters.
package kernel

// CommandRequest describes a /moai command invocation.
type CommandRequest struct {
	Command string         `json:"command"`
	Args    string         `json:"args,omitempty"`
	CWD     string         `json:"cwd"`
	Session map[string]any `json:"session,omitempty"`
}

// Artifact is a file or runtime artifact produced by a command.
type Artifact struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name,omitempty"`
}

// Message is a user-visible command message.
type Message struct {
	Level string `json:"level"`
	Text  string `json:"text"`
}

// CommandResult is the runtime-neutral result of executing a command.
type CommandResult struct {
	Command   string         `json:"command"`
	OK        bool           `json:"ok"`
	Messages  []Message      `json:"messages"`
	Artifacts []Artifact     `json:"artifacts,omitempty"`
	UIState   map[string]any `json:"uiState,omitempty"`
	Data      map[string]any `json:"data,omitempty"`
}
