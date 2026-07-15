/*
Package azurekms implements kms.KeyManager using Azure Key Vault Encrypt/Decrypt.

The adapter wraps an EncryptDecryptAPI (satisfied by *azkeys.Client). Tests
inject a fake client; production callers use New or NewFromAPI.
*/
package azurekms
