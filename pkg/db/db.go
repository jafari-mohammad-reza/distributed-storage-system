package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/redis/go-redis/v9"
)

type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"password"`
	CreatedAt time.Time `json:"created_at"`
	Agents    []Agent   `json:"agents"`
	Files     []File    `json:"files"`
}

type Agent struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	LastRequest time.Time `json:"last_request"`
}

type File struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	Path       string    `json:"path"`
	UploadedAt time.Time `json:"uploaded_at"`
	UploadedBy string     `json:"uploaded_by"`
	Versions   []FileVersion
}

type FileVersion struct {
	ID        string    `json:"id"`
	Hash      string    `json:"hash"`
	CreatedAt time.Time `json:"created_at"`
}

func InsertMany(ctx context.Context, redisClient *redis.Client, data map[string]any) error {
	for key, val := range data {
		if err := Insert(ctx, redisClient, key, val); err != nil {
			return fmt.Errorf("failed to insert key %s: %w", key, err)
		}
	}
	return nil
}

func Insert(ctx context.Context, redisClient *redis.Client, key string, val any) error {
	result, err := redisClient.JSONSet(ctx, key, "$", val).Result()
	slog.Info("Insert into Redis as JSON", "key", key, "value", val, "result", result)
	if err != nil {
		return fmt.Errorf("failed to insert key %s: %w", key, err)
	}
	return nil
}

func Get(ctx context.Context, redisClient *redis.Client, key string) (string, error) {
	result, err := redisClient.JSONGet(ctx, key, "$").Result()
	if err == redis.Nil {
		return "", nil
	} else if err != nil {
		return "", fmt.Errorf("error checking key existence: %w", err)
	}
	return result, nil
}

func ValExists(ctx context.Context, redisClient *redis.Client, key, val string, path ...string) (bool, error) {
	result, err := redisClient.JSONGet(ctx, key, path...).Result()
	if err == redis.Nil {
		return false, nil
	} else if err != nil {
		return false, fmt.Errorf("error checking value existence: %w", err)
	}

	if result == val {
		return true, nil
	}
	return false, nil
}

func AppendArray(ctx context.Context, redisClient *redis.Client, key string, val any, path string) error {
	_, err := redisClient.JSONArrAppend(ctx, key, path, val).Result()
	if err != nil {
		return fmt.Errorf("failed to append value to array: %w", err)
	}
	return nil
}

func PopArray(ctx context.Context, redisClient *redis.Client, key, path, val string) error {
	arr, err := redisClient.JSONGet(ctx, key, path).Result()
	if err != nil {
		return fmt.Errorf("failed to get array: %w", err)
	}

	var values []string
	if err := json.Unmarshal([]byte(arr), &values); err != nil {
		return fmt.Errorf("failed to unmarshal array: %w", err)
	}

	index := -1
	for i, v := range values {
		if v == val {
			index = i
			break
		}
	}

	if index == -1 {
		return fmt.Errorf("value not found in array")
	}

	_, err = redisClient.JSONArrPop(ctx, key, path, index).Result()
	if err != nil {
		return fmt.Errorf("failed to pop value from array at index %d: %w", index, err)
	}
	return nil
}
