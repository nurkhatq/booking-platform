package cache

import (
    "context"
    "encoding/json"
    "fmt"
    "time"
    
    "github.com/go-redis/redis/v8"
    
    "booking-platform/shared/config"
)

var Client *redis.Client

func Initialize(cfg *config.Config) error {
    Client = redis.NewClient(&redis.Options{
        Addr:     fmt.Sprintf("%s:%d", cfg.Redis.Host, cfg.Redis.Port),
        Password: cfg.Redis.Password,
        DB:       cfg.Redis.DB,
        PoolSize: cfg.Redis.MaxConnections,
    })
    
    // Test connection
    ctx := context.Background()
    if err := Client.Ping(ctx).Err(); err != nil {
        return fmt.Errorf("failed to connect to Redis: %w", err)
    }
    
    fmt.Println("Redis connection established successfully")
    return nil
}

func Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
    data, err := json.Marshal(value)
    if err != nil {
        return err
    }
    return Client.Set(ctx, key, data, expiration).Err()
}

func Get(ctx context.Context, key string, dest interface{}) error {
    data, err := Client.Get(ctx, key).Result()
    if err != nil {
        return err
    }
    return json.Unmarshal([]byte(data), dest)
}

func Delete(ctx context.Context, key string) error {
    return Client.Del(ctx, key).Err()
}

func Exists(ctx context.Context, key string) bool {
    count := Client.Exists(ctx, key).Val()
    return count > 0
}

func Close() error {
    if Client != nil {
        return Client.Close()
    }
    return nil
}
