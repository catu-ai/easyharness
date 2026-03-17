package templates

import (
	_ "embed"
	"fmt"
	"regexp"
)

var (
	//go:embed plan-template.md
	planTemplate string

	templateVersionPattern = regexp.MustCompile(`(?m)^template_version:\s*([^\s]+)\s*$`)
)

func PlanTemplate() string {
	return planTemplate
}

func PlanTemplateVersion() (string, error) {
	matches := templateVersionPattern.FindStringSubmatch(planTemplate)
	if len(matches) != 2 {
		return "", fmt.Errorf("embedded plan template is missing template_version")
	}
	return matches[1], nil
}
