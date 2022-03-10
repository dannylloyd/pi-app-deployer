package redis

import (
	"context"
	"fmt"

	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/go-redis/redis/v8"
)

const (
	updateConditionStatusPrefix = config.RepoPushStatusTopic
)

type Redis struct {
	client redis.Client
}

func NewRedisClient(redisURL string) (Redis, error) {
	r := Redis{}
	options, err := redis.ParseURL(redisURL)
	options.TLSConfig.InsecureSkipVerify = true
	if err != nil {
		return r, err
	}
	r.client = *redis.NewClient(options)

	return r, nil
}

func (r *Redis) ReadCondition(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, fmt.Sprintf("%s/%s", updateConditionStatusPrefix, key)).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}

func (r *Redis) WriteCondition(ctx context.Context, key string, value string) error {
	d := r.client.Set(ctx, fmt.Sprintf("%s/%s", updateConditionStatusPrefix, key), value, 0)
	err := d.Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) ReadAll(ctx context.Context) (map[string]string, error) {
	state := make(map[string]string)
	keys := r.client.Keys(ctx, fmt.Sprintf("%s/*", updateConditionStatusPrefix)).Val()
	for _, k := range keys {
		val, err := r.client.Get(ctx, k).Result()
		if err != nil {
			return state, err
		}
		state[k] = val
	}
	return state, nil
}
