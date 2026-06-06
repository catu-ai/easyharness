package repoconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	Dir            = ".harness"
	File           = ".harness/config.yaml"
	CurrentVersion = 1
	DefaultContent = "version: 1\n"
)

type Config struct {
	Version int `yaml:"version"`
}

type LoadResult struct {
	Config   Config
	Path     string
	Exists   bool
	Valid    bool
	Warnings []string
}

func Load(workdir string) LoadResult {
	path := filepath.Join(workdir, filepath.FromSlash(File))
	result := LoadResult{
		Config: Config{Version: CurrentVersion},
		Path:   path,
		Valid:  true,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return result
		}
		return invalid(path, fmt.Sprintf("unable to read config: %v", err))
	}
	result.Exists = true

	var node yaml.Node
	if err := yaml.Unmarshal(data, &node); err != nil {
		return invalid(path, fmt.Sprintf("malformed YAML: %v", err))
	}
	if len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return invalid(path, "config must be a YAML object")
	}
	mapping := node.Content[0]
	fields := map[string]*yaml.Node{}
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key := strings.TrimSpace(mapping.Content[i].Value)
		if key == "" {
			return invalid(path, "config contains an empty field name")
		}
		if _, ok := fields[key]; ok {
			return invalid(path, fmt.Sprintf("config contains duplicate field %q", key))
		}
		fields[key] = mapping.Content[i+1]
	}
	versionNode, ok := fields["version"]
	if !ok {
		return invalid(path, "missing required field version")
	}
	if versionNode.Kind != yaml.ScalarNode || versionNode.Tag != "!!int" {
		return invalid(path, "field version must be the integer 1")
	}
	var version int
	if err := versionNode.Decode(&version); err != nil {
		return invalid(path, fmt.Sprintf("field version must be the integer 1: %v", err))
	}
	if version != CurrentVersion {
		return invalid(path, fmt.Sprintf("unsupported version %d", version))
	}
	for key := range fields {
		if key != "version" {
			return invalid(path, fmt.Sprintf("unsupported field %q", key))
		}
	}

	result.Config.Version = version
	return result
}

func invalid(path, reason string) LoadResult {
	return LoadResult{
		Config: Config{Version: CurrentVersion},
		Path:   path,
		Exists: true,
		Valid:  false,
		Warnings: []string{
			fmt.Sprintf("Ignoring %s because %s; using built-in defaults.", filepath.ToSlash(path), reason),
		},
	}
}
