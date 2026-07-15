// Package openapi provides lightweight OpenAPI 3.x document helper stubs.
//
// This is a scaffolding package for attaching path/operation metadata and
// marshaling a minimal OpenAPI document. It is not a full codegen or Echo
// middleware bridge; use it to prototype specs until a richer generator lands.
package openapi

import (
	"encoding/json"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Document is a minimal OpenAPI 3 document.
type Document struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
}

// Info holds API metadata.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// PathItem describes operations on a path.
type PathItem struct {
	Get    *Operation `json:"get,omitempty"`
	Post   *Operation `json:"post,omitempty"`
	Put    *Operation `json:"put,omitempty"`
	Patch  *Operation `json:"patch,omitempty"`
	Delete *Operation `json:"delete,omitempty"`
}

// Operation describes a single HTTP operation.
type Operation struct {
	OperationID string               `json:"operationId,omitempty"`
	Summary     string               `json:"summary,omitempty"`
	Description string               `json:"description,omitempty"`
	Tags        []string             `json:"tags,omitempty"`
	Responses   map[string]*Response `json:"responses,omitempty"`
}

// Response is a minimal response object.
type Response struct {
	Description string `json:"description"`
}

// Components holds reusable schemas (stub).
type Components struct {
	Schemas map[string]interface{} `json:"schemas,omitempty"`
}

// NewDocument creates an OpenAPI 3.0.3 document stub.
func NewDocument(title, version string) *Document {
	if title == "" {
		title = "API"
	}
	if version == "" {
		version = "0.0.0"
	}
	return &Document{
		OpenAPI: "3.0.3",
		Info:    Info{Title: title, Version: version},
		Paths:   make(map[string]*PathItem),
	}
}

// AddOperation registers an operation on path (e.g. "/users", "get").
func (d *Document) AddOperation(path, method string, op Operation) error {
	if d == nil {
		return errors.InvalidArgument("document is nil", nil)
	}
	if path == "" || method == "" {
		return errors.InvalidArgument("path and method are required", nil)
	}
	item := d.Paths[path]
	if item == nil {
		item = &PathItem{}
		d.Paths[path] = item
	}
	if op.Responses == nil {
		op.Responses = map[string]*Response{
			"200": {Description: "OK"},
		}
	}
	switch method {
	case "get", "GET":
		item.Get = &op
	case "post", "POST":
		item.Post = &op
	case "put", "PUT":
		item.Put = &op
	case "patch", "PATCH":
		item.Patch = &op
	case "delete", "DELETE":
		item.Delete = &op
	default:
		return errors.InvalidArgument("unsupported HTTP method: "+method, nil)
	}
	return nil
}

// MarshalJSON returns the OpenAPI document as JSON.
func (d *Document) MarshalJSON() ([]byte, error) {
	if d == nil {
		return nil, errors.InvalidArgument("document is nil", nil)
	}
	type alias Document
	return json.Marshal((*alias)(d))
}
