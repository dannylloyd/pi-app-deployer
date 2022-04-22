package redis

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/status"
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

func (r *Redis) ReadCondition(ctx context.Context, repoName, manifestName string) (string, error) {
	val, err := r.client.Get(ctx, conditionKey(repoName, manifestName)).Result()
	if err != nil {
		return "", err
	}

	return val, nil
}

func (r *Redis) WriteCondition(ctx context.Context, uc status.UpdateCondition) error {
	key := conditionKey(uc.RepoName, uc.ManifestName)
	value, err := json.Marshal(uc)
	if err != nil {
		return fmt.Errorf("marshalling json: %s", err)
	}
	d := r.client.Set(ctx, key, value, 0)
	err = d.Err()
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

func conditionKey(repoName, manifestName string) string {
	key := fmt.Sprintf("%s/%s", repoName, manifestName)
	return fmt.Sprintf("%s/%s", updateConditionStatusPrefix, key)
}
