package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfigValidConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skillctl.yaml")
	data := []byte(`
sources:
  superpowers:
    repo: https://github.com/example/skills.git
    ref: main
    skills:
      brainstorming:
        path: skills/brainstorming
      writing-plans:
        path: skills/writing-plans
profiles:
  trae-workspace:
    target: /cloudide/workspace/.trae/skills
    skills:
      - brainstorming
      - writing-plans
`)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}

	got := cfg.Sources["superpowers"].Skills["brainstorming"].Path
	if got != "skills/brainstorming" {
		t.Fatalf("brainstorming path = %q, want %q", got, "skills/brainstorming")
	}
}

func TestConfigValidate(t *testing.T) {
	validConfig := func() Config {
		return Config{
			Sources: map[string]Source{
				"superpowers": {
					Repo: "https://github.com/example/skills.git",
					Ref:  "main",
					Skills: map[string]Skill{
						"brainstorming": {Path: "skills/brainstorming"},
					},
				},
			},
			Profiles: map[string]Profile{
				"trae-workspace": {
					Target: "/cloudide/workspace/.trae/skills",
					Skills: []string{"brainstorming"},
				},
			},
		}
	}

	tests := []struct {
		name   string
		mutate func(*Config)
	}{
		{
			name: "missing sources",
			mutate: func(c *Config) {
				c.Sources = nil
			},
		},
		{
			name: "missing source repo",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Repo = ""
				c.Sources["superpowers"] = source
			},
		},
		{
			name: "missing source ref",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Ref = ""
				c.Sources["superpowers"] = source
			},
		},
		{
			name: "missing source skills",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Skills = nil
				c.Sources["superpowers"] = source
			},
		},
		{
			name: "missing skill path",
			mutate: func(c *Config) {
				c.Sources["superpowers"].Skills["brainstorming"] = Skill{}
			},
		},
		{
			name: "missing profile target",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Target = ""
				c.Profiles["trae-workspace"] = profile
			},
		},
		{
			name: "missing profile skills",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Skills = nil
				c.Profiles["trae-workspace"] = profile
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			tt.mutate(&cfg)

			if err := cfg.Validate(); err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
		})
	}
}
