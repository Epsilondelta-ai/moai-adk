package kernel

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/modu-ai/moai-adk/internal/taskstore"
)

var specIDRe = regexp.MustCompile(`SPEC-[A-Z][A-Z0-9]*-\d{3}`)

// Kernel executes MoAI workflows independent of the host runtime.
type Kernel struct{}

// New creates a kernel.
func New() *Kernel { return &Kernel{} }

// ExecuteCommand runs a MoAI command.
func (k *Kernel) ExecuteCommand(ctx context.Context, req CommandRequest) (*CommandResult, error) {
	if req.CWD == "" {
		wd, err := os.Getwd()
		if err != nil {
			return nil, err
		}
		req.CWD = wd
	}
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	switch req.Command {
	case "plan", "default":
		return k.executePlan(req)
	case "run":
		return k.executeRun(req)
	case "sync":
		return k.executeSync(req)
	case "review", "coverage", "e2e", "gate":
		return k.executeQualityCommand(req)
	case "design":
		return k.executeDesign(req)
	case "clean", "mx", "project", "db", "feedback", "brain", "fix", "loop", "codemaps":
		return k.executeUtilityCommand(req)
	default:
		return &CommandResult{Command: req.Command, OK: false, Messages: []Message{{Level: "error", Text: "unknown MoAI command: " + req.Command}}}, nil
	}
}

func (k *Kernel) executePlan(req CommandRequest) (*CommandResult, error) {
	request := strings.TrimSpace(req.Args)
	if request == "" {
		return blockerResult("plan", "Plan requires a feature description", "Run /moai plan \"describe the desired behavior\"."), nil
	}

	now := time.Now().UTC()
	date := now.Format("2006-01-02")
	specID := nextSpecID(req.CWD, "PI")
	specDir := filepath.Join(req.CWD, ".moai", "specs", specID)
	if err := os.MkdirAll(specDir, 0o755); err != nil {
		return nil, err
	}

	files := map[string]string{
		"spec.md":           specDocument(specID, request, date),
		"plan.md":           planDocument(specID, request, now),
		"acceptance.md":     acceptanceDocument(specID, request),
		"workflow.json":     workflowGraphJSON(specID, "plan", []string{"manager-spec", "plan-auditor"}, now),
		"delegation.md":     delegationContract(specID, "plan", []string{"manager-spec", "plan-auditor"}),
		"clarifications.md": clarificationLedger(request, now),
		"status.json":       statusJSON(specID, "planned", "plan", now),
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(specDir, name), []byte(content), 0o644); err != nil {
			return nil, err
		}
	}

	_, _ = taskstore.New(req.CWD).Create("Plan "+specID, request, map[string]any{"specId": specID, "phase": "plan"})
	return &CommandResult{
		Command: "plan",
		OK:      true,
		Messages: []Message{
			{Level: "info", Text: "Created canonical SPEC " + specID},
			{Level: "next", Text: "Review spec.md, plan.md, and acceptance.md before /moai run."},
		},
		Artifacts: []Artifact{
			{Type: "spec", Path: specDir, Name: specID},
			{Type: "specification", Path: filepath.Join(specDir, "spec.md")},
			{Type: "plan", Path: filepath.Join(specDir, "plan.md")},
			{Type: "acceptance", Path: filepath.Join(specDir, "acceptance.md")},
			{Type: "workflow", Path: filepath.Join(specDir, "workflow.json")},
			{Type: "delegation", Path: filepath.Join(specDir, "delegation.md")},
		},
		UIState: map[string]any{"activeSpecId": specID, "phase": "plan", "qualityStatus": "pending"},
		Data:    map[string]any{"specId": specID, "specDir": specDir, "canonical": true},
	}, nil
}

func (k *Kernel) executeRun(req CommandRequest) (*CommandResult, error) {
	specID := resolveSpecID(req)
	if specID == "" {
		return blockerResult("run", "No SPEC found for run", "Create one first with /moai plan \"feature description\"."), nil
	}
	specDir := filepath.Join(req.CWD, ".moai", "specs", specID)
	if missing := missingCanonicalSpecFiles(specDir); len(missing) > 0 {
		return blockerResult("run", "SPEC is not canonical: missing "+strings.Join(missing, ", "), "Run /moai plan again or restore the missing files before /moai run."), nil
	}

	now := time.Now().UTC()
	runPlanPath := filepath.Join(specDir, "run-plan.md")
	runPlan := fmt.Sprintf(`# Run Plan: %s

## Phase

run

## Execution Strategy

1. Load and verify spec.md, plan.md, and acceptance.md.
2. Select development mode from project quality configuration.
3. Delegate to manager-ddd or manager-tdd according to quality.yaml.
4. Fan out to expert implementation/testing agents as required by the SPEC.
5. Run quality, LSP, and MX gates before sync.

## Manager Workflow

- primary: manager-ddd or manager-tdd
- support: manager-quality, expert-testing, expert-refactoring
- no-user-prompt: subagents must return blocker reports to orchestrator

## Current Pi Foundation Handoff

- SPEC validated: yes
- Runtime: pi
- Next implementation phase: Agent/TDD-DDD orchestration
- Generated at: %s
`, specID, now.Format(time.RFC3339))
	if err := os.WriteFile(runPlanPath, []byte(runPlan), 0o644); err != nil {
		return nil, err
	}
	progressPath := filepath.Join(specDir, "progress.md")
	progress := fmt.Sprintf("# Progress\n\n- spec: %s\n- phase: run\n- runtime: pi\n- updated_at: %s\n- status: ready-for-agent-orchestration\n", specID, now.Format(time.RFC3339))
	if err := os.WriteFile(progressPath, []byte(progress), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(specDir, "workflow.json"), []byte(workflowGraphJSON(specID, "run", []string{"manager-ddd", "manager-tdd", "expert-testing", "manager-quality"}, now)), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(specDir, "delegation.md"), []byte(delegationContract(specID, "run", []string{"manager-ddd", "manager-tdd", "expert-testing", "manager-quality"})), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(specDir, "status.json"), []byte(statusJSON(specID, "running", "run", now)), 0o644); err != nil {
		return nil, err
	}
	_, _ = taskstore.New(req.CWD).Create("Run "+specID, "Execute implementation workflow", map[string]any{"specId": specID, "phase": "run"})
	return &CommandResult{
		Command: "run",
		OK:      true,
		Messages: []Message{
			{Level: "info", Text: "Validated canonical SPEC " + specID},
			{Level: "next", Text: "Run phase is ready for agent/TDD-DDD orchestration."},
		},
		Artifacts: []Artifact{{Type: "run-plan", Path: runPlanPath}, {Type: "progress", Path: progressPath}, {Type: "workflow", Path: filepath.Join(specDir, "workflow.json")}, {Type: "delegation", Path: filepath.Join(specDir, "delegation.md")}},
		UIState:   map[string]any{"activeSpecId": specID, "phase": "run", "qualityStatus": "pending"},
		Data:      map[string]any{"specId": specID, "specDir": specDir, "validated": true, "next": "agent-orchestration"},
	}, nil
}

func (k *Kernel) executeSync(req CommandRequest) (*CommandResult, error) {
	specID := resolveSpecID(req)
	if specID == "" {
		return blockerResult("sync", "No SPEC found for sync", "Create one with /moai plan, then run it with /moai run."), nil
	}
	specDir := filepath.Join(req.CWD, ".moai", "specs", specID)
	if missing := missingCanonicalSpecFiles(specDir); len(missing) > 0 {
		return blockerResult("sync", "SPEC is not canonical: missing "+strings.Join(missing, ", "), "Restore canonical SPEC files before /moai sync."), nil
	}
	if !pathExists(filepath.Join(specDir, "progress.md")) || !pathExists(filepath.Join(specDir, "run-plan.md")) {
		return blockerResult("sync", "Run artifacts are missing for "+specID, "Execute /moai run "+specID+" before /moai sync."), nil
	}

	now := time.Now().UTC()
	syncPath := filepath.Join(specDir, "sync.md")
	sync := fmt.Sprintf(`# Sync Plan: %s

## Phase

sync

## Actions

1. Verify implementation artifacts and quality results.
2. Delegate documentation updates to manager-docs.
3. Validate and update MX tags.
4. Record remaining follow-ups and final SPEC status.

## Manager Workflow

- primary: manager-docs
- support: manager-quality, expert-refactoring
- no-user-prompt: subagents must return blocker reports to orchestrator

## Current Pi Foundation Handoff

- Run artifacts detected: yes
- Runtime: pi
- Next sync phase: documentation/MX/status orchestration
- Generated at: %s
`, specID, now.Format(time.RFC3339))
	if err := os.WriteFile(syncPath, []byte(sync), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(specDir, "workflow.json"), []byte(workflowGraphJSON(specID, "sync", []string{"manager-docs", "manager-quality"}, now)), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(specDir, "delegation.md"), []byte(delegationContract(specID, "sync", []string{"manager-docs", "manager-quality"})), 0o644); err != nil {
		return nil, err
	}
	if err := os.WriteFile(filepath.Join(specDir, "status.json"), []byte(statusJSON(specID, "synced", "sync", now)), 0o644); err != nil {
		return nil, err
	}
	return &CommandResult{
		Command: "sync",
		OK:      true,
		Messages: []Message{
			{Level: "info", Text: "Validated run artifacts for " + specID},
			{Level: "next", Text: "Sync phase is ready for documentation and MX orchestration."},
		},
		Artifacts: []Artifact{{Type: "sync", Path: syncPath}, {Type: "workflow", Path: filepath.Join(specDir, "workflow.json")}, {Type: "delegation", Path: filepath.Join(specDir, "delegation.md")}},
		UIState:   map[string]any{"activeSpecId": specID, "phase": "sync", "qualityStatus": "synced"},
		Data:      map[string]any{"specId": specID, "specDir": specDir, "validated": true, "next": "documentation-mx-orchestration"},
	}, nil
}

func (k *Kernel) executeQualityCommand(req CommandRequest) (*CommandResult, error) {
	return &CommandResult{Command: req.Command, OK: true, Messages: []Message{{Level: "info", Text: "Quality command dispatched: " + req.Command}}, UIState: map[string]any{"phase": req.Command, "qualityStatus": "pending"}, Data: map[string]any{"command": req.Command}}, nil
}

func (k *Kernel) executeUtilityCommand(req CommandRequest) (*CommandResult, error) {
	return &CommandResult{Command: req.Command, OK: true, Messages: []Message{{Level: "info", Text: "MoAI command dispatched: " + req.Command}}, UIState: map[string]any{"phase": req.Command}, Data: map[string]any{"command": req.Command, "args": req.Args}}, nil
}

func specDocument(specID, request, date string) string {
	return fmt.Sprintf(`---
id: %s
version: "0.1.0"
status: draft
created_at: %s
updated_at: %s
author: moai-pi
priority: Medium
labels: [pi, workflow]
issue_number: null
---

# %s

## HISTORY

- %s: Created from Pi /moai plan.

## WHY

The user requested the following behavior:

> %s

## WHAT

MoAI shall turn the user request into canonical SPEC artifacts that can be executed by the Run and Sync phases.

## REQUIREMENTS

- REQ-001: The MoAI Pi runtime shall create a canonical SPEC directory with spec.md, plan.md, and acceptance.md.
- REQ-002: When the user invokes /moai run for this SPEC, the runtime shall validate canonical SPEC files before preparing implementation orchestration.
- REQ-003: When the user invokes /moai sync for this SPEC, the runtime shall require run artifacts before preparing documentation synchronization.

## EARS REQUIREMENTS

- When /moai plan receives a feature description, the system shall create canonical SPEC artifacts and a manager workflow graph.
- When /moai run receives a canonical SPEC, the system shall prepare DDD/TDD manager delegation and quality gates.
- When /moai sync receives completed run artifacts, the system shall prepare docs/MX/status synchronization delegation.

## ACCEPTANCE CRITERIA

- AC-001: Given a Pi /moai plan request, when planning completes, then the system shall create spec.md, plan.md, and acceptance.md in one SPEC directory.
- AC-002: Given a canonical SPEC exists, when /moai run is invoked, then the system shall validate required SPEC files before writing run artifacts.
- AC-003: Given run artifacts exist, when /moai sync is invoked, then the system shall create sync artifacts and update SPEC status.

## Exclusions (What NOT to Build)

- Do not implement full agent/TDD-DDD execution in this planning phase.
- Do not perform documentation synchronization until /moai sync.
`, specID, date, date, specID, date, request)
}

func planDocument(specID, request string, now time.Time) string {
	return fmt.Sprintf(`# Implementation Plan: %s

## Request

%s

## Milestones

1. Validate canonical SPEC artifact structure.
2. Prepare run orchestration handoff.
3. Prepare sync orchestration handoff.
4. Execute quality, LSP, and MX gates in later parity phases.

## Technical Approach

- Runtime: Pi bridge through shared MoAI Kernel.
- Persistence: .moai/specs/%s/.
- Interaction: moai_ask_user when clarification is required.

## Risks

- Current Pi command parity is foundational; full agent orchestration is delegated to later phases.

## Generated

%s
`, specID, request, specID, now.Format(time.RFC3339))
}

func acceptanceDocument(specID, request string) string {
	return fmt.Sprintf(`# Acceptance: %s

## Request Under Test

%s

## Scenarios

### AC-001 — Canonical SPEC creation

Given a user provides a feature description, when /moai plan completes, then spec.md, plan.md, and acceptance.md exist under the new SPEC directory.

### AC-002 — Run validation

Given a canonical SPEC exists, when /moai run %s is invoked, then the runtime validates required files before producing run-plan.md and progress.md.

### AC-003 — Sync validation

Given run-plan.md and progress.md exist, when /moai sync %s is invoked, then the runtime produces sync.md and updates status.json.

## Definition of Done

- Canonical files exist.
- Run artifacts are present before sync.
- Command results include artifacts and next actions.
`, specID, request, specID, specID)
}

func workflowGraphJSON(specID, phase string, agents []string, now time.Time) string {
	var b strings.Builder
	b.WriteString("{\n")
	b.WriteString(fmt.Sprintf("  \"specId\": %q,\n  \"phase\": %q,\n  \"runtime\": \"pi\",\n  \"generatedAt\": %q,\n  \"agents\": [", specID, phase, now.Format(time.RFC3339)))
	for i, agent := range agents {
		if i > 0 {
			b.WriteString(", ")
		}
		b.WriteString(fmt.Sprintf("%q", agent))
	}
	b.WriteString("],\n  \"subagentPromptPolicy\": \"blocker-report-only\"\n}\n")
	return b.String()
}

func delegationContract(specID, phase string, agents []string) string {
	return fmt.Sprintf("# Delegation Contract: %s\n\n- phase: %s\n- agents: %s\n- user interaction: orchestrator only via moai_ask_user\n- subagent blockers: return structured blocker report, never prompt user directly\n- quality: run quality/LSP/MX gates before completion\n", specID, phase, strings.Join(agents, ", "))
}

func clarificationLedger(request string, now time.Time) string {
	return fmt.Sprintf("# Clarifications\n\n- created_at: %s\n- initial_request: %s\n- status: no additional clarification captured in non-interactive bridge mode\n", now.Format(time.RFC3339), request)
}

func statusJSON(specID, status, phase string, now time.Time) string {
	return fmt.Sprintf("{\n  \"specId\": %q,\n  \"status\": %q,\n  \"phase\": %q,\n  \"updatedAt\": %q\n}\n", specID, status, phase, now.Format(time.RFC3339))
}

func blockerResult(command, message, recovery string) *CommandResult {
	return &CommandResult{
		Command: command,
		OK:      false,
		Messages: []Message{
			{Level: "blocker", Text: message},
			{Level: "recovery", Text: recovery},
		},
		UIState: map[string]any{"phase": command, "qualityStatus": "blocked"},
		Data:    map[string]any{"blocker": true, "reason": message, "recovery": recovery},
	}
}

func resolveSpecID(req CommandRequest) string {
	specID := extractSpecID(req.Args)
	if specID == "" {
		specID = latestSpecID(filepath.Join(req.CWD, ".moai", "specs"))
	}
	return specID
}

func missingCanonicalSpecFiles(specDir string) []string {
	var missing []string
	for _, name := range []string{"spec.md", "plan.md", "acceptance.md"} {
		if !pathExists(filepath.Join(specDir, name)) {
			missing = append(missing, name)
		}
	}
	return missing
}

func pathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func nextSpecID(cwd, domain string) string {
	specDir := filepath.Join(cwd, ".moai", "specs")
	entries, err := os.ReadDir(specDir)
	if err != nil {
		return fmt.Sprintf("SPEC-%s-001", domain)
	}
	prefix := "SPEC-" + domain + "-"
	maxID := 0
	for _, entry := range entries {
		if !entry.IsDir() || !strings.HasPrefix(entry.Name(), prefix) {
			continue
		}
		n, err := strconv.Atoi(strings.TrimPrefix(entry.Name(), prefix))
		if err == nil && n > maxID {
			maxID = n
		}
	}
	return fmt.Sprintf("SPEC-%s-%03d", domain, maxID+1)
}

func extractSpecID(text string) string {
	return specIDRe.FindString(text)
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
	if len(candidates) == 0 {
		return ""
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].mod.Equal(candidates[j].mod) {
			return candidates[i].name > candidates[j].name
		}
		return candidates[i].mod.After(candidates[j].mod)
	})
	return candidates[0].name
}
