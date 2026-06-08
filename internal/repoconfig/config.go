package repoconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	Dir                      = ".harness"
	File                     = ".harness/config.yaml"
	CurrentVersion           = 1
	DefaultContent           = "version: 1\n"
	DefaultActivePlansRoot   = "docs/plans/active"
	DefaultArchivedPlansRoot = "docs/plans/archived"
	DefaultLocalRuntimeRoot  = ".local/harness"
)

type Config struct {
	Version int         `yaml:"version"`
	Paths   PathsConfig `yaml:"paths,omitempty"`
}

type PathsConfig struct {
	Plans        PlanPathsConfig `yaml:"plans,omitempty"`
	LocalRuntime string          `yaml:"local_runtime,omitempty"`
}

type PlanPathsConfig struct {
	Active   string `yaml:"active,omitempty"`
	Archived string `yaml:"archived,omitempty"`
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
		Config: DefaultConfig(),
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
		if key != "version" && key != "paths" {
			return invalid(path, fmt.Sprintf("unsupported field %q", key))
		}
	}

	result.Config.Version = version
	if pathsNode, ok := fields["paths"]; ok {
		paths, err := parsePaths(pathsNode)
		if err != nil {
			return invalid(path, err.Error())
		}
		result.Config.Paths = paths
	}
	return result
}

func DefaultConfig() Config {
	return Config{
		Version: CurrentVersion,
		Paths: PathsConfig{
			Plans: PlanPathsConfig{
				Active:   DefaultActivePlansRoot,
				Archived: DefaultArchivedPlansRoot,
			},
			LocalRuntime: DefaultLocalRuntimeRoot,
		},
	}
}

func invalid(path, reason string) LoadResult {
	return LoadResult{
		Config: DefaultConfig(),
		Path:   path,
		Exists: true,
		Valid:  false,
		Warnings: []string{
			fmt.Sprintf("Ignoring %s because %s; using built-in defaults.", filepath.ToSlash(path), reason),
		},
	}
}

func parsePaths(node *yaml.Node) (PathsConfig, error) {
	paths := DefaultConfig().Paths
	if node.Kind != yaml.MappingNode {
		return paths, fmt.Errorf("field paths must be a YAML object")
	}
	fields, err := mappingFields("paths", node)
	if err != nil {
		return paths, err
	}
	for key, value := range fields {
		switch key {
		case "plans":
			plans, err := parsePlanPaths(value, paths.Plans)
			if err != nil {
				return paths, err
			}
			paths.Plans = plans
		case "local_runtime":
			localRuntime, err := parseRepoRelativePath("paths.local_runtime", value)
			if err != nil {
				return paths, err
			}
			paths.LocalRuntime = localRuntime
		default:
			return paths, fmt.Errorf("unsupported field %q", "paths."+key)
		}
	}
	if err := validateDistinctRoots(paths); err != nil {
		return paths, err
	}
	return paths, nil
}

func parsePlanPaths(node *yaml.Node, defaults PlanPathsConfig) (PlanPathsConfig, error) {
	plans := defaults
	if node.Kind != yaml.MappingNode {
		return plans, fmt.Errorf("field paths.plans must be a YAML object")
	}
	fields, err := mappingFields("paths.plans", node)
	if err != nil {
		return plans, err
	}
	for key, value := range fields {
		switch key {
		case "active":
			active, err := parseRepoRelativePath("paths.plans.active", value)
			if err != nil {
				return plans, err
			}
			plans.Active = active
		case "archived":
			archived, err := parseRepoRelativePath("paths.plans.archived", value)
			if err != nil {
				return plans, err
			}
			plans.Archived = archived
		default:
			return plans, fmt.Errorf("unsupported field %q", "paths.plans."+key)
		}
	}
	return plans, nil
}

func mappingFields(prefix string, node *yaml.Node) (map[string]*yaml.Node, error) {
	fields := map[string]*yaml.Node{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		key := strings.TrimSpace(node.Content[i].Value)
		if key == "" {
			return nil, fmt.Errorf("%s contains an empty field name", prefix)
		}
		if _, ok := fields[key]; ok {
			return nil, fmt.Errorf("%s contains duplicate field %q", prefix, key)
		}
		fields[key] = node.Content[i+1]
	}
	return fields, nil
}

func parseRepoRelativePath(label string, node *yaml.Node) (string, error) {
	if node.Kind != yaml.ScalarNode || node.Tag != "!!str" {
		return "", fmt.Errorf("%s must be a string path", label)
	}
	value := strings.TrimSpace(node.Value)
	if value == "" {
		return "", fmt.Errorf("%s must not be empty", label)
	}
	if filepath.IsAbs(value) || strings.HasPrefix(value, "~") {
		return "", fmt.Errorf("%s must be repo-relative", label)
	}
	if strings.Contains(value, "\\") {
		return "", fmt.Errorf("%s must use slash-separated repo-relative paths", label)
	}
	clean := filepath.ToSlash(filepath.Clean(filepath.FromSlash(value)))
	if clean == "." {
		return "", fmt.Errorf("%s must not resolve to the repository root", label)
	}
	if strings.HasPrefix(clean, "../") || clean == ".." {
		return "", fmt.Errorf("%s must stay within the repository", label)
	}
	return clean, nil
}

func validateDistinctRoots(paths PathsConfig) error {
	roots := []struct {
		label string
		path  string
	}{
		{label: "paths.plans.active", path: paths.Plans.Active},
		{label: "paths.plans.archived", path: paths.Plans.Archived},
		{label: "paths.local_runtime", path: paths.LocalRuntime},
	}
	for i := 0; i < len(roots); i++ {
		for j := i + 1; j < len(roots); j++ {
			if pathsOverlap(roots[i].path, roots[j].path) {
				return fmt.Errorf("configured path roots must not overlap: %s=%s and %s=%s", roots[i].label, roots[i].path, roots[j].label, roots[j].path)
			}
		}
	}
	return nil
}

func pathsOverlap(a, b string) bool {
	a = filepath.ToSlash(filepath.Clean(filepath.FromSlash(a)))
	b = filepath.ToSlash(filepath.Clean(filepath.FromSlash(b)))
	if a == b {
		return true
	}
	return strings.HasPrefix(a, b+"/") || strings.HasPrefix(b, a+"/")
}
