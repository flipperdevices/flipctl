package viewdoc

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/santhosh-tekuri/jsonschema/v6"
)

func compileViewDocumentSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	compiler := jsonschema.NewCompiler()
	schema, err := compiler.Compile(filepath.Join("..", "..", "schemas", "view-document.json"))
	if err != nil {
		t.Fatalf("compile ViewDocument schema: %v", err)
	}
	return schema
}

func validateJSONFile(t *testing.T, schema *jsonschema.Schema, path string) error {
	t.Helper()
	f, err := os.Open(path)
	if err != nil {
		t.Fatalf("open %s: %v", path, err)
	}
	defer f.Close()
	instance, err := jsonschema.UnmarshalJSON(f)
	if err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	return schema.Validate(instance)
}

func TestViewDocumentValidFixturesPassDraft2020Schema(t *testing.T) {
	schema := compileViewDocumentSchema(t)
	paths, err := filepath.Glob(filepath.Join("..", "..", "schemas", "view-document-fixtures", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("fixtures: %v count=%d", err, len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if err := validateJSONFile(t, schema, path); err != nil {
				t.Fatalf("valid fixture rejected: %v", err)
			}
		})
	}
}

func TestViewDocumentInvalidFixturesFailDraft2020Schema(t *testing.T) {
	schema := compileViewDocumentSchema(t)
	paths, err := filepath.Glob(filepath.Join("..", "..", "schemas", "invalid-view-document-fixtures", "*.json"))
	if err != nil || len(paths) == 0 {
		t.Fatalf("invalid fixtures: %v count=%d", err, len(paths))
	}
	for _, path := range paths {
		t.Run(filepath.Base(path), func(t *testing.T) {
			if err := validateJSONFile(t, schema, path); err == nil {
				t.Fatalf("invalid fixture accepted")
			}
		})
	}
}
