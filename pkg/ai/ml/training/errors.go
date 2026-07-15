package training

import "github.com/chris-alexander-pop/go-hyperforge/pkg/errors"

// Sentinel errors for training operations.
var (
	// ErrJobNotFound is returned when a training job does not exist.
	ErrJobNotFound = errors.NotFound("training job not found", nil)

	// ErrJobAlreadyExists is returned when starting a duplicate job name/id.
	ErrJobAlreadyExists = errors.Conflict("training job already exists", nil)

	// ErrInvalidJob is returned when a job config is malformed.
	ErrInvalidJob = errors.InvalidArgument("invalid training job config", nil)

	// ErrNotRunning is returned when stopping a job that is not running.
	ErrNotRunning = errors.FailedPrecondition("training job is not running", nil)

	// ErrBackendUnavailable is returned when the training backend cannot be reached.
	ErrBackendUnavailable = errors.Unavailable("training backend unavailable", nil)

	// ErrClosed is returned when the trainer was used after Close.
	ErrClosed = errors.New("TRAINING_CLOSED", "trainer is closed", nil)
)
