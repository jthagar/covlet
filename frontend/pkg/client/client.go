package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/jthagar/covlet/backend/pkg/config"
)

// Client talks to the covlet HTTP API.
type Client struct {
	BaseURL string
	HTTP    *http.Client
}

// New returns a client with BaseURL trimmed (default http://127.0.0.1:8080).
func New(base string) *Client {
	base = strings.TrimRight(strings.TrimSpace(base), "/")
	if base == "" {
		base = "http://127.0.0.1:8080"
	}
	return &Client{
		BaseURL: base,
		HTTP:    &http.Client{Timeout: 3 * time.Minute},
	}
}

func (c *Client) api(path string) string {
	return c.BaseURL + path
}

func (c *Client) do(req *http.Request) (*http.Response, error) {
	if c.HTTP == nil {
		c.HTTP = &http.Client{Timeout: 2 * time.Minute}
	}
	return c.HTTP.Do(req)
}

// Health checks GET /api/v1/health.
func (c *Client) Health() error {
	req, err := http.NewRequest(http.MethodGet, c.api("/api/v1/health"), nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

// ListFiles returns template paths relative to the templates root.
func (c *Client) ListFiles() ([]string, error) {
	req, err := http.NewRequest(http.MethodGet, c.api("/api/v1/files"), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("list files: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out struct {
		Files []string `json:"files"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return nil, err
	}
	return out.Files, nil
}

// GetFile reads a template file by relative path.
func (c *Client) GetFile(relPath string) ([]byte, error) {
	u, err := url.Parse(c.api("/api/v1/file"))
	if err != nil {
		return nil, err
	}
	q := u.Query()
	q.Set("path", relPath)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}
	resp, err := c.do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("get file: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return b, nil
}

// PutFile writes template content.
func (c *Client) PutFile(relPath string, content []byte) error {
	u, err := url.Parse(c.api("/api/v1/file"))
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("path", relPath)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodPut, u.String(), bytes.NewReader(content))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "text/plain; charset=utf-8")
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("put file: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

// CreateFile creates an empty template under parent (use "" for templates root). Returns relative path.
func (c *Client) CreateFile(parent, filename string) (string, error) {
	payload := map[string]string{"parent": parent, "filename": filename}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.api("/api/v1/file"), bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusCreated {
		return "", fmt.Errorf("create file: %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}
	var out struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(body, &out); err != nil {
		return "", err
	}
	return out.Path, nil
}

// DeleteFile removes a template by relative path.
func (c *Client) DeleteFile(relPath string) error {
	u, err := url.Parse(c.api("/api/v1/file"))
	if err != nil {
		return err
	}
	q := u.Query()
	q.Set("path", relPath)
	u.RawQuery = q.Encode()
	req, err := http.NewRequest(http.MethodDelete, u.String(), nil)
	if err != nil {
		return err
	}
	resp, err := c.do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("delete file: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return nil
}

// Render executes the template on the server and returns plain text.
func (c *Client) Render(template string, resume config.Resume, overrides map[string]string) (string, error) {
	payload := map[string]interface{}{
		"template":  template,
		"resume":    resume,
		"overrides": overrides,
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.api("/api/v1/render"), bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	b, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("render: %s: %s", resp.Status, strings.TrimSpace(string(b)))
	}
	return string(b), nil
}

// ExportPDF calls POST /api/v1/export/pdf and returns PDF bytes and a suggested filename.
func (c *Client) ExportPDF(title, text string) (pdf []byte, filename string, err error) {
	payload := map[string]string{"title": title, "text": text}
	raw, err := json.Marshal(payload)
	if err != nil {
		return nil, "", err
	}
	req, err := http.NewRequest(http.MethodPost, c.api("/api/v1/export/pdf"), bytes.NewReader(raw))
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", err
	}
	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("export pdf: %s: %s", resp.Status, strings.TrimSpace(string(data)))
	}
	cd := resp.Header.Get("Content-Disposition")
	_, params, err := mime.ParseMediaType(cd)
	if err == nil {
		if fn := params["filename"]; fn != "" {
			filename = fn
		}
	}
	if filename == "" {
		filename = "document.pdf"
	}
	return data, filename, nil
}
