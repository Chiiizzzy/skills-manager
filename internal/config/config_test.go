package config

import (
	"os"
	"path/filepath"
	"strings"
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
		name              string
		mutate            func(*Config)
		wantErrSubstrings []string
	}{
		{
			name: "valid config",
		},
		{
			name: "missing sources",
			mutate: func(c *Config) {
				c.Sources = nil
			},
			wantErrSubstrings: []string{"sources must not be empty"},
		},
		{
			name: "missing profiles",
			mutate: func(c *Config) {
				c.Profiles = nil
			},
			wantErrSubstrings: []string{"profiles must not be empty"},
		},
		{
			name: "missing source repo",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Repo = ""
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "repo"},
		},
		{
			name: "missing source ref",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Ref = ""
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "ref"},
		},
		{
			name: "missing source skills",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Skills = nil
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "skills"},
		},
		{
			name: "missing skill path",
			mutate: func(c *Config) {
				c.Sources["superpowers"].Skills["brainstorming"] = Skill{}
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "skill \"brainstorming\"", "path"},
		},
		{
			name: "duplicate skill name across sources",
			mutate: func(c *Config) {
				c.Sources["local-overrides"] = Source{
					Repo: "https://github.com/example/local-skills.git",
					Ref:  "main",
					Skills: map[string]Skill{
						"brainstorming": {Path: "skills/brainstorming"},
					},
				}
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "source \"local-overrides\"", "skill \"brainstorming\"", "duplicates"},
		},
		{
			name: "missing profile target",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Target = ""
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "target"},
		},
		{
			name: "missing profile skills",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Skills = nil
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "skills"},
		},
		{
			name: "profile references unknown skill",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Skills = []string{"unknown-skill"}
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "unknown skill \"unknown-skill\""},
		},
		{
			name: "duplicate skill in one profile",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Skills = []string{"brainstorming", "brainstorming"}
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "skill \"brainstorming\"", "duplicated"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := validConfig()
			if tt.mutate != nil {
				tt.mutate(&cfg)
			}

			err := cfg.Validate()
			if len(tt.wantErrSubstrings) == 0 {
				if err != nil {
					t.Fatalf("Validate() error = %v, want nil", err)
				}
				return
			}
			if err == nil {
				t.Fatal("Validate() error = nil, want error")
			}
			for _, want := range tt.wantErrSubstrings {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("Validate() error = %q, want substring %q", err, want)
				}
			}
		})
	}
}
