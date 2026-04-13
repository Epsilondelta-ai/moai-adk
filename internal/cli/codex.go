package cli

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"

	codexruntime "github.com/modu-ai/moai-adk/internal/codex"
)

var codexCmd = &cobra.Command{
	Use:     "codex",
	Short:   "Codex-specific workflow helpers",
	GroupID: "project",
	Long:    "Codex-specific helpers for checking whether the current project is ready for the MoAI Codex workflow pack.",
}

var codexDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check Codex workflow readiness for this project",
	Long: strings.Join([]string{
		"Run Codex-specific readiness checks for the current project.",
		"It verifies shared MoAI state, deployed Codex skill assets, and the local workflow pack without implying Claude-specific runtime parity.",
	}, " "),
	RunE: runCodexDoctor,
}

func init() {
	rootCmd.AddCommand(codexCmd)
	codexCmd.AddCommand(codexDoctorCmd)

	codexDoctorCmd.Flags().BoolP("verbose", "v", false, "Show detailed Codex readiness information")
	codexDoctorCmd.Flags().Bool("json", false, "Emit machine-readable JSON output")
}

func runCodexDoctor(cmd *cobra.Command, _ []string) error {
	root, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("get working directory: %w", err)
	}

	return writeCodexDoctorReport(
		cmd.OutOrStdout(),
		root,
		getBoolFlag(cmd, "verbose"),
		getBoolFlag(cmd, "json"),
		codexruntime.NewInspector(),
	)
}

func writeCodexDoctorReport(out io.Writer, root string, verbose, jsonOutput bool, inspector codexruntime.Inspector) error {
	report := inspector.Inspect(root, verbose)

	if jsonOutput {
		encoder := json.NewEncoder(out)
		encoder.SetIndent("", "  ")
		return encoder.Encode(report)
	}

	maxLabel := 0
	for _, check := range report.Checks {
		if len(check.Name) > maxLabel {
			maxLabel = len(check.Name)
		}
	}

	lines := make([]string, 0, len(report.Checks)+2)
	for _, check := range report.Checks {
		lines = append(lines, renderStatusLine(codexStatusToCheckStatus(check.Status), check.Name, check.Message, maxLabel))
		if verbose && check.Detail != "" {
			lines = append(lines, fmt.Sprintf("    %s", cliMuted.Render(check.Detail)))
		}
	}
	lines = append(lines, "", renderSummaryLine(report.Summary.OK, report.Summary.Warn, report.Summary.Fail))

	if _, err := fmt.Fprintln(out, renderCard("Codex Readiness", strings.Join(lines, "\n"))); err != nil {
		return err
	}

	if advice := codexAdvice(report); len(advice) > 0 {
		_, err := fmt.Fprintln(out, "\n"+renderInfoCard("Suggested Next Steps", strings.Join(advice, "\n")))
		return err
	}

	return nil
}

func codexStatusToCheckStatus(status codexruntime.Status) CheckStatus {
	switch status {
	case codexruntime.StatusOK:
		return CheckOK
	case codexruntime.StatusWarn:
		return CheckWarn
	case codexruntime.StatusFail:
		return CheckFail
	default:
		return CheckWarn
	}
}

func codexAdvice(report codexruntime.Report) []string {
	var advice []string
	for _, check := range report.Checks {
		if check.Status == codexruntime.StatusOK || check.Detail == "" {
			continue
		}
		advice = append(advice, "- "+check.Detail)
	}
	return advice
}
