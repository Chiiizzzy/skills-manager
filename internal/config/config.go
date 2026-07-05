package config

import (
	"fmt"
	"os"
	"regexp"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

var versionTagRefRE = regexp.MustCompile(`^v?[0-9]+[.][0-9]+[.][0-9]+([-+][0-9A-Za-z][0-9A-Za-z.-]*)?$`)

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
		if isBlank(name) {
			return fmt.Errorf("source %q name must not be empty", name)
		}
		if hasSurroundingWhitespace(name) {
			return fmt.Errorf("source %q name must not have leading or trailing whitespace", name)
		}
		if isBlank(source.Repo) {
			return fmt.Errorf("source %q repo %q must not be empty", name, source.Repo)
		}
		if hasSurroundingWhitespace(source.Repo) {
			return fmt.Errorf("source %q repo %q must not have leading or trailing whitespace", name, source.Repo)
		}
		if isBlank(source.Ref) {
			return fmt.Errorf("source %q ref %q must not be empty", name, source.Ref)
		}
		if hasSurroundingWhitespace(source.Ref) {
			return fmt.Errorf("source %q ref %q must not have leading or trailing whitespace", name, source.Ref)
		}
		if err := validateSourceRef(name, source.Ref); err != nil {
			return err
		}
		if len(source.Skills) == 0 {
			return fmt.Errorf("source %q skills must not be empty", name)
		}

		for skillName, skill := range source.Skills {
			if isBlank(skillName) {
				return fmt.Errorf("source %q skill %q name must not be empty", name, skillName)
			}
			if hasSurroundingWhitespace(skillName) {
				return fmt.Errorf("source %q skill %q name must not have leading or trailing whitespace", name, skillName)
			}
			if isBlank(skill.Path) {
				return fmt.Errorf("source %q skill %q path %q must not be empty", name, skillName, skill.Path)
			}
			if hasSurroundingWhitespace(skill.Path) {
				return fmt.Errorf("source %q skill %q path %q must not have leading or trailing whitespace", name, skillName, skill.Path)
			}
			if existingSource, ok := skillSources[skillName]; ok {
				return fmt.Errorf("source %q skill %q duplicates skill from source %q", name, skillName, existingSource)
			}
			skillSources[skillName] = name
		}
	}

	for name, profile := range c.Profiles {
		if isBlank(name) {
			return fmt.Errorf("profile %q name must not be empty", name)
		}
		if hasSurroundingWhitespace(name) {
			return fmt.Errorf("profile %q name must not have leading or trailing whitespace", name)
		}
		if isBlank(profile.Target) {
			return fmt.Errorf("profile %q target %q must not be empty", name, profile.Target)
		}
		if hasSurroundingWhitespace(profile.Target) {
			return fmt.Errorf("profile %q target %q must not have leading or trailing whitespace", name, profile.Target)
		}
		if len(profile.Skills) == 0 {
			return fmt.Errorf("profile %q skills must not be empty", name)
		}
		profileSkills := make(map[string]struct{})
		for _, skillName := range profile.Skills {
			if isBlank(skillName) {
				return fmt.Errorf("profile %q skill %q name must not be empty", name, skillName)
			}
			if hasSurroundingWhitespace(skillName) {
				return fmt.Errorf("profile %q skill %q name must not have leading or trailing whitespace", name, skillName)
			}
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

func isBlank(value string) bool {
	return strings.TrimSpace(value) == ""
}

func hasSurroundingWhitespace(value string) bool {
	return strings.TrimSpace(value) != value
}

func validateSourceRef(sourceName, ref string) error {
	if strings.HasPrefix(ref, "origin/") ||
		strings.HasPrefix(ref, "refs/") ||
		versionTagRefRE.MatchString(ref) ||
		isHexRef(ref) ||
		strings.ContainsAny(ref, "~^:?*[\\") ||
		strings.Contains(ref, "//") ||
		strings.HasSuffix(ref, "/") ||
		strings.IndexFunc(ref, unicode.IsSpace) >= 0 {
		return fmt.Errorf("source %q ref %q must be a plain branch name", sourceName, ref)
	}
	return nil
}

func isHexRef(ref string) bool {
	if len(ref) < 7 || len(ref) > 40 {
		return false
	}
	for _, r := range ref {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') && (r < 'A' || r > 'F') {
			return false
		}
	}
	return true
}
