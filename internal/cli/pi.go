package cli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"

	pibridge "github.com/modu-ai/moai-adk/internal/pi/bridge"
	"github.com/modu-ai/moai-adk/internal/pi/resourcesync"
	"github.com/modu-ai/moai-adk/internal/pi/skillconvert"
)

var piCmd = &cobra.Command{
	Use:     "pi",
	Short:   "Pi extension runtime support",
	GroupID: "tools",
	Long:    "Commands used by the MoAI Pi extension adapter.",
}

func init() {
	rootCmd.AddCommand(piCmd)

	piCmd.AddCommand(&cobra.Command{
		Use:   "bridge",
		Short: "Execute a Pi bridge request from stdin",
		Long:  "Reads a JSON bridge request from stdin and writes a JSON bridge response to stdout.",
		RunE:  runPiBridge,
	})

	piCmd.AddCommand(&cobra.Command{
		Use:   "doctor",
		Short: "Check Pi extension bridge readiness",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			res := pibridge.NewHandler().Handle(cmd.Context(), pibridge.Request{Kind: "doctor", CWD: cwd})
			return writeJSON(os.Stdout, res)
		},
	})

	piCmd.AddCommand(&cobra.Command{
		Use:   "capabilities",
		Short: "List Pi bridge capabilities",
		RunE: func(cmd *cobra.Command, args []string) error {
			res := pibridge.NewHandler().Handle(cmd.Context(), pibridge.Request{Kind: "capabilities"})
			return writeJSON(os.Stdout, res)
		},
	})

	syncResourcesCmd := &cobra.Command{
		Use:   "sync-resources",
		Short: "Generate Pi prompts and skills from MoAI Claude assets",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("get working directory: %w", err)
			}
			check, _ := cmd.Flags().GetBool("check")
			var prompts *resourcesync.Result
			var skills *skillconvert.Result
			if check {
				prompts, err = resourcesync.Check(cwd)
			} else {
				prompts, err = resourcesync.Sync(cwd)
			}
			if err != nil {
				return err
			}
			if check {
				skills, err = skillconvert.Check(cwd)
			} else {
				skills, err = skillconvert.Convert(cwd)
			}
			if err != nil {
				return err
			}
			result := map[string]any{"prompts": prompts.Prompts, "skills": skills.Skills, "stale": append(prompts.Stale, skills.Stale...), "checked": check}
			if check && len(append(prompts.Stale, skills.Stale...)) > 0 {
				_ = writeJSON(os.Stdout, result)
				return fmt.Errorf("Pi resources are stale; run moai pi sync-resources")
			}
			return writeJSON(os.Stdout, result)
		},
	}
	syncResourcesCmd.Flags().Bool("check", false, "check for generated Pi resource drift without writing")
	piCmd.AddCommand(syncResourcesCmd)
}

func runPiBridge(cmd *cobra.Command, _ []string) error {
	data, err := io.ReadAll(os.Stdin)
	if err != nil {
		return fmt.Errorf("read bridge request: %w", err)
	}
	var req pibridge.Request
	if err := json.Unmarshal(data, &req); err != nil {
		res := pibridge.Failure("unknown", "invalid_json", err.Error())
		return writeJSON(os.Stdout, res)
	}
	ctx := cmd.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	res := pibridge.NewHandler().Handle(ctx, req)
	return writeJSON(os.Stdout, res)
}

func writeJSON(w io.Writer, value any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(value)
}
