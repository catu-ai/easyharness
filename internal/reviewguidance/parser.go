package reviewguidance

import (
	"fmt"
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

var namePattern = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)

// Definition is one reusable reviewer-guidance fragment read from Markdown.
type Definition struct {
	Name         string
	Description  string
	Instructions string
}

// ValidName reports whether name is a stable reviewer-guidance identifier.
func ValidName(name string) bool {
	return namePattern.MatchString(strings.TrimSpace(name))
}

// ParseFile reads one reviewer-guidance Markdown file.
func ParseFile(path string) (Definition, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Definition{}, fmt.Errorf("unable to read guidance file: %v", err)
	}
	return Parse(string(data))
}

// Parse validates the common repo and plan-scoped reviewer-guidance format.
func Parse(content string) (Definition, error) {
	frontmatter, body, err := splitFrontmatter(content)
	if err != nil {
		return Definition{}, err
	}
	var metadata struct {
		Name        string `yaml:"name"`
		Description string `yaml:"description"`
	}
	var node yaml.Node
	if err := yaml.Unmarshal([]byte(frontmatter), &node); err != nil {
		return Definition{}, fmt.Errorf("malformed YAML frontmatter: %v", err)
	}
	if len(node.Content) != 1 || node.Content[0].Kind != yaml.MappingNode {
		return Definition{}, fmt.Errorf("frontmatter must be a YAML object")
	}
	for i := 0; i+1 < len(node.Content[0].Content); i += 2 {
		key := strings.TrimSpace(node.Content[0].Content[i].Value)
		if key != "name" && key != "description" {
			return Definition{}, fmt.Errorf("unsupported frontmatter field %q", key)
		}
	}
	if err := yaml.Unmarshal([]byte(frontmatter), &metadata); err != nil {
		return Definition{}, fmt.Errorf("malformed YAML frontmatter: %v", err)
	}
	metadata.Name = strings.TrimSpace(metadata.Name)
	metadata.Description = strings.TrimSpace(metadata.Description)
	body = strings.TrimSpace(body)
	if !ValidName(metadata.Name) {
		return Definition{}, fmt.Errorf("field name must use lowercase alphanumeric segments separated by single hyphens")
	}
	if metadata.Description == "" {
		return Definition{}, fmt.Errorf("field description must not be empty")
	}
	if body == "" {
		return Definition{}, fmt.Errorf("instruction body must not be empty")
	}
	return Definition{
		Name:         metadata.Name,
		Description:  metadata.Description,
		Instructions: body,
	}, nil
}

func splitFrontmatter(content string) (string, string, error) {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	if !strings.HasPrefix(content, "---\n") {
		return "", "", fmt.Errorf("must start with YAML frontmatter")
	}
	rest := strings.TrimPrefix(content, "---\n")
	index := strings.Index(rest, "\n---")
	if index < 0 {
		return "", "", fmt.Errorf("missing closing YAML frontmatter marker")
	}
	frontmatter := rest[:index]
	body := rest[index+len("\n---"):]
	body = strings.TrimPrefix(body, "\n")
	return frontmatter, body, nil
}
