package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
)

// RedisInterface defines the interface for Redis operations
type RedisInterface interface {
	Set(key string, value interface{}, ttl time.Duration) error
	SetWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	Get(key string, dest interface{}) error
	GetWithContext(ctx context.Context, key string, dest interface{}) error
	Delete(key string) error
	DeleteWithContext(ctx context.Context, key string) error
	DeletePattern(pattern string) error
	DeletePatternWithContext(ctx context.Context, pattern string) error
	Exists(key string) (bool, error)
	GetTTL(key string) (time.Duration, error)
	FlushDB() error
	GetStats() (map[string]interface{}, error)
	Close() error
}

// RedisClient wraps the redis client with additional functionality
type RedisClient struct {
	client         *redis.Client
	defaultTimeout time.Duration
}

var (
	redisClient *RedisClient
	once        sync.Once
)

// redisConfig holds Redis connection configuration
type redisConfig struct {
	URL                   string
	DB                    int
	Password              string
	MaxRetries            int
	DialTimeout           time.Duration
	ReadTimeout           time.Duration
	WriteTimeout          time.Duration
	RedisOperationTimeout time.Duration
}

// InitRedis initializes Redis using configuration values from the configuration package
func InitRedis() error {
	var err error
	once.Do(func() {
		config := redisConfig{
			URL:                   configuration.Config.RedisURL,
			DB:                    configuration.Config.RedisDB,
			Password:              configuration.Config.RedisPassword,
			MaxRetries:            configuration.Config.RedisMaxRetries,
			DialTimeout:           configuration.Config.RedisDialTimeout,
			ReadTimeout:           configuration.Config.RedisReadTimeout,
			WriteTimeout:          configuration.Config.RedisWriteTimeout,
			RedisOperationTimeout: configuration.Config.RedisOperationTimeout,
		}

		opts, parseErr := redis.ParseURL(config.URL)
		if parseErr != nil {
			err = fmt.Errorf("failed to parse Redis URL: %w", parseErr)
			return
		}

		// Override with config values if provided
		if config.DB != 0 {
			opts.DB = config.DB
		}
		if config.Password != "" {
			opts.Password = config.Password
		}
		if config.MaxRetries != 0 {
			opts.MaxRetries = config.MaxRetries
		}
		if config.DialTimeout != 0 {
			opts.DialTimeout = config.DialTimeout
		}
		if config.ReadTimeout != 0 {
			opts.ReadTimeout = config.ReadTimeout
		}
		if config.WriteTimeout != 0 {
			opts.WriteTimeout = config.WriteTimeout
		}

		// Configure connection pool using values from configuration
		opts.PoolSize = configuration.Config.RedisPoolSize
		opts.MinIdleConns = configuration.Config.RedisMinIdleConns
		opts.MaxIdleConns = configuration.Config.RedisPoolSize / 2
		opts.PoolTimeout = configuration.Config.RedisPoolTimeout
		opts.ConnMaxIdleTime = 5 * time.Minute  // Close idle connections after 5 minutes
		opts.ConnMaxLifetime = 30 * time.Minute // Maximum connection lifetime

		client := redis.NewClient(opts)

		// Test connection
		ctx := context.Background()
		_, pingErr := client.Ping(ctx).Result()
		if pingErr != nil {
			err = fmt.Errorf("failed to connect to Redis: %w", pingErr)
			return
		}

		redisClient = &RedisClient{
			client:         client,
			defaultTimeout: config.RedisOperationTimeout,
		}

		log.Info().
			Str("component", "redis").
			Str("url", logger.SanitizeConnectionURL(config.URL)).
			Int("db", opts.DB).
			Msg("Redis client initialized successfully")
	})

	if err != nil {
		log.Error().
			Str("component", "redis").
			Err(err).
			Msg("Failed to initialize Redis cache")
		return fmt.Errorf("failed to initialize Redis cache: %w", err)
	}

	return nil
}

// GetRedisClient returns the singleton Redis client
func GetRedisClient() *RedisClient {
	if redisClient == nil {
		log.Error().
			Str("component", "redis").
			Msg("Redis client not initialized - call InitRedis first")
		// Return a nil client - callers should handle this gracefully
		return nil
	}
	return redisClient
}

// IsRedisAvailable checks if Redis client is available and connected
func IsRedisAvailable() bool {
	if redisClient == nil {
		return false
	}

	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	_, err := redisClient.client.Ping(ctx).Result()
	return err == nil
}

// Set stores a value in Redis with TTL
func (r *RedisClient) Set(key string, value interface{}, ttl time.Duration) error {
	return r.SetWithContext(context.Background(), key, value, ttl)
}

// SetWithContext stores a value in Redis with TTL using provided context
func (r *RedisClient) SetWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.defaultTimeout)
	defer cancel()

	data, err := json.Marshal(value)
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "set").
			Str("key", key).
			Err(err).
			Msg("Failed to marshal value")
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	err = r.client.Set(ctx, key, data, ttl).Err()
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "set").
			Str("key", key).
			Err(err).
			Msg("Failed to set value in Redis")
		return fmt.Errorf("failed to set value in Redis: %w", err)
	}

	log.Debug().
		Str("component", "redis").
		Str("operation", "set").
		Str("key", key).
		Dur("ttl", ttl).
		Msg("Value stored in Redis")

	return nil
}

// Get retrieves a value from Redis
func (r *RedisClient) Get(key string, dest interface{}) error {
	return r.GetWithContext(context.Background(), key, dest)
}

// GetWithContext retrieves a value from Redis using provided context
func (r *RedisClient) GetWithContext(ctx context.Context, key string, dest interface{}) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.defaultTimeout)
	defer cancel()

	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			log.Debug().
				Str("component", "redis").
				Str("operation", "get").
				Str("key", key).
				Msg("Key not found in Redis")
			return ErrCacheMiss
		}

		log.Error().
			Str("component", "redis").
			Str("operation", "get").
			Str("key", key).
			Err(err).
			Msg("Failed to get value from Redis")
		return fmt.Errorf("failed to get value from Redis: %w", err)
	}

	err = json.Unmarshal([]byte(data), dest)
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "get").
			Str("key", key).
			Err(err).
			Msg("Failed to unmarshal value")
		return fmt.Errorf("failed to unmarshal value: %w", err)
	}

	log.Debug().
		Str("component", "redis").
		Str("operation", "get").
		Str("key", key).
		Msg("Value retrieved from Redis")

	return nil
}

// Delete removes a key from Redis
func (r *RedisClient) Delete(key string) error {
	return r.DeleteWithContext(context.Background(), key)
}

// DeleteWithContext removes a key from Redis using provided context
func (r *RedisClient) DeleteWithContext(ctx context.Context, key string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, r.defaultTimeout)
	defer cancel()

	err := r.client.Del(ctx, key).Err()
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "delete").
			Str("key", key).
			Err(err).
			Msg("Failed to delete key from Redis")
		return fmt.Errorf("failed to delete key from Redis: %w", err)
	}

	log.Debug().
		Str("component", "redis").
		Str("operation", "delete").
		Str("key", key).
		Msg("Key deleted from Redis")

	return nil
}

// DeletePattern removes all keys matching a pattern using SCAN
func (r *RedisClient) DeletePattern(pattern string) error {
	return r.DeletePatternWithContext(context.Background(), pattern)
}

// DeletePatternWithContext removes all keys matching a pattern using SCAN with provided context
func (r *RedisClient) DeletePatternWithContext(ctx context.Context, pattern string) error {
	// Create context with timeout
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second) // Longer timeout for pattern operations
	defer cancel()

	var cursor uint64
	var totalDeleted int
	batchSize := 100 // Process keys in batches to avoid memory issues

	for {
		keys, nextCursor, err := r.client.Scan(ctx, cursor, pattern, int64(batchSize)).Result()
		if err != nil {
			log.Error().
				Str("component", "redis").
				Str("operation", "delete_pattern_scan").
				Str("pattern", pattern).
				Uint64("cursor", cursor).
				Err(err).
				Msg("Failed to scan keys matching pattern")
			return fmt.Errorf("failed to scan keys matching pattern: %w", err)
		}

		if len(keys) > 0 {
			err = r.client.Del(ctx, keys...).Err()
			if err != nil {
				log.Error().
					Str("component", "redis").
					Str("operation", "delete_pattern_batch").
					Str("pattern", pattern).
					Int("key_count", len(keys)).
					Err(err).
					Msg("Failed to delete keys batch")
				return fmt.Errorf("failed to delete keys batch: %w", err)
			}
			totalDeleted += len(keys)
		}

		cursor = nextCursor
		if cursor == 0 {
			break
		}
	}

	log.Debug().
		Str("component", "redis").
		Str("operation", "delete_pattern").
		Str("pattern", pattern).
		Int("deleted_count", totalDeleted).
		Msg("Keys deleted from Redis using SCAN")

	return nil
}

// Exists checks if a key exists in Redis
func (r *RedisClient) Exists(key string) (bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.defaultTimeout)
	defer cancel()

	result, err := r.client.Exists(ctx, key).Result()
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "exists").
			Str("key", key).
			Err(err).
			Msg("Failed to check key existence")
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}

	exists := result > 0
	log.Debug().
		Str("component", "redis").
		Str("operation", "exists").
		Str("key", key).
		Bool("exists", exists).
		Msg("Key existence checked")

	return exists, nil
}

// GetTTL returns the TTL of a key
func (r *RedisClient) GetTTL(key string) (time.Duration, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.defaultTimeout)
	defer cancel()

	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "get_ttl").
			Str("key", key).
			Err(err).
			Msg("Failed to get TTL")
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}

	log.Debug().
		Str("component", "redis").
		Str("operation", "get_ttl").
		Str("key", key).
		Dur("ttl", ttl).
		Msg("TTL retrieved")

	return ttl, nil
}

// FlushDB clears all keys from the current database
func (r *RedisClient) FlushDB() error {
	ctx, cancel := context.WithTimeout(context.Background(), r.defaultTimeout)
	defer cancel()

	err := r.client.FlushDB(ctx).Err()
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "flush_db").
			Err(err).
			Msg("Failed to flush database")
		return fmt.Errorf("failed to flush database: %w", err)
	}

	log.Info().
		Str("component", "redis").
		Str("operation", "flush_db").
		Msg("Database flushed")

	return nil
}

// GetStats returns Redis statistics
func (r *RedisClient) GetStats() (map[string]interface{}, error) {
	ctx, cancel := context.WithTimeout(context.Background(), r.defaultTimeout)
	defer cancel()

	info, err := r.client.Info(ctx).Result()
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "get_stats").
			Err(err).
			Msg("Failed to get Redis info")
		return nil, fmt.Errorf("failed to get Redis info: %w", err)
	}

	dbSize, err := r.client.DBSize(ctx).Result()
	if err != nil {
		log.Error().
			Str("component", "redis").
			Str("operation", "get_stats").
			Err(err).
			Msg("Failed to get database size")
		return nil, fmt.Errorf("failed to get database size: %w", err)
	}

	stats := map[string]interface{}{
		"info":    info,
		"db_size": dbSize,
	}

	log.Debug().
		Str("component", "redis").
		Str("operation", "get_stats").
		Int64("db_size", dbSize).
		Msg("Redis stats retrieved")

	return stats, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if r.client != nil {
		err := r.client.Close()
		if err != nil {
			log.Error().
				Str("component", "redis").
				Str("operation", "close").
				Err(err).
				Msg("Failed to close Redis client")
			return fmt.Errorf("failed to close Redis client: %w", err)
		}

		log.Info().
			Str("component", "redis").
			Str("operation", "close").
			Msg("Redis client closed")
	}
	return nil
}

// CloseRedis closes the singleton Redis client connection
func CloseRedis() error {
	if redisClient != nil {
		return redisClient.Close()
	}
	return nil
}

// ErrCacheMiss is returned when a key is not found in cache
var ErrCacheMiss = fmt.Errorf("cache miss")
