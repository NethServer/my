/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package dashboard

import (
	"slices"
	"strings"
	"unicode/utf8"

	"github.com/nethesis/my/backend/models"
)

const (
	// MinWidgets and MaxWidgets bound every selection, whoever produced it.
	MinWidgets = 3
	MaxWidgets = 7

	maxReasonRunes = 160
	maxTitleRunes  = 80
)

// selection is what the model answers with, before any validation.
type selection struct {
	Title   string           `json:"title"`
	Widgets []selectedWidget `json:"widgets"`
}

type selectedWidget struct {
	ID     string `json:"id"`
	Size   string `json:"size"`
	Reason string `json:"reason"`
}

// fallbackSelect is the deterministic selection: no model, no randomness, no
// clock. It walks the topics in the order the user sent them, emitting each
// topic's widgets in catalog order, then tops up from whatever is left. Same
// input always yields the same output, which is what lets CI and offline
// development run the whole feature with no API key.
func fallbackSelect(allowed []Widget, topics []Topic, max int) []Widget {
	chosen := make([]Widget, 0, max)

	appendWidget := func(widget Widget) bool {
		if len(chosen) >= max {
			return false
		}
		if slices.ContainsFunc(chosen, func(c Widget) bool { return c.ID == widget.ID }) {
			return true
		}
		chosen = append(chosen, widget)
		return true
	}

	for _, topic := range topics {
		for _, widget := range allowed {
			if widget.Topic != topic {
				continue
			}
			if !appendWidget(widget) {
				return chosen
			}
		}
	}

	for _, widget := range allowed {
		if !appendWidget(widget) {
			break
		}
	}

	return chosen
}

// validateSelection turns a model answer into widgets that are safe to render.
//
// The security-relevant case and the merely sloppy one share a single code
// path: an id the model invented and an id the caller is not allowed to see are
// both simply absent from allowed, and both get dropped. The enum in the
// response schema already constrains the model to the caller's ids; this is the
// check that does not depend on the model having honoured it.
//
// It returns the widgets and whether the answer survived well enough to still
// be called the model's work.
func validateSelection(raw selection, allowed []Widget, topics []Topic) ([]models.DashboardWidget, bool) {
	byID := make(map[string]Widget, len(allowed))
	for _, widget := range allowed {
		byID[widget.ID] = widget
	}

	chosen := make([]models.DashboardWidget, 0, MaxWidgets)
	seen := make(map[string]struct{}, MaxWidgets)

	for _, candidate := range raw.Widgets {
		if len(chosen) >= MaxWidgets {
			break
		}
		widget, ok := byID[candidate.ID]
		if !ok {
			continue
		}
		if _, duplicate := seen[widget.ID]; duplicate {
			continue
		}
		seen[widget.ID] = struct{}{}
		chosen = append(chosen, models.DashboardWidget{
			ID:     widget.ID,
			Size:   normalizeSize(candidate.Size, widget.DefaultSize),
			Reason: truncateRunes(strings.TrimSpace(candidate.Reason), maxReasonRunes),
		})
	}

	if len(chosen) == 0 {
		return fromWidgets(fallbackSelect(allowed, topics, MaxWidgets)), false
	}

	if len(chosen) < MinWidgets {
		for _, widget := range fallbackSelect(allowed, topics, MaxWidgets) {
			if len(chosen) >= MinWidgets {
				break
			}
			if _, duplicate := seen[widget.ID]; duplicate {
				continue
			}
			seen[widget.ID] = struct{}{}
			chosen = append(chosen, models.DashboardWidget{
				ID:   widget.ID,
				Size: string(widget.DefaultSize),
			})
		}
	}

	return chosen, true
}

// fromWidgets renders catalog entries at their default size, with no reason:
// the fallback path does not invent prose it cannot justify.
func fromWidgets(widgets []Widget) []models.DashboardWidget {
	out := make([]models.DashboardWidget, 0, len(widgets))
	for _, widget := range widgets {
		out = append(out, models.DashboardWidget{
			ID:   widget.ID,
			Size: string(widget.DefaultSize),
		})
	}
	return out
}

func normalizeSize(size string, fallback Size) string {
	switch Size(size) {
	case SizeSmall, SizeMedium, SizeLarge:
		return size
	}
	return string(fallback)
}

func truncateRunes(s string, max int) string {
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	return string([]rune(s)[:max])
}

// parseTopics maps the request's validated strings onto the topic type.
func parseTopics(raw []string) []Topic {
	topics := make([]Topic, 0, len(raw))
	for _, value := range raw {
		topic := Topic(value)
		if !slices.Contains(topics, topic) {
			topics = append(topics, topic)
		}
	}
	return topics
}

// trimTitle collapses whitespace so a model answer padded with newlines does
// not become a multi-line heading.
func trimTitle(title string) string {
	return strings.Join(strings.Fields(title), " ")
}
