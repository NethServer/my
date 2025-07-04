/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package syncmd

import (
	"testing"
	"time"

	"github.com/spf13/viper"
	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/sync/internal/sync"
)

func TestOutputResult(t *testing.T) {
	// Save original viper setting
	originalOutput := viper.GetString("output")
	defer viper.Set("output", originalOutput)

	// Create a test result
	result := &sync.Result{
		Success:  true,
		Duration: 5 * time.Second,
		DryRun:   false,
		Summary: &sync.Summary{
			ResourcesCreated: 2,
			ResourcesUpdated: 1,
			RolesCreated:     3,
		},
		Operations: []sync.Operation{
			{
				Type:        "resource",
				Action:      "create",
				Resource:    "systems",
				Description: "Created systems resource",
				Success:     true,
				Timestamp:   time.Now(),
			},
		},
	}

	t.Run("text output format", func(t *testing.T) {
		viper.Set("output", "text")

		err := OutputResult(result)
		assert.NoError(t, err)
	})

	t.Run("json output format", func(t *testing.T) {
		viper.Set("output", "json")

		err := OutputResult(result)
		assert.NoError(t, err)
	})

	t.Run("yaml output format", func(t *testing.T) {
		viper.Set("output", "yaml")

		err := OutputResult(result)
		assert.NoError(t, err)
	})

	t.Run("default output format", func(t *testing.T) {
		viper.Set("output", "unknown")

		// Should default to text format
		err := OutputResult(result)
		assert.NoError(t, err)
	})

	t.Run("empty output format defaults to text", func(t *testing.T) {
		viper.Set("output", "")

		err := OutputResult(result)
		assert.NoError(t, err)
	})
}

func TestOutputResultWithDifferentResults(t *testing.T) {
	// Save original viper setting
	originalOutput := viper.GetString("output")
	defer viper.Set("output", originalOutput)

	t.Run("failed result output", func(t *testing.T) {
		viper.Set("output", "text")

		failedResult := &sync.Result{
			Success:  false,
			Duration: 2 * time.Second,
			DryRun:   false,
			Summary:  &sync.Summary{},
			Operations: []sync.Operation{
				{
					Type:        "role",
					Action:      "create",
					Resource:    "admin",
					Description: "Failed to create admin role",
					Success:     false,
					Error:       "permission denied",
					Timestamp:   time.Now(),
				},
			},
			Errors: []string{"Configuration error", "Network timeout"},
		}

		err := OutputResult(failedResult)
		assert.NoError(t, err)
	})

	t.Run("dry run result output", func(t *testing.T) {
		viper.Set("output", "json")

		dryRunResult := &sync.Result{
			Success:  true,
			Duration: 1 * time.Second,
			DryRun:   true,
			Summary: &sync.Summary{
				ResourcesCreated: 1,
			},
			Operations: []sync.Operation{
				{
					Type:        "resource",
					Action:      "create",
					Resource:    "test-resource",
					Description: "Would create test resource",
					Success:     true,
					Timestamp:   time.Now(),
				},
			},
		}

		err := OutputResult(dryRunResult)
		assert.NoError(t, err)
	})
}
