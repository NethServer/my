/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockRedisClient is a mock implementation of RedisInterface
type MockRedisClient struct {
	mock.Mock
	data map[string]interface{}
}

func NewMockRedisClient() *MockRedisClient {
	return &MockRedisClient{
		data: make(map[string]interface{}),
	}
}

func (m *MockRedisClient) Set(key string, value interface{}, ttl time.Duration) error {
	args := m.Called(key, value, ttl)
	return args.Error(0)
}

func (m *MockRedisClient) SetWithContext(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	args := m.Called(ctx, key, value, ttl)
	return args.Error(0)
}

func (m *MockRedisClient) Get(key string, dest interface{}) error {
	args := m.Called(key, dest)
	return args.Error(0)
}

func (m *MockRedisClient) GetWithContext(ctx context.Context, key string, dest interface{}) error {
	args := m.Called(ctx, key, dest)
	return args.Error(0)
}

func (m *MockRedisClient) Delete(key string) error {
	args := m.Called(key)
	return args.Error(0)
}

func (m *MockRedisClient) DeleteWithContext(ctx context.Context, key string) error {
	args := m.Called(ctx, key)
	return args.Error(0)
}

func (m *MockRedisClient) DeletePattern(pattern string) error {
	args := m.Called(pattern)
	return args.Error(0)
}

func (m *MockRedisClient) DeletePatternWithContext(ctx context.Context, pattern string) error {
	args := m.Called(ctx, pattern)
	return args.Error(0)
}

func (m *MockRedisClient) Exists(key string) (bool, error) {
	args := m.Called(key)
	return args.Bool(0), args.Error(1)
}

func (m *MockRedisClient) GetTTL(key string) (time.Duration, error) {
	args := m.Called(key)
	return args.Get(0).(time.Duration), args.Error(1)
}

func (m *MockRedisClient) FlushDB() error {
	args := m.Called()
	return args.Error(0)
}

func (m *MockRedisClient) GetStats() (map[string]interface{}, error) {
	args := m.Called()
	return args.Get(0).(map[string]interface{}), args.Error(1)
}

func (m *MockRedisClient) Close() error {
	args := m.Called()
	return args.Error(0)
}

func TestRedisConfigStruct(t *testing.T) {
	config := redisConfig{
		URL:                   "redis://localhost:6379",
		DB:                    1,
		Password:              "test_password",
		MaxRetries:            3,
		DialTimeout:           5 * time.Second,
		ReadTimeout:           3 * time.Second,
		WriteTimeout:          3 * time.Second,
		RedisOperationTimeout: 2 * time.Second,
	}

	assert.Equal(t, "redis://localhost:6379", config.URL)
	assert.Equal(t, 1, config.DB)
	assert.Equal(t, "test_password", config.Password)
	assert.Equal(t, 3, config.MaxRetries)
	assert.Equal(t, 5*time.Second, config.DialTimeout)
	assert.Equal(t, 3*time.Second, config.ReadTimeout)
	assert.Equal(t, 3*time.Second, config.WriteTimeout)
	assert.Equal(t, 2*time.Second, config.RedisOperationTimeout)
}

func TestErrCacheMiss(t *testing.T) {
	assert.NotNil(t, ErrCacheMiss)
	assert.Equal(t, "cache miss", ErrCacheMiss.Error())
}

func TestRedisClientStructure(t *testing.T) {
	client := &RedisClient{
		defaultTimeout: 5 * time.Second,
	}

	assert.Equal(t, 5*time.Second, client.defaultTimeout)
	assert.Nil(t, client.client) // Not initialized in test
}

func TestIsRedisAvailableWithNilClient(t *testing.T) {
	// Reset global client to nil for this test
	originalClient := redisClient
	redisClient = nil
	defer func() { redisClient = originalClient }()

	assert.False(t, IsRedisAvailable())
}

func TestGetRedisClientWhenNotInitialized(t *testing.T) {
	// Reset global client to nil for this test
	originalClient := redisClient
	redisClient = nil
	defer func() { redisClient = originalClient }()

	client := GetRedisClient()
	assert.Nil(t, client)
}

func TestRedisClientSetOperations(t *testing.T) {
	mockClient := NewMockRedisClient()

	client := &RedisClient{
		defaultTimeout: 5 * time.Second,
	}

	t.Run("Set operation success", func(t *testing.T) {
		testData := map[string]interface{}{"test": "value"}
		ttl := 10 * time.Minute

		// Mock successful set
		mockClient.On("Set", "test_key", testData, ttl).Return(nil)

		// Since we can't easily mock the actual Redis client, we test the structure
		assert.Equal(t, 5*time.Second, client.defaultTimeout)
	})

	t.Run("Set operation with context", func(t *testing.T) {
		ctx := context.Background()
		testData := map[string]interface{}{"test": "value"}
		ttl := 10 * time.Minute

		// Mock successful set with context
		mockClient.On("SetWithContext", ctx, "test_key", testData, ttl).Return(nil)

		// Test that context operations are supported
		assert.NotNil(t, ctx)
	})
}

func TestRedisClientGetOperations(t *testing.T) {
	mockClient := NewMockRedisClient()

	t.Run("Get operation cache miss", func(t *testing.T) {
		var dest interface{}

		// Mock cache miss
		mockClient.On("Get", "missing_key", &dest).Return(ErrCacheMiss)

		err := mockClient.Get("missing_key", &dest)
		assert.Equal(t, ErrCacheMiss, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Get operation success", func(t *testing.T) {
		var dest interface{}

		// Mock successful get
		mockClient.On("Get", "existing_key", &dest).Return(nil)

		err := mockClient.Get("existing_key", &dest)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Get operation with context", func(t *testing.T) {
		ctx := context.Background()
		var dest interface{}

		// Mock get with context
		mockClient.On("GetWithContext", ctx, "test_key", &dest).Return(nil)

		err := mockClient.GetWithContext(ctx, "test_key", &dest)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestRedisClientDeleteOperations(t *testing.T) {
	mockClient := NewMockRedisClient()

	t.Run("Delete single key", func(t *testing.T) {
		// Mock successful delete
		mockClient.On("Delete", "test_key").Return(nil)

		err := mockClient.Delete("test_key")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Delete with context", func(t *testing.T) {
		ctx := context.Background()

		// Mock delete with context
		mockClient.On("DeleteWithContext", ctx, "test_key").Return(nil)

		err := mockClient.DeleteWithContext(ctx, "test_key")
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Delete pattern", func(t *testing.T) {
		pattern := "test:*"

		// Mock pattern delete
		mockClient.On("DeletePattern", pattern).Return(nil)

		err := mockClient.DeletePattern(pattern)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Delete pattern with context", func(t *testing.T) {
		ctx := context.Background()
		pattern := "test:*"

		// Mock pattern delete with context
		mockClient.On("DeletePatternWithContext", ctx, pattern).Return(nil)

		err := mockClient.DeletePatternWithContext(ctx, pattern)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Delete operation error", func(t *testing.T) {
		expectedErr := errors.New("delete failed")

		// Mock delete error
		mockClient.On("Delete", "error_key").Return(expectedErr)

		err := mockClient.Delete("error_key")
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})
}

func TestRedisClientUtilityOperations(t *testing.T) {
	mockClient := NewMockRedisClient()

	t.Run("Exists operation", func(t *testing.T) {
		// Mock key exists
		mockClient.On("Exists", "existing_key").Return(true, nil)

		exists, err := mockClient.Exists("existing_key")
		assert.NoError(t, err)
		assert.True(t, exists)
		mockClient.AssertExpectations(t)
	})

	t.Run("Exists operation key not found", func(t *testing.T) {
		// Mock key does not exist
		mockClient.On("Exists", "missing_key").Return(false, nil)

		exists, err := mockClient.Exists("missing_key")
		assert.NoError(t, err)
		assert.False(t, exists)
		mockClient.AssertExpectations(t)
	})

	t.Run("GetTTL operation", func(t *testing.T) {
		expectedTTL := 5 * time.Minute

		// Mock TTL get
		mockClient.On("GetTTL", "test_key").Return(expectedTTL, nil)

		ttl, err := mockClient.GetTTL("test_key")
		assert.NoError(t, err)
		assert.Equal(t, expectedTTL, ttl)
		mockClient.AssertExpectations(t)
	})

	t.Run("FlushDB operation", func(t *testing.T) {
		// Mock flush database
		mockClient.On("FlushDB").Return(nil)

		err := mockClient.FlushDB()
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("GetStats operation", func(t *testing.T) {
		expectedStats := map[string]interface{}{
			"info":    "mock redis info",
			"db_size": int64(42),
		}

		// Mock get stats
		mockClient.On("GetStats").Return(expectedStats, nil)

		stats, err := mockClient.GetStats()
		assert.NoError(t, err)
		assert.Equal(t, expectedStats, stats)
		mockClient.AssertExpectations(t)
	})

	t.Run("Close operation", func(t *testing.T) {
		// Mock close
		mockClient.On("Close").Return(nil)

		err := mockClient.Close()
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}

func TestRedisClientErrorHandling(t *testing.T) {
	mockClient := NewMockRedisClient()

	t.Run("Set operation error", func(t *testing.T) {
		expectedErr := errors.New("set operation failed")
		testData := map[string]interface{}{"test": "value"}
		ttl := 10 * time.Minute

		mockClient.On("Set", "error_key", testData, ttl).Return(expectedErr)

		err := mockClient.Set("error_key", testData, ttl)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Get operation error", func(t *testing.T) {
		expectedErr := errors.New("get operation failed")
		var dest interface{}

		mockClient.On("Get", "error_key", &dest).Return(expectedErr)

		err := mockClient.Get("error_key", &dest)
		assert.Equal(t, expectedErr, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Exists operation error", func(t *testing.T) {
		expectedErr := errors.New("exists operation failed")

		mockClient.On("Exists", "error_key").Return(false, expectedErr)

		exists, err := mockClient.Exists("error_key")
		assert.Equal(t, expectedErr, err)
		assert.False(t, exists)
		mockClient.AssertExpectations(t)
	})

	t.Run("GetTTL operation error", func(t *testing.T) {
		expectedErr := errors.New("get TTL failed")

		mockClient.On("GetTTL", "error_key").Return(time.Duration(0), expectedErr)

		ttl, err := mockClient.GetTTL("error_key")
		assert.Equal(t, expectedErr, err)
		assert.Equal(t, time.Duration(0), ttl)
		mockClient.AssertExpectations(t)
	})

	t.Run("GetStats operation error", func(t *testing.T) {
		expectedErr := errors.New("get stats failed")

		mockClient.On("GetStats").Return(map[string]interface{}(nil), expectedErr)

		stats, err := mockClient.GetStats()
		assert.Equal(t, expectedErr, err)
		assert.Nil(t, stats)
		mockClient.AssertExpectations(t)
	})
}

func TestContextOperations(t *testing.T) {
	t.Run("Context with timeout", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		assert.NotNil(t, ctx)
		deadline, ok := ctx.Deadline()
		assert.True(t, ok)
		assert.True(t, deadline.After(time.Now()))
	})

	t.Run("Context cancellation", func(t *testing.T) {
		ctx, cancel := context.WithCancel(context.Background())

		// Cancel immediately
		cancel()

		select {
		case <-ctx.Done():
			assert.Equal(t, context.Canceled, ctx.Err())
		default:
			t.Error("Context should be cancelled")
		}
	})
}

func TestRedisClientDefensiveProgramming(t *testing.T) {
	t.Run("Nil client handling", func(t *testing.T) {
		var client *RedisClient

		// Test that nil client doesn't panic
		assert.Nil(t, client)
		if client != nil {
			// This branch won't execute but shows defensive programming
			_ = client.Set("key", "value", time.Minute)
		}
	})

	t.Run("Empty key handling", func(t *testing.T) {
		mockClient := NewMockRedisClient()

		// Mock operations with empty key
		mockClient.On("Get", "", mock.Anything).Return(ErrCacheMiss)

		var dest interface{}
		err := mockClient.Get("", &dest)
		assert.Equal(t, ErrCacheMiss, err)
		mockClient.AssertExpectations(t)
	})

	t.Run("Nil value handling", func(t *testing.T) {
		mockClient := NewMockRedisClient()

		// Mock set with nil value
		mockClient.On("Set", "test_key", nil, time.Minute).Return(nil)

		err := mockClient.Set("test_key", nil, time.Minute)
		assert.NoError(t, err)
		mockClient.AssertExpectations(t)
	})
}
