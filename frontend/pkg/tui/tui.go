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
	"github.com/jthagar/covlet/pkg/tplparse"
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
	ed.Placeholder = "Use {{ .Name }} style fields for dynamic values (right panel). Tab cycles panes. Ctrl+S / Ctrl+R / Ctrl+O / Ctrl+P"
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
		focusZone:    1, // 0=list, 1=editor, 2=preview
	}
	p := tea.NewProgram(m, tea.WithAltScreen(), tea.WithMouseCellMotion())
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

// listBlockLines matches the list section height in View (header + rows or empty hint).
func listBlockLines(files []string) int {
	if len(files) == 0 {
		return 2
	}
	return 1 + min(6, len(files))
}

type screen int

const (
	screenEdit screen = iota
	screenSavePDF
	screenOverrides
	screenNewFile
	screenDeleteConfirm
)

// layoutBodyChrome is lines above the editor inner + between panes + borders, excluding the footer.
// footerReserve(screen) adds status/help or a fixed modal band so total chrome matches the View.
const layoutBodyChrome = 11

func footerReserve(s screen) int {
	switch s {
	case screenSavePDF, screenOverrides, screenNewFile, screenDeleteConfirm:
		return 10
	default:
		return 3
	}
}

func previewLineCount(text string) int {
	t := strings.TrimSpace(text)
	if t == "" {
		return 1
	}
	return strings.Count(text, "\n") + 1
}

func maxPreviewCap(avail int) int {
	minEd, minPv := 6, 4
	if avail < 2 {
		return 1
	}
	if avail <= minEd+minPv {
		return max(1, avail-max(1, avail/2))
	}
	ed := max(minEd, avail*45/100)
	pv := avail - ed
	if pv < minPv {
		pv = minPv
		ed = avail - pv
		if ed < minEd {
			ed = minEd
			pv = avail - ed
			if pv < 1 {
				pv = 1
			}
		}
	}
	return pv
}

// layoutCompute splits vertical space: editor grows when preview needs fewer lines.
// Preview height grows with content up to maxPreviewCap (same cap as the old fixed split).
func layoutCompute(termHeight, termWidth int, files []string, previewLines int, s screen, hasVars bool) (edInnerH, pvInnerH, editorTextW, varsColW int) {
	L := listBlockLines(files)
	avail := termHeight - layoutBodyChrome - L - footerReserve(s)
	if avail < 2 {
		edInnerH, pvInnerH = 1, 1
	} else {
		maxCap := maxPreviewCap(avail)
		minPv := 3
		lines := max(1, previewLines)
		pvInnerH = min(maxCap, max(minPv, lines))
		edInnerH = avail - pvInnerH
		minEd := 6
		if edInnerH < minEd {
			need := minEd - edInnerH
			pvInnerH = max(minPv, pvInnerH-need)
			edInnerH = avail - pvInnerH
		}
	}
	contentW := max(20, termWidth-4)
	varsColW = 0
	if hasVars {
		varsColW = min(38, max(22, termWidth/4))
		if varsColW >= contentW-12 {
			varsColW = max(18, contentW/5)
		}
	}
	editorTextW = contentW - varsColW
	if hasVars {
		editorTextW -= 1
	}
	editorTextW = max(20, editorTextW)
	return edInnerH, pvInnerH, editorTextW, varsColW
}

type tplVarRow struct {
	Name  string
	Input textinput.Model
}

type model struct {
	cl     *client.Client
	resume config.Resume

	files     []string
	sel       int
	path      string
	focusZone int // 0=list, 1=editor, 2=vars (if any), 3=preview; else 2=preview

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

	tplRows       []tplVarRow
	tplVarSig     string
	tplVarFocus   int
	varsScroll    int
	layoutEditorW int
	layoutVarsW   int

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

func (m model) tplVarsEnabled() bool {
	if strings.TrimSpace(m.path) == "" {
		return true
	}
	return strings.HasSuffix(strings.ToLower(m.path), ".tpl")
}

func (m model) varsPanelActive() bool {
	return len(m.tplRows) > 0 && m.tplVarsEnabled()
}

func (m model) maxFocusZone() int {
	if m.varsPanelActive() {
		return 3
	}
	return 2
}

func (m model) normalizeFocusZone() model {
	if mz := m.maxFocusZone(); m.focusZone > mz {
		m.focusZone = mz
	}
	return m
}

func defaultResumeField(r config.Resume, name string) string {
	switch name {
	case "Name":
		return r.Name
	case "Email":
		return r.Email
	case "Phone":
		return r.Phone
	case "Address":
		return r.Address
	case "Website":
		return r.Website
	case "Github":
		return r.Github
	case "CompanyToApplyTo":
		return r.CompanyToApplyTo
	case "RoleToApplyTo":
		return r.RoleToApplyTo
	default:
		return ""
	}
}

func (m model) rebuildTplRowsFromEditor() (model, bool) {
	names := tplparse.ParseTopLevelVars(m.editor.Value())
	sig := strings.Join(names, "\x00")
	if sig == m.tplVarSig && len(names) == len(m.tplRows) {
		return m, false
	}
	prev := map[string]string{}
	for _, row := range m.tplRows {
		prev[row.Name] = row.Input.Value()
	}
	m.tplVarSig = sig
	m.tplRows = nil
	for _, name := range names {
		ti := textinput.New()
		ti.Placeholder = "value"
		ti.CharLimit = 768
		if v, ok := prev[name]; ok {
			ti.SetValue(v)
		} else {
			ti.SetValue(defaultResumeField(m.resume, name))
		}
		m.tplRows = append(m.tplRows, tplVarRow{Name: name, Input: ti})
	}
	if len(m.tplRows) == 0 {
		m.tplVarFocus = 0
	} else if m.tplVarFocus >= len(m.tplRows) {
		m.tplVarFocus = len(m.tplRows) - 1
	}
	m = m.ensureVarsScroll()
	return m, true
}

func (m model) ensureVarsScroll() model {
	if len(m.tplRows) == 0 {
		m.varsScroll = 0
		return m
	}
	if m.tplVarFocus < m.varsScroll {
		m.varsScroll = m.tplVarFocus
	}
	// Match varsPanelView: one line is reserved for the hint; rest are field rows.
	vis := max(1, m.editor.Height()-1)
	if m.tplVarFocus >= m.varsScroll+vis {
		m.varsScroll = m.tplVarFocus - vis + 1
	}
	if m.varsScroll < 0 {
		m.varsScroll = 0
	}
	return m
}

func (m model) relayout() model {
	if m.width <= 0 || m.height <= 0 {
		return m
	}
	pl := previewLineCount(m.previewText)
	hasVars := m.varsPanelActive()
	edH, pvH, edW, vW := layoutCompute(m.height, m.width, m.files, pl, m.screen, hasVars)
	m.layoutEditorW = edW
	m.layoutVarsW = vW
	m.editor.SetWidth(edW)
	m.editor.SetHeight(max(1, edH))
	m.preview.Width = max(20, m.width-4)
	m.preview.Height = max(1, pvH)
	if hasVars {
		inW := vW - 4
		if inW < 6 {
			inW = 6
		}
		for i := range m.tplRows {
			ti := m.tplRows[i].Input
			ti.Width = inW
			m.tplRows[i].Input = ti
		}
	}
	m = m.normalizeFocusZone()
	m = m.syncAllFocus()
	return m
}

func (m model) syncAllFocus() model {
	if m.focusZone == 1 {
		m.editor.Focus()
	} else {
		m.editor.Blur()
	}
	for i := range m.tplRows {
		if m.varsPanelActive() && m.focusZone == 2 && i == m.tplVarFocus {
			m.tplRows[i].Input.Focus()
		} else {
			m.tplRows[i].Input.Blur()
		}
	}
	return m
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
		m, _ = m.rebuildTplRowsFromEditor()
		m = m.relayout()
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
		if m.width > 0 && m.height > 0 {
			m, _ = m.rebuildTplRowsFromEditor()
			m = m.relayout()
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
		m, _ = m.rebuildTplRowsFromEditor()
		m = m.relayout()
		return m, nil

	case createFileMsg:
		m.screen = screenEdit
		m.newFile.Blur()
		if msg.err != nil {
			m.status = "create: " + msg.err.Error()
			m = m.relayout()
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
			m = m.relayout()
			return m, nil
		}
		m.status = "deleted " + rel
		if m.path == rel {
			m.path = ""
			m.editor.SetValue("")
		}
		m, _ = m.rebuildTplRowsFromEditor()
		m = m.relayout()
		return m, m.refreshFilesCmd

	case renderResultMsg:
		if msg.err != nil {
			m.status = "render: " + msg.err.Error()
			return m, nil
		}
		m.previewText = msg.text
		m.preview.SetContent(msg.text)
		m.status = "rendered"
		m = m.relayout()
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
			m = m.relayout()
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
		m = m.relayout()
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
			nz := m.maxFocusZone() + 1
			m.focusZone = (m.focusZone + 1) % nz
			m = m.syncAllFocus()
			return m, nil
		}

		if m.focusZone == 0 {
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
				m = m.relayout()
				return m, textinput.Blink
			case "d":
				if len(m.files) == 0 || m.sel < 0 || m.sel >= len(m.files) {
					return m, nil
				}
				m.pendingDelete = m.files[m.sel]
				m.screen = screenDeleteConfirm
				m.status = fmt.Sprintf("Delete %q? (y/N)", m.pendingDelete)
				m = m.relayout()
				return m, nil
			}
			return m, nil
		}

		if m.varsPanelActive() && m.focusZone == 2 {
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
				m = m.relayout()
				return m, textinput.Blink
			case "up", "k":
				if m.tplVarFocus > 0 {
					m.tplVarFocus--
					m = m.ensureVarsScroll()
					m = m.syncAllFocus()
				}
				return m, nil
			case "down", "j":
				if m.tplVarFocus < len(m.tplRows)-1 {
					m.tplVarFocus++
					m = m.ensureVarsScroll()
					m = m.syncAllFocus()
				}
				return m, nil
			default:
				row := m.tplRows[m.tplVarFocus]
				var cmd tea.Cmd
				row.Input, cmd = row.Input.Update(msg)
				m.tplRows[m.tplVarFocus] = row
				return m, cmd
			}
		}

		if m.focusZone == m.maxFocusZone() {
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
				m = m.relayout()
				return m, textinput.Blink
			default:
				var cmd tea.Cmd
				m.preview, cmd = m.preview.Update(msg)
				return m, cmd
			}
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
			m = m.relayout()
			return m, textinput.Blink
		}

		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		m2, changed := m.rebuildTplRowsFromEditor()
		m = m2
		if changed {
			m = m.relayout()
		}
		return m, cmd

	case tea.MouseMsg:
		if m.screen != screenEdit {
			return m, nil
		}
		if m.focusZone != m.maxFocusZone() {
			return m, nil
		}
		var cmd tea.Cmd
		m.preview, cmd = m.preview.Update(msg)
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
		m = m.relayout()
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
		m = m.relayout()
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
		m = m.relayout()
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
		m = m.relayout()
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
	for _, row := range m.tplRows {
		if v := strings.TrimSpace(row.Input.Value()); v != "" {
			o[row.Name] = v
		}
	}
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
	m = m.relayout()
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
	dimStyle   = lipgloss.NewStyle().Foreground(lipgloss.Color("245"))
)

// varsPanelView returns exactly targetLines rows so the vars box matches the editor
// inner height; otherwise JoinHorizontal misaligns rounded borders when the column is shorter.
func (m model) varsPanelView(targetLines, contentW int) string {
	if len(m.tplRows) == 0 || contentW < 8 {
		return ""
	}
	if targetLines < 1 {
		targetLines = 1
	}
	labelW := min(12, max(6, contentW/3))
	inpW := contentW - labelW - 2
	if inpW < 4 {
		inpW = 4
	}
	hint := dimStyle.Render(truncate("↑/↓ field · Tab next pane", contentW))
	if targetLines == 1 {
		return hint
	}
	dataLines := targetLines - 1
	end := min(len(m.tplRows), m.varsScroll+dataLines)
	var lines []string
	for i := m.varsScroll; i < end; i++ {
		row := m.tplRows[i]
		prefix := " "
		if i == m.tplVarFocus && m.varsPanelActive() && m.focusZone == 2 {
			prefix = ">"
		}
		label := truncate(row.Name+":", labelW)
		line := lipgloss.JoinHorizontal(lipgloss.Top,
			lipgloss.NewStyle().Width(1).Render(prefix),
			lipgloss.NewStyle().Width(labelW).Render(label),
			row.Input.View(),
		)
		lines = append(lines, truncate(line, contentW))
	}
	for len(lines) < dataLines {
		lines = append(lines, "")
	}
	if len(lines) > dataLines {
		lines = lines[:dataLines]
	}
	lines = append(lines, hint)
	for len(lines) < targetLines {
		lines = append(lines, "")
	}
	if len(lines) > targetLines {
		lines = lines[:targetLines]
	}
	return strings.Join(lines, "\n")
}

func (m model) renderFooter() string {
	h := footerReserve(m.screen)
	var lines []string
	switch m.screen {
	case screenEdit:
		help := "Tab panes · j/k list · Enter · r · n · d · Ctrl+S · Ctrl+R · Ctrl+O resume · Ctrl+P PDF · Ctrl+C quit"
		if strings.TrimSpace(m.previewText) != "" {
			help += fmt.Sprintf(" · preview %.0f%%", m.preview.ScrollPercent()*100)
		}
		lines = []string{m.status, help}
	case screenSavePDF:
		lines = []string{
			titleStyle.Render("Save PDF"),
			"Directory: " + m.savePath.View(),
			"(Enter to save, Esc cancel)",
		}
	case screenOverrides:
		lines = []string{
			titleStyle.Render("Overrides (render / PDF)"),
			"Company: " + m.overrideCo.View(),
			"Role:    " + m.overrideRo.View(),
			"(Enter or Esc to close, Tab switch)",
		}
	case screenNewFile:
		lines = []string{
			titleStyle.Render("New template"),
			"Filename: " + m.newFile.View(),
		}
	case screenDeleteConfirm:
		lines = []string{
			titleStyle.Render("Confirm delete"),
			"(y to confirm, n or Esc cancel)",
		}
	}
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}
	return strings.Join(lines, "\n")
}

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
	if m.focusZone == 0 {
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

	edStar := map[bool]string{true: " *", false: ""}[m.focusZone == 1]
	b.WriteString("\neditor" + edStar)
	if m.varsPanelActive() {
		b.WriteString(dimStyle.Render("  │  ") + lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("{{…}} fields") +
			map[bool]string{true: " *", false: ""}[m.focusZone == 2])
	}
	b.WriteString("\n")

	edW := m.layoutEditorW
	vW := m.layoutVarsW
	if edW == 0 && m.height > 0 {
		has := len(m.tplRows) > 0 && m.tplVarsEnabled()
		_, _, edW, vW = layoutCompute(m.height, m.width, m.files, previewLineCount(m.previewText), m.screen, has)
	}
	if m.varsPanelActive() && vW > 0 {
		innerVars := vW - 2
		if innerVars < 6 {
			innerVars = 6
		}
		left := boxStyle.Width(edW).Render(m.editor.View())
		right := boxStyle.Width(vW).Render(m.varsPanelView(m.editor.Height(), innerVars))
		b.WriteString(lipgloss.JoinHorizontal(lipgloss.Top, left, " ", right) + "\n")
	} else {
		b.WriteString(boxStyle.Width(w).Render(m.editor.View()) + "\n")
	}

	pvStar := map[bool]string{true: " *", false: ""}[m.focusZone == m.maxFocusZone()]
	b.WriteString("\n\npreview" + pvStar + "\n")
	b.WriteString(boxStyle.Width(w).Render(m.preview.View()) + "\n")

	b.WriteString("\n" + m.renderFooter() + "\n")

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
