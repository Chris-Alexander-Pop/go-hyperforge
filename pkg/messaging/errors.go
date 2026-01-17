package messaging

import "github.com/chris-alexander-pop/system-design-library/pkg/errors"

// Error codes for messaging operations.
const (
	CodeConnectionFailed      = "MESSAGING_CONN_FAILED"
	CodeTopicNotFound         = "MESSAGING_TOPIC_NOT_FOUND"
	CodePublishFailed         = "MESSAGING_PUBLISH_FAILED"
	CodeConsumeFailed         = "MESSAGING_CONSUME_FAILED"
	CodeTimeout               = "MESSAGING_TIMEOUT"
	CodeClosed                = "MESSAGING_CLOSED"
	CodeInvalidConfig         = "MESSAGING_INVALID_CONFIG"
	CodeAckFailed             = "MESSAGING_ACK_FAILED"
	CodeNackFailed            = "MESSAGING_NACK_FAILED"
	CodeSerializationFailed   = "MESSAGING_SERIALIZATION_FAILED"
	CodeQueueFull             = "MESSAGING_QUEUE_FULL"
	CodeConsumerGroupConflict = "MESSAGING_GROUP_CONFLICT"
)

// Error constructors for messaging operations.
// These use the pkg/errors patterns for consistent error handling.

// ErrConnectionFailed creates an error for broker connection failures.
func ErrConnectionFailed(err error) *errors.AppError {
	return errors.New(CodeConnectionFailed, "failed to connect to message broker", err)
}

// ErrTopicNotFound creates an error for missing topic/queue.
func ErrTopicNotFound(topic string, err error) *errors.AppError {
	return errors.New(CodeTopicNotFound, "topic or queue not found: "+topic, err)
}

// ErrPublishFailed creates an error for publish failures.
func ErrPublishFailed(err error) *errors.AppError {
	return errors.New(CodePublishFailed, "failed to publish message", err)
}

// ErrConsumeFailed creates an error for consume failures.
func ErrConsumeFailed(err error) *errors.AppError {
	return errors.New(CodeConsumeFailed, "failed to consume message", err)
}

// ErrTimeout creates an error for operation timeouts.
func ErrTimeout(operation string, err error) *errors.AppError {
	return errors.New(CodeTimeout, "messaging operation timed out: "+operation, err)
}

// ErrClosed creates an error for closed connections.
func ErrClosed(err error) *errors.AppError {
	return errors.New(CodeClosed, "broker connection is closed", err)
}

// ErrInvalidConfig creates an error for invalid configuration.
func ErrInvalidConfig(msg string, err error) *errors.AppError {
	return errors.New(CodeInvalidConfig, "invalid messaging configuration: "+msg, err)
}

// ErrAckFailed creates an error for acknowledgment failures.
func ErrAckFailed(err error) *errors.AppError {
	return errors.New(CodeAckFailed, "failed to acknowledge message", err)
}

// ErrNackFailed creates an error for negative acknowledgment failures.
func ErrNackFailed(err error) *errors.AppError {
	return errors.New(CodeNackFailed, "failed to nack message", err)
}

// ErrSerializationFailed creates an error for serialization failures.
func ErrSerializationFailed(err error) *errors.AppError {
	return errors.New(CodeSerializationFailed, "failed to serialize/deserialize message", err)
}

// ErrQueueFull creates an error for full producer queues.
func ErrQueueFull(err error) *errors.AppError {
	return errors.New(CodeQueueFull, "producer queue is full", err)
}

// ErrConsumerGroupConflict creates an error for consumer group conflicts.
func ErrConsumerGroupConflict(group string, err error) *errors.AppError {
	return errors.New(CodeConsumerGroupConflict, "consumer group conflict: "+group, err)
}
