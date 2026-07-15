/*
Package gcpkms implements kms.KeyManager using Google Cloud KMS Encrypt/Decrypt.

The adapter wraps an EncryptDecryptAPI (satisfied by *kms.KeyManagementClient).
Tests inject a fake client; production callers use New or NewFromAPI.
*/
package gcpkms
