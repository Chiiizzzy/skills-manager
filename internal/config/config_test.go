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

func TestLoadConfigInvalidConfigReturnsValidationError(t *testing.T) {
	path := filepath.Join(t.TempDir(), "skillctl.yaml")
	data := []byte(`
sources:
  superpowers:
    repo: "   "
    ref: main
    skills:
      brainstorming:
        path: skills/brainstorming
profiles:
  trae-workspace:
    target: /cloudide/workspace/.trae/skills
    skills:
      - brainstorming
`)

	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	_, err := LoadConfig(path)
	if err == nil {
		t.Fatal("LoadConfig() error = nil, want validation error")
	}
	for _, want := range []string{"source \"superpowers\"", "repo"} {
		if !strings.Contains(err.Error(), want) {
			t.Fatalf("LoadConfig() error = %q, want substring %q", err, want)
		}
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
			name: "blank source name",
			mutate: func(c *Config) {
				c.Sources[" \t "] = c.Sources["superpowers"]
				delete(c.Sources, "superpowers")
			},
			wantErrSubstrings: []string{"source \" \\t \"", "name"},
		},
		{
			name: "source name with surrounding whitespace",
			mutate: func(c *Config) {
				c.Sources[" superpowers "] = c.Sources["superpowers"]
				delete(c.Sources, "superpowers")
			},
			wantErrSubstrings: []string{"source \" superpowers \"", "name", "whitespace"},
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
			name: "blank source repo",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Repo = " \t\n"
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "repo"},
		},
		{
			name: "source repo with surrounding whitespace",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Repo = " https://github.com/example/skills.git "
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "repo \" https://github.com/example/skills.git \"", "whitespace"},
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
			name: "blank source ref",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Ref = " \t\n"
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "ref"},
		},
		{
			name: "source ref with surrounding whitespace",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Ref = " main "
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "ref \" main \"", "whitespace"},
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
			name: "blank skill name",
			mutate: func(c *Config) {
				c.Sources["superpowers"].Skills[" \t "] = Skill{Path: "skills/blank"}
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "skill \" \\t \"", "name"},
		},
		{
			name: "skill name with surrounding whitespace",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Skills[" brainstorming "] = source.Skills["brainstorming"]
				delete(source.Skills, "brainstorming")
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "skill \" brainstorming \"", "name", "whitespace"},
		},
		{
			name: "missing skill path",
			mutate: func(c *Config) {
				c.Sources["superpowers"].Skills["brainstorming"] = Skill{}
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "skill \"brainstorming\"", "path"},
		},
		{
			name: "blank skill path",
			mutate: func(c *Config) {
				c.Sources["superpowers"].Skills["brainstorming"] = Skill{Path: " \t\n"}
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "skill \"brainstorming\"", "path"},
		},
		{
			name: "skill path with surrounding whitespace",
			mutate: func(c *Config) {
				c.Sources["superpowers"].Skills["brainstorming"] = Skill{Path: " skills/brainstorming "}
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "skill \"brainstorming\"", "path \" skills/brainstorming \"", "whitespace"},
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
			name: "skill name with surrounding whitespace in additional source",
			mutate: func(c *Config) {
				c.Sources["local-overrides"] = Source{
					Repo: "https://github.com/example/local-skills.git",
					Ref:  "main",
					Skills: map[string]Skill{
						" brainstorming ": {Path: "skills/brainstorming"},
					},
				}
			},
			wantErrSubstrings: []string{"source \"local-overrides\"", "skill \" brainstorming \"", "name", "whitespace"},
		},
		{
			name: "profile no longer references source skill after trimming source key",
			mutate: func(c *Config) {
				source := c.Sources["superpowers"]
				source.Skills = map[string]Skill{
					" brainstorming ": {Path: "skills/brainstorming"},
				}
				c.Sources["superpowers"] = source
			},
			wantErrSubstrings: []string{"source \"superpowers\"", "skill \" brainstorming \"", "name", "whitespace"},
		},
		{
			name: "profile skill entry with surrounding whitespace",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Skills = []string{" brainstorming "}
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "skill \" brainstorming \"", "name", "whitespace"},
		},
		{
			name: "blank profile name",
			mutate: func(c *Config) {
				c.Profiles[" \t "] = c.Profiles["trae-workspace"]
				delete(c.Profiles, "trae-workspace")
			},
			wantErrSubstrings: []string{"profile \" \\t \"", "name"},
		},
		{
			name: "profile name with surrounding whitespace",
			mutate: func(c *Config) {
				c.Profiles[" trae-workspace "] = c.Profiles["trae-workspace"]
				delete(c.Profiles, "trae-workspace")
			},
			wantErrSubstrings: []string{"profile \" trae-workspace \"", "name", "whitespace"},
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
			name: "blank profile target",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Target = " \t\n"
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "target"},
		},
		{
			name: "profile target with surrounding whitespace",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Target = " /cloudide/workspace/.trae/skills "
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "target \" /cloudide/workspace/.trae/skills \"", "whitespace"},
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
			name: "blank profile skill name",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Skills = []string{" \t "}
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "skill \" \\t \"", "name"},
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
		{
			name: "profile duplicate check rejects surrounding whitespace before trimming",
			mutate: func(c *Config) {
				profile := c.Profiles["trae-workspace"]
				profile.Skills = []string{"brainstorming", " brainstorming "}
				c.Profiles["trae-workspace"] = profile
			},
			wantErrSubstrings: []string{"profile \"trae-workspace\"", "skill \" brainstorming \"", "name", "whitespace"},
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
