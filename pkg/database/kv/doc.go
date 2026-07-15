/*
Package kv provides a unified interface for key-value database access.

Shipping backends:
  - Redis: Production-grade in-memory key-value store
  - Memory: In-memory store for testing

Planned (not implemented): Cassandra wide-column adapter (DriverCassandra reserved).

Basic usage:

	import (
		"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv"
		"github.com/chris-alexander-pop/go-hyperforge/pkg/database/kv/adapters/redis"
	)

	cfg := kv.Config{
		Driver:   "redis",
		Host:     "localhost",
		Port:     "6379",
		Password: "",
	}

	client, err := redis.New(cfg)
	if err != nil {
		log.Fatal(err)
	}
	defer client.Close()

	// Store and retrieve values
	err = client.Set(ctx, "mykey", []byte("myvalue"), time.Hour)
	value, err := client.Get(ctx, "mykey")
*/
package kv
