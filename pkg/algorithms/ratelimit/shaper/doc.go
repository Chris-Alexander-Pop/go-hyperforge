/*
Package shaper provides a traffic shaping implementation using a Leaky Bucket algorithm.
Unlike the rate limiter (which drops requests), the shaper queues and delays requests to smooth out bursts.
*/
package shaper
