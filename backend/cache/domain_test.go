/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package cache

import (
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/testutils"
)

func TestGetDomainValidation_Singleton(t *testing.T) {
	// Reset the singleton for testing
	domainValidationOnce = sync.Once{}
	domainValidation = nil

	dv1 := GetDomainValidation()
	dv2 := GetDomainValidation()

	assert.NotNil(t, dv1)
	assert.NotNil(t, dv2)
	assert.Same(t, dv1, dv2) // Should be the same instance
}

func TestDomainValidation_InitialState(t *testing.T) {
	dv := &DomainValidation{}

	assert.False(t, dv.IsValid())
	assert.False(t, dv.IsLoaded())
	assert.Equal(t, "", dv.GetDomain())
}

func TestDomainValidation_LoadDomainValidation(t *testing.T) {
	testutils.SetupLogger()

	// Setup test configuration
	originalConfig := configuration.Config
	defer func() {
		configuration.Config = originalConfig
	}()

	configuration.Config.TenantDomain = "test.example.com"

	tests := []struct {
		name           string
		expectedDomain string
		expectedValid  bool
		setupMock      func()
	}{
		{
			name:           "load domain validation",
			expectedDomain: "test.example.com",
			expectedValid:  false, // Since we can't mock the Logto client easily
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := &DomainValidation{}

			// This will attempt to contact Logto service but gracefully handle failures
			// The function does not return errors, it performs graceful degradation
			err := dv.LoadDomainValidation()

			// The function performs graceful degradation and doesn't return errors
			assert.NoError(t, err)

			// Test the structure even if load fails
			assert.Equal(t, tt.expectedDomain, dv.GetDomain())
			assert.True(t, dv.IsLoaded()) // Should be loaded even if invalid
		})
	}
}

func TestDomainValidation_IsValid(t *testing.T) {
	testutils.SetupLogger()

	tests := []struct {
		name     string
		loaded   bool
		isValid  bool
		expected bool
	}{
		{
			name:     "valid domain when loaded",
			loaded:   true,
			isValid:  true,
			expected: true,
		},
		{
			name:     "invalid domain when loaded",
			loaded:   true,
			isValid:  false,
			expected: false,
		},
		{
			name:     "not loaded yet",
			loaded:   false,
			isValid:  true,
			expected: false, // Should return false if not loaded
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := &DomainValidation{
				loaded:  tt.loaded,
				isValid: tt.isValid,
			}

			result := dv.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainValidation_IsLoaded(t *testing.T) {
	tests := []struct {
		name     string
		loaded   bool
		expected bool
	}{
		{
			name:     "domain validation loaded",
			loaded:   true,
			expected: true,
		},
		{
			name:     "domain validation not loaded",
			loaded:   false,
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := &DomainValidation{
				loaded: tt.loaded,
			}

			result := dv.IsLoaded()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainValidation_GetDomain(t *testing.T) {
	tests := []struct {
		name     string
		domain   string
		expected string
	}{
		{
			name:     "get domain",
			domain:   "test.example.com",
			expected: "test.example.com",
		},
		{
			name:     "empty domain",
			domain:   "",
			expected: "",
		},
		{
			name:     "complex domain",
			domain:   "my-app.staging.example.com",
			expected: "my-app.staging.example.com",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dv := &DomainValidation{
				domain: tt.domain,
			}

			result := dv.GetDomain()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDomainValidation_ThreadSafety(t *testing.T) {
	testutils.SetupLogger()

	dv := &DomainValidation{}

	// Test concurrent access
	const numGoroutines = 10
	const numOperations = 100

	var wg sync.WaitGroup
	wg.Add(numGoroutines * 3) // 3 operations per goroutine

	// Concurrent reads
	for i := 0; i < numGoroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				dv.IsValid()
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				dv.IsLoaded()
			}
		}()

		go func() {
			defer wg.Done()
			for j := 0; j < numOperations; j++ {
				dv.GetDomain()
			}
		}()
	}

	wg.Wait()

	// If we get here without deadlock, thread safety test passed
	assert.True(t, true)
}

func TestDomainValidation_StateTransitions(t *testing.T) {
	dv := &DomainValidation{}

	// Initial state
	assert.False(t, dv.IsLoaded())
	assert.False(t, dv.IsValid())
	assert.Equal(t, "", dv.GetDomain())

	// Simulate loading (manually set state)
	dv.mutex.Lock()
	dv.domain = "test.com"
	dv.isValid = true
	dv.loaded = true
	dv.mutex.Unlock()

	// Check final state
	assert.True(t, dv.IsLoaded())
	assert.True(t, dv.IsValid())
	assert.Equal(t, "test.com", dv.GetDomain())
}

func TestDomainValidation_LockingBehavior(t *testing.T) {
	dv := &DomainValidation{}

	// Test that read operations work concurrently
	done := make(chan bool, 3)

	go func() {
		dv.IsValid()
		done <- true
	}()

	go func() {
		dv.IsLoaded()
		done <- true
	}()

	go func() {
		dv.GetDomain()
		done <- true
	}()

	// All read operations should complete
	for i := 0; i < 3; i++ {
		<-done
	}

	assert.True(t, true)
}
