// Package openapi provides OpenAPI 3.x document helpers generated from route metadata,
// plus Echo↔net/http adapter bridge utilities.
package openapi

import (
	"encoding/json"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Document is an OpenAPI 3 document.
type Document struct {
	OpenAPI    string               `json:"openapi"`
	Info       Info                 `json:"info"`
	Paths      map[string]*PathItem `json:"paths"`
	Components *Components          `json:"components,omitempty"`
	Tags       []Tag                `json:"tags,omitempty"`
}

// Info holds API metadata.
type Info struct {
	Title       string `json:"title"`
	Version     string `json:"version"`
	Description string `json:"description,omitempty"`
}

// Tag is an OpenAPI tag object.
type Tag struct {
	Name        string `json:"name"`
	Description string `json:"description,omitempty"`
}

// PathItem describes operations on a path.
type PathItem struct {
	Get     *Operation `json:"get,omitempty"`
	Post    *Operation `json:"post,omitempty"`
	Put     *Operation `json:"put,omitempty"`
	Patch   *Operation `json:"patch,omitempty"`
	Delete  *Operation `json:"delete,omitempty"`
	Head    *Operation `json:"head,omitempty"`
	Options *Operation `json:"options,omitempty"`
}

// Parameter is a minimal OpenAPI parameter object.
type Parameter struct {
	Name        string  `json:"name"`
	In          string  `json:"in"` // path, query, header, cookie
	Required    bool    `json:"required,omitempty"`
	Description string  `json:"description,omitempty"`
	Schema      *Schema `json:"schema,omitempty"`
}

// Schema is a minimal OpenAPI schema object.
type Schema struct {
	Type   string `json:"type,omitempty"`
	Format string `json:"format,omitempty"`
}

// Operation describes a single HTTP operation.
type Operation struct {
	OperationID string                `json:"operationId,omitempty"`
	Summary     string                `json:"summary,omitempty"`
	Description string                `json:"description,omitempty"`
	Tags        []string              `json:"tags,omitempty"`
	Parameters  []Parameter           `json:"parameters,omitempty"`
	Responses   map[string]*Response  `json:"responses,omitempty"`
	Security    []map[string][]string `json:"security,omitempty"`
}

// Response is a minimal response object.
type Response struct {
	Description string `json:"description"`
}

// Components holds reusable schemas and security schemes.
type Components struct {
	Schemas         map[string]interface{} `json:"schemas,omitempty"`
	SecuritySchemes map[string]interface{} `json:"securitySchemes,omitempty"`
}

// RouteMeta describes a single HTTP route for OpenAPI generation.
type RouteMeta struct {
	Path        string
	Method      string
	OperationID string
	Summary     string
	Description string
	Tags        []string
	Parameters  []Parameter
	Responses   map[string]*Response
	Security    []map[string][]string
}

// NewDocument creates an OpenAPI 3.0.3 document.
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

// FromRoutes builds an OpenAPI document from route metadata.
func FromRoutes(title, version string, routes []RouteMeta) (*Document, error) {
	doc := NewDocument(title, version)
	seenTags := make(map[string]struct{})
	for _, r := range routes {
		op := Operation{
			OperationID: r.OperationID,
			Summary:     r.Summary,
			Description: r.Description,
			Tags:        r.Tags,
			Parameters:  r.Parameters,
			Responses:   r.Responses,
			Security:    r.Security,
		}
		if err := doc.AddOperation(r.Path, r.Method, op); err != nil {
			return nil, err
		}
		for _, tag := range r.Tags {
			if _, ok := seenTags[tag]; !ok {
				seenTags[tag] = struct{}{}
				doc.Tags = append(doc.Tags, Tag{Name: tag})
			}
		}
	}
	return doc, nil
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
	// Infer path parameters from {param} segments when none provided.
	if len(op.Parameters) == 0 {
		op.Parameters = pathParamsFromPath(path)
	}
	switch strings.ToUpper(method) {
	case "GET":
		item.Get = &op
	case "POST":
		item.Post = &op
	case "PUT":
		item.Put = &op
	case "PATCH":
		item.Patch = &op
	case "DELETE":
		item.Delete = &op
	case "HEAD":
		item.Head = &op
	case "OPTIONS":
		item.Options = &op
	default:
		return errors.InvalidArgument("unsupported HTTP method: "+method, nil)
	}
	return nil
}

func pathParamsFromPath(path string) []Parameter {
	var params []Parameter
	for _, part := range strings.Split(path, "/") {
		if len(part) >= 2 && part[0] == '{' && part[len(part)-1] == '}' {
			name := part[1 : len(part)-1]
			if name == "" {
				continue
			}
			params = append(params, Parameter{
				Name:     name,
				In:       "path",
				Required: true,
				Schema:   &Schema{Type: "string"},
			})
		}
	}
	return params
}

// MarshalJSON returns the OpenAPI document as JSON.
func (d *Document) MarshalJSON() ([]byte, error) {
	if d == nil {
		return nil, errors.InvalidArgument("document is nil", nil)
	}
	type alias Document
	return json.Marshal((*alias)(d))
}
