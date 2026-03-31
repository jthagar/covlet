package render

import (
	"bytes"
	"fmt"
	"strings"
	"text/template"

	"github.com/jthagar/covlet/backend/pkg/config"
)

// Render executes the template with the given resume data and returns plain text.
func Render(tpl *template.Template, resume config.Resume) (string, error) {
	if tpl == nil {
		return "", fmt.Errorf("template is nil")
	}
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, resume); err != nil {
		return "", err
	}
	return strings.TrimSpace(buf.String()), nil
}
