package redis

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"time"
)

var (
	rPool *redis.Pool
	redisHost = "127.0.0.1:6379"
)

func newRedisPool() *redis.Pool {
	return &redis.Pool{
		MaxIdle: 50,
		MaxActive: 30,
		IdleTimeout: 300*time.Second,
		Dial: func() (redis.Conn, error) {
			//打开连接
			conn, err := redis.Dial("tcp", redisHost)
			if err != nil {
				fmt.Println(err.Error())
				return nil, err
			}
			return conn, nil
		},
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			if time.Since(t) < time.Minute {
				return nil
			}
			_, err := c.Do("PING")
			return err
		},
	}
}

func init()  {
	rPool = newRedisPool()
}

func RedisPool() *redis.Pool {
	return rPool
}
