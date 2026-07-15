/*
Package s3 provides an AWS S3 (and S3-compatible) adapter implementing blob.Store.

Missing keys (NoSuchKey / NotFound) are mapped to pkg/errors NotFound.
*/
package s3
