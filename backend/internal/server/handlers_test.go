package server

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/jthagar/covlet/backend/pkg/config"
)

func TestHandleHealth(t *testing.T) {
	t.Setenv("COVLET_HOME", t.TempDir())
	app := New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/health", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}

func TestHandleListFiles_Empty(t *testing.T) {
	t.Setenv("COVLET_HOME", t.TempDir())
	app := New()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/files", nil)
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	body, _ := io.ReadAll(resp.Body)
	if !bytes.Contains(body, []byte(`"files"`)) {
		t.Fatalf("expected files key: %s", body)
	}
}

func TestHandleRender(t *testing.T) {
	t.Setenv("COVLET_HOME", t.TempDir())
	app := New()
	payload := map[string]interface{}{
		"template": "Hello {{ .Name }}",
		"resume":   config.Resume{Name: "World"},
	}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/render", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	out, _ := io.ReadAll(resp.Body)
	if string(out) != "Hello World" {
		t.Fatalf("got %q", out)
	}
}

func TestHandleRender_DynamicOverride(t *testing.T) {
	t.Setenv("COVLET_HOME", t.TempDir())
	app := New()
	payload := map[string]interface{}{
		"template":  "Role: {{ .RoleHint }} at {{ .CompanyToApplyTo }}",
		"resume":    config.Resume{CompanyToApplyTo: "Acme"},
		"overrides": map[string]string{"RoleHint": "Engineer"},
	}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/render", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	out, _ := io.ReadAll(resp.Body)
	if string(out) != "Role: Engineer at Acme" {
		t.Fatalf("got %q", out)
	}
}

func TestHandleExportPDF(t *testing.T) {
	t.Setenv("COVLET_HOME", t.TempDir())
	app := New()
	payload := map[string]string{"title": "Test", "text": "Body line one.\n\nTwo."}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/export/pdf", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	if ct := resp.Header.Get("Content-Type"); ct != "application/pdf" {
		t.Fatalf("Content-Type: %q", ct)
	}
	cd := resp.Header.Get("Content-Disposition")
	if !strings.Contains(cd, "attachment") || !strings.Contains(cd, "Test") {
		t.Fatalf("Content-Disposition: %q", cd)
	}
	b, _ := io.ReadAll(resp.Body)
	if len(b) < 500 || !bytes.HasPrefix(b, []byte("%PDF")) {
		t.Fatalf("invalid pdf: len=%d", len(b))
	}
}

func TestHandleVars(t *testing.T) {
	t.Setenv("COVLET_HOME", t.TempDir())
	app := New()
	payload := map[string]string{"content": "{{ .Name }} {{ .Email }}"}
	raw, _ := json.Marshal(payload)
	req := httptest.NewRequest(http.MethodPost, "/api/v1/vars", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req, -1)
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
}
