/*
Copyright (C) 2025 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package cache

import (
	"crypto/rsa"
	"math/big"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockHTTPClient is a mock implementation of HTTPClient
type MockHTTPClient struct {
	mock.Mock
}

func (m *MockHTTPClient) Get(url string) (*http.Response, error) {
	args := m.Called(url)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*http.Response), args.Error(1)
}

func TestJWKStruct(t *testing.T) {
	jwk := JWK{
		Kid: "key_123",
		Kty: "RSA",
		Use: "sig",
		N:   "test_modulus",
		E:   "AQAB",
	}

	assert.Equal(t, "key_123", jwk.Kid)
	assert.Equal(t, "RSA", jwk.Kty)
	assert.Equal(t, "sig", jwk.Use)
	assert.Equal(t, "test_modulus", jwk.N)
	assert.Equal(t, "AQAB", jwk.E)
}

func TestJWKSetStruct(t *testing.T) {
	jwks := JWKSet{
		Keys: []JWK{
			{
				Kid: "key_1",
				Kty: "RSA",
				Use: "sig",
				N:   "modulus_1",
				E:   "AQAB",
			},
			{
				Kid: "key_2",
				Kty: "RSA",
				Use: "sig",
				N:   "modulus_2",
				E:   "AQAB",
			},
		},
	}

	assert.Len(t, jwks.Keys, 2)
	assert.Equal(t, "key_1", jwks.Keys[0].Kid)
	assert.Equal(t, "key_2", jwks.Keys[1].Kid)
}

func TestJWKSCacheStruct(t *testing.T) {
	now := time.Now()

	// Create test RSA public key
	pubKey := &rsa.PublicKey{
		N: big.NewInt(12345),
		E: 65537,
	}

	keys := map[string]*rsa.PublicKey{
		"key_1": pubKey,
	}

	cache := JWKSCache{
		Keys:      keys,
		CachedAt:  now,
		ExpiresAt: now.Add(time.Hour),
	}

	assert.Len(t, cache.Keys, 1)
	assert.Contains(t, cache.Keys, "key_1")
	assert.Equal(t, pubKey, cache.Keys["key_1"])
	assert.Equal(t, now, cache.CachedAt)
	assert.True(t, cache.ExpiresAt.After(now))
}

func TestJWKSCacheManagerStruct(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mockHTTP := &MockHTTPClient{}
	ttl := 5 * time.Minute
	endpoint := "https://example.com/.well-known/jwks.json"

	manager := &JWKSCacheManager{
		redis:      mockRedis,
		ttl:        ttl,
		endpoint:   endpoint,
		httpClient: mockHTTP,
	}

	assert.Equal(t, mockRedis, manager.redis)
	assert.Equal(t, ttl, manager.ttl)
	assert.Equal(t, endpoint, manager.endpoint)
	assert.Equal(t, mockHTTP, manager.httpClient)
}

func TestJWKSCacheManagerGetPublicKey(t *testing.T) {
	mockRedis := NewMockRedisClient()
	mockHTTP := &MockHTTPClient{}
	manager := &JWKSCacheManager{
		redis:      mockRedis,
		ttl:        5 * time.Minute,
		endpoint:   "https://example.com/.well-known/jwks.json",
		httpClient: mockHTTP,
	}

	kid := "test_key_id"
	cacheKey := "jwks:keys"

	t.Run("Cache hit with valid key", func(t *testing.T) {
		now := time.Now()
		pubKey := &rsa.PublicKey{
			N: big.NewInt(12345),
			E: 65537,
		}

		cached := JWKSCache{
			Keys: map[string]*rsa.PublicKey{
				kid: pubKey,
			},
			CachedAt:  now,
			ExpiresAt: now.Add(time.Hour), // Not expired
		}

		mockRedis.On("Get", cacheKey, mock.AnythingOfType("*cache.JWKSCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*JWKSCache)
			*arg = cached
		})

		key, err := manager.GetPublicKey(kid)
		assert.NoError(t, err)
		assert.Equal(t, pubKey, key)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Cache hit but expired", func(t *testing.T) {
		mockRedis := NewMockRedisClient() // Fresh mock for this test
		mockHTTP := &MockHTTPClient{}
		manager := &JWKSCacheManager{
			redis:      mockRedis,
			ttl:        5 * time.Minute,
			endpoint:   "https://example.com/.well-known/jwks.json",
			httpClient: mockHTTP,
		}

		// For expired cache, the method checks time.Now().Before(cached.ExpiresAt)
		// So we need ExpiresAt to be in the past for it to be considered expired
		now := time.Now()
		pubKey := &rsa.PublicKey{
			N: big.NewInt(12345),
			E: 65537,
		}

		cached := JWKSCache{
			Keys: map[string]*rsa.PublicKey{
				kid: pubKey,
			},
			CachedAt:  now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(-time.Hour), // Expired (in the past)
		}

		mockRedis.On("Get", cacheKey, mock.AnythingOfType("*cache.JWKSCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*JWKSCache)
			*arg = cached
		})

		// Mock HTTP client to return an error when trying to fetch JWKS
		mockHTTP.On("Get", "https://example.com/.well-known/jwks.json").Return(nil, assert.AnError)

		// Since fetchAndCacheJWKS tries to make HTTP requests, it will fail with mock error
		key, err := manager.GetPublicKey(kid)
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), "failed to fetch JWKS")
		mockRedis.AssertExpectations(t)
		mockHTTP.AssertExpectations(t)
	})

	t.Run("Cache miss", func(t *testing.T) {
		mockRedis := NewMockRedisClient() // Fresh mock for this test
		mockHTTP := &MockHTTPClient{}
		manager := &JWKSCacheManager{
			redis:      mockRedis,
			ttl:        5 * time.Minute,
			endpoint:   "https://example.com/.well-known/jwks.json",
			httpClient: mockHTTP,
		}

		mockRedis.On("Get", cacheKey, mock.AnythingOfType("*cache.JWKSCache")).Return(ErrCacheMiss)

		// Mock HTTP client to return an error when trying to fetch JWKS
		mockHTTP.On("Get", "https://example.com/.well-known/jwks.json").Return(nil, assert.AnError)

		// Since fetchAndCacheJWKS uses mock HTTP client, we expect an error
		key, err := manager.GetPublicKey(kid)
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), "failed to fetch JWKS")
		mockRedis.AssertExpectations(t)
		mockHTTP.AssertExpectations(t)
	})

	t.Run("Cache hit but key not found", func(t *testing.T) {
		mockRedis := NewMockRedisClient() // Fresh mock for this test
		mockHTTP := &MockHTTPClient{}
		manager := &JWKSCacheManager{
			redis:      mockRedis,
			ttl:        5 * time.Minute,
			endpoint:   "https://example.com/.well-known/jwks.json",
			httpClient: mockHTTP,
		}

		now := time.Now()
		cached := JWKSCache{
			Keys: map[string]*rsa.PublicKey{
				"different_key": {N: big.NewInt(12345), E: 65537},
			},
			CachedAt:  now,
			ExpiresAt: now.Add(time.Hour),
		}

		mockRedis.On("Get", cacheKey, mock.AnythingOfType("*cache.JWKSCache")).Return(nil).Run(func(args mock.Arguments) {
			arg := args.Get(1).(*JWKSCache)
			*arg = cached
		})

		// Mock HTTP client to return an error when trying to fetch JWKS
		mockHTTP.On("Get", "https://example.com/.well-known/jwks.json").Return(nil, assert.AnError)

		// Since the requested key is not in cache, it will try to fetch from endpoint
		key, err := manager.GetPublicKey(kid)
		assert.Error(t, err)
		assert.Nil(t, key)
		assert.Contains(t, err.Error(), "failed to fetch JWKS")
		mockRedis.AssertExpectations(t)
		mockHTTP.AssertExpectations(t)
	})
}

func TestJWKSCacheManagerClearCache(t *testing.T) {
	expectedPattern := "jwks:*"

	t.Run("Successful clear", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		mockHTTP := &MockHTTPClient{}
		manager := &JWKSCacheManager{
			redis:      mockRedis,
			ttl:        5 * time.Minute,
			endpoint:   "https://example.com/.well-known/jwks.json",
			httpClient: mockHTTP,
		}

		mockRedis.On("DeletePattern", expectedPattern).Return(nil)

		err := manager.ClearCache()
		assert.NoError(t, err)
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during clear", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		mockHTTP := &MockHTTPClient{}
		manager := &JWKSCacheManager{
			redis:      mockRedis,
			ttl:        5 * time.Minute,
			endpoint:   "https://example.com/.well-known/jwks.json",
			httpClient: mockHTTP,
		}

		mockRedis.On("DeletePattern", expectedPattern).Return(assert.AnError)

		err := manager.ClearCache()
		assert.Error(t, err)
		assert.Equal(t, assert.AnError, err)
		mockRedis.AssertExpectations(t)
	})
}

func TestJWKSCacheManagerGetStats(t *testing.T) {
	t.Run("Successful stats retrieval", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		mockHTTP := &MockHTTPClient{}
		manager := &JWKSCacheManager{
			redis:      mockRedis,
			ttl:        5 * time.Minute,
			endpoint:   "https://example.com/.well-known/jwks.json",
			httpClient: mockHTTP,
		}

		expectedRedisStats := map[string]interface{}{
			"connections": 10,
			"memory_used": "1MB",
		}

		mockRedis.On("GetStats").Return(expectedRedisStats, nil)

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Equal(t, 5.0, stats["ttl_minutes"])
		assert.Equal(t, "https://example.com/.well-known/jwks.json", stats["endpoint"])
		assert.Equal(t, expectedRedisStats, stats["redis_stats"])
		assert.Equal(t, "jwks:", stats["cache_prefix"])
		mockRedis.AssertExpectations(t)
	})

	t.Run("Redis error during stats retrieval", func(t *testing.T) {
		mockRedis := NewMockRedisClient()
		mockHTTP := &MockHTTPClient{}
		manager := &JWKSCacheManager{
			redis:      mockRedis,
			ttl:        5 * time.Minute,
			endpoint:   "https://example.com/.well-known/jwks.json",
			httpClient: mockHTTP,
		}

		mockRedis.On("GetStats").Return(map[string]interface{}(nil), assert.AnError)

		stats := manager.GetStats()
		assert.NotNil(t, stats)
		assert.Contains(t, stats, "error")
		assert.Equal(t, "failed to get Redis stats", stats["error"])
		assert.Equal(t, 5.0, stats["ttl_minutes"])
		mockRedis.AssertExpectations(t)
	})
}

func TestJwkToRSAPublicKey(t *testing.T) {
	t.Run("Valid JWK conversion", func(t *testing.T) {
		// Standard RSA exponent (65537) encoded in base64url
		// AQAB = [1, 0, 1] = 65537
		jwk := JWK{
			Kid: "test_key",
			Kty: "RSA",
			Use: "sig",
			N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS",
			E:   "AQAB",
		}

		pubKey, err := JwkToRSAPublicKey(jwk)
		assert.NoError(t, err)
		assert.NotNil(t, pubKey)
		assert.Equal(t, 65537, pubKey.E)
		assert.NotNil(t, pubKey.N)
		assert.True(t, pubKey.N.Cmp(big.NewInt(0)) > 0)
	})

	t.Run("Invalid modulus encoding", func(t *testing.T) {
		jwk := JWK{
			Kid: "test_key",
			Kty: "RSA",
			Use: "sig",
			N:   "invalid_base64!!!",
			E:   "AQAB",
		}

		pubKey, err := JwkToRSAPublicKey(jwk)
		assert.Error(t, err)
		assert.Nil(t, pubKey)
		assert.Contains(t, err.Error(), "failed to decode modulus")
	})

	t.Run("Invalid exponent encoding", func(t *testing.T) {
		jwk := JWK{
			Kid: "test_key",
			Kty: "RSA",
			Use: "sig",
			N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS",
			E:   "invalid_base64!!!",
		}

		pubKey, err := JwkToRSAPublicKey(jwk)
		assert.Error(t, err)
		assert.Nil(t, pubKey)
		assert.Contains(t, err.Error(), "failed to decode exponent")
	})

	t.Run("Empty modulus", func(t *testing.T) {
		jwk := JWK{
			Kid: "test_key",
			Kty: "RSA",
			Use: "sig",
			N:   "",
			E:   "AQAB",
		}

		pubKey, err := JwkToRSAPublicKey(jwk)
		// Empty string decodes to empty byte slice, which creates a zero big.Int
		// This is technically valid but creates a weak key
		assert.NoError(t, err)
		assert.NotNil(t, pubKey)
		assert.Equal(t, 65537, pubKey.E)
		assert.Equal(t, big.NewInt(0), pubKey.N)
	})

	t.Run("Empty exponent", func(t *testing.T) {
		jwk := JWK{
			Kid: "test_key",
			Kty: "RSA",
			Use: "sig",
			N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS",
			E:   "",
		}

		pubKey, err := JwkToRSAPublicKey(jwk)
		// Empty string decodes to empty byte slice, which creates a zero exponent
		// This is technically valid but creates an invalid RSA key
		assert.NoError(t, err)
		assert.NotNil(t, pubKey)
		assert.Equal(t, 0, pubKey.E)
		assert.NotNil(t, pubKey.N)
	})

	t.Run("Small exponent value", func(t *testing.T) {
		// Encode small exponent (3) in base64url
		// [3] = "Aw"
		jwk := JWK{
			Kid: "test_key",
			Kty: "RSA",
			Use: "sig",
			N:   "0vx7agoebGcQSuuPiLJXZptN9nndrQmbXEps2aiAFbWhM78LhWx4cbbfAAtVT86zwu1RK7aPFFxuhDR1L6tSoc_BJECPebWKRXjBZCiFV4n3oknjhMstn64tZ_2W-5JsGY4Hc5n9yBXArwl93lqt7_RN5w6Cf0h4QyQ5v-65YGjQR0_FDW2QvzqY368QQMicAtaSqzs8KJZgnYb9c7d0zgdAZHzu6qMQvRL5hajrn1n91CbOpbIS",
			E:   "Aw",
		}

		pubKey, err := JwkToRSAPublicKey(jwk)
		assert.NoError(t, err)
		assert.NotNil(t, pubKey)
		assert.Equal(t, 3, pubKey.E)
	})
}

func TestJWKSEdgeCases(t *testing.T) {
	t.Run("JWK with different key types", func(t *testing.T) {
		jwkRSA := JWK{
			Kid: "rsa_key",
			Kty: "RSA",
			Use: "sig",
			N:   "test_modulus",
			E:   "AQAB",
		}

		jwkEC := JWK{
			Kid: "ec_key",
			Kty: "EC",
			Use: "sig",
			N:   "",
			E:   "",
		}

		assert.Equal(t, "RSA", jwkRSA.Kty)
		assert.Equal(t, "EC", jwkEC.Kty)
		assert.NotEmpty(t, jwkRSA.N)
		assert.Empty(t, jwkEC.N)
	})

	t.Run("JWK with different use values", func(t *testing.T) {
		jwkSig := JWK{
			Kid: "sig_key",
			Kty: "RSA",
			Use: "sig",
			N:   "test_modulus",
			E:   "AQAB",
		}

		jwkEnc := JWK{
			Kid: "enc_key",
			Kty: "RSA",
			Use: "enc",
			N:   "test_modulus",
			E:   "AQAB",
		}

		assert.Equal(t, "sig", jwkSig.Use)
		assert.Equal(t, "enc", jwkEnc.Use)
	})

	t.Run("Empty JWKSet", func(t *testing.T) {
		jwks := JWKSet{
			Keys: []JWK{},
		}

		assert.Len(t, jwks.Keys, 0)
		assert.NotNil(t, jwks.Keys)
	})

	t.Run("JWKSCache with empty keys map", func(t *testing.T) {
		now := time.Now()
		cache := JWKSCache{
			Keys:      make(map[string]*rsa.PublicKey),
			CachedAt:  now,
			ExpiresAt: now.Add(time.Hour),
		}

		assert.Len(t, cache.Keys, 0)
		assert.NotNil(t, cache.Keys)
	})

	t.Run("JWKSCache with nil keys map", func(t *testing.T) {
		now := time.Now()
		cache := JWKSCache{
			Keys:      nil,
			CachedAt:  now,
			ExpiresAt: now.Add(time.Hour),
		}

		assert.Nil(t, cache.Keys)
	})
}

func TestJWKStructFieldTypes(t *testing.T) {
	jwk := JWK{}

	// Verify field types are correct for JSON serialization
	assert.IsType(t, "", jwk.Kid)
	assert.IsType(t, "", jwk.Kty)
	assert.IsType(t, "", jwk.Use)
	assert.IsType(t, "", jwk.N)
	assert.IsType(t, "", jwk.E)
}

func TestJWKSetStructFieldTypes(t *testing.T) {
	jwks := JWKSet{}

	// Verify field types are correct for JSON serialization
	assert.IsType(t, []JWK{}, jwks.Keys)
}

func TestJWKSCacheStructFieldTypes(t *testing.T) {
	cache := JWKSCache{}

	// Verify field types are correct
	assert.IsType(t, map[string]*rsa.PublicKey{}, cache.Keys)
	assert.IsType(t, time.Time{}, cache.CachedAt)
	assert.IsType(t, time.Time{}, cache.ExpiresAt)
}
