// Package glacier provides a thin archive.ArchiveStore over S3 Glacier-style APIs.
//
// It uses PutObject with GLACIER/DEEP_ARCHIVE storage class and RestoreObject for
// restores. Inject ObjectAPI via NewFromAPI for tests; New builds the AWS SDK client.
// Restore completion is best-effort / simulated when using a fake API.
package glacier
