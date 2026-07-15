/*
Package milvus implements vector.Store against the Milvus REST API (v2).

This is a thin HTTP client: Search/Upsert/Delete map to /v2/vectordb/entities/*
endpoints. Tests inject BaseURL via vector.Config.Host + httptest.
*/
package milvus
