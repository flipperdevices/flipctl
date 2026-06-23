package tuirender

import (
	"encoding/json"
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/flipperdevices/flipctl/internal/viewdoc"
)

var update = flag.Bool("update", false, "update golden files")

func TestRendererGoldens(t *testing.T) {
	fixtures := []string{
		"home-app-list", "empty-job-history", "job-history",
		"ping-form", "ping-running", "ping-success", "ping-failure",
		"nmap-form", "nmap-running", "nmap-success", "nmap-failure",
	}
	for _, name := range fixtures {
		doc := loadFixture(t, name)
		for _, profile := range []ColorProfile{Plain, ANSI} {
			profile := profile
			t.Run(name+"/"+string(profile), func(t *testing.T) {
				got := RenderDocument(doc, Options{Width: 160, Height: 40, ColorProfile: profile})
				if strings.Contains(got, "Apps (↑/↓") || strings.Contains(got, "Jobs (c cancel") {
					t.Fatalf("renderer fell back to bespoke app/job list: %q", got)
				}
				assertSemanticCoverage(t, doc, got)
				golden := filepath.Join("testdata", "golden", name+"."+string(profile)+".golden")
				if *update {
					if err := os.WriteFile(golden, []byte(got), 0o644); err != nil {
						t.Fatal(err)
					}
				}
				want, err := os.ReadFile(golden)
				if err != nil {
					t.Fatal(err)
				}
				if got != string(want) {
					t.Fatalf("golden mismatch for %s\n--- got ---\n%s\n--- want ---\n%s", golden, got, string(want))
				}
			})
		}
	}
}

func TestNoninteractiveRenderEntryPoint(t *testing.T) {
	doc := loadFixture(t, "ping-running")
	out := RenderDocument(doc, Options{Width: 80, Height: 24, ColorProfile: Plain})
	for _, token := range []string{"FlipCTL Ping Running", "progress", "log", "sequence=2", "2 of 4 replies received"} {
		if !strings.Contains(out, token) {
			t.Fatalf("noninteractive render missing %q in\n%s", token, out)
		}
	}
}

func loadFixture(t *testing.T, name string) viewdoc.Document {
	t.Helper()
	b, err := os.ReadFile(filepath.Join("..", "..", "schemas", "view-document-fixtures", name+".json"))
	if err != nil {
		t.Fatal(err)
	}
	var doc viewdoc.Document
	if err := json.Unmarshal(b, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

func assertSemanticCoverage(t *testing.T, doc viewdoc.Document, got string) {
	t.Helper()
	for _, token := range []string{doc.Title, doc.ViewID} {
		if token != "" && !strings.Contains(got, token) {
			t.Fatalf("missing token %q in\n%s", token, got)
		}
	}
	for _, block := range doc.Blocks {
		for _, token := range []string{block.ID, block.Kind, block.Title, block.Text, block.Severity} {
			if token != "" && !strings.Contains(got, token) {
				t.Fatalf("missing block token %q in\n%s", token, got)
			}
		}
		for _, f := range block.Fields {
			for _, token := range []string{f.Label, f.Default, f.Validation} {
				if token != "" && !strings.Contains(got, token) {
					t.Fatalf("missing form token %q in\n%s", token, got)
				}
			}
		}
		for _, l := range block.Lines {
			for _, token := range []string{l.ID, l.Source, l.Text, l.Severity} {
				if token != "" && !strings.Contains(got, token) {
					t.Fatalf("missing log token %q in\n%s", token, got)
				}
			}
		}
		if block.Progress != nil {
			for _, token := range []string{block.Progress.Label, block.Progress.Text} {
				if token != "" && !strings.Contains(got, token) {
					t.Fatalf("missing progress token %q in\n%s", token, got)
				}
			}
		}
		for _, p := range block.Pairs {
			for _, token := range []string{p.Key, p.Value} {
				if token != "" && !strings.Contains(got, token) {
					t.Fatalf("missing key/value token %q in\n%s", token, got)
				}
			}
		}
		for _, row := range block.Rows {
			for _, token := range row.Cells {
				if token != "" && !strings.Contains(got, token) {
					t.Fatalf("missing table token %q in\n%s", token, got)
				}
			}
		}
	}
}
