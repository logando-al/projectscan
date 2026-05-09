package config

import (
	"errors"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
)

type ProjectscanConfig struct {
	IgnorePatterns []string                   `toml:"ignore_patterns"`
	Projects       map[string]ProjectOverride `toml:"projects"`
}

type ProjectOverride struct {
	Label          string `toml:"label"`
	DisplayName    string `toml:"display_name"`
	ArchivedReason string `toml:"archived_reason"`
	Pinned         bool   `toml:"pinned"`
}

func Load(rootPath string, explicitPath string) (ProjectscanConfig, error) {
	config := ProjectscanConfig{
		Projects: map[string]ProjectOverride{},
	}
	configPath := explicitPath
	if configPath == "" {
		configPath = filepath.Join(rootPath, ".projectscan.toml")
	}
	if _, err := os.Stat(configPath); errors.Is(err, os.ErrNotExist) {
		return config, nil
	}
	if _, err := toml.DecodeFile(configPath, &config); err != nil {
		return config, err
	}
	if config.Projects == nil {
		config.Projects = map[string]ProjectOverride{}
	}
	return config, nil
}

func LoadIgnorePatterns(rootPath string, configPatterns []string) ([]string, error) {
	patterns := append([]string{}, configPatterns...)
	path := filepath.Join(rootPath, ".projectscanignore")
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return patterns, nil
	}
	if err != nil {
		return nil, err
	}
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, nil
}
