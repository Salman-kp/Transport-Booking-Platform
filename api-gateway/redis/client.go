package redis

import (
	"context"
	"log"

	"github.com/redis/go-redis/v9"
)

// return redis client
func Client(redisHost, redisPort string) *redis.Client {

	rdb := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort,
	})

	err := rdb.Set(context.Background(), "test", "working", 0).Err()
	if err != nil {
		log.Print("redis connection failed")
	}

	_, err = rdb.Get(context.Background(), "test").Result()
	if err != nil {
		log.Print("redis Failed, reason")
	} else {
		log.Print("redis connection succeeded")
	}

	return rdb
}
