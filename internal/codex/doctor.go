package codex

import (
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"

	"github.com/modu-ai/moai-adk/internal/defs"
)

// Status describes the outcome of a Codex readiness check.
type Status string

const (
	StatusOK   Status = "ok"
	StatusWarn Status = "warn"
	StatusFail Status = "fail"
)

const (
	skillPath    = ".codex/skills/moai/SKILL.md"
	workflowsDir = ".codex/skills/moai/workflows"
)

var expectedWorkflowDocs = []string{
	"project.md",
	"plan.md",
	"run.md",
	"sync.md",
	"review.md",
	"clean.md",
	"loop.md",
}

// Check captures the result of a single Codex readiness check.
type Check struct {
	Name    string `json:"name"`
	Status  Status `json:"status"`
	Message string `json:"message"`
	Detail  string `json:"detail,omitempty"`
}

// Summary holds aggregate check counts.
type Summary struct {
	OK   int `json:"ok"`
	Warn int `json:"warn"`
	Fail int `json:"fail"`
}

// Report is the machine-readable Codex readiness payload.
type Report struct {
	Ready   bool    `json:"ready"`
	Summary Summary `json:"summary"`
	Checks  []Check `json:"checks"`
}

// Inspector evaluates whether a project is ready for the Codex workflow pack.
type Inspector struct {
	LookPath func(string) (string, error)
}

// NewInspector returns an Inspector wired to the real environment.
func NewInspector() Inspector {
	return Inspector{LookPath: exec.LookPath}
}

// Inspect evaluates Codex readiness for the given project root.
func (i Inspector) Inspect(root string, verbose bool) Report {
	checks := []Check{
		i.checkGit(verbose),
		checkMoAIConfig(root, verbose),
		checkCodexSkill(root, verbose),
		checkCodexWorkflows(root, verbose),
	}

	report := Report{Checks: checks}
	for _, check := range checks {
		switch check.Status {
		case StatusOK:
			report.Summary.OK++
		case StatusWarn:
			report.Summary.Warn++
		case StatusFail:
			report.Summary.Fail++
		}
	}
	report.Ready = report.Summary.Fail == 0
	return report
}

func (i Inspector) checkGit(verbose bool) Check {
	check := Check{Name: "Git"}
	lookPath := i.LookPath
	if lookPath == nil {
		lookPath = exec.LookPath
	}

	gitPath, err := lookPath("git")
	if err != nil {
		check.Status = StatusWarn
		check.Message = "git not found in PATH"
		check.Detail = "Install git before relying on git-based MoAI workflows like sync, review, or clean."
		return check
	}

	check.Status = StatusOK
	check.Message = "git available"
	if verbose {
		check.Detail = "path: " + gitPath
	}
	return check
}

func checkMoAIConfig(root string, verbose bool) Check {
	check := Check{Name: "MoAI Config"}

	moaiDir := filepath.Join(root, defs.MoAIDir)
	info, err := os.Stat(moaiDir)
	if err != nil || !info.IsDir() {
		check.Status = StatusFail
		check.Message = ".moai/ directory not found"
		check.Detail = "Run 'moai init .' in the project root before using the Codex workflow pack."
		return check
	}

	configDir := filepath.Join(moaiDir, defs.SectionsSubdir)
	if info, err := os.Stat(configDir); err != nil || !info.IsDir() {
		check.Status = StatusFail
		check.Message = ".moai/config/sections/ not found"
		check.Detail = "Run 'moai init .' or repair the project configuration before using Codex workflows."
		return check
	}

	check.Status = StatusOK
	check.Message = "shared MoAI project state found"
	if verbose {
		check.Detail = "path: " + moaiDir
	}
	return check
}

func checkCodexSkill(root string, verbose bool) Check {
	check := Check{Name: "Codex Skill"}
	path := filepath.Join(root, skillPath)
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		check.Status = StatusFail
		check.Message = ".codex/skills/moai/SKILL.md not found"
		check.Detail = "Run 'moai update --templates-only' to deploy or refresh the Codex workflow pack."
		return check
	}

	check.Status = StatusOK
	check.Message = "$moai Codex entrypoint found"
	if verbose {
		check.Detail = "path: " + path
	}
	return check
}

func checkCodexWorkflows(root string, verbose bool) Check {
	check := Check{Name: "Codex Workflows"}
	dir := filepath.Join(root, workflowsDir)

	entries, err := os.ReadDir(dir)
	if err != nil {
		check.Status = StatusFail
		check.Message = ".codex/skills/moai/workflows/ not found"
		check.Detail = "Run 'moai update --templates-only' to restore the Codex workflow docs."
		return check
	}

	found := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		found = append(found, entry.Name())
	}
	slices.Sort(found)

	var missing []string
	for _, name := range expectedWorkflowDocs {
		if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
			missing = append(missing, name)
		}
	}

	if len(missing) > 0 {
		check.Status = StatusFail
		check.Message = "workflow pack is incomplete"
		check.Detail = "Missing: " + strings.Join(missing, ", ")
		return check
	}

	check.Status = StatusOK
	check.Message = "workflow pack found"
	if verbose {
		check.Detail = "files: " + strings.Join(found, ", ")
	}
	return check
}
