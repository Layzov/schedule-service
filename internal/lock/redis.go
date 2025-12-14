package lock

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

type Locker interface {
	Lock(ctx context.Context, key string, ttl time.Duration) (bool, error)
	Unlock(ctx context.Context, key string) error
}

type RedisLock struct {
	client *redis.Client
}

func NewRedisLock(redisAddr string) (*RedisLock, error) {
	const op = "lock.NewRedisLock"

	client := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("%s: %w", op, err)
	}

	return &RedisLock{client: client}, nil
}

func (r *RedisLock) Lock(ctx context.Context, key string, ttl time.Duration) (bool, error) {
	const op = "lock.RedisLock.Lock"

	lockKey := fmt.Sprintf("lock:%s", key)
	result, err := r.client.SetNX(ctx, lockKey, "1", ttl).Result()
	if err != nil {
		return false, fmt.Errorf("%s: %w", op, err)
	}

	return result, nil
}

func (r *RedisLock) Unlock(ctx context.Context, key string) error {
	const op = "lock.RedisLock.Unlock"

	lockKey := fmt.Sprintf("lock:%s", key)
	_, err := r.client.Del(ctx, lockKey).Result()
	if err != nil {
		return fmt.Errorf("%s: %w", op, err)
	}

	return nil
}

func (r *RedisLock) Close() error {
	return r.client.Close()
}

