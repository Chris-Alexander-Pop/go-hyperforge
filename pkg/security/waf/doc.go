// Package waf provides Web Application Firewall control interfaces.
//
// Adapters:
//   - adapters/memory — in-process IP block list
//   - adapters/cloudflare — Cloudflare Firewall IP access rules API
//
// AWS WAF remains a reserved Provider name without an in-tree adapter.
package waf
