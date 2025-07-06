/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"

	"github.com/nethesis/my/sync/internal/client"
)

func TestNewEngineUtils(t *testing.T) {
	mockClient := &client.LogtoClient{}

	t.Run("with options", func(t *testing.T) {
		options := &Options{
			DryRun:          true,
			Verbose:         true,
			SkipResources:   false,
			SkipRoles:       false,
			SkipPermissions: true,
			Cleanup:         false,
			APIBaseURL:      "https://api.example.com",
		}

		engine := NewEngine(mockClient, options)

		assert.NotNil(t, engine)
		assert.Equal(t, mockClient, engine.client)
		assert.Equal(t, options, engine.options)
		assert.True(t, engine.options.DryRun)
		assert.True(t, engine.options.Verbose)
		assert.True(t, engine.options.SkipPermissions)
		assert.Equal(t, "https://api.example.com", engine.options.APIBaseURL)
	})

	t.Run("with nil options", func(t *testing.T) {
		engine := NewEngine(mockClient, nil)

		assert.NotNil(t, engine)
		assert.Equal(t, mockClient, engine.client)
		assert.NotNil(t, engine.options)

		// Should have default empty options
		assert.False(t, engine.options.DryRun)
		assert.False(t, engine.options.Verbose)
		assert.False(t, engine.options.SkipResources)
		assert.False(t, engine.options.SkipRoles)
		assert.False(t, engine.options.SkipPermissions)
		assert.False(t, engine.options.Cleanup)
		assert.Empty(t, engine.options.APIBaseURL)
	})
}

func TestOptions(t *testing.T) {
	t.Run("options struct", func(t *testing.T) {
		options := Options{
			DryRun:          true,
			Verbose:         false,
			SkipResources:   true,
			SkipRoles:       false,
			SkipPermissions: true,
			Cleanup:         false,
			APIBaseURL:      "https://test.example.com",
		}

		assert.True(t, options.DryRun)
		assert.False(t, options.Verbose)
		assert.True(t, options.SkipResources)
		assert.False(t, options.SkipRoles)
		assert.True(t, options.SkipPermissions)
		assert.False(t, options.Cleanup)
		assert.Equal(t, "https://test.example.com", options.APIBaseURL)
	})
}

func TestResult(t *testing.T) {
	startTime := time.Now()
	endTime := startTime.Add(5 * time.Second)

	t.Run("result structure", func(t *testing.T) {
		result := Result{
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  endTime.Sub(startTime),
			DryRun:    true,
			Success:   true,
			Summary: &Summary{
				ResourcesCreated: 2,
				RolesCreated:     3,
				ScopesCreated:    5,
			},
			Operations: []Operation{
				{
					Type:        "resource",
					Action:      "create",
					Resource:    "test-resource",
					Description: "Created test resource",
					Success:     true,
					Timestamp:   startTime,
				},
			},
			Errors: []string{},
		}

		assert.Equal(t, startTime, result.StartTime)
		assert.Equal(t, endTime, result.EndTime)
		assert.Equal(t, 5*time.Second, result.Duration)
		assert.True(t, result.DryRun)
		assert.True(t, result.Success)
		assert.NotNil(t, result.Summary)
		assert.Len(t, result.Operations, 1)
		assert.Empty(t, result.Errors)
	})

	t.Run("result with errors", func(t *testing.T) {
		result := Result{
			StartTime: startTime,
			EndTime:   endTime,
			Duration:  endTime.Sub(startTime),
			Success:   false,
			Errors:    []string{"Error 1", "Error 2"},
		}

		assert.False(t, result.Success)
		assert.Len(t, result.Errors, 2)
		assert.Contains(t, result.Errors, "Error 1")
		assert.Contains(t, result.Errors, "Error 2")
	})
}

func TestSummary(t *testing.T) {
	t.Run("summary totals", func(t *testing.T) {
		summary := Summary{
			ResourcesCreated:   3,
			ResourcesUpdated:   2,
			ResourcesDeleted:   1,
			RolesCreated:       4,
			RolesUpdated:       3,
			RolesDeleted:       0,
			PermissionsCreated: 10,
			PermissionsUpdated: 5,
			PermissionsDeleted: 2,
			ScopesCreated:      8,
			ScopesUpdated:      3,
			ScopesDeleted:      1,
		}

		// Calculate totals
		totalCreated := summary.ResourcesCreated + summary.RolesCreated + summary.PermissionsCreated + summary.ScopesCreated
		totalUpdated := summary.ResourcesUpdated + summary.RolesUpdated + summary.PermissionsUpdated + summary.ScopesUpdated
		totalDeleted := summary.ResourcesDeleted + summary.RolesDeleted + summary.PermissionsDeleted + summary.ScopesDeleted

		assert.Equal(t, 25, totalCreated)
		assert.Equal(t, 13, totalUpdated)
		assert.Equal(t, 4, totalDeleted)
	})

	t.Run("empty summary", func(t *testing.T) {
		summary := Summary{}

		assert.Equal(t, 0, summary.ResourcesCreated)
		assert.Equal(t, 0, summary.RolesCreated)
		assert.Equal(t, 0, summary.PermissionsCreated)
		assert.Equal(t, 0, summary.ScopesCreated)
	})
}

func TestOperation(t *testing.T) {
	timestamp := time.Now()

	t.Run("successful operation", func(t *testing.T) {
		op := Operation{
			Type:        "role",
			Action:      "update",
			Resource:    "admin-role",
			Description: "Updated admin role permissions",
			Success:     true,
			Timestamp:   timestamp,
		}

		assert.Equal(t, "role", op.Type)
		assert.Equal(t, "update", op.Action)
		assert.Equal(t, "admin-role", op.Resource)
		assert.True(t, op.Success)
		assert.Empty(t, op.Error)
		assert.Equal(t, timestamp, op.Timestamp)
	})

	t.Run("failed operation", func(t *testing.T) {
		op := Operation{
			Type:        "resource",
			Action:      "delete",
			Resource:    "old-resource",
			Description: "Attempted to delete old resource",
			Success:     false,
			Error:       "Resource not found",
			Timestamp:   timestamp,
		}

		assert.Equal(t, "resource", op.Type)
		assert.Equal(t, "delete", op.Action)
		assert.False(t, op.Success)
		assert.Equal(t, "Resource not found", op.Error)
	})
}

func TestResultJSONSerialization(t *testing.T) {
	startTime := time.Now().Truncate(time.Second) // Truncate for consistent testing
	result := Result{
		StartTime: startTime,
		EndTime:   startTime.Add(10 * time.Second),
		Duration:  10 * time.Second,
		DryRun:    true,
		Success:   true,
		Summary: &Summary{
			ResourcesCreated: 2,
			RolesCreated:     1,
		},
		Operations: []Operation{
			{
				Type:        "resource",
				Action:      "create",
				Resource:    "test-resource",
				Description: "Created test resource",
				Success:     true,
				Timestamp:   startTime,
			},
		},
		Errors: []string{},
	}

	t.Run("marshal and unmarshal JSON", func(t *testing.T) {
		jsonData, err := json.Marshal(result)
		require.NoError(t, err)

		var unmarshaled Result
		err = json.Unmarshal(jsonData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, result.DryRun, unmarshaled.DryRun)
		assert.Equal(t, result.Success, unmarshaled.Success)
		assert.Equal(t, result.Summary.ResourcesCreated, unmarshaled.Summary.ResourcesCreated)
		assert.Len(t, unmarshaled.Operations, 1)
		assert.Equal(t, "resource", unmarshaled.Operations[0].Type)
	})

	t.Run("JSON contains expected fields", func(t *testing.T) {
		jsonData, err := json.Marshal(result)
		require.NoError(t, err)

		jsonStr := string(jsonData)
		assert.Contains(t, jsonStr, `"dry_run":true`)
		assert.Contains(t, jsonStr, `"success":true`)
		assert.Contains(t, jsonStr, `"resources_created":2`)
		assert.Contains(t, jsonStr, `"type":"resource"`)
	})
}

func TestResultYAMLSerialization(t *testing.T) {
	result := Result{
		StartTime: time.Now(),
		DryRun:    false,
		Success:   true,
		Summary: &Summary{
			ResourcesCreated: 1,
			RolesUpdated:     2,
		},
		Operations: []Operation{
			{
				Type:     "role",
				Action:   "update",
				Resource: "admin",
				Success:  true,
			},
		},
		Errors: []string{"Warning: deprecated field"},
	}

	t.Run("marshal and unmarshal YAML", func(t *testing.T) {
		yamlData, err := yaml.Marshal(result)
		require.NoError(t, err)

		var unmarshaled Result
		err = yaml.Unmarshal(yamlData, &unmarshaled)
		require.NoError(t, err)

		assert.Equal(t, result.DryRun, unmarshaled.DryRun)
		assert.Equal(t, result.Success, unmarshaled.Success)
		assert.Equal(t, result.Summary.ResourcesCreated, unmarshaled.Summary.ResourcesCreated)
		assert.Len(t, unmarshaled.Operations, 1)
		assert.Len(t, unmarshaled.Errors, 1)
	})

	t.Run("YAML contains expected fields", func(t *testing.T) {
		yamlData, err := yaml.Marshal(result)
		require.NoError(t, err)

		yamlStr := string(yamlData)
		assert.Contains(t, yamlStr, "dry_run: false")
		assert.Contains(t, yamlStr, "success: true")
		assert.Contains(t, yamlStr, "resources_created: 1")
		assert.Contains(t, yamlStr, "type: role")
		assert.Contains(t, yamlStr, "- 'Warning: deprecated field'")
	})
}

func TestResultOutputMethods(t *testing.T) {
	result := Result{
		StartTime: time.Now(),
		DryRun:    true,
		Success:   true,
		Summary: &Summary{
			ResourcesCreated: 3,
			RolesCreated:     2,
		},
		Operations: []Operation{
			{
				Type:        "resource",
				Action:      "create",
				Resource:    "api-resource",
				Description: "Created API resource",
				Success:     true,
			},
		},
	}

	t.Run("OutputJSON", func(t *testing.T) {
		var buf bytes.Buffer
		err := result.OutputJSON(&buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, `"dry_run": true`)
		assert.Contains(t, output, `"success": true`)
		assert.Contains(t, output, `"resources_created": 3`)

		// Verify it's valid JSON
		var parsedResult Result
		err = json.Unmarshal(buf.Bytes(), &parsedResult)
		assert.NoError(t, err)
	})

	t.Run("OutputYAML", func(t *testing.T) {
		var buf bytes.Buffer
		err := result.OutputYAML(&buf)
		require.NoError(t, err)

		output := buf.String()
		assert.Contains(t, output, "dry_run: true")
		assert.Contains(t, output, "success: true")
		assert.Contains(t, output, "resources_created: 3")

		// Verify it's valid YAML
		var parsedResult Result
		err = yaml.Unmarshal(buf.Bytes(), &parsedResult)
		assert.NoError(t, err)
	})

	t.Run("OutputText", func(t *testing.T) {
		var buf bytes.Buffer
		err := result.OutputText(&buf)
		require.NoError(t, err)

		output := buf.String()

		// Check for common text output elements
		assert.Contains(t, output, "Synchronization")
		assert.Contains(t, output, "Dry Run") // Since result.DryRun is true
		assert.Contains(t, output, "SUCCESS") // Since result.Success is true

		// Check for summary information
		lines := strings.Split(output, "\n")
		assert.NotEmpty(t, lines)

		// Should contain information about created resources
		summaryFound := false
		for _, line := range lines {
			if strings.Contains(line, "Resources") && strings.Contains(line, "3") {
				summaryFound = true
				break
			}
		}
		assert.True(t, summaryFound, "Output should contain resource creation summary")
	})
}

func TestEngineStructure(t *testing.T) {
	t.Run("engine initialization", func(t *testing.T) {
		mockClient := &client.LogtoClient{}
		options := &Options{
			DryRun:     true,
			Verbose:    false,
			APIBaseURL: "https://test.api.com",
		}

		engine := NewEngine(mockClient, options)

		assert.NotNil(t, engine)
		assert.Equal(t, mockClient, engine.client)
		assert.Equal(t, options, engine.options)
		assert.True(t, engine.options.DryRun)
		assert.Equal(t, "https://test.api.com", engine.options.APIBaseURL)
	})
}
