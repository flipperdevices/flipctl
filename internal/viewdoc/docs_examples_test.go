package viewdoc

import (
	"bytes"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func TestViewDocumentMarkdownExamplesPassDraft2020Schema(t *testing.T) {
	schema := compileViewDocumentSchema(t)
	docs := []struct {
		path         string
		minValidated int
	}{
		{path: filepath.Join("..", "..", "docs", "view-document.md"), minValidated: 2},
		{path: filepath.Join("..", "..", "docs", "evidence", "architecture-demo.md"), minValidated: 1},
	}

	for _, doc := range docs {
		t.Run(filepath.ToSlash(doc.path), func(t *testing.T) {
			b, err := os.ReadFile(doc.path)
			if err != nil {
				t.Fatalf("read %s: %v", doc.path, err)
			}

			examples := extractViewDocumentMarkdownJSONExamples(t, doc.path, b)
			validated := 0
			for _, example := range examples {
				t.Run(example.name(), func(t *testing.T) {
					instance, err := jsonschema.UnmarshalJSON(bytes.NewReader([]byte(example.json)))
					if err != nil {
						t.Fatalf("parse JSON fence at line %d: %v", example.line, err)
					}
					if example.mode == "skip" {
						return
					}
					validated++
					if err := schema.Validate(instance); err != nil {
						t.Fatalf("ViewDocument JSON fence at line %d rejected by schema: %v", example.line, err)
					}
				})
			}
			if validated < doc.minValidated {
				t.Fatalf("validated %d ViewDocument examples in %s, want at least %d", validated, doc.path, doc.minValidated)
			}
		})
	}
}

type markdownJSONExample struct {
	line   int
	mode   string
	reason string
	json   string
}

func (e markdownJSONExample) name() string {
	return strings.ReplaceAll(e.mode+"_line_"+strconv.Itoa(e.line)+"_"+e.reason, " ", "_")
}

func extractViewDocumentMarkdownJSONExamples(t *testing.T, docPath string, b []byte) []markdownJSONExample {
	t.Helper()
	lines := strings.Split(string(b), "\n")
	var examples []markdownJSONExample
	for i := 0; i < len(lines); i++ {
		if strings.TrimSpace(lines[i]) != "```json" {
			continue
		}
		if i == 0 {
			t.Fatalf("JSON fence at line %d missing viewdocument-json marker", i+1)
		}
		mode, reason, ok := parseViewDocumentJSONMarker(lines[i-1])
		if !ok {
			t.Fatalf("JSON fence at line %d must be immediately preceded by <!-- viewdocument-json: validate <reason> --> or <!-- viewdocument-json: skip <reason> -->", i+1)
		}
		start := i + 1
		var body []string
		for i++; i < len(lines); i++ {
			if strings.TrimSpace(lines[i]) == "```" {
				break
			}
			body = append(body, lines[i])
		}
		if i == len(lines) {
			t.Fatalf("JSON fence at line %d is not closed", start)
		}
		examples = append(examples, markdownJSONExample{line: start, mode: mode, reason: reason, json: strings.Join(body, "\n")})
	}
	if len(examples) == 0 {
		t.Fatalf("no JSON fences found in %s", docPath)
	}
	return examples
}

func parseViewDocumentJSONMarker(line string) (mode, reason string, ok bool) {
	trimmed := strings.TrimSpace(line)
	const prefix = "<!-- viewdocument-json:"
	if !strings.HasPrefix(trimmed, prefix) || !strings.HasSuffix(trimmed, "-->") {
		return "", "", false
	}
	content := strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(trimmed, prefix), "-->"))
	mode, reason, ok = strings.Cut(content, " ")
	reason = strings.TrimSpace(reason)
	if !ok || reason == "" || (mode != "validate" && mode != "skip") {
		return "", "", false
	}
	return mode, reason, true
}
