/*
Package vault implements secrets.SecretManager against HashiCorp Vault KV v2.

Uses the Vault HTTP API with token authentication (X-Vault-Token). Mount path
defaults to "secret" (KV v2). Suitable for production when pointed at a real
Vault; tests inject a custom HTTPClient / BaseURL.

Rotate writes a new KV v2 version (does not call Vault's rotate API for
dynamic secrets engines).
*/
package vault
