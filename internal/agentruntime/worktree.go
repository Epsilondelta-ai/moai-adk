package agentruntime

import (
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	coregit "github.com/modu-ai/moai-adk/internal/core/git"
)

var unsafeBranchChars = regexp.MustCompile(`[^a-zA-Z0-9._/-]+`)

// WorktreeOptions describes optional worktree isolation for write-capable agents.
type WorktreeOptions struct {
	Enabled bool   `json:"enabled"`
	BaseDir string `json:"baseDir,omitempty"`
	Branch  string `json:"branch,omitempty"`
	Keep    bool   `json:"keep,omitempty"`
}

// PrepareWorktree creates an isolated git worktree for an agent request.
func PrepareWorktree(ctx context.Context, cwd string, agent string, opts WorktreeOptions) (string, string, func(), error) {
	if !opts.Enabled {
		return cwd, "", func() {}, nil
	}
	root, err := gitRoot(ctx, cwd)
	if err != nil {
		return "", "", func() {}, err
	}
	branch := opts.Branch
	if branch == "" {
		branch = fmt.Sprintf("moai/pi/%s/%d", sanitizeBranchPart(agent), time.Now().UnixNano())
	}
	baseDir := opts.BaseDir
	if baseDir == "" {
		baseDir = filepath.Join(filepath.Dir(root), ".moai-worktrees")
	}
	path := filepath.Join(baseDir, sanitizeBranchPart(branch))
	mgr := coregit.NewWorktreeManager(root)
	if err := mgr.Add(path, branch); err != nil {
		return "", "", func() {}, err
	}
	cleanup := func() {
		if !opts.Keep {
			_ = mgr.Remove(path, true)
		}
	}
	return path, branch, cleanup, nil
}

func gitRoot(ctx context.Context, cwd string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--show-toplevel")
	cmd.Dir = cwd
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("detect git root: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func sanitizeBranchPart(value string) string {
	value = strings.TrimSpace(value)
	value = unsafeBranchChars.ReplaceAllString(value, "-")
	value = strings.Trim(value, "/.-")
	if value == "" {
		return "agent"
	}
	return value
}
