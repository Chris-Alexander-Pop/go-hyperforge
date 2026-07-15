/*
Package redis implements analytics.Tracker using Redis HyperLogLog.

Commands used: PFADD, PFCOUNT, PFMERGE, DEL, EXISTS.

Precision from analytics.Config is not applicable — Redis manages a fixed
HyperLogLog structure per key. Keys are stored under the prefix
"analytics:hll:" by default (override with WithKeyPrefix).

	client := goredis.NewClient(&goredis.Options{Addr: "localhost:6379"})
	tracker := redis.New(client)
	defer tracker.Close()
*/
package redis
