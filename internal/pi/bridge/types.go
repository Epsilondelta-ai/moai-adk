// Package bridge defines the stable JSON bridge used by the Pi extension.
package bridge

import "time"

// Request is the envelope sent by the TypeScript Pi adapter to the MoAI Go core.
type Request struct {
	Version string         `json:"version,omitempty"`
	Kind    string         `json:"kind"`
	CWD     string         `json:"cwd,omitempty"`
	Session *SessionInfo   `json:"session,omitempty"`
	Payload map[string]any `json:"payload,omitempty"`
}

// SessionInfo contains Pi session metadata that is useful for branch-aware state.
type SessionInfo struct {
	File   string `json:"file,omitempty"`
	LeafID string `json:"leafId,omitempty"`
	Branch string `json:"branch,omitempty"`
	Mode   string `json:"mode,omitempty"`
}

// Response is the stable result envelope returned to the Pi adapter.
type Response struct {
	OK          bool           `json:"ok"`
	Kind        string         `json:"kind,omitempty"`
	Message     string         `json:"message,omitempty"`
	Data        map[string]any `json:"data,omitempty"`
	Error       *BridgeError   `json:"error,omitempty"`
	GeneratedAt string         `json:"generatedAt"`
}

// BridgeError is a structured error suitable for Pi tool details and rendering.
type BridgeError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Success returns a successful bridge response.
func Success(kind, message string, data map[string]any) Response {
	return Response{
		OK:          true,
		Kind:        kind,
		Message:     message,
		Data:        data,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
}

// Failure returns a failed bridge response.
func Failure(kind, code, message string) Response {
	return Response{
		OK:          false,
		Kind:        kind,
		Error:       &BridgeError{Code: code, Message: message},
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
	}
}
