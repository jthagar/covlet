package config

import (
	"bytes"
	"os"
	"path/filepath"
	"text/template"
	"testing"
)

func TestLoadConfig_ProjectURLs(t *testing.T) {
	path := filepath.Join(t.TempDir(), "resume.yml")
	content := `name: "Test"
email: "t@example.com"
projects:
  - name: "Cover Letter Generator"
    description: "tooling"
    url: "https://www.github.com/johndoe/cover-letter-generator"
  - name: "Site"
    description: "portfolio"
    url: "johndoe.com"
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	c, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if len(c.Resume.Projects) != 2 {
		t.Fatalf("projects: got %d want 2", len(c.Resume.Projects))
	}
	if got := c.Resume.Projects[0].URL; got != "https://www.github.com/johndoe/cover-letter-generator" {
		t.Fatalf("first URL: got %q", got)
	}
	if got := c.Resume.Projects[1].URL; got != "johndoe.com" {
		t.Fatalf("second URL: got %q", got)
	}
}

func TestResume_ProjectURLInTextTemplate(t *testing.T) {
	r := Resume{
		Projects: []Project{
			{Name: "P", Description: "d", URL: "https://example.com/foo"},
		},
	}
	tpl := template.Must(template.New("").Parse(`{{ range .Projects }}{{ .URL }}{{ end }}`))
	var buf bytes.Buffer
	if err := tpl.Execute(&buf, r); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "https://example.com/foo" {
		t.Fatalf("template output: got %q want URL", buf.String())
	}
}
