package web3

import (
	"fmt"

	"github.com/chris-alexander-pop/go-hyperforge/pkg/errors"
)

// Error codes for Web3 operations.
const (
	CodeConnectionFailed = "WEB3_CONN_FAILED"
	CodeRPCFailed        = "WEB3_RPC_FAILED"
	CodeNotFound         = "WEB3_NOT_FOUND"
	CodeInvalidConfig    = "WEB3_INVALID_CONFIG"
	CodeInvalidSignature = "WEB3_INVALID_SIGNATURE"
	CodeNonceReused      = "WEB3_NONCE_REUSED"
	CodeMessageExpired   = "WEB3_MESSAGE_EXPIRED"
	CodeMessageNotYet    = "WEB3_MESSAGE_NOT_YET"
	CodeNoSigner         = "WEB3_NO_SIGNER"
	CodeStorageFailed    = "WEB3_STORAGE_FAILED"
	CodeTimeout          = "WEB3_TIMEOUT"
	CodeCanceled         = "WEB3_CANCELED"
)

// ErrConnectionFailed creates an error for RPC/node connection failures.
func ErrConnectionFailed(err error) *errors.AppError {
	return errors.New(CodeConnectionFailed, "failed to connect to Web3 endpoint", err)
}

// ErrRPCFailed creates an error for RPC call failures.
func ErrRPCFailed(operation string, err error) *errors.AppError {
	return errors.New(CodeRPCFailed, "RPC call failed: "+operation, err)
}

// ErrNotFound creates an error when a transaction or content is missing.
func ErrNotFound(resource string, err error) *errors.AppError {
	return errors.New(CodeNotFound, resource+" not found", err)
}

// ErrInvalidConfig creates an error for invalid configuration.
func ErrInvalidConfig(msg string, err error) *errors.AppError {
	return errors.New(CodeInvalidConfig, "invalid Web3 configuration: "+msg, err)
}

// ErrInvalidSignature creates an error for SIWE/signature verification failures.
func ErrInvalidSignature(msg string, err error) *errors.AppError {
	return errors.New(CodeInvalidSignature, msg, err)
}

// ErrNonceReused creates an error when a SIWE nonce has already been consumed.
func ErrNonceReused(nonce string) *errors.AppError {
	return errors.New(CodeNonceReused, fmt.Sprintf("SIWE nonce already used: %s", nonce), nil)
}

// ErrMessageExpired creates an error when a SIWE message is past ExpirationTime.
func ErrMessageExpired() *errors.AppError {
	return errors.New(CodeMessageExpired, "SIWE message expired", nil)
}

// ErrMessageNotYetValid creates an error when a SIWE message is before NotBefore.
func ErrMessageNotYetValid() *errors.AppError {
	return errors.New(CodeMessageNotYet, "SIWE message not yet valid", nil)
}

// ErrNoSigner creates an error when a transfer is attempted without a private key.
func ErrNoSigner() *errors.AppError {
	return errors.New(CodeNoSigner, "no signer configured", nil)
}

// ErrStorageFailed creates an error for IPFS/storage operations.
func ErrStorageFailed(operation string, err error) *errors.AppError {
	return errors.New(CodeStorageFailed, "storage operation failed: "+operation, err)
}

// ErrTimeout creates an error for timed-out Web3 operations.
func ErrTimeout(operation string, err error) *errors.AppError {
	return errors.New(CodeTimeout, "Web3 operation timed out: "+operation, err)
}

// ErrCanceled creates an error when context is canceled during a Web3 operation.
func ErrCanceled(operation string, err error) *errors.AppError {
	return errors.New(CodeCanceled, "Web3 operation canceled: "+operation, err)
}
