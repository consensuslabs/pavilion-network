// cache.go
package main

import (
	"context"
	"log"

	"github.com/go-redis/redis/v8"
)

var ctx = context.Background()
var Cache *redis.Client

func ConnectRedis() {
	Cache = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379", // adjust if using a different host/port
		Password: "",               // no password set
		DB:       0,                // use default DB
	})

	// Test the connection
	pong, err := Cache.Ping(ctx).Result()
	if err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Printf("Redis connected: %s", pong)
}
