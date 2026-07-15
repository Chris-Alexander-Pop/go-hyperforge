// Package memory provides in-memory DID resolvers for did:ethr and did:web.
package memory

import (
	"context"
	"fmt"
	"strings"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3"
	"github.com/chris-alexander-pop/go-hyperforge/pkg/web3/identity"
)

// Ensure compile-time compliance.
var (
	_ web3.DIDResolver = (*EthrResolver)(nil)
	_ web3.DIDResolver = (*WebResolver)(nil)
	_ web3.DIDResolver = (*Registry)(nil)
)

// EthrResolver resolves did:ethr DIDs into in-memory documents.
type EthrResolver struct {
	docs map[string]*web3.DIDDocument
}

// NewEthrResolver creates an ethr DID resolver with optional seed documents.
func NewEthrResolver(seed map[string]*web3.DIDDocument) *EthrResolver {
	docs := make(map[string]*web3.DIDDocument)
	for k, v := range seed {
		docs[strings.ToLower(k)] = v
	}
	return &EthrResolver{docs: docs}
}

// Method returns "ethr".
func (r *EthrResolver) Method() string { return "ethr" }

// Resolve implements web3.DIDResolver for did:ethr.
func (r *EthrResolver) Resolve(ctx context.Context, did string) (*web3.DIDDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parsed, err := identity.ParseDID(did)
	if err != nil {
		return nil, err
	}
	if parsed.Method != "ethr" {
		return nil, web3.ErrInvalidConfig("ethr resolver cannot resolve method "+parsed.Method, nil)
	}
	key := strings.ToLower(did)
	if doc, ok := r.docs[key]; ok {
		return cloneDoc(doc), nil
	}
	// Synthesize a minimal document from the address identifier.
	addr := parsed.Identifier
	id := fmt.Sprintf("did:ethr:%s", addr)
	doc := &web3.DIDDocument{
		ID:         id,
		Controller: []string{id},
		VerificationMethod: []web3.DIDVerificationMethod{{
			ID:           id + "#controller",
			Type:         "EcdsaSecp256k1RecoveryMethod2020",
			Controller:   id,
			PublicKeyHex: addr,
		}},
		Authentication: []string{id + "#controller"},
	}
	r.docs[key] = doc
	return cloneDoc(doc), nil
}

// Put registers or replaces a document (test helper).
func (r *EthrResolver) Put(did string, doc *web3.DIDDocument) {
	r.docs[strings.ToLower(did)] = doc
}

// WebResolver resolves did:web DIDs from an in-memory map (no HTTP fetch).
type WebResolver struct {
	docs map[string]*web3.DIDDocument
}

// NewWebResolver creates a did:web memory resolver.
func NewWebResolver(seed map[string]*web3.DIDDocument) *WebResolver {
	docs := make(map[string]*web3.DIDDocument)
	for k, v := range seed {
		docs[strings.ToLower(k)] = v
	}
	return &WebResolver{docs: docs}
}

// Method returns "web".
func (r *WebResolver) Method() string { return "web" }

// Resolve implements web3.DIDResolver for did:web.
func (r *WebResolver) Resolve(ctx context.Context, did string) (*web3.DIDDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parsed, err := identity.ParseDID(did)
	if err != nil {
		return nil, err
	}
	if parsed.Method != "web" {
		return nil, web3.ErrInvalidConfig("web resolver cannot resolve method "+parsed.Method, nil)
	}
	key := strings.ToLower(did)
	if doc, ok := r.docs[key]; ok {
		return cloneDoc(doc), nil
	}
	return nil, web3.ErrNotFound("did document", nil)
}

// Put registers a did:web document.
func (r *WebResolver) Put(did string, doc *web3.DIDDocument) {
	r.docs[strings.ToLower(did)] = doc
}

// Registry dispatches Resolve to method-specific resolvers.
type Registry struct {
	byMethod map[string]web3.DIDResolver
}

// NewRegistry creates a multi-method DID resolver registry.
func NewRegistry(resolvers ...web3.DIDResolver) *Registry {
	m := make(map[string]web3.DIDResolver)
	for _, r := range resolvers {
		if r != nil {
			m[r.Method()] = r
		}
	}
	return &Registry{byMethod: m}
}

// Method returns "*" for the composite registry.
func (r *Registry) Method() string { return "*" }

// Resolve dispatches to the resolver registered for the DID method.
func (r *Registry) Resolve(ctx context.Context, did string) (*web3.DIDDocument, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	parsed, err := identity.ParseDID(did)
	if err != nil {
		return nil, err
	}
	res, ok := r.byMethod[parsed.Method]
	if !ok {
		return nil, web3.ErrInvalidConfig("no resolver for DID method "+parsed.Method, nil)
	}
	return res.Resolve(ctx, did)
}

func cloneDoc(d *web3.DIDDocument) *web3.DIDDocument {
	if d == nil {
		return nil
	}
	cp := *d
	cp.Controller = append([]string(nil), d.Controller...)
	cp.Authentication = append([]string(nil), d.Authentication...)
	cp.AlsoKnownAs = append([]string(nil), d.AlsoKnownAs...)
	cp.VerificationMethod = append([]web3.DIDVerificationMethod(nil), d.VerificationMethod...)
	cp.Service = append([]web3.DIDService(nil), d.Service...)
	return &cp
}
