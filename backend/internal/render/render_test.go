package render

import (
	"strings"
	"testing"
	"text/template"

	"github.com/jthagar/covlet/backend/pkg/config"
)

func TestRender_Basic(t *testing.T) {
	tpl := template.Must(template.New("t").Parse("Hello {{ .Name }}"))
	out, err := Render(tpl, config.Resume{Name: "World"})
	if err != nil {
		t.Fatal(err)
	}
	if out != "Hello World" {
		t.Fatalf("got %q", out)
	}
}

func TestRender_NilTemplate(t *testing.T) {
	_, err := Render(nil, config.Resume{})
	if err == nil || !strings.Contains(err.Error(), "nil") {
		t.Fatalf("expected nil template error, got %v", err)
	}
}
