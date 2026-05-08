package teamruntime

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type workflowYAML struct {
	Workflow struct {
		Team struct {
			RoleProfiles map[string]struct {
				Mode        string `yaml:"mode"`
				Model       string `yaml:"model"`
				Isolation   string `yaml:"isolation"`
				Description string `yaml:"description"`
			} `yaml:"role_profiles"`
		} `yaml:"team"`
	} `yaml:"workflow"`
}

// LoadProfiles loads team role profiles from .moai/config/sections/workflow.yaml.
func LoadProfiles(cwd string) (map[string]RoleProfile, error) {
	path := filepath.Join(cwd, ".moai", "config", "sections", "workflow.yaml")
	bytes, err := os.ReadFile(path)
	if err != nil {
		return DefaultProfiles(), nil
	}
	var raw workflowYAML
	if err := yaml.Unmarshal(bytes, &raw); err != nil {
		return nil, err
	}
	profiles := DefaultProfiles()
	for name, profile := range raw.Workflow.Team.RoleProfiles {
		profiles[name] = RoleProfile{Name: name, Mode: profile.Mode, Model: profile.Model, Isolation: profile.Isolation, Description: profile.Description}
	}
	return profiles, nil
}

// DefaultProfiles returns built-in role profile defaults.
func DefaultProfiles() map[string]RoleProfile {
	return map[string]RoleProfile{
		"researcher":  {Name: "researcher", Model: "haiku", Isolation: "none"},
		"analyst":     {Name: "analyst", Model: "sonnet", Isolation: "none"},
		"architect":   {Name: "architect", Model: "opus", Isolation: "none"},
		"implementer": {Name: "implementer", Model: "sonnet", Isolation: "worktree"},
		"tester":      {Name: "tester", Model: "sonnet", Isolation: "worktree"},
		"designer":    {Name: "designer", Model: "sonnet", Isolation: "worktree"},
		"reviewer":    {Name: "reviewer", Model: "sonnet", Isolation: "none"},
	}
}
