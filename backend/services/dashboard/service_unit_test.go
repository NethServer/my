/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package dashboard

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/models"
)

// withModel swaps the model call for the duration of one test. The service is
// otherwise exercised end to end, key or no key.
func withModel(t *testing.T, fn func(context.Context, promptInput) (selection, error)) {
	t.Helper()
	original := modelSelect
	modelSelect = fn
	t.Cleanup(func() { modelSelect = original })

	configuration.Config.DashboardAIEnabled = true
	configuration.Config.GeminiAPIKey = "test-key"
	configuration.Config.GeminiModel = "gemini-test"
	configuration.Config.DashboardAITimeout = time.Second
	t.Cleanup(func() {
		configuration.Config.GeminiAPIKey = ""
		configuration.Config.GeminiModel = ""
	})
}

func customerUser() *models.User {
	return userWith("read:systems", "read:applications", "read:entitlements")
}

func TestGenerateFallsBackWithNoAPIKey(t *testing.T) {
	configuration.Config.DashboardAIEnabled = true
	configuration.Config.GeminiAPIKey = ""

	got := Generate(context.Background(), customerUser(), models.DashboardGenerateRequest{
		Topics: []string{"systems"},
	})

	assert.Equal(t, GeneratedByFallback, got.GeneratedBy)
	assert.Empty(t, got.Model, "the fallback names no model")
	assert.GreaterOrEqual(t, len(got.Widgets), MinWidgets)
	assert.Equal(t, "Your dashboard", got.Title)
}

func TestGenerateFallsBackWhenTheModelIsDisabled(t *testing.T) {
	configuration.Config.DashboardAIEnabled = false
	configuration.Config.GeminiAPIKey = "test-key"
	t.Cleanup(func() {
		configuration.Config.DashboardAIEnabled = true
		configuration.Config.GeminiAPIKey = ""
	})

	called := false
	original := modelSelect
	modelSelect = func(context.Context, promptInput) (selection, error) {
		called = true
		return selection{}, nil
	}
	t.Cleanup(func() { modelSelect = original })

	got := Generate(context.Background(), customerUser(), models.DashboardGenerateRequest{
		Topics: []string{"alerts"},
	})

	assert.False(t, called, "a disabled model must not be called at all")
	assert.Equal(t, GeneratedByFallback, got.GeneratedBy)
}

func TestGenerateUsesTheModelAnswer(t *testing.T) {
	withModel(t, func(context.Context, promptInput) (selection, error) {
		return selection{
			Title: "  Operations   board ",
			Widgets: []selectedWidget{
				{ID: "alerts_firing_now", Size: "md", Reason: "you watch alerts"},
				{ID: "systems_counter", Size: "sm"},
				{ID: "addons_expiring", Size: "md"},
			},
		}, nil
	})

	got := Generate(context.Background(), customerUser(), models.DashboardGenerateRequest{
		Topics: []string{"alerts", "systems"},
	})

	assert.Equal(t, GeneratedByAI, got.GeneratedBy)
	assert.Equal(t, "gemini-test", got.Model)
	assert.Equal(t, "Operations board", got.Title)
	require.Len(t, got.Widgets, 3)
	assert.Equal(t, "alerts_firing_now", got.Widgets[0].ID)
	assert.Equal(t, "you watch alerts", got.Widgets[0].Reason)
}

func TestGenerateFallsBackOnAModelError(t *testing.T) {
	withModel(t, func(context.Context, promptInput) (selection, error) {
		return selection{}, errors.New("upstream is down")
	})

	got := Generate(context.Background(), customerUser(), models.DashboardGenerateRequest{
		Topics: []string{"systems"},
	})

	assert.Equal(t, GeneratedByFallback, got.GeneratedBy)
	assert.GreaterOrEqual(t, len(got.Widgets), MinWidgets)
}

func TestGenerateFallsBackWhenNoWidgetSurvivesValidation(t *testing.T) {
	withModel(t, func(context.Context, promptInput) (selection, error) {
		return selection{Widgets: []selectedWidget{
			{ID: "distributors_counter", Size: "sm"},
			{ID: "invented", Size: "lg"},
		}}, nil
	})

	got := Generate(context.Background(), customerUser(), models.DashboardGenerateRequest{
		Topics: []string{"organizations"},
	})

	assert.Equal(t, GeneratedByFallback, got.GeneratedBy)
	for _, widget := range got.Widgets {
		assert.NotContains(t,
			[]string{"distributors_counter", "resellers_counter", "customers_counter"},
			widget.ID)
	}
}

func TestGenerateNeverNamesAWidgetTheCallerCannotRead(t *testing.T) {
	// The model answers with an organization widget mixed into valid picks: the
	// enum should have prevented it, and the validator drops it regardless.
	withModel(t, func(context.Context, promptInput) (selection, error) {
		return selection{Widgets: []selectedWidget{
			{ID: "alerts_counter", Size: "sm"},
			{ID: "customers_counter", Size: "sm"},
			{ID: "systems_counter", Size: "sm"},
			{ID: "addons_expiring", Size: "md"},
		}}, nil
	})

	got := Generate(context.Background(), customerUser(), models.DashboardGenerateRequest{
		Topics: []string{"organizations", "alerts"},
	})

	for _, widget := range got.Widgets {
		assert.NotEqual(t, "customers_counter", widget.ID)
	}
}

func TestGenerateHonoursTheLocaleForTheDefaultTitle(t *testing.T) {
	configuration.Config.DashboardAIEnabled = true
	configuration.Config.GeminiAPIKey = ""

	got := Generate(context.Background(), customerUser(), models.DashboardGenerateRequest{
		Topics: []string{"systems"},
		Locale: "it",
	})

	assert.Equal(t, "La tua dashboard", got.Title)
}

func TestGenerateWithAnEmptyCatalog(t *testing.T) {
	got := Generate(context.Background(), &models.User{}, models.DashboardGenerateRequest{
		Topics: []string{"systems"},
	})

	// third_party_apps needs no permission, so even a permissionless user has
	// something. A nil user has nothing, and must still get a valid answer.
	assert.Equal(t, GeneratedByFallback, got.GeneratedBy)
	assert.NotNil(t, got.Widgets)
}

func TestCatalogResponseExposesOnlyAllowedTopics(t *testing.T) {
	got := CatalogResponse(customerUser())

	assert.NotContains(t, got.Topics, string(TopicOrganizations))
	assert.NotContains(t, got.Topics, string(TopicUsers))
	assert.Contains(t, got.Topics, string(TopicAlerts))

	for _, widget := range got.Widgets {
		assert.NotEqual(t, "customers_counter", widget.ID)
		assert.NotEmpty(t, widget.DefaultSize)
	}
}
