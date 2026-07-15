/*
Package tokenbucket implements the token-bucket rate limiting strategy.

Local / InMemoryLimiter are process-local. DistLimiter persists bucket state
in a shared cache.Cache so peers sharing the store observe the same budget.
*/
package tokenbucket
