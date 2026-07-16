// Package services contains Hyperforge microservices.
//
// See SERVICE_CATALOG.md for the full service catalog and docs/services.md
// for layout and bootstrap conventions.
//
// Identity cluster (v1):
//   - services/auth     — register / login, JWT issuance
//   - services/user     — user profiles
//   - services/gateway  — edge reverse proxy + JWT verification
//   - services/platform — shared bootstrap helpers
package services
