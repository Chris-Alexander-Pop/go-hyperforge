package storage

// Kind identifies a storage capability within the storage domain.
type Kind string

const (
	KindBlob       Kind = "blob"
	KindFile       Kind = "file"
	KindBlock      Kind = "block"
	KindArchive    Kind = "archive"
	KindController Kind = "controller"
)

// Shared / testing driver.
const (
	DriverMemory = "memory"
)

// Blob storage drivers.
const (
	DriverLocal     = "local"
	DriverS3        = "s3"
	DriverGCS       = "gcs"
	DriverAzureBlob = "azureblob"
)

// File storage drivers (memory is the only shipping adapter today).
const (
	DriverNFS        = "nfs"
	DriverEFS        = "efs"
	DriverAzureFiles = "azure-files"
	DriverGCSFuse    = "gcs-fuse"
)

// Block storage drivers (memory is the only shipping adapter today).
const (
	DriverEBS             = "ebs"
	DriverAzureDisk       = "azure-disk"
	DriverGCPPersistent   = "gcp-persistent"
	DriverCeph            = "ceph"
	DriverOpenStackCinder = "cinder"
)

// Controller volume drivers (memory/lvm/ceph/csi adapters ship under controller/).
const (
	DriverLVM = "lvm"
	DriverCSI = "csi"
)

// Archive (cold storage) drivers (memory is the only shipping adapter today).
const (
	DriverGlacier      = "glacier"
	DriverGlacierDeep  = "glacier-deep"
	DriverAzureArchive = "azure-archive"
	DriverGCSArchive   = "gcs-archive"
)
