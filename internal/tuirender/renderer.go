package tuirender

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/flipperdevices/flipctl/internal/viewdoc"
)

// ColorProfile selects deterministic terminal styling for snapshot tests and
// noninteractive evidence capture.
type ColorProfile string

const (
	Plain ColorProfile = "plain"
	ANSI  ColorProfile = "ansi"
)

// Options fixes the terminal surface so rendering is deterministic.
type Options struct {
	Width        int
	Height       int
	ColorProfile ColorProfile
}

// RenderDocument renders the canonical generic ViewDocument into deterministic
// terminal text. The renderer intentionally switches only on generic block
// kinds from the public contract and does not know about Ping/Nmap parser
// fields or app-specific public models.
func RenderDocument(doc viewdoc.Document, opts Options) string {
	if opts.Width <= 0 {
		opts.Width = 80
	}
	if opts.Height < 0 {
		opts.Height = 0
	}
	if opts.ColorProfile == "" {
		opts.ColorProfile = Plain
	}
	r := renderer{opts: opts}
	r.line(r.style("FlipCTL", "title") + " " + clean(doc.Title))
	r.line(dim(r.opts, fmt.Sprintf("view=%s revision=%d session=%s", doc.ViewID, doc.Revision, empty(doc.SessionID, "-"))))
	if len(doc.Actions) > 0 {
		parts := make([]string, 0, len(doc.Actions))
		for _, a := range doc.Actions {
			label := a.Label
			if label == "" {
				label = a.ID
			}
			if !a.Enabled {
				label += " (disabled)"
			}
			parts = append(parts, fmt.Sprintf("%s[%s]", label, a.ID))
		}
		r.line("Actions: " + strings.Join(parts, " | "))
	}
	for _, b := range doc.Blocks {
		r.blank()
		r.line(r.style(blockTitle(b), "block"))
		switch b.Kind {
		case "text":
			r.text(b.Text)
		case "notice":
			r.notice(b)
		case "list":
			r.list(b.Items)
		case "form":
			r.form(b.Fields)
		case "key_value":
			r.pairs(b.Pairs)
		case "table":
			r.table(b.Columns, b.Rows)
		case "log":
			r.logs(b.Lines)
		case "progress":
			r.progress(b.Progress)
		default:
			r.line("unknown block kind: " + b.Kind)
		}
	}
	return r.output()
}

type renderer struct {
	opts  Options
	lines []string
}

func (r *renderer) blank()        { r.lines = append(r.lines, "") }
func (r *renderer) line(s string) { r.lines = append(r.lines, fitANSI(s, r.opts.Width)) }
func (r *renderer) output() string {
	lines := r.lines
	if r.opts.Height > 0 && len(lines) > r.opts.Height {
		lines = lines[:r.opts.Height]
	}
	return strings.Join(lines, "\n") + "\n"
}
func (r *renderer) style(s, kind string) string {
	if r.opts.ColorProfile != ANSI {
		return s
	}
	switch kind {
	case "title":
		return "\x1b[1;36m" + s + "\x1b[0m"
	case "block":
		return "\x1b[1m" + s + "\x1b[0m"
	case "error":
		return "\x1b[31m" + s + "\x1b[0m"
	case "ok":
		return "\x1b[32m" + s + "\x1b[0m"
	default:
		return s
	}
}
func (r *renderer) text(s string) {
	for _, line := range splitLines(s) {
		r.line(line)
	}
}
func (r *renderer) notice(b viewdoc.Block) {
	prefix := b.Severity
	if prefix == "" {
		prefix = "notice"
	}
	line := prefix + ": " + b.Text
	if b.Severity == "error" || b.Severity == "warning" {
		line = r.style(line, "error")
	}
	r.line(line)
}
func (r *renderer) list(items []viewdoc.ListItem) {
	for _, it := range items {
		line := "• " + firstNonEmpty(it.Label, it.Value)
		if it.Value != "" && it.Value != it.Label {
			line += " [" + it.Value + "]"
		}
		if it.Description != "" {
			line += " — " + it.Description
		}
		r.line(line)
	}
}
func (r *renderer) form(fields []viewdoc.FormField) {
	for _, f := range fields {
		req := ""
		if f.Required {
			req = " required"
		}
		val := empty(f.Default, "-")
		extra := ""
		if f.Validation != "" {
			extra = " — " + f.Validation
		}
		if len(f.Options) > 0 {
			extra += " options=" + strings.Join(f.Options, ",")
		}
		r.line(fmt.Sprintf("• %s (%s%s): %s%s", f.Label, f.Type, req, val, extra))
	}
}
func (r *renderer) pairs(pairs []viewdoc.KeyValuePair) {
	for _, p := range pairs {
		r.line(fmt.Sprintf("• %s: %s", p.Key, p.Value))
	}
}
func (r *renderer) table(cols []viewdoc.TableColumn, rows []viewdoc.TableRow) {
	if len(cols) > 0 {
		labels := make([]string, 0, len(cols))
		for _, c := range cols {
			labels = append(labels, firstNonEmpty(c.Label, c.ID))
		}
		r.line(strings.Join(labels, " | "))
	}
	for _, row := range rows {
		cells := make([]string, 0, len(cols))
		for _, c := range cols {
			cells = append(cells, row.Cells[c.ID])
		}
		if len(cells) == 0 {
			for k, v := range row.Cells {
				cells = append(cells, k+"="+v)
			}
		}
		r.line(strings.Join(cells, " | "))
	}
}
func (r *renderer) logs(lines []viewdoc.LogLine) {
	for _, l := range lines {
		id := ""
		if l.ID != "" {
			id = "[" + l.ID + "] "
		}
		src := ""
		if l.Source != "" {
			src = l.Source + " "
		}
		r.line(id + src + l.Text)
	}
}
func (r *renderer) progress(p *viewdoc.Progress) {
	if p == nil {
		return
	}
	pct := ""
	if p.Max > 0 {
		pct = fmt.Sprintf(" %.1f%%", (p.Value/p.Max)*100)
	}
	r.line(strings.TrimSpace(fmt.Sprintf("%s%s: %s", p.Label, pct, p.Text)))
}

func blockTitle(b viewdoc.Block) string {
	t := b.Kind + " " + b.ID
	if b.Title != "" {
		t += " — " + b.Title
	}
	return t
}
func clean(s string) string { return strings.Join(strings.Fields(s), " ") }
func empty(s, fallback string) string {
	if s == "" {
		return fallback
	}
	return s
}
func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}
func dim(opts Options, s string) string {
	if opts.ColorProfile == ANSI {
		return "\x1b[2m" + s + "\x1b[0m"
	}
	return s
}
func splitLines(s string) []string {
	s = strings.TrimRight(s, "\n")
	if s == "" {
		return nil
	}
	return strings.Split(s, "\n")
}
func fitANSI(s string, width int) string {
	if width <= 0 {
		return s
	}
	var b strings.Builder
	visible := 0
	for i := 0; i < len(s); {
		if s[i] == '\x1b' {
			j := i + 1
			for j < len(s) && s[j] != 'm' {
				j++
			}
			if j < len(s) {
				j++
			}
			b.WriteString(s[i:j])
			i = j
			continue
		}
		rr, size := utf8.DecodeRuneInString(s[i:])
		if visible >= width {
			break
		}
		b.WriteRune(rr)
		visible++
		i += size
	}
	return b.String()
}
