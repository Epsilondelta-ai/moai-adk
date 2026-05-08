package bridge

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/modu-ai/moai-adk/internal/taskstore"
)

// UIState is the canonical state rendered by the Pi MoAI footer/widget.
type UIState struct {
	ActiveSpecID        string         `json:"activeSpecId,omitempty"`
	Phase               string         `json:"phase,omitempty"`
	DevelopmentMode     string         `json:"developmentMode,omitempty"`
	HarnessLevel        string         `json:"harnessLevel,omitempty"`
	ProjectName         string         `json:"projectName,omitempty"`
	MoAIVersion         string         `json:"moaiVersion,omitempty"`
	MoAILatestVersion   string         `json:"moaiLatestVersion,omitempty"`
	PiVersion           string         `json:"piVersion,omitempty"`
	SessionStartedAt    int64          `json:"sessionStartedAt,omitempty"`
	ShortWindowPercent  int            `json:"shortWindowPercent"`
	WeeklyWindowPercent int            `json:"weeklyWindowPercent"`
	GitBranch           string         `json:"gitBranch,omitempty"`
	GitAdded            int            `json:"gitAdded"`
	GitModified         int            `json:"gitModified"`
	GitUntracked        int            `json:"gitUntracked"`
	WorktreePath        string         `json:"worktreePath,omitempty"`
	TaskTotal           int            `json:"taskTotal"`
	TaskCompleted       int            `json:"taskCompleted"`
	TaskInProgress      int            `json:"taskInProgress"`
	QualityStatus       string         `json:"qualityStatus,omitempty"`
	LspErrors           int            `json:"lspErrors"`
	MXWarnings          int            `json:"mxWarnings"`
	ClipboardImagePath  string         `json:"clipboardImagePath,omitempty"`
	LastUpdated         string         `json:"lastUpdated"`
	Details             map[string]any `json:"details,omitempty"`
}

func loadUIState(cwd string, payload map[string]any) map[string]any {
	added, modified, untracked := gitStatusCounts(cwd)
	version, latest := moaiVersions(cwd)
	state := UIState{
		ActiveSpecID:        latestSpecID(filepath.Join(cwd, ".moai", "specs")),
		Phase:               stringValue(payload["phase"], inferPhase(payload)),
		DevelopmentMode:     readDevelopmentMode(filepath.Join(cwd, ".moai", "config", "sections", "quality.yaml")),
		HarnessLevel:        readHarnessLevel(filepath.Join(cwd, ".moai", "config", "sections", "harness.yaml")),
		ProjectName:         filepath.Base(cwd),
		MoAIVersion:         version,
		MoAILatestVersion:   latest,
		PiVersion:           stringValue(payload["piVersion"], ""),
		SessionStartedAt:    sessionStartedAt(cwd),
		ShortWindowPercent:  intFromPayload(payload, "shortWindowPercent"),
		WeeklyWindowPercent: intFromPayload(payload, "weeklyWindowPercent"),
		GitBranch:           gitBranch(cwd),
		GitAdded:            added,
		GitModified:         modified,
		GitUntracked:        untracked,
		WorktreePath:        cwd,
		QualityStatus:       "unknown",
		LastUpdated:         time.Now().UTC().Format(time.RFC3339),
		Details:             map[string]any{},
	}
	if specID, ok := payload["activeSpecId"].(string); ok && specID != "" {
		state.ActiveSpecID = specID
	}
	if quality, ok := payload["qualityStatus"].(string); ok && quality != "" {
		state.QualityStatus = quality
	}
	if lspErrors, ok := payload["lspErrors"].(float64); ok {
		state.LspErrors = int(lspErrors)
	}
	if tasks, err := taskstore.New(cwd).List(); err == nil {
		state.TaskTotal = len(tasks)
		for _, task := range tasks {
			switch task.Status {
			case taskstore.TaskStatusCompleted:
				state.TaskCompleted++
			case taskstore.TaskStatusInProgress:
				state.TaskInProgress++
			}
		}
	}
	if mx := scanMX(cwd); mx != nil {
		if counts, ok := mx["counts"].(map[string]int); ok {
			state.MXWarnings = counts["WARN"] + counts["TODO"]
		}
		state.Details["mx"] = mx
	}
	state.ClipboardImagePath = latestClipboardImage()
	state.Details["source"] = "pi-bridge"
	bytes, _ := json.Marshal(state)
	var asMap map[string]any
	_ = json.Unmarshal(bytes, &asMap)
	return asMap
}

func latestSpecID(specDir string) string {
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return ""
	}
	type candidate struct {
		name string
		mod  time.Time
	}
	var candidates []candidate
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), "SPEC-") {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		candidates = append(candidates, candidate{name: entry.Name(), mod: info.ModTime()})
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].mod.After(candidates[j].mod) })
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].name
}

func readDevelopmentMode(path string) string {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "unknown"
	}
	text := string(bytes)
	if strings.Contains(text, "development_mode: tdd") {
		return "tdd"
	}
	if strings.Contains(text, "development_mode: ddd") {
		return "ddd"
	}
	return "unknown"
}

func readHarnessLevel(path string) string {
	bytes, err := os.ReadFile(path)
	if err != nil {
		return "standard"
	}
	for _, line := range strings.Split(string(bytes), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "level:") || strings.HasPrefix(trimmed, "default:") {
			parts := strings.SplitN(trimmed, ":", 2)
			if len(parts) == 2 && strings.TrimSpace(parts[1]) != "" {
				return strings.Trim(strings.TrimSpace(parts[1]), `"'`)
			}
		}
	}
	return "standard"
}

func gitBranch(cwd string) string {
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(out))
}

func gitStatusCounts(cwd string) (added, modified, untracked int) {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return 0, 0, 0
	}
	for _, line := range strings.Split(string(out), "\n") {
		if len(line) < 2 {
			continue
		}
		x, y := line[0], line[1]
		if x == '?' && y == '?' {
			untracked++
			continue
		}
		if x == 'A' || y == 'A' {
			added++
		}
		if x == 'M' || y == 'M' || x == 'D' || y == 'D' || x == 'R' || y == 'R' {
			modified++
		}
	}
	return added, modified, untracked
}

func moaiVersions(cwd string) (current, latest string) {
	bytes, err := os.ReadFile(filepath.Join(cwd, ".moai", "manifest.json"))
	if err != nil {
		return "", ""
	}
	var manifest struct {
		Version       string `json:"version"`
		LatestVersion string `json:"latest_version"`
	}
	if err := json.Unmarshal(bytes, &manifest); err != nil {
		return "", ""
	}
	return manifest.Version, manifest.LatestVersion
}

func latestClipboardImage() string {
	patterns := []string{
		filepath.Join(os.TempDir(), "pi-clipboard-*.png"),
		filepath.Join(os.TempDir(), "pi-clipboard-*.jpg"),
		filepath.Join(os.TempDir(), "pi-clipboard-*.jpeg"),
		filepath.Join(os.TempDir(), "pi-clipboard-*.webp"),
	}
	type candidate struct {
		path string
		mod  time.Time
	}
	var candidates []candidate
	for _, pattern := range patterns {
		matches, _ := filepath.Glob(pattern)
		for _, match := range matches {
			info, err := os.Stat(match)
			if err == nil && !info.IsDir() {
				candidates = append(candidates, candidate{path: match, mod: info.ModTime()})
			}
		}
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].mod.After(candidates[j].mod) })
	if len(candidates) == 0 {
		return ""
	}
	return candidates[0].path
}

func sessionStartedAt(cwd string) int64 {
	path := filepath.Join(cwd, ".moai", "runtime", "pi-session.json")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return time.Now().UTC().UnixMilli()
	}
	var state struct {
		SessionFile string         `json:"sessionFile"`
		Data        map[string]any `json:"data"`
	}
	if err := json.Unmarshal(bytes, &state); err == nil {
		if startedAt := int64FromAny(state.Data["sessionStartedAt"]); startedAt > 0 {
			return startedAt
		}
		if startedAt := sessionStartedAtFromSessionFile(state.SessionFile); startedAt > 0 {
			return startedAt
		}
	}
	return time.Now().UTC().UnixMilli()
}

func intFromPayload(payload map[string]any, key string) int {
	switch value := payload[key].(type) {
	case float64:
		return int(value)
	case int:
		return value
	default:
		return 0
	}
}

func stringValue(value any, fallback string) string {
	if text, ok := value.(string); ok && strings.TrimSpace(text) != "" {
		return strings.TrimSpace(text)
	}
	return fallback
}

func inferPhase(payload map[string]any) string {
	if event, _ := payload["event"].(string); event != "" {
		return event
	}
	return "idle"
}
