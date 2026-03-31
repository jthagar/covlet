package server

import (
	"fmt"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/gofiber/fiber/v2"

	"github.com/jthagar/covlet/backend/internal/pdf"
	"github.com/jthagar/covlet/backend/internal/render"
	"github.com/jthagar/covlet/backend/internal/templatevars"
	"github.com/jthagar/covlet/backend/pkg/config"
)

func templateExtensionsOK(name string) bool {
	switch strings.ToLower(filepath.Ext(name)) {
	case ".tpl", ".tmpl", ".txt", ".md", ".gohtml", ".html":
		return true
	default:
		return false
	}
}

func isHidden(name string) bool {
	return len(name) > 0 && name[0] == '.'
}

// RootsResponse mirrors a dual-tree layout (templates root plus optional base/ and partials/ subfolders).
type RootsResponse struct {
	TemplatesRoot string `json:"templates_root"`
	Left          string `json:"left"`
	Right         string `json:"right"`
}

func computeRoots(templatesRoot string) (left, right string) {
	left = templatesRoot
	right = templatesRoot
	base := filepath.Join(templatesRoot, "base")
	if fi, err := os.Stat(base); err == nil && fi.IsDir() {
		left = base
	}
	partials := filepath.Join(templatesRoot, "partials")
	if fi, err := os.Stat(partials); err == nil && fi.IsDir() {
		right = partials
	}
	return left, right
}

func handleRoots(c *fiber.Ctx) error {
	root := config.TemplatesDir()
	l, r := computeRoots(root)
	return c.JSON(RootsResponse{
		TemplatesRoot: root,
		Left:          l,
		Right:         r,
	})
}

func handleListFiles(c *fiber.Ctx) error {
	templatesRoot, err := config.EnsureTemplatesDir()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	var paths []string
	err = filepath.WalkDir(templatesRoot, func(path string, d os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if isHidden(d.Name()) {
			return nil
		}
		if !templateExtensionsOK(path) {
			return nil
		}
		rel, err := filepath.Rel(templatesRoot, path)
		if err != nil {
			return err
		}
		paths = append(paths, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.JSON(fiber.Map{"files": paths})
}

func handleGetFile(c *fiber.Ctx) error {
	rel := strings.TrimSpace(c.Query("path"))
	if rel == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing path query")
	}
	templatesRoot, err := config.EnsureTemplatesDir()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	full, err := JoinUnder(templatesRoot, rel)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	b, err := os.ReadFile(full)
	if err != nil {
		if os.IsNotExist(err) {
			return fiber.NewError(fiber.StatusNotFound, "file not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.Send(b)
}

func handlePutFile(c *fiber.Ctx) error {
	rel := strings.TrimSpace(c.Query("path"))
	if rel == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing path query")
	}
	templatesRoot, err := config.EnsureTemplatesDir()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	full, err := JoinUnder(templatesRoot, rel)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	body := c.Body()
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := os.WriteFile(full, body, 0o644); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type createFileReq struct {
	Parent   string `json:"parent"`
	Filename string `json:"filename"`
}

func handlePostFile(c *fiber.Ctx) error {
	var req createFileReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	name := strings.TrimSpace(req.Filename)
	if name == "" {
		return fiber.NewError(fiber.StatusBadRequest, "filename required")
	}
	if strings.ContainsAny(name, `/\`) {
		return fiber.NewError(fiber.StatusBadRequest, "filename must not contain path separators")
	}
	if !strings.Contains(name, ".") {
		name += ".tpl"
	}
	templatesRoot, err := config.EnsureTemplatesDir()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	parent := strings.TrimSpace(req.Parent)
	full, err := JoinUnder(templatesRoot, filepath.Join(parent, name))
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if _, err := os.Stat(full); err == nil {
		return fiber.NewError(fiber.StatusConflict, "file already exists")
	}
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	if err := os.WriteFile(full, []byte(""), 0o644); err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	rel, _ := filepath.Rel(templatesRoot, full)
	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"path": filepath.ToSlash(rel),
	})
}

func handleDeleteFile(c *fiber.Ctx) error {
	rel := strings.TrimSpace(c.Query("path"))
	if rel == "" {
		return fiber.NewError(fiber.StatusBadRequest, "missing path query")
	}
	templatesRoot, err := config.EnsureTemplatesDir()
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	full, err := JoinUnder(templatesRoot, rel)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	if err := os.Remove(full); err != nil {
		if os.IsNotExist(err) {
			return fiber.NewError(fiber.StatusNotFound, "file not found")
		}
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	return c.SendStatus(fiber.StatusNoContent)
}

type varsReq struct {
	Content string `json:"content"`
}

func handleVars(c *fiber.Ctx) error {
	var req varsReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	vars := templatevars.ParseTopLevelVars(req.Content)
	return c.JSON(fiber.Map{"vars": vars})
}

type renderReq struct {
	Template  string            `json:"template"`
	Resume    config.Resume     `json:"resume"`
	Overrides map[string]string `json:"overrides"`
}

func handleRender(c *fiber.Ctx) error {
	var req renderReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	t, err := template.New("body").Parse(req.Template)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "template parse: "+err.Error())
	}
	data := templatevars.ApplyOverrides(req.Resume, req.Overrides)
	out, err := render.Render(t, data)
	if err != nil {
		return fiber.NewError(fiber.StatusBadRequest, "template execute: "+err.Error())
	}
	c.Set("Content-Type", "text/plain; charset=utf-8")
	return c.SendString(out)
}

type pdfReq struct {
	Title string `json:"title"`
	Text  string `json:"text"`
}

func handlePDF(c *fiber.Ctx) error {
	var req pdfReq
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(fiber.StatusBadRequest, err.Error())
	}
	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = "Document"
	}
	pdfBytes, err := pdf.TextToPDF(title, req.Text)
	if err != nil {
		return fiber.NewError(fiber.StatusInternalServerError, err.Error())
	}
	base := templatevars.SanitizeFileName(title)
	if base == "" {
		base = "document"
	}
	fn := base + ".pdf"
	c.Set("Content-Type", "application/pdf")
	cd := mime.FormatMediaType("attachment", map[string]string{"filename": fn})
	if cd == "" {
		c.Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", fn))
	} else {
		c.Set("Content-Disposition", cd)
	}
	return c.Send(pdfBytes)
}

func handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok"})
}

func registerAPI(app *fiber.App) {
	api := app.Group("/api/v1")
	api.Get("/health", handleHealth)
	api.Get("/roots", handleRoots)
	api.Get("/files", handleListFiles)
	api.Get("/file", handleGetFile)
	api.Put("/file", handlePutFile)
	api.Post("/file", handlePostFile)
	api.Delete("/file", handleDeleteFile)
	api.Post("/vars", handleVars)
	api.Post("/render", handleRender)
	api.Post("/export/pdf", handlePDF)
}
