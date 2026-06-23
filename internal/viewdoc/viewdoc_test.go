package viewdoc

import (
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

func TestViewDocumentFixturesValidateContract(t *testing.T) {
	paths, err := filepath.Glob(filepath.Join("..", "..", "schemas", "view-document-fixtures", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v count=%d", err, len(paths))
	}
	seenKinds, seenViews := map[string]bool{}, map[string]bool{}
	for _, path := range paths {
		b, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var doc Document
		if err := json.Unmarshal(b, &doc); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		if doc.APIVersion != CanonicalAPIVersion {
			t.Fatalf("%s: apiVersion=%q", path, doc.APIVersion)
		}
		if doc.SessionID == "" || doc.ViewID == "" || doc.Revision < 0 || doc.Title == "" || len(doc.Blocks) == 0 {
			t.Fatalf("%s: missing required root fields", path)
		}
		seenViews[doc.ViewID] = true
		for _, block := range doc.Blocks {
			if block.ID == "" || block.Kind == "" {
				t.Fatalf("%s: block missing id/kind", path)
			}
			seenKinds[block.Kind] = true
			switch block.Kind {
			case "text":
				if block.Text == "" {
					t.Fatalf("%s: invalid text block", path)
				}
			case "notice":
				if block.Text == "" || block.Severity == "" {
					t.Fatalf("%s: invalid notice block", path)
				}
			case "list":
				if len(block.Items) == 0 {
					t.Fatalf("%s: invalid list block", path)
				}
			case "form":
				if len(block.Fields) == 0 {
					t.Fatalf("%s: invalid form block", path)
				}
			case "key_value":
				if len(block.Pairs) == 0 {
					t.Fatalf("%s: invalid key_value block", path)
				}
			case "table":
				if len(block.Columns) == 0 {
					t.Fatalf("%s: invalid table block", path)
				}
			case "log":
				if len(block.Lines) == 0 {
					t.Fatalf("%s: invalid log block", path)
				}
			case "progress":
				if block.Progress == nil || block.Progress.Label == "" {
					t.Fatalf("%s: invalid progress block", path)
				}
			default:
				t.Fatalf("%s: unknown kind %q", path, block.Kind)
			}
		}
	}
	for _, k := range []string{"text", "notice", "list", "form", "key_value", "table", "log", "progress"} {
		if !seenKinds[k] {
			t.Fatalf("missing block kind %s", k)
		}
	}
	for _, v := range []string{"home-app-list", "ping-form", "ping-running", "ping-success", "ping-failure", "nmap-form", "nmap-running", "nmap-success", "nmap-failure", "empty-job-history", "job-history"} {
		if !seenViews[v] {
			t.Fatalf("missing fixture %s", v)
		}
	}
}

func TestViewDocumentSchemaIsCanonicalAndGeneric(t *testing.T) {
	b, err := os.ReadFile(filepath.Join("..", "..", "schemas", "view-document.json"))
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	for _, token := range []string{CanonicalAPIVersion, "apiVersion", "sessionId", "viewId", "revision", "title", "blocks", "text", "notice", "list", "form", "key_value", "table", "log", "progress"} {
		if !strings.Contains(s, token) {
			t.Fatalf("schema missing token %s", token)
		}
	}
	for _, token := range []string{"pingReplies", "pingSummary", "nmapProgress", "nmapPortStates"} {
		if strings.Contains(s, token) {
			t.Fatalf("schema contains forbidden public field %s", token)
		}
	}
}

func TestRepositoryViewDocumentVersionAndForbiddenPublicFields(t *testing.T) {
	root := filepath.Join("..", "..")
	versionRe := regexp.MustCompile(`viewdoc\.flipctl\.dev/[A-Za-z0-9._-]+`)
	obsoleteVersionRe := regexp.MustCompile(`flipctl\.view/v[0-9A-Za-z._-]+`)
	allowedVersion := CanonicalAPIVersion
	forbidden := []string{"ping" + "Replies", "ping" + "Summary", "nmap" + "Progress", "nmap" + "PortStates", "Log" + "Stream", "Result" + "Block"}
	include := func(path string) bool {
		path = filepath.ToSlash(path)
		if strings.Contains(path, "/.git/") || strings.Contains(path, "/node_modules/") || strings.Contains(path, "/dist/") {
			return false
		}
		return strings.HasPrefix(path, "internal/viewdoc/") || strings.HasPrefix(path, "schemas/") || strings.HasPrefix(path, "web/src/viewDocument") || strings.HasPrefix(path, "scripts/") || path == "docs/view-document.md" || strings.HasPrefix(path, "docs/adr/") || path == "README.md" || path == "Makefile"
	}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, _ := filepath.Rel(root, path)
		if !include(rel) {
			return nil
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		s := string(b)
		for _, m := range versionRe.FindAllString(s, -1) {
			if m != allowedVersion {
				t.Fatalf("%s contains non-canonical ViewDocument version %s", rel, m)
			}
		}
		if m := obsoleteVersionRe.FindString(s); m != "" {
			t.Fatalf("%s contains obsolete ViewDocument version %s", rel, m)
		}
		relSlash := filepath.ToSlash(rel)
		if strings.HasPrefix(relSlash, "schemas/invalid-view-document-fixtures/") {
			return nil
		}
		if strings.HasPrefix(relSlash, "schemas/") || strings.HasPrefix(relSlash, "web/src/viewDocument") || strings.HasPrefix(relSlash, "scripts/") {
			for _, token := range forbidden {
				if strings.Contains(s, token) {
					t.Fatalf("%s contains forbidden public contract field %s", rel, token)
				}
			}
		}
		if strings.HasPrefix(relSlash, "scripts/") && strings.Contains(s, "Progress"+"Block") {
			t.Fatalf("%s contains obsolete block kind %s", rel, "Progress"+"Block")
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
