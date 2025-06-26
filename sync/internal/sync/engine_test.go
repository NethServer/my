/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"bytes"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/nethesis/my/sync/internal/client"
	"github.com/nethesis/my/sync/internal/logger"
	"gopkg.in/yaml.v3"
)

func TestNewEngine(t *testing.T) {
	logtoClient := &client.LogtoClient{}

	t.Run("with options", func(t *testing.T) {
		options := &Options{
			DryRun:  true,
			Verbose: true,
		}
		engine := NewEngine(logtoClient, options)

		if engine.client != logtoClient {
			t.Error("expected client to be set")
		}
		if engine.options != options {
			t.Error("expected options to be set")
		}
		if !engine.options.DryRun {
			t.Error("expected DryRun to be true")
		}
	})

	t.Run("with nil options", func(t *testing.T) {
		engine := NewEngine(logtoClient, nil)

		if engine.client != logtoClient {
			t.Error("expected client to be set")
		}
		if engine.options == nil {
			t.Error("expected options to be initialized")
		}
		if engine.options.DryRun {
			t.Error("expected DryRun to be false by default")
		}
	})
}

func TestEngineAddOperation(t *testing.T) {
	// Initialize logger for testing
	logger.Init(logger.InfoLevel)
	
	logtoClient := &client.LogtoClient{}
	engine := NewEngine(logtoClient, &Options{})
	result := &Result{
		Operations: []Operation{},
	}

	t.Run("successful operation", func(t *testing.T) {
		engine.addOperation(result, "role", "create", "admin", "Created admin role", nil)

		if len(result.Operations) != 1 {
			t.Fatalf("expected 1 operation, got %d", len(result.Operations))
		}

		op := result.Operations[0]
		if op.Type != "role" {
			t.Errorf("expected type 'role', got %q", op.Type)
		}
		if op.Action != "create" {
			t.Errorf("expected action 'create', got %q", op.Action)
		}
		if op.Resource != "admin" {
			t.Errorf("expected resource 'admin', got %q", op.Resource)
		}
		if op.Description != "Created admin role" {
			t.Errorf("expected description 'Created admin role', got %q", op.Description)
		}
		if !op.Success {
			t.Error("expected operation to be successful")
		}
		if op.Error != "" {
			t.Errorf("expected no error, got %q", op.Error)
		}
		if op.Timestamp.IsZero() {
			t.Error("expected timestamp to be set")
		}
	})

	t.Run("failed operation", func(t *testing.T) {
		testErr := fmt.Errorf("test error")
		engine.addOperation(result, "resource", "delete", "systems", "Failed to delete systems", testErr)

		if len(result.Operations) != 2 {
			t.Fatalf("expected 2 operations, got %d", len(result.Operations))
		}

		op := result.Operations[1]
		if op.Success {
			t.Error("expected operation to be failed")
		}
		if op.Error != "test error" {
			t.Errorf("expected error 'test error', got %q", op.Error)
		}
	})
}

func TestResultOutputText(t *testing.T) {
	result := &Result{
		Success:  true,
		Duration: 5 * time.Second,
		DryRun:   false,
		Summary: &Summary{
			ResourcesCreated: 2,
			ResourcesUpdated: 1,
			RolesCreated:     3,
			RolesUpdated:     0,
			RolesDeleted:     1,
		},
		Operations: []Operation{
			{
				Type:        "resource",
				Action:      "create",
				Resource:    "systems",
				Description: "Created systems resource",
				Success:     true,
				Timestamp:   time.Now(),
			},
			{
				Type:        "role",
				Action:      "delete",
				Resource:    "old-role",
				Description: "Failed to delete old role",
				Success:     false,
				Error:       "role not found",
				Timestamp:   time.Now(),
			},
		},
		Errors: []string{"Some error occurred"},
	}

	var buf bytes.Buffer
	err := result.OutputText(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()

	// Check that output contains expected content
	expectedStrings := []string{
		"Synchronization Results",
		"Status: SUCCESS",
		"Duration: 5s",
		"Dry Run: false",
		"Resources: 2 created, 1 updated, 0 deleted",
		"Roles: 3 created, 0 updated, 1 deleted",
		"✓ resource create systems",
		"✗ role delete old-role",
		"Errors:",
		"Some error occurred",
	}

	for _, expected := range expectedStrings {
		if !containsString(output, expected) {
			t.Errorf("expected output to contain %q", expected)
		}
	}
}

func TestResultOutputJSON(t *testing.T) {
	result := &Result{
		Success:  true,
		Duration: 5 * time.Second,
		DryRun:   true,
		Summary: &Summary{
			ResourcesCreated: 1,
		},
		Operations: []Operation{
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

	var buf bytes.Buffer
	err := result.OutputJSON(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the output is valid JSON
	var decoded Result
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}

	if decoded.Success != result.Success {
		t.Errorf("expected success %v, got %v", result.Success, decoded.Success)
	}
	if decoded.DryRun != result.DryRun {
		t.Errorf("expected dry run %v, got %v", result.DryRun, decoded.DryRun)
	}
	if decoded.Summary.ResourcesCreated != result.Summary.ResourcesCreated {
		t.Errorf("expected resources created %d, got %d", result.Summary.ResourcesCreated, decoded.Summary.ResourcesCreated)
	}
}

func TestResultOutputYAML(t *testing.T) {
	result := &Result{
		Success:  false,
		Duration: 10 * time.Second,
		DryRun:   false,
		Summary: &Summary{
			RolesCreated: 2,
		},
		Errors: []string{"First error", "Second error"},
	}

	var buf bytes.Buffer
	err := result.OutputYAML(&buf)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that the output is valid YAML
	var decoded Result
	if err := yaml.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode YAML: %v", err)
	}

	if decoded.Success != result.Success {
		t.Errorf("expected success %v, got %v", result.Success, decoded.Success)
	}
	if len(decoded.Errors) != len(result.Errors) {
		t.Errorf("expected %d errors, got %d", len(result.Errors), len(decoded.Errors))
	}
	if decoded.Summary.RolesCreated != result.Summary.RolesCreated {
		t.Errorf("expected roles created %d, got %d", result.Summary.RolesCreated, decoded.Summary.RolesCreated)
	}
}

func TestOptionsDefaults(t *testing.T) {
	options := &Options{}

	// Test that defaults are false
	if options.DryRun {
		t.Error("expected DryRun to default to false")
	}
	if options.Verbose {
		t.Error("expected Verbose to default to false")
	}
	if options.SkipResources {
		t.Error("expected SkipResources to default to false")
	}
	if options.SkipRoles {
		t.Error("expected SkipRoles to default to false")
	}
	if options.SkipPermissions {
		t.Error("expected SkipPermissions to default to false")
	}
	if options.Cleanup {
		t.Error("expected Cleanup to default to false")
	}
}

func TestSummaryStructure(t *testing.T) {
	summary := &Summary{}

	// Test that summary starts with zeros
	if summary.ResourcesCreated != 0 {
		t.Error("expected ResourcesCreated to start at 0")
	}
	if summary.RolesCreated != 0 {
		t.Error("expected RolesCreated to start at 0")
	}
	if summary.PermissionsCreated != 0 {
		t.Error("expected PermissionsCreated to start at 0")
	}

	// Test summary field assignment
	summary.ResourcesCreated = 5
	summary.ResourcesUpdated = 3
	summary.ResourcesDeleted = 1
	summary.RolesCreated = 2
	summary.RolesUpdated = 1
	summary.RolesDeleted = 0

	if summary.ResourcesCreated != 5 {
		t.Errorf("expected ResourcesCreated 5, got %d", summary.ResourcesCreated)
	}
	if summary.RolesUpdated != 1 {
		t.Errorf("expected RolesUpdated 1, got %d", summary.RolesUpdated)
	}
}

func TestOperationStructure(t *testing.T) {
	op := Operation{
		Type:        "role",
		Action:      "create",
		Resource:    "admin",
		Description: "Created admin role",
		Success:     true,
		Timestamp:   time.Now(),
	}

	if op.Type != "role" {
		t.Errorf("expected type 'role', got %q", op.Type)
	}
	if op.Action != "create" {
		t.Errorf("expected action 'create', got %q", op.Action)
	}
	if op.Resource != "admin" {
		t.Errorf("expected resource 'admin', got %q", op.Resource)
	}
	if !op.Success {
		t.Error("expected success to be true")
	}
	if op.Error != "" {
		t.Errorf("expected error to be empty, got %q", op.Error)
	}
}

func TestResultStructure(t *testing.T) {
	startTime := time.Now()
	result := &Result{
		StartTime:  startTime,
		EndTime:    startTime.Add(5 * time.Second),
		Duration:   5 * time.Second,
		DryRun:     true,
		Success:    true,
		Summary:    &Summary{ResourcesCreated: 1},
		Operations: []Operation{},
		Errors:     []string{},
	}

	if result.StartTime != startTime {
		t.Error("expected start time to be set correctly")
	}
	if result.Duration != 5*time.Second {
		t.Errorf("expected duration 5s, got %v", result.Duration)
	}
	if !result.DryRun {
		t.Error("expected dry run to be true")
	}
	if !result.Success {
		t.Error("expected success to be true")
	}
	if result.Summary == nil {
		t.Error("expected summary to be non-nil")
	}
	if result.Operations == nil {
		t.Error("expected operations to be non-nil")
	}
	if result.Errors == nil {
		t.Error("expected errors to be non-nil")
	}
}

// Helper function to check if a string contains a substring
func containsString(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}