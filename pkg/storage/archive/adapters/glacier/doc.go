// Package glacier provides an S3 Glacier / DEEP_ARCHIVE archive.ArchiveStore adapter.
//
// Shipped: Archive/Delete/List/GetObject via S3; RestoreObject + job tracking;
// Download after completed restore; InstantRestore / CompleteRestore for tests.
//
// Remaining gaps (honest): no multipart upload, inventory/job polling against
// real Glacier API, or Azure/GCS archive backends.
package glacier
