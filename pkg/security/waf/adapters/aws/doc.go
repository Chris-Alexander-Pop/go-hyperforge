/*
Package aws implements waf.Manager by updating an AWS WAFv2 IP set.

The adapter wraps an IPSetAPI (satisfied by *wafv2.Client). Tests inject a
fake client via NewFromAPI; production callers use New or NewFromAPI.
*/
package aws
