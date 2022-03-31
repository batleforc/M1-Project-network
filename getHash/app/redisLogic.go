package app

import (
	"context"

	"github.com/go-redis/redis/v8"
)

func Get(name string, redisIP string) (string, error) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisIP,
		Password: "tpuser",
		DB:       0, // use default DB
	})
	defer rdb.Close()

	return rdb.Get(ctx, name).Result()
}

func GetKeys(redisIP string) []string {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisIP,
		Password: "tpuser",
		DB:       0, // use default DB
	})
	defer rdb.Close()

	var cursor uint64
	keys, _, _ := rdb.Scan(ctx, cursor, "prefix:*", 0).Result()
	return keys;
}

func HashExist(hash string, redisIP string) bool {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisIP,
		Password: "tpuser",
		DB:       0, // use default DB
	})
	defer rdb.Close()

	var cursor uint64
	keys, _, err := rdb.Scan(ctx, cursor, "prefix:*", 0).Result()
	if err != nil {
		panic(err)
	}

	for _, key := range keys {
		value, _ := rdb.Get(ctx, key).Result()
		if value == hash {
			return true
		}
	}
	return false
}

func Insert(key, value string , redisIP string) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisIP,
		Password: "tpuser",
		DB:       0, // use default DB
	})
	defer rdb.Close()

	err := rdb.Set(ctx, key, value, 0).Err()
    if err != nil {
        panic(err)
    }
}

func Drop(redisIP string) {
	var ctx = context.Background()
	rdb := redis.NewClient(&redis.Options{
		Addr:     redisIP,
		Password: "tpuser",
		DB:       0, // use default DB
	})
	defer rdb.Close()

	err := rdb.Del(ctx, "prefix:*").Err()
    if err != nil {
        panic(err)
    }
}