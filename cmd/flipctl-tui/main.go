package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
	"github.com/flipperdevices/flipctl/internal/tuirender"
	"github.com/flipperdevices/flipctl/internal/viewdoc"
)

type focusPane string

const (
	focusDocument focusPane = "document"
	focusActions  focusPane = "actions"
)

type model struct {
	base         string
	sessionID    string
	viewDoc      *viewdoc.Document
	focus        focusPane
	blockCursor  int
	itemCursor   int
	actionCursor int
	fieldCursor  int
	editBuffers  map[string]string
	connected    bool
	err          error
	notice       string
	width        int
	height       int
	detail       viewport.Model
}

type loaded struct {
	sessionID string
	viewDoc   *viewdoc.Document
}
type statusMsg string
type sseChangedMsg struct{}
type sseStatusMsg string

type tuiEvent struct {
	Type      string `json:"type"`
	SessionID string `json:"sessionId"`
	Revision  int    `json:"revision"`
}

var (
	titleStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("14"))
	blockStyle    = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("15"))
	mutedStyle    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	badStyle      = lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Bold(true)
	ruleStyle     = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("0")).Background(lipgloss.Color("14"))
)

func main() {
	base := flag.String("addr", "http://localhost:8080", "FlipCTL daemon base URL")
	renderViewDoc := flag.String("render-viewdoc", "", "render a ViewDocument JSON file noninteractively and exit")
	renderWidth := flag.Int("render-width", 80, "fixed width for --render-viewdoc")
	renderHeight := flag.Int("render-height", 40, "fixed height for --render-viewdoc; 0 disables clipping")
	renderColor := flag.String("render-color", "plain", "color profile for --render-viewdoc: plain or ansi")
	flag.Parse()
	if *renderViewDoc != "" {
		if err := renderFixture(*renderViewDoc, *renderWidth, *renderHeight, *renderColor); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		return
	}
	m := model{base: *base, focus: focusDocument, notice: "Ready. Canonical ViewDocument session mode.", editBuffers: map[string]string{}}
	if _, err := tea.NewProgram(m, tea.WithAltScreen()).Run(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}

func (m model) Init() tea.Cmd { return tea.Batch(m.refresh, m.watchEvents) }

func (m model) refresh() tea.Msg {
	id := m.sessionID
	if id == "" {
		ss, err := createCanonicalSession(m.base)
		if err != nil {
			return err
		}
		id = ss.ID
	}
	doc, err := fetchCanonicalView(m.base, id)
	if err != nil {
		return err
	}
	return loaded{sessionID: id, viewDoc: doc}
}

type canonicalSession struct {
	ID        string `json:"id"`
	SessionID string `json:"sessionId"`
}

func createCanonicalSession(base string) (canonicalSession, error) {
	resp, err := http.Post(base+"/api/v1/sessions", "application/json", bytes.NewReader(nil))
	if err != nil {
		return canonicalSession{}, err
	}
	defer resp.Body.Close()
	var ss canonicalSession
	if err := json.NewDecoder(resp.Body).Decode(&ss); err != nil {
		return canonicalSession{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return canonicalSession{}, fmt.Errorf("create session failed: HTTP %d", resp.StatusCode)
	}
	if ss.ID == "" {
		ss.ID = ss.SessionID
	}
	if ss.ID == "" {
		return canonicalSession{}, fmt.Errorf("create session response missing session id")
	}
	return ss, nil
}

func fetchCanonicalView(base, sessionID string) (*viewdoc.Document, error) {
	var doc viewdoc.Document
	if err := getJSON(base+"/api/v1/sessions/"+sessionID+"/view", &doc); err != nil {
		return nil, err
	}
	if doc.APIVersion != viewdoc.CanonicalAPIVersion {
		return nil, fmt.Errorf("unexpected ViewDocument apiVersion %q", doc.APIVersion)
	}
	return &doc, nil
}

func getJSON(url string, v any) error {
	resp, err := http.Get(url)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return fmt.Errorf("GET %s failed: HTTP %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(v)
}

func (m model) watchEvents() tea.Msg {
	resp, err := http.Get(m.base + "/api/v1/events")
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	scanner := bufio.NewScanner(resp.Body)
	var data strings.Builder
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "data:") {
			data.WriteString(strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			continue
		}
		if line == "" && data.Len() > 0 {
			var e tuiEvent
			if json.Unmarshal([]byte(data.String()), &e) == nil && e.Type == "view.changed" && (m.sessionID == "" || e.SessionID == "" || e.SessionID == m.sessionID) {
				return sseChangedMsg{}
			}
			data.Reset()
		}
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return sseStatusMsg("event stream closed")
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch x := msg.(type) {
	case loaded:
		m.sessionID, m.viewDoc, m.connected, m.err = x.sessionID, x.viewDoc, true, nil
		m.syncEditBuffers()
		m.clampCursors()
	case sseChangedMsg:
		m.notice = "view.changed received; refreshing canonical view"
		return m, tea.Batch(m.refresh, m.watchEvents)
	case sseStatusMsg:
		m.connected = false
		m.notice = string(x)
		return m, m.watchEvents
	case statusMsg:
		m.notice = string(x)
		return m, nil
	case error:
		m.err = x
		m.connected = false
	case tea.WindowSizeMsg:
		m.width, m.height = x.Width, x.Height
		m.resizeDetail()
	case tea.KeyMsg:
		switch x.String() {
		case "q", "ctrl+c":
			return m, tea.Quit
		case "tab", "right":
			if m.focus == focusDocument {
				m.focus = focusActions
			} else {
				m.focus = focusDocument
			}
		case "shift+tab", "left":
			if m.focus == focusActions {
				m.focus = focusDocument
			} else {
				m.focus = focusActions
			}
		case "up", "k":
			m.move(-1)
		case "down", "j":
			m.move(1)
		case "r":
			return m, m.refresh
		case "backspace":
			m.backspaceEdit()
		case "enter", "s":
			return m, m.invokeSelectedAction
		case "c":
			return m, m.invokeActionByID("job.cancel")
		}
		if len(x.String()) == 1 && !x.Alt {
			m.appendEdit(x.String())
		}
	}
	return m, nil
}

func (m model) invokeSelectedAction() tea.Msg {
	if m.viewDoc == nil {
		return statusMsg("no canonical view loaded")
	}
	if act, ok := m.selectedAction(); ok {
		return m.invokeViewAction(act)
	}
	return statusMsg("no enabled action selected")
}
func (m model) invokeActionByID(id string) tea.Cmd {
	return func() tea.Msg { return m.invokeAction(id) }
}

func (m model) invokeAction(actionID string) tea.Msg {
	action, ok := m.actionByID(actionID)
	if !ok || !action.Enabled {
		return statusMsg("no enabled action selected")
	}
	return m.invokeViewAction(action)
}

func (m model) invokeViewAction(action viewdoc.Action) tea.Msg {
	if m.viewDoc == nil || m.sessionID == "" {
		return statusMsg("no canonical session loaded")
	}
	body := map[string]any{"actionId": action.ID, "viewId": m.viewDoc.ViewID, "revision": m.viewDoc.Revision}
	if len(action.Params) > 0 {
		params := make(map[string]string, len(action.Params))
		for k, v := range action.Params {
			params[k] = v
		}
		body["params"] = params
	}
	if action.ID == "form.update" {
		body["values"] = m.formValues()
	}
	b, _ := json.Marshal(body)
	resp, err := http.Post(m.base+"/api/v1/sessions/"+m.sessionID+"/actions", "application/json", bytes.NewReader(b))
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusConflict {
		doc, err := fetchCanonicalView(m.base, m.sessionID)
		if err != nil {
			return err
		}
		return loaded{sessionID: m.sessionID, viewDoc: doc}
	}
	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		var p apiProblem
		_ = json.NewDecoder(resp.Body).Decode(&p)
		return statusMsg(formatProblem(p, resp.StatusCode))
	}
	var doc viewdoc.Document
	if err := json.NewDecoder(resp.Body).Decode(&doc); err != nil {
		return err
	}
	return loaded{sessionID: m.sessionID, viewDoc: &doc}
}

func (m model) View() string {
	if m.width <= 0 {
		m.width = 120
	}
	if m.height <= 0 {
		m.height = 36
	}
	if m.err != nil {
		return fitLines([]string{truncateCell(badStyle.Render("error: "+m.err.Error()), m.width)}, m.width, m.height)
	}
	top := dashboardLine(m, m.width)
	notice := truncateCell("Notice: "+m.notice, m.width)
	footer := footerLine(m, m.width)
	bodyH := max(1, m.height-5)
	body := m.renderBody(m.width, bodyH)
	lines := []string{top, ruleLine(m.width), notice, ruleLine(m.width)}
	lines = append(lines, padLinesPreserveSpaces(strings.Split(strings.TrimRight(body, "\n"), "\n"), m.width, bodyH)...)
	lines = append(lines, footer)
	return fitLines(lines, m.width, m.height)
}

func (m model) renderBody(width, height int) string {
	docW := max(40, width-32)
	actW := max(24, width-docW-1)
	if width < 90 {
		docW, actW = width, width
	}
	m.detail.Width, m.detail.Height = max(1, docW-4), max(1, height-3)
	m.detail.SetContent(renderDocumentContent(m, m.detail.Width))
	docPane := pane("Canonical ViewDocument", m.focus == focusDocument, docW, height, m.detail.View())
	if width < 90 {
		if m.focus == focusActions {
			return pane("Actions", true, width, height, renderActions(m, max(1, width-4), max(1, height-3)))
		}
		return docPane
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, docPane, " ", pane("Actions", m.focus == focusActions, actW, height, renderActions(m, max(1, actW-4), max(1, height-3))))
}

func renderDocumentContent(m model, width int) string {
	if m.viewDoc == nil {
		return "Loading canonical ViewDocument…"
	}
	r := strings.TrimRight(tuirender.RenderDocument(*m.viewDoc, tuirender.Options{Width: width, Height: 0, ColorProfile: tuirender.Plain}), "\n")
	if sb := renderListSelection(m, width); sb != "" {
		r += "\n\n" + sb
	}
	if fb := renderFormEditor(m, width); fb != "" {
		r += "\n\n" + fb
	}
	return r
}

func renderListSelection(m model, width int) string {
	items := m.listItems()
	if len(items) == 0 {
		return ""
	}
	lines := []string{"Selectable list items:"}
	for i, item := range items {
		mark := " "
		if i == m.itemCursor && m.focus == focusDocument {
			mark = ">"
		}
		actionHint := ""
		if item.Action != nil && item.Action.Enabled {
			actionHint = " · enter " + item.Action.Label
		}
		lines = append(lines, truncateCell(fmt.Sprintf("%s %s [%s]%s", mark, item.Label, item.Value, actionHint), width))
	}
	return strings.Join(lines, "\n")
}

func renderFormEditor(m model, width int) string {
	fields := m.formFields()
	if len(fields) == 0 {
		return ""
	}
	lines := []string{"Editable form fields:"}
	for i, f := range fields {
		mark := " "
		if i == m.fieldCursor && m.focus == focusDocument {
			mark = ">"
		}
		lines = append(lines, truncateCell(fmt.Sprintf("%s %s: %s", mark, f.Label, m.editBuffers[f.ID]), width))
	}
	return strings.Join(lines, "\n")
}
func renderActions(m model, width, height int) string {
	if m.viewDoc == nil {
		return "No actions."
	}
	var lines []string
	for i, a := range m.actions() {
		mark := " "
		if i == m.actionCursor {
			mark = ">"
		}
		state := ""
		if !a.Enabled {
			state = " disabled"
		}
		line := truncateCell(fmt.Sprintf("%s %s [%s]%s", mark, a.Label, a.ID, state), width)
		if i == m.actionCursor && m.focus == focusActions {
			line = selectedStyle.Render(line)
		}
		lines = append(lines, line)
	}
	if len(lines) == 0 {
		lines = []string{"No offered actions."}
	}
	return strings.Join(tailVisible(lines, height), "\n")
}

func dashboardLine(m model, width int) string {
	view := "loading"
	rev := 0
	if m.viewDoc != nil {
		view = m.viewDoc.ViewID
		rev = m.viewDoc.Revision
	}
	conn := "offline"
	if m.connected {
		conn = "live"
	}
	return truncateCell(titleStyle.Render(fmt.Sprintf("FlipCTL TUI │ session %s │ view=%s │ rev=%d │ %s │ focus %s", empty(m.sessionID, "-"), view, rev, conn, m.focus)), width)
}
func footerLine(m model, width int) string {
	return truncateCell(mutedStyle.Render("q quit  r refresh  tab focus  j/k move  text edit  backspace  enter action  c cancel"), width)
}
func ruleLine(width int) string { return ruleStyle.Render(strings.Repeat("─", max(0, width))) }

func (m *model) syncEditBuffers() {
	if m.editBuffers == nil {
		m.editBuffers = map[string]string{}
	}
	for _, f := range m.formFields() {
		if _, ok := m.editBuffers[f.ID]; !ok {
			m.editBuffers[f.ID] = f.Default
		}
	}
}
func (m model) formFields() []viewdoc.FormField {
	if m.viewDoc == nil {
		return nil
	}
	for _, b := range m.viewDoc.Blocks {
		if b.Kind == "form" {
			return b.Fields
		}
	}
	return nil
}
func (m model) formValues() map[string]string {
	vals := map[string]string{}
	for _, f := range m.formFields() {
		vals[f.ID] = m.editBuffers[f.ID]
	}
	return vals
}
func (m *model) move(delta int) {
	if m.focus == focusActions {
		m.actionCursor = clamp(m.actionCursor+delta, 0, max(0, len(m.actions())-1))
		return
	}
	if len(m.formFields()) > 0 {
		m.fieldCursor = clamp(m.fieldCursor+delta, 0, len(m.formFields())-1)
		return
	}
	m.itemCursor = clamp(m.itemCursor+delta, 0, max(0, len(m.listItems())-1))
}
func (m *model) clampCursors() {
	m.actionCursor = clamp(m.actionCursor, 0, max(0, len(m.actions())-1))
	m.itemCursor = clamp(m.itemCursor, 0, max(0, len(m.listItems())-1))
	m.fieldCursor = clamp(m.fieldCursor, 0, max(0, len(m.formFields())-1))
}
func (m model) actions() []viewdoc.Action {
	if m.viewDoc == nil {
		return nil
	}
	acts := make([]viewdoc.Action, 0, len(m.viewDoc.Actions)+1)
	if itemAction, ok := m.selectedListAction(); ok {
		acts = append(acts, itemAction)
	}
	acts = append(acts, m.viewDoc.Actions...)
	return acts
}
func (m model) selectedAction() (viewdoc.Action, bool) {
	acts := m.actions()
	if len(acts) == 0 || m.actionCursor < 0 || m.actionCursor >= len(acts) {
		return viewdoc.Action{}, false
	}
	if !acts[m.actionCursor].Enabled {
		return viewdoc.Action{}, false
	}
	return acts[m.actionCursor], true
}
func (m model) actionByID(id string) (viewdoc.Action, bool) {
	for _, a := range m.actions() {
		if a.ID == id {
			return a, true
		}
	}
	return viewdoc.Action{}, false
}
func (m model) listItems() []viewdoc.ListItem {
	if m.viewDoc == nil {
		return nil
	}
	for _, b := range m.viewDoc.Blocks {
		if b.Kind == "list" {
			return b.Items
		}
	}
	return nil
}
func (m model) selectedListItem() (viewdoc.ListItem, bool) {
	items := m.listItems()
	if len(items) == 0 || m.itemCursor < 0 || m.itemCursor >= len(items) {
		return viewdoc.ListItem{}, false
	}
	return items[m.itemCursor], true
}
func (m model) selectedListAction() (viewdoc.Action, bool) {
	item, ok := m.selectedListItem()
	if !ok || item.Action == nil {
		return viewdoc.Action{}, false
	}
	return *item.Action, true
}
func (m *model) appendEdit(s string) {
	fs := m.formFields()
	if len(fs) == 0 || m.fieldCursor >= len(fs) {
		return
	}
	id := fs[m.fieldCursor].ID
	m.editBuffers[id] += s
}
func (m *model) backspaceEdit() {
	fs := m.formFields()
	if len(fs) == 0 || m.fieldCursor >= len(fs) {
		return
	}
	id := fs[m.fieldCursor].ID
	if v := m.editBuffers[id]; v != "" {
		m.editBuffers[id] = v[:len(v)-1]
	}
}

func (m *model) resizeDetail() {
	m.detail.Width = max(1, m.width-4)
	m.detail.Height = max(1, m.height-8)
	m.detail.SetContent(renderDocumentContent(*m, m.detail.Width))
}
func truncateCell(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = strings.ReplaceAll(s, "\n", " ")
	s = strings.Join(strings.Fields(s), " ")
	if ansi.StringWidth(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "…")
}
func fitLines(lines []string, width, height int) string {
	out := make([]string, 0, len(lines))
	for _, l := range lines {
		out = append(out, truncateLinePreserveSpaces(l, width))
	}
	for len(out) < height {
		out = append(out, "")
	}
	if len(out) > height {
		out = out[:height]
	}
	return strings.Join(out, "\n") + "\n"
}
func truncateLinePreserveSpaces(s string, width int) string {
	if width <= 0 {
		return ""
	}
	s = strings.ReplaceAll(s, "\n", " ")
	if ansi.StringWidth(s) <= width {
		return s
	}
	return ansi.Truncate(s, width, "…")
}
func padLinesPreserveSpaces(lines []string, width, height int) []string {
	out := make([]string, 0, height)
	for _, l := range lines {
		if len(out) >= height {
			break
		}
		out = append(out, padCellPreserveSpaces(l, width))
	}
	for len(out) < height {
		out = append(out, strings.Repeat(" ", max(0, width)))
	}
	return out
}
func padCellPreserveSpaces(s string, width int) string {
	s = truncateLinePreserveSpaces(s, width)
	if pad := width - ansi.StringWidth(s); pad > 0 {
		s += strings.Repeat(" ", pad)
	}
	return s
}
func pane(title string, focused bool, width, height int, content string) string {
	if width <= 0 || height <= 0 {
		return ""
	}
	innerW := max(0, width-2)
	innerH := max(0, height-2)
	contentLines := []string{blockStyle.Render(focusLabel(title, focused))}
	if strings.TrimSpace(content) != "" {
		contentLines = append(contentLines, strings.Split(strings.TrimRight(content, "\n"), "\n")...)
	}
	if len(contentLines) > innerH {
		contentLines = contentLines[:innerH]
	}
	out := []string{"╭" + strings.Repeat("─", innerW) + "╮"}
	for _, line := range contentLines {
		out = append(out, "│"+padCellPreserveSpaces(line, innerW)+"│")
	}
	for len(out) < height-1 {
		out = append(out, "│"+strings.Repeat(" ", innerW)+"│")
	}
	out = append(out, "╰"+strings.Repeat("─", innerW)+"╯")
	return strings.Join(out, "\n")
}
func tailVisible(lines []string, height int) []string {
	if height <= 0 {
		return nil
	}
	if len(lines) <= height {
		return lines
	}
	return lines[:height]
}
func focusLabel(label string, focused bool) string {
	if focused {
		return "◆ " + label
	}
	return label
}
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
func clamp(v, lo, hi int) int {
	if hi < lo {
		return lo
	}
	if v < lo {
		return lo
	}
	if v > hi {
		return hi
	}
	return v
}
func empty(s, f string) string {
	if s == "" {
		return f
	}
	return s
}

type apiProblem struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Field   string `json:"field"`
}

func formatProblem(p apiProblem, status int) string {
	code := p.Code
	if code == "" {
		code = fmt.Sprintf("http_%d", status)
	}
	msg := p.Message
	if msg == "" {
		msg = "request failed"
	}
	if p.Field != "" {
		return fmt.Sprintf("%s: %s (%s)", code, msg, p.Field)
	}
	return fmt.Sprintf("%s: %s", code, msg)
}

func renderFixture(path string, width, height int, color string) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc viewdoc.Document
	if err := json.Unmarshal(b, &doc); err != nil {
		return err
	}
	profile := tuirender.ColorProfile(color)
	if profile != tuirender.ANSI {
		profile = tuirender.Plain
	}
	fmt.Print(tuirender.RenderDocument(doc, tuirender.Options{Width: width, Height: height, ColorProfile: profile}))
	return nil
}
