// Package memory provides an in-memory streaming.Client for local development and tests.
//
// It honors streaming.Config.BufferSize as a hard capacity on retained records.
// PutRecord returns streaming.ErrBufferFull when the buffer is at capacity.
package memory
