package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/andrewmarklloyd/pi-app-deployer/api/v1/status"
	"github.com/andrewmarklloyd/pi-app-deployer/internal/pkg/config"
	"github.com/go-redis/redis/v8"
)

const (
	updateConditionStatusPrefix = config.RepoPushStatusTopic
	agentInventoryPrefix        = config.AgentInventoryTopic
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

func (r *Redis) ReadConditions(ctx context.Context, repoName, manifestName string) (map[string]status.UpdateCondition, error) {
	state := make(map[string]status.UpdateCondition)
	k := getReadKey(repoName, manifestName)
	keys := r.client.Keys(ctx, fmt.Sprintf("%s*", k)).Val()
	p := strings.ReplaceAll(k, "*", "")
	for _, k := range keys {
		host := strings.ReplaceAll(k, p, "")
		val, err := r.client.Get(ctx, k).Result()
		if err != nil {
			return state, err
		}
		var uc status.UpdateCondition
		err = json.Unmarshal([]byte(val), &uc)
		if err != nil {
			return state, err
		}
		state[host] = uc
	}
	return state, nil
}

func (r *Redis) DeleteConditions(ctx context.Context, repoName, manifestName string) error {
	m, err := r.ReadConditions(ctx, repoName, manifestName)
	if err != nil {
		return err
	}
	for k := range m {
		_, err := r.client.Del(ctx, k).Result()
		if err != nil {
			return err
		}
	}

	return nil
}

func (r *Redis) WriteCondition(ctx context.Context, uc status.UpdateCondition) error {
	key := getWriteKey(uc.RepoName, uc.ManifestName, uc.Host)
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

func (r *Redis) WriteAgentInventory(ctx context.Context, c config.AgentInventoryPayload, expiration time.Duration) error {
	key := getAgentInventoryWriteKey(c.RepoName, c.ManifestName, c.Host)
	d := r.client.Set(ctx, key, c.Timestamp, expiration)
	err := d.Err()
	if err != nil {
		return err
	}
	return nil
}

func (r *Redis) ReadAgentInventory(ctx context.Context, repoName, manifestName string) (map[string]time.Time, error) {
	agents := make(map[string]time.Time, 0)
	readKey := getAgentInventoryReadKey(repoName, manifestName)
	keys := r.client.Keys(ctx, readKey).Val()
	p := strings.ReplaceAll(readKey, "*", "")
	for _, k := range keys {
		host := strings.ReplaceAll(k, p, "")
		val, err := r.client.Get(ctx, k).Result()
		if err != nil {
			return agents, err
		}
		n, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return agents, err
		}
		agents[host] = time.Unix(n, 0)
	}

	return agents, nil
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

func getWriteKey(repoName, manifestName, host string) string {
	key := fmt.Sprintf("%s/%s/%s", repoName, manifestName, host)
	return fmt.Sprintf("%s/%s", updateConditionStatusPrefix, key)
}

func getReadKey(repoName, manifestName string) string {
	key := fmt.Sprintf("%s/%s/*", repoName, manifestName)
	return fmt.Sprintf("%s/%s", updateConditionStatusPrefix, key)
}

func getAgentInventoryWriteKey(repoName, manifestName, host string) string {
	key := fmt.Sprintf("%s/%s/%s", repoName, manifestName, host)
	return fmt.Sprintf("%s/%s", agentInventoryPrefix, key)
}

func getAgentInventoryReadKey(repoName, manifestName string) string {
	key := fmt.Sprintf("%s/%s/*", repoName, manifestName)
	return fmt.Sprintf("%s/%s", agentInventoryPrefix, key)
}
