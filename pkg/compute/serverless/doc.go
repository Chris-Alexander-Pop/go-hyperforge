// Package serverless provides a unified interface for serverless function management.
//
// Shipping backends:
//   - Memory: in-memory runtime for tests
//   - Lambda: AWS Lambda
//   - GCF: Google Cloud Functions
//   - Azure Functions: HTTP Invoke when InvokeBaseURL is set; ARM CRUD Unimplemented
package serverless
