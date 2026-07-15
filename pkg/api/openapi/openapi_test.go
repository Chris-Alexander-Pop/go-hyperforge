package openapi_test

import (
	"encoding/json"
	"testing"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/api/openapi"
)

func TestDocumentStub(t *testing.T) {
	doc := openapi.NewDocument("Demo", "1.0.0")
	if err := doc.AddOperation("/health", "get", openapi.Operation{
		OperationID: "healthCheck",
		Summary:     "Liveness",
	}); err != nil {
		t.Fatalf("AddOperation: %v", err)
	}
	raw, err := doc.MarshalJSON()
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	var got map[string]interface{}
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if got["openapi"] != "3.0.3" {
		t.Fatalf("openapi=%v", got["openapi"])
	}
	paths, ok := got["paths"].(map[string]interface{})
	if !ok || paths["/health"] == nil {
		t.Fatalf("paths=%v", got["paths"])
	}
}
