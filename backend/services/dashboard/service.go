/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package dashboard

import (
	"context"
	"time"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
	"github.com/nethesis/my/backend/models"
)

const (
	// GeneratedByAI marks a selection the model actually produced.
	GeneratedByAI = "ai"
	// GeneratedByFallback marks the deterministic rule-based selection.
	GeneratedByFallback = "fallback"
)

// defaultTitles is what a board is called when the model did not name it, or
// named it with something unusable.
var defaultTitles = map[string]string{
	"en": "Your dashboard",
	"it": "La tua dashboard",
}

// modelSelect is the model call, injectable so tests exercise the whole service
// without the SDK, a key, or a network.
var modelSelect = geminiSelect

// Generate picks the widgets for one user's dashboard.
//
// It returns no error on purpose: every path produces a usable selection. The
// model being disabled, unreachable, slow or incoherent is a normal condition,
// not a failure of the request, so the caller answers 200 with
// generated_by="fallback" rather than an error the frontend would have to
// invent a dashboard for anyway.
func Generate(ctx context.Context, user *models.User, req models.DashboardGenerateRequest) models.DashboardGenerateResponse {
	allowed := CatalogFor(user)
	topics := parseTopics(req.Topics)
	locale := req.Locale
	if locale == "" {
		locale = "en"
	}

	fallback := func() models.DashboardGenerateResponse {
		return models.DashboardGenerateResponse{
			Title:       defaultTitle(locale),
			GeneratedBy: GeneratedByFallback,
			Widgets:     fromWidgets(fallbackSelect(allowed, topics, MaxWidgets)),
		}
	}

	if len(allowed) == 0 {
		return models.DashboardGenerateResponse{
			Title:       defaultTitle(locale),
			GeneratedBy: GeneratedByFallback,
			Widgets:     []models.DashboardWidget{},
		}
	}

	if !aiEnabled() {
		return fallback()
	}

	callCtx, cancel := context.WithTimeout(ctx, configuration.Config.DashboardAITimeout)
	defer cancel()

	started := time.Now()
	raw, err := modelSelect(callCtx, promptInput{
		Allowed: allowed,
		Topics:  topics,
		Hint:    req.Hint,
		Locale:  locale,
	})
	latency := time.Since(started).Milliseconds()

	if err != nil {
		logger.ComponentLogger("dashboard").Warn().
			Err(err).
			Str("operation", "dashboard_generate").
			Str("model", configuration.Config.GeminiModel).
			Int64("latency_ms", latency).
			Int("catalog_size", len(allowed)).
			Msg("model selection failed, answering with the rule-based fallback")
		return fallback()
	}

	widgets, fromModel := validateSelection(raw, allowed, topics)
	if !fromModel {
		logger.ComponentLogger("dashboard").Warn().
			Str("operation", "dashboard_generate").
			Str("model", configuration.Config.GeminiModel).
			Int64("latency_ms", latency).
			Int("catalog_size", len(allowed)).
			Msg("model selection contained no usable widget, answering with the rule-based fallback")
		return fallback()
	}

	title := truncateRunes(trimTitle(raw.Title), maxTitleRunes)
	if title == "" {
		title = defaultTitle(locale)
	}

	logger.ComponentLogger("dashboard").Info().
		Str("operation", "dashboard_generate").
		Str("model", configuration.Config.GeminiModel).
		Str("generated_by", GeneratedByAI).
		Int64("latency_ms", latency).
		Int("widget_count", len(widgets)).
		Int("topics_count", len(topics)).
		Int("catalog_size", len(allowed)).
		Msg("dashboard generated")

	return models.DashboardGenerateResponse{
		Title:       title,
		GeneratedBy: GeneratedByAI,
		Model:       configuration.Config.GeminiModel,
		Widgets:     widgets,
	}
}

// CatalogResponse is the caller's filtered catalog, for the wizard.
func CatalogResponse(user *models.User) models.DashboardCatalogResponse {
	allowed := CatalogFor(user)

	widgets := make([]models.DashboardCatalogWidget, 0, len(allowed))
	for _, widget := range allowed {
		widgets = append(widgets, models.DashboardCatalogWidget{
			ID:          widget.ID,
			Title:       widget.Title,
			Description: widget.Description,
			Topic:       string(widget.Topic),
			DefaultSize: string(widget.DefaultSize),
		})
	}

	topics := make([]string, 0, len(allowed))
	for _, topic := range Topics(allowed) {
		topics = append(topics, string(topic))
	}

	return models.DashboardCatalogResponse{Widgets: widgets, Topics: topics}
}

// aiEnabled reports whether the model path may be attempted at all. An absent
// key is a normal deployment, not a misconfiguration.
func aiEnabled() bool {
	return configuration.Config.DashboardAIEnabled && configuration.Config.GeminiAPIKey != ""
}

func defaultTitle(locale string) string {
	if title, ok := defaultTitles[locale]; ok {
		return title
	}
	return defaultTitles["en"]
}
