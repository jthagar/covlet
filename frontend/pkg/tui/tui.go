package tui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	"github.com/charmbracelet/lipgloss"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jthagar/covlet/backend/pkg/config"
	"github.com/jthagar/covlet/frontend/pkg/client"
)

// Run starts the Bubble Tea UI.
func Run(apiURL string, resume config.Resume) error {
	cl := client.New(apiURL)

	sp := textinput.New()
	sp.Placeholder = filepath.Join("~", "Downloads")
	if d := defaultDownloads(); d != "" {
		sp.SetValue(d)
	}
	sp.CharLimit = 4096

	oc := textinput.New()
	oc.Placeholder = "Company to apply to"
	oc.SetValue(resume.CompanyToApplyTo)
	oc.CharLimit = 512

	or := textinput.New()
	or.Placeholder = "Role"
	or.SetValue(resume.RoleToApplyTo)
	or.CharLimit = 512

	nf := textinput.New()
	nf.Placeholder = "filename.tpl"
	nf.CharLimit = 256

	ed := textarea.New()
	ed.Placeholder = "Select a template or paste content. Tab: list/editor · Ctrl+S save · Ctrl+R render · Ctrl+P PDF · Ctrl+O overrides"
	ed.ShowLineNumbers = true
	ed.Prompt = ""
	ed.Focus()

	pv := viewport.New(80, 12)
	pv.SetContent("(render preview — Ctrl+R)")

	m := model{
		cl:           cl,
		resume:       resume,
		editor:       ed,
		preview:      pv,
		savePath:     sp,
		overrideCo:   oc,
		overrideRo:   or,
		newFile:      nf,
		overrideFocus: 0,
	}
	p := tea.NewProgram(m, tea.WithAltScreen())
	_, err := p.Run()
	return err
}

func defaultDownloads() string {
	h, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(h, "Downloads")
}

type screen int

const (
	screenEdit screen = iota
	screenSavePDF
	screenOverrides
	screenNewFile
	screenDeleteConfirm
)

type model struct {
	cl     *client.Client
	resume config.Resume

	files     []string
	sel       int
	path      string
	focusList bool

	editor       textarea.Model
	preview      viewport.Model
	previewText  string
	savePath     textinput.Model
	screen       screen

	overrideCo    textinput.Model
	overrideRo    textinput.Model
	overrideFocus int

	newFile       textinput.Model
	pendingDelete string

	pdfData []byte
	pdfName string

	status string
	width  int
	height int
}

type filesMsg struct {
	files []string
	err   error
}

type healthMsg struct {
	err error
}

type loadedMsg struct {
	path    string
	content []byte
	err     error
}

type renderResultMsg struct {
	text string
	err  error
}

type saveResultMsg struct {
	err error
}

type pdfMsg struct {
	data []byte
	name string
	err  error
}

type createFileMsg struct {
	path string
	err  error
}

type deleteMsg struct {
	err error
}

func (m model) Init() tea.Cmd {
	return tea.Sequence(m.healthCmd(), m.refreshFilesCmd)
}

func (m model) healthCmd() tea.Cmd {
	return func() tea.Msg {
		err := m.cl.Health()
		return healthMsg{err: err}
	}
}

func (m model) refreshFilesCmd() tea.Msg {
	files, err := m.cl.ListFiles()
	return filesMsg{files: files, err: err}
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		half := m.height / 2
		if half < 6 {
			half = 6
		}
		top := m.height - half - 6
		if top < 8 {
			top = 8
		}
		m.editor.SetWidth(max(20, m.width-4))
		m.editor.SetHeight(max(6, top))
		m.preview.Width = max(20, m.width-4)
		m.preview.Height = max(4, half)
		var eCmd, pCmd tea.Cmd
		m.editor, eCmd = m.editor.Update(msg)
		m.preview, pCmd = m.preview.Update(msg)
		return m, tea.Batch(eCmd, pCmd)

	case healthMsg:
		if msg.err != nil {
			m.status = "API: " + msg.err.Error()
		} else {
			if strings.HasPrefix(m.status, "API:") {
				m.status = ""
			}
		}
		return m, nil

	case filesMsg:
		if msg.err != nil {
			m.status = "list: " + msg.err.Error()
			return m, nil
		}
		m.files = msg.files
		if m.status == "" || strings.HasPrefix(m.status, "list:") {
			m.status = fmt.Sprintf("%d template(s)", len(m.files))
		}
		return m, nil

	case loadedMsg:
		if msg.err != nil {
			m.status = "load: " + msg.err.Error()
			return m, nil
		}
		m.path = msg.path
		m.editor.SetValue(string(msg.content))
		m.status = "loaded " + m.path
		return m, nil

	case createFileMsg:
		m.screen = screenEdit
		m.newFile.Blur()
		if msg.err != nil {
			m.status = "create: " + msg.err.Error()
			return m, nil
		}
		m.status = "created " + msg.path
		return m, tea.Sequence(m.refreshFilesCmd, m.loadFileCmd(msg.path))

	case deleteMsg:
		m.screen = screenEdit
		rel := m.pendingDelete
		m.pendingDelete = ""
		if msg.err != nil {
			m.status = "delete: " + msg.err.Error()
			return m, nil
		}
		m.status = "deleted " + rel
		if m.path == rel {
			m.path = ""
			m.editor.SetValue("")
		}
		return m, m.refreshFilesCmd

	case renderResultMsg:
		if msg.err != nil {
			m.status = "render: " + msg.err.Error()
			return m, nil
		}
		m.previewText = msg.text
		m.preview.SetContent(msg.text)
		m.status = "rendered"
		return m, nil

	case saveResultMsg:
		if msg.err != nil {
			m.status = "save: " + msg.err.Error()
			return m, nil
		}
		m.status = "saved " + m.path
		return m, nil

	case pdfMsg:
		if msg.err != nil {
			m.status = "pdf: " + msg.err.Error()
			m.screen = screenEdit
			return m, nil
		}
		m.pdfData = msg.data
		m.pdfName = msg.name
		m.screen = screenSavePDF
		if d := defaultDownloads(); d != "" {
			m.savePath.SetValue(d)
		}
		m.savePath.Focus()
		m.status = "PDF ready — Enter directory to save, Esc cancel"
		return m, textinput.Blink

	case tea.KeyMsg:
		switch m.screen {
		case screenSavePDF:
			return m.updateSavePDF(msg)
		case screenOverrides:
			return m.updateOverrides(msg)
		case screenNewFile:
			return m.updateNewFile(msg)
		case screenDeleteConfirm:
			return m.updateDeleteConfirm(msg)
		}

		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "tab":
			m.focusList = !m.focusList
			if m.focusList {
				m.editor.Blur()
			} else {
				m.editor.Focus()
			}
			return m, nil
		}

		if m.focusList {
			switch msg.String() {
			case "up", "k":
				if m.sel > 0 {
					m.sel--
				}
			case "down", "j":
				if m.sel < len(m.files)-1 {
					m.sel++
				}
			case "enter", " ":
				if len(m.files) == 0 {
					return m, nil
				}
				p := m.files[m.sel]
				return m, m.loadFileCmd(p)
			case "r":
				return m, m.refreshFilesCmd
			case "n":
				m.screen = screenNewFile
				m.newFile.SetValue("")
				m.newFile.Focus()
				m.status = "New file — Enter name, Esc cancel"
				return m, textinput.Blink
			case "d":
				if len(m.files) == 0 || m.sel < 0 || m.sel >= len(m.files) {
					return m, nil
				}
				m.pendingDelete = m.files[m.sel]
				m.screen = screenDeleteConfirm
				m.status = fmt.Sprintf("Delete %q? (y/N)", m.pendingDelete)
				return m, nil
			}
			return m, nil
		}

		switch msg.String() {
		case "ctrl+s":
			return m, m.enqueueSave()
		case "ctrl+r":
			return m, m.enqueueRender()
		case "ctrl+p":
			return m, m.enqueueExportPDF()
		case "ctrl+o":
			m.screen = screenOverrides
			m.overrideFocus = 0
			m.overrideRo.Blur()
			m.overrideCo.Focus()
			m.status = "Overrides — Tab switch field, Enter/Esc close"
			return m, textinput.Blink
		}

		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}

	var eCmd, pCmd tea.Cmd
	m.editor, eCmd = m.editor.Update(msg)
	m.preview, pCmd = m.preview.Update(msg)
	return m, tea.Batch(eCmd, pCmd)
}

func (m model) updateSavePDF(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenEdit
		m.savePath.Blur()
		m.pdfData = nil
		m.status = "save cancelled"
		return m, nil
	case "enter":
		return m.finishSavePDF()
	}
	var cmd tea.Cmd
	m.savePath, cmd = m.savePath.Update(msg)
	return m, cmd
}

func (m model) updateOverrides(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc", "enter":
		m.screen = screenEdit
		m.overrideCo.Blur()
		m.overrideRo.Blur()
		m.status = "overrides saved"
		return m, nil
	case "tab":
		if m.overrideFocus == 0 {
			m.overrideFocus = 1
			m.overrideCo.Blur()
			m.overrideRo.Focus()
		} else {
			m.overrideFocus = 0
			m.overrideRo.Blur()
			m.overrideCo.Focus()
		}
		return m, textinput.Blink
	}
	var cmd tea.Cmd
	if m.overrideFocus == 0 {
		m.overrideCo, cmd = m.overrideCo.Update(msg)
	} else {
		m.overrideRo, cmd = m.overrideRo.Update(msg)
	}
	return m, cmd
}

func (m model) updateNewFile(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "esc":
		m.screen = screenEdit
		m.newFile.Blur()
		m.status = "cancelled"
		return m, nil
	case "enter":
		name := strings.TrimSpace(m.newFile.Value())
		if name == "" {
			return m, nil
		}
		return m, m.createFileCmd(name)
	}
	var cmd tea.Cmd
	m.newFile, cmd = m.newFile.Update(msg)
	return m, cmd
}

func (m model) updateDeleteConfirm(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch strings.ToLower(msg.String()) {
	case "y":
		rel := m.pendingDelete
		return m, m.deleteCmd(rel)
	case "n", "esc":
		m.screen = screenEdit
		m.pendingDelete = ""
		m.status = "cancelled"
		return m, nil
	}
	return m, nil
}

func (m model) createFileCmd(filename string) tea.Cmd {
	return func() tea.Msg {
		path, err := m.cl.CreateFile("", filename)
		return createFileMsg{path: path, err: err}
	}
}

func (m model) deleteCmd(rel string) tea.Cmd {
	return func() tea.Msg {
		err := m.cl.DeleteFile(rel)
		return deleteMsg{err: err}
	}
}

func (m model) enqueueSave() tea.Cmd {
	return func() tea.Msg {
		if strings.TrimSpace(m.path) == "" {
			return saveResultMsg{err: fmt.Errorf("no file selected — open a template from the list first")}
		}
		err := m.cl.PutFile(m.path, []byte(m.editor.Value()))
		return saveResultMsg{err: err}
	}
}

func (m model) renderOverrides() map[string]string {
	o := make(map[string]string)
	if v := strings.TrimSpace(m.overrideCo.Value()); v != "" {
		o["CompanyToApplyTo"] = v
	}
	if v := strings.TrimSpace(m.overrideRo.Value()); v != "" {
		o["RoleToApplyTo"] = v
	}
	if len(o) == 0 {
		return nil
	}
	return o
}

func (m model) enqueueRender() tea.Cmd {
	return func() tea.Msg {
		text, err := m.cl.Render(m.editor.Value(), m.resume, m.renderOverrides())
		return renderResultMsg{text: text, err: err}
	}
}

func (m model) enqueueExportPDF() tea.Cmd {
	return func() tea.Msg {
		title := pdfTitle(m.previewText)
		if title == "" {
			title = "Document"
		}
		text := strings.TrimSpace(m.previewText)
		if text == "" {
			text = strings.TrimSpace(m.editor.Value())
		}
		b, name, err := m.cl.ExportPDF(title, text)
		return pdfMsg{data: b, name: name, err: err}
	}
}

func pdfTitle(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return ""
	}
	for _, line := range strings.Split(s, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		return strings.TrimSpace(strings.TrimPrefix(line, "#"))
	}
	return ""
}

func (m model) finishSavePDF() (tea.Model, tea.Cmd) {
	dir, err := expandPath(m.savePath.Value())
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		m.status = err.Error()
		return m, nil
	}
	out := filepath.Join(dir, m.pdfName)
	if err := os.WriteFile(out, m.pdfData, 0o644); err != nil {
		m.status = err.Error()
		return m, nil
	}
	m.screen = screenEdit
	m.savePath.Blur()
	m.pdfData = nil
	m.status = "wrote " + out
	return m, nil
}

func expandPath(p string) (string, error) {
	p = strings.TrimSpace(p)
	if p == "" {
		return "", fmt.Errorf("empty directory")
	}
	if strings.HasPrefix(p, "~/") {
		h, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		p = filepath.Join(h, p[2:])
	}
	return filepath.Abs(p)
}

func (m model) loadFileCmd(rel string) tea.Cmd {
	return func() tea.Msg {
		b, err := m.cl.GetFile(rel)
		return loadedMsg{path: rel, content: b, err: err}
	}
}

var (
	titleStyle = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("62"))
	boxStyle   = lipgloss.NewStyle().Border(lipgloss.RoundedBorder())
)

func (m model) View() string {
	if m.width == 0 {
		m.width = 80
	}
	w := m.width - 4

	var b strings.Builder
	b.WriteString(titleStyle.Render("covlet"))
	b.WriteString("\n")
	if m.path != "" {
		b.WriteString("file: " + m.path + "\n")
	} else {
		b.WriteString("file: (none)\n")
	}

	listLines := min(6, max(1, len(m.files)))
	if m.focusList {
		b.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("[list]") + "\n")
	} else {
		b.WriteString("[list — Tab to focus]\n")
	}
	start := 0
	if m.sel >= listLines {
		start = m.sel - listLines + 1
	}
	for i := 0; i < listLines && start+i < len(m.files); i++ {
		idx := start + i
		line := m.files[idx]
		if idx == m.sel {
			line = "> " + line
		} else {
			line = "  " + line
		}
		b.WriteString(truncate(line, w) + "\n")
	}
	if len(m.files) == 0 {
		b.WriteString("  (no files — n new, or add under templates on server)\n")
	}

	b.WriteString("\neditor" + map[bool]string{true: " *", false: ""}[!m.focusList] + "\n")
	b.WriteString(boxStyle.Width(w).Render(m.editor.View()) + "\n")

	b.WriteString("\npreview\n")
	b.WriteString(boxStyle.Width(w).Render(m.preview.View()) + "\n")

	b.WriteString("\n" + m.status + "\n")
	b.WriteString("Tab list/editor · j/k · Enter · r · n new · d delete · Ctrl+S · Ctrl+R · Ctrl+O overrides · Ctrl+P PDF · Ctrl+C quit\n")

	switch m.screen {
	case screenSavePDF:
		b.WriteString("\n" + titleStyle.Render("Save PDF") + "\n")
		b.WriteString("Directory: " + m.savePath.View() + "\n")
		b.WriteString("(Enter to save, Esc cancel)\n")
	case screenOverrides:
		b.WriteString("\n" + titleStyle.Render("Overrides (render / PDF)") + "\n")
		b.WriteString("Company: " + m.overrideCo.View() + "\n")
		b.WriteString("Role:    " + m.overrideRo.View() + "\n")
		b.WriteString("(Enter or Esc to close, Tab switch)\n")
	case screenNewFile:
		b.WriteString("\n" + titleStyle.Render("New template") + "\n")
		b.WriteString("Filename: " + m.newFile.View() + "\n")
	case screenDeleteConfirm:
		b.WriteString("\n" + titleStyle.Render("Confirm delete") + "\n")
		b.WriteString("(y to confirm, n or Esc cancel)\n")
	}

	return b.String()
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func truncate(s string, maxW int) string {
	if maxW <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= maxW {
		return s
	}
	return string(r[:maxW-1]) + "…"
}
