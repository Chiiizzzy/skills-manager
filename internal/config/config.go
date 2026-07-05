package config

import (
	"fmt"
	"os"

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
		if source.Repo == "" {
			return fmt.Errorf("source %q repo must not be empty", name)
		}
		if source.Ref == "" {
			return fmt.Errorf("source %q ref must not be empty", name)
		}
		if len(source.Skills) == 0 {
			return fmt.Errorf("source %q skills must not be empty", name)
		}

		for skillName, skill := range source.Skills {
			if skill.Path == "" {
				return fmt.Errorf("source %q skill %q path must not be empty", name, skillName)
			}
			if existingSource, ok := skillSources[skillName]; ok {
				return fmt.Errorf("source %q skill %q duplicates skill from source %q", name, skillName, existingSource)
			}
			skillSources[skillName] = name
		}
	}

	for name, profile := range c.Profiles {
		if profile.Target == "" {
			return fmt.Errorf("profile %q target must not be empty", name)
		}
		if len(profile.Skills) == 0 {
			return fmt.Errorf("profile %q skills must not be empty", name)
		}
		profileSkills := make(map[string]struct{})
		for _, skillName := range profile.Skills {
			if _, ok := profileSkills[skillName]; ok {
				return fmt.Errorf("profile %q skill %q must not be duplicated", name, skillName)
			}
			profileSkills[skillName] = struct{}{}

			if _, ok := skillSources[skillName]; !ok {
				return fmt.Errorf("profile %q references unknown skill %q", name, skillName)
			}
		}
	}

	return nil
}
