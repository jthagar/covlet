package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"
)

// Render executes the template with the given data (typically map[string]interface{}
// from templatevars.ResumeTemplateData) and returns plain text.
func Render(tpl *template.Template, data any) (string, error) {
	if tpl == nil {
		return "", fmt.Errorf("template is nil")
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, data); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
