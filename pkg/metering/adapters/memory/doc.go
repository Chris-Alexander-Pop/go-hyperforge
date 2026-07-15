// Package memory provides an in-process Meter and Rater for tests and local development.
//
// It is the only metering driver currently implemented. State is process-local
// and lost when the process exits.
package memory
