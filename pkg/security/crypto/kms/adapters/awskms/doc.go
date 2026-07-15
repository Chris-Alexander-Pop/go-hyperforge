/*
Package awskms implements kms.KeyManager using AWS KMS Encrypt/Decrypt.

The adapter wraps an EncryptDecryptAPI (satisfied by *kms.Client from
aws-sdk-go-v2/service/kms). Tests inject a fake client; production callers
use New or NewFromAPI.
*/
package awskms
