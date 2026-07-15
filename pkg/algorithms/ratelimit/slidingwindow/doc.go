/*
Package slidingwindow implements a sliding-window rate limiter.

Local is a process-local sliding-window log (exact timestamps).

Limiter is a distributed sliding-window counter over cache.Cache: it weights
the previous and current fixed windows by elapsed time so the effective window
slides continuously rather than resetting at period boundaries.
*/
package slidingwindow
