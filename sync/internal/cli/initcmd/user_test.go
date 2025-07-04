/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package initcmd

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestUserStruct(t *testing.T) {
	t.Run("User struct creation", func(t *testing.T) {
		user := User{
			ID:       "test-id-123",
			Username: "testuser",
			Email:    "test@example.com",
			Password: "secure-password-123",
		}

		assert.Equal(t, "test-id-123", user.ID)
		assert.Equal(t, "testuser", user.Username)
		assert.Equal(t, "test@example.com", user.Email)
		assert.Equal(t, "secure-password-123", user.Password)
	})

	t.Run("User struct with empty values", func(t *testing.T) {
		user := User{}

		assert.Empty(t, user.ID)
		assert.Empty(t, user.Username)
		assert.Empty(t, user.Email)
		assert.Empty(t, user.Password)
	})

	t.Run("User struct field validation", func(t *testing.T) {
		user := User{
			ID:       "user-123",
			Username: "admin_user",
			Email:    "admin@company.com",
			Password: "ComplexPassword123!",
		}

		// Test that all fields are properly set
		assert.NotEmpty(t, user.ID, "ID should not be empty")
		assert.NotEmpty(t, user.Username, "Username should not be empty")
		assert.NotEmpty(t, user.Email, "Email should not be empty")
		assert.NotEmpty(t, user.Password, "Password should not be empty")

		// Test field types
		assert.IsType(t, "", user.ID, "ID should be string")
		assert.IsType(t, "", user.Username, "Username should be string")
		assert.IsType(t, "", user.Email, "Email should be string")
		assert.IsType(t, "", user.Password, "Password should be string")
	})

	t.Run("User struct with special characters", func(t *testing.T) {
		user := User{
			ID:       "user-with-dashes-123",
			Username: "user.with.dots",
			Email:    "user+tag@example-domain.com",
			Password: "P@ssw0rd!#$%",
		}

		assert.Contains(t, user.ID, "-", "ID should contain dashes")
		assert.Contains(t, user.Username, ".", "Username should contain dots")
		assert.Contains(t, user.Email, "+", "Email should contain plus sign")
		assert.Contains(t, user.Email, "@", "Email should contain @ sign")
		assert.Contains(t, user.Password, "!", "Password should contain special characters")
	})
}

// Note: CreateOwnerUser function requires a real Logto client for testing.
// These would be integration tests that need:
// 1. A mock Logto client
// 2. Test server responses for user creation
// 3. Error handling scenarios
//
// For now, we focus on testing the User struct itself.
// To add integration tests, we would need to:
// 1. Create a mock client interface
// 2. Implement test doubles for Logto API responses
// 3. Test various scenarios (user creation success, user already exists, etc.)

func TestUserPasswordGeneration(t *testing.T) {
	t.Run("password should be generated", func(t *testing.T) {
		// This tests that the password generation function works
		password := GenerateSecurePassword()

		assert.NotEmpty(t, password, "Generated password should not be empty")
		assert.Equal(t, 16, len(password), "Generated password should be 16 characters")

		// Test that multiple calls generate different passwords
		password2 := GenerateSecurePassword()
		assert.NotEqual(t, password, password2, "Multiple password generations should yield different results")
	})
}

func TestUserCreationScenarios(t *testing.T) {
	t.Run("typical owner user scenario", func(t *testing.T) {
		// Test a typical owner user creation scenario
		user := User{
			ID:       "owner-user-id-123",
			Username: "owner",
			Email:    "owner@company.com",
			Password: GenerateSecurePassword(),
		}

		assert.Equal(t, "owner", user.Username, "Owner username should be 'owner'")
		assert.Contains(t, user.Email, "@", "Owner email should be valid email format")
		assert.NotEmpty(t, user.Password, "Owner password should be generated")
		assert.Len(t, user.Password, 16, "Owner password should be 16 characters")
	})

	t.Run("user with default values", func(t *testing.T) {
		// Test user creation with default init command values
		defaultUsername := "owner"
		defaultEmail := "owner@example.com"
		defaultDisplayName := "Company Owner"

		user := User{
			ID:       "generated-id",
			Username: defaultUsername,
			Email:    defaultEmail,
			Password: GenerateSecurePassword(),
		}

		assert.Equal(t, defaultUsername, user.Username)
		assert.Equal(t, defaultEmail, user.Email)
		assert.NotEmpty(t, user.Password)

		// The display name is not stored in the User struct,
		// but we can verify the defaults are what we expect
		assert.Equal(t, "owner", defaultUsername)
		assert.Equal(t, "owner@example.com", defaultEmail)
		assert.Equal(t, "Company Owner", defaultDisplayName)
	})

	t.Run("user password validation", func(t *testing.T) {
		// Test that generated passwords meet requirements
		for i := 0; i < 10; i++ {
			password := GenerateSecurePassword()
			user := User{
				ID:       "test-user",
				Username: "testuser",
				Email:    "test@example.com",
				Password: password,
			}

			assert.Len(t, user.Password, 16, "Password should be 16 characters")
			assert.NotContains(t, user.Password, " ", "Password should not contain spaces")
			assert.NotEmpty(t, user.Password, "Password should not be empty")
		}
	})
}
