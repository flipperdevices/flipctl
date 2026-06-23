package viewdoc

const CanonicalAPIVersion = "viewdoc.flipctl.dev/v1alpha1"

// Document is the canonical renderer-neutral semantic contract consumed by all
// FlipCTL frontends. It describes generic user-visible blocks only; app/tool
// parser details belong in backend/runtime models, not in this public contract.
type Document struct {
	APIVersion string   `json:"apiVersion"`
	SessionID  string   `json:"sessionId"`
	ViewID     string   `json:"viewId"`
	Revision   int      `json:"revision"`
	Title      string   `json:"title"`
	Blocks     []Block  `json:"blocks"`
	Actions    []Action `json:"actions,omitempty"`
}

type Action struct {
	ID      string            `json:"id"`
	Label   string            `json:"label"`
	Enabled bool              `json:"enabled"`
	Role    string            `json:"role,omitempty"`
	Params  map[string]string `json:"params,omitempty"`
}

type Block struct {
	ID       string         `json:"id"`
	Kind     string         `json:"kind"`
	Title    string         `json:"title,omitempty"`
	Text     string         `json:"text,omitempty"`
	Severity string         `json:"severity,omitempty"`
	Items    []ListItem     `json:"items,omitempty"`
	Fields   []FormField    `json:"fields,omitempty"`
	Pairs    []KeyValuePair `json:"pairs,omitempty"`
	Columns  []TableColumn  `json:"columns,omitempty"`
	Rows     []TableRow     `json:"rows,omitempty"`
	Lines    []LogLine      `json:"lines,omitempty"`
	Progress *Progress      `json:"progress,omitempty"`
}

type ListItem struct {
	Label       string  `json:"label"`
	Value       string  `json:"value,omitempty"`
	Description string  `json:"description,omitempty"`
	Action      *Action `json:"action,omitempty"`
}
type FormField struct {
	ID         string   `json:"id"`
	Label      string   `json:"label"`
	Type       string   `json:"type"`
	Default    string   `json:"default,omitempty"`
	Required   bool     `json:"required,omitempty"`
	Options    []string `json:"options,omitempty"`
	Validation string   `json:"validation,omitempty"`
}
type KeyValuePair struct {
	Key   string `json:"key"`
	Value string `json:"value"`
}
type TableColumn struct {
	ID    string `json:"id"`
	Label string `json:"label"`
}
type TableRow struct {
	Cells map[string]string `json:"cells"`
}
type LogLine struct {
	ID       string `json:"id,omitempty"`
	Source   string `json:"source,omitempty"`
	Text     string `json:"text"`
	Severity string `json:"severity,omitempty"`
}
type Progress struct {
	Label string  `json:"label"`
	Value float64 `json:"value,omitempty"`
	Max   float64 `json:"max,omitempty"`
	Text  string  `json:"text,omitempty"`
}
