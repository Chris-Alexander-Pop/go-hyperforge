package controller

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

var (
	// ErrVolumeNotFound is returned when a requested volume does not exist.
	ErrVolumeNotFound = errors.NotFound("volume not found", nil)

	// ErrVolumeAttached is returned when attempting to delete or modify a volume that is currently attached.
	ErrVolumeAttached = errors.Conflict("volume is attached", nil)

	// ErrInvalidSize is returned when the requested size is invalid (e.g. shrinking).
	ErrInvalidSize = errors.InvalidArgument("invalid volume size", nil)
)
