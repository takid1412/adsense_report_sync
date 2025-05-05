package db

import (
	"github.com/redis/go-redis/v9"
	"os"
	"strings"
)

var rdb *redis.ClusterClient = nil

func GetRedisClient() *redis.ClusterClient {
	if rdb == nil {
		rdb = redis.NewClusterClient(&redis.ClusterOptions{
			Addrs: strings.Split(os.Getenv("REDIS_CLUSTER"), ","),
		})
	}
	return rdb

}
