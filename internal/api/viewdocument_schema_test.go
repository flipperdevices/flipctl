package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sync"
	"testing"

	"github.com/flipperdevices/flipctl/internal/viewdoc"
	"github.com/santhosh-tekuri/jsonschema/v6"
)

var (
	viewDocumentSchemaOnce sync.Once
	viewDocumentSchema     *jsonschema.Schema
	viewDocumentSchemaErr  error
)

func compiledViewDocumentSchema(t *testing.T) *jsonschema.Schema {
	t.Helper()
	viewDocumentSchemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		viewDocumentSchema, viewDocumentSchemaErr = compiler.Compile(filepath.Join("..", "..", "schemas", "view-document.json"))
	})
	if viewDocumentSchemaErr != nil {
		t.Fatalf("compile ViewDocument schema: %v", viewDocumentSchemaErr)
	}
	return viewDocumentSchema
}

// ValidateViewDocument serializes the live Go projector output and validates it
// against the checked-in ViewDocument JSON Schema used by fixture validation.
func ValidateViewDocument(t *testing.T, doc viewdoc.Document) {
	t.Helper()
	if err := validateViewDocument(doc); err != nil {
		t.Fatalf("validate live ViewDocument viewId=%q revision=%d: %v", doc.ViewID, doc.Revision, err)
	}
}

func validateViewDocument(doc viewdoc.Document) error {
	b, err := json.Marshal(doc)
	if err != nil {
		return fmt.Errorf("marshal ViewDocument: %w", err)
	}
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(b))
	if err != nil {
		return fmt.Errorf("parse marshaled ViewDocument JSON: %w", err)
	}
	viewDocumentSchemaOnce.Do(func() {
		compiler := jsonschema.NewCompiler()
		viewDocumentSchema, viewDocumentSchemaErr = compiler.Compile(filepath.Join("..", "..", "schemas", "view-document.json"))
	})
	if viewDocumentSchemaErr != nil {
		return fmt.Errorf("compile ViewDocument schema: %w", viewDocumentSchemaErr)
	}
	if err := viewDocumentSchema.Validate(instance); err != nil {
		return fmt.Errorf("schema validation failed: %w", err)
	}
	return nil
}

func validateHTTPViewDocument(t *testing.T, raw []byte) viewdoc.Document {
	t.Helper()
	instance, err := jsonschema.UnmarshalJSON(bytes.NewReader(raw))
	if err != nil {
		t.Fatalf("parse HTTP ViewDocument response: %v", err)
	}
	if err := compiledViewDocumentSchema(t).Validate(instance); err != nil {
		t.Fatalf("HTTP ViewDocument response failed schema validation: %v", err)
	}
	var doc viewdoc.Document
	if err := json.Unmarshal(raw, &doc); err != nil {
		t.Fatalf("decode HTTP ViewDocument response: %v", err)
	}
	return doc
}
