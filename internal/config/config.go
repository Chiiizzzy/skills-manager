package config

import (
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Sources  map[string]Source  `yaml:"sources"`
	Profiles map[string]Profile `yaml:"profiles"`
}

type Source struct {
	Repo   string           `yaml:"repo"`
	Ref    string           `yaml:"ref"`
	Skills map[string]Skill `yaml:"skills"`
}

type Skill struct {
	Path string `yaml:"path"`
}

type Profile struct {
	Target string   `yaml:"target"`
	Skills []string `yaml:"skills"`
}

type Lock struct {
	Skills map[string]LockedSkill `yaml:"skills"`
}

type LockedSkill struct {
	Source         string `yaml:"source"`
	UpstreamCommit string `yaml:"upstream_commit"`
	UpstreamPath   string `yaml:"upstream_path"`
}

func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, err
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func (c *Config) Validate() error {
	if len(c.Sources) == 0 {
		return fmt.Errorf("sources must not be empty")
	}
	if len(c.Profiles) == 0 {
		return fmt.Errorf("profiles must not be empty")
	}

	skillSources := make(map[string]string)
	for name, source := range c.Sources {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("source %q name must not be empty", name)
		}
		if strings.TrimSpace(source.Repo) == "" {
			return fmt.Errorf("source %q repo must not be empty", name)
		}
		if strings.TrimSpace(source.Ref) == "" {
			return fmt.Errorf("source %q ref must not be empty", name)
		}
		if len(source.Skills) == 0 {
			return fmt.Errorf("source %q skills must not be empty", name)
		}

		for skillName, skill := range source.Skills {
			trimmedSkillName := strings.TrimSpace(skillName)
			if trimmedSkillName == "" {
				return fmt.Errorf("source %q skill %q name must not be empty", name, skillName)
			}
			if strings.TrimSpace(skill.Path) == "" {
				return fmt.Errorf("source %q skill %q path must not be empty", name, skillName)
			}
			if existingSource, ok := skillSources[trimmedSkillName]; ok {
				return fmt.Errorf("source %q skill %q duplicates skill from source %q", name, skillName, existingSource)
			}
			skillSources[trimmedSkillName] = name
		}
	}

	for name, profile := range c.Profiles {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("profile %q name must not be empty", name)
		}
		if strings.TrimSpace(profile.Target) == "" {
			return fmt.Errorf("profile %q target must not be empty", name)
		}
		if len(profile.Skills) == 0 {
			return fmt.Errorf("profile %q skills must not be empty", name)
		}
		profileSkills := make(map[string]struct{})
		for _, skillName := range profile.Skills {
			trimmedSkillName := strings.TrimSpace(skillName)
			if trimmedSkillName == "" {
				return fmt.Errorf("profile %q skill %q name must not be empty", name, skillName)
			}
			if _, ok := profileSkills[trimmedSkillName]; ok {
				return fmt.Errorf("profile %q skill %q must not be duplicated", name, skillName)
			}
			profileSkills[trimmedSkillName] = struct{}{}

			if _, ok := skillSources[trimmedSkillName]; !ok {
				return fmt.Errorf("profile %q references unknown skill %q", name, skillName)
			}
		}
	}

	return nil
}
