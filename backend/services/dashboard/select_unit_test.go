/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package dashboard

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func chosenIDs(widgets []Widget) []string {
	out := make([]string, 0, len(widgets))
	for _, widget := range widgets {
		out = append(out, widget.ID)
	}
	return out
}

// customerCatalog is the filtered catalog of a persona that cannot read any
// organization resource — the case the leakage tests hang on.
func customerCatalog() []Widget {
	return CatalogFor(userWith("read:systems", "read:applications", "read:entitlements"))
}

func TestFallbackSelectFollowsTopicOrder(t *testing.T) {
	allowed := customerCatalog()

	first := fallbackSelect(allowed, []Topic{TopicEntitlements, TopicSystems}, MaxWidgets)
	require.NotEmpty(t, first)

	widget, ok := ByID(first[0].ID)
	require.True(t, ok)
	assert.Equal(t, TopicEntitlements, widget.Topic,
		"the first topic the user asked for should lead the board")

	reversed := fallbackSelect(allowed, []Topic{TopicSystems, TopicEntitlements}, MaxWidgets)
	require.NotEmpty(t, reversed)
	assert.NotEqual(t, first[0].ID, reversed[0].ID)
}

func TestFallbackSelectIsDeterministic(t *testing.T) {
	allowed := customerCatalog()
	topics := []Topic{TopicAlerts, TopicEntitlements}

	assert.Equal(t,
		chosenIDs(fallbackSelect(allowed, topics, MaxWidgets)),
		chosenIDs(fallbackSelect(allowed, topics, MaxWidgets)))
}

func TestFallbackSelectRespectsThePermissionFilter(t *testing.T) {
	allowed := customerCatalog()

	// The user asks for organizations explicitly; nothing in that topic survived
	// the filter, so the board is topped up from what did.
	chosen := fallbackSelect(allowed, []Topic{TopicOrganizations}, MaxWidgets)

	assert.NotEmpty(t, chosen)
	for _, id := range []string{"distributors_counter", "resellers_counter", "customers_counter"} {
		assert.NotContains(t, chosenIDs(chosen), id)
	}
}

func TestFallbackSelectFillsTheBoardWithoutTopics(t *testing.T) {
	chosen := fallbackSelect(customerCatalog(), nil, MaxWidgets)

	assert.GreaterOrEqual(t, len(chosen), MinWidgets)
	assert.LessOrEqual(t, len(chosen), MaxWidgets)
}

func TestFallbackSelectNeverRepeatsAWidget(t *testing.T) {
	chosen := fallbackSelect(customerCatalog(), []Topic{TopicAlerts, TopicAlerts, TopicSystems}, MaxWidgets)

	seen := make(map[string]struct{}, len(chosen))
	for _, widget := range chosen {
		_, duplicate := seen[widget.ID]
		assert.False(t, duplicate, "widget %s appears twice", widget.ID)
		seen[widget.ID] = struct{}{}
	}
}

func TestValidateSelectionDropsWidgetsTheCallerCannotSee(t *testing.T) {
	allowed := customerCatalog()

	raw := selection{
		Title: "Overview",
		Widgets: []selectedWidget{
			// Real ids, but this caller holds none of read:distributors|resellers|customers.
			{ID: "distributors_counter", Size: "sm"},
			{ID: "customers_counter", Size: "sm"},
			// Invented outright.
			{ID: "definitely_not_a_widget", Size: "lg"},
			{ID: "alerts_counter", Size: "sm"},
			{ID: "systems_counter", Size: "sm"},
			{ID: "addons_expiring", Size: "md"},
		},
	}

	widgets, fromModel := validateSelection(raw, allowed, []Topic{TopicAlerts})

	assert.True(t, fromModel)
	ids := make([]string, 0, len(widgets))
	for _, widget := range widgets {
		ids = append(ids, widget.ID)
	}
	assert.Equal(t, []string{"alerts_counter", "systems_counter", "addons_expiring"}, ids)
}

func TestValidateSelectionPreservesModelOrderAndDedupes(t *testing.T) {
	raw := selection{Widgets: []selectedWidget{
		{ID: "addons_expiring", Size: "md"},
		{ID: "alerts_counter", Size: "sm"},
		{ID: "addons_expiring", Size: "lg"},
		{ID: "systems_counter", Size: "sm"},
	}}

	widgets, fromModel := validateSelection(raw, customerCatalog(), []Topic{TopicAlerts})

	require.True(t, fromModel)
	require.Len(t, widgets, 3)
	assert.Equal(t, "addons_expiring", widgets[0].ID)
	assert.Equal(t, "alerts_counter", widgets[1].ID)
	assert.Equal(t, "md", widgets[0].Size, "the first occurrence wins")
}

func TestValidateSelectionNormalizesSize(t *testing.T) {
	raw := selection{Widgets: []selectedWidget{
		{ID: "alerts_counter", Size: "enormous"},
		{ID: "systems_counter", Size: ""},
		{ID: "addons_expiring", Size: "lg"},
	}}

	widgets, _ := validateSelection(raw, customerCatalog(), nil)

	require.Len(t, widgets, 3)
	assert.Equal(t, "sm", widgets[0].Size, "an unknown size falls back to the catalog default")
	assert.Equal(t, "sm", widgets[1].Size)
	assert.Equal(t, "lg", widgets[2].Size, "a valid size the model chose is kept")
}

func TestValidateSelectionTruncatesTheReason(t *testing.T) {
	raw := selection{Widgets: []selectedWidget{
		{ID: "alerts_counter", Size: "sm", Reason: strings.Repeat("è", maxReasonRunes+40)},
		{ID: "systems_counter", Size: "sm"},
		{ID: "addons_expiring", Size: "md"},
	}}

	widgets, _ := validateSelection(raw, customerCatalog(), nil)

	assert.Equal(t, strings.Repeat("è", maxReasonRunes), widgets[0].Reason)
}

func TestValidateSelectionClampsToMaxWidgets(t *testing.T) {
	allowed := customerCatalog()
	raw := selection{}
	for _, widget := range allowed {
		raw.Widgets = append(raw.Widgets, selectedWidget{ID: widget.ID, Size: string(widget.DefaultSize)})
	}
	require.Greater(t, len(raw.Widgets), MaxWidgets)

	widgets, _ := validateSelection(raw, allowed, nil)

	assert.Len(t, widgets, MaxWidgets)
}

func TestValidateSelectionTopsUpAShortAnswer(t *testing.T) {
	raw := selection{Widgets: []selectedWidget{{ID: "alerts_counter", Size: "sm"}}}

	widgets, fromModel := validateSelection(raw, customerCatalog(), []Topic{TopicSystems})

	assert.True(t, fromModel, "one good widget is still the model's work")
	assert.Len(t, widgets, MinWidgets)
	assert.Equal(t, "alerts_counter", widgets[0].ID, "the model's pick keeps its place")

	seen := make(map[string]struct{}, len(widgets))
	for _, widget := range widgets {
		_, duplicate := seen[widget.ID]
		assert.False(t, duplicate)
		seen[widget.ID] = struct{}{}
	}
}

func TestValidateSelectionFallsBackWhenNothingSurvives(t *testing.T) {
	raw := selection{Widgets: []selectedWidget{
		{ID: "distributors_counter", Size: "sm"},
		{ID: "nope", Size: "lg"},
	}}

	widgets, fromModel := validateSelection(raw, customerCatalog(), []Topic{TopicSystems})

	assert.False(t, fromModel, "an answer with no usable widget is not the model's work")
	assert.GreaterOrEqual(t, len(widgets), MinWidgets)
	for _, widget := range widgets {
		assert.Empty(t, widget.Reason, "the fallback path invents no prose")
	}
}

func TestParseTopicsDedupes(t *testing.T) {
	assert.Equal(t,
		[]Topic{TopicAlerts, TopicSystems},
		parseTopics([]string{"alerts", "systems", "alerts"}))
}

func TestTrimTitleCollapsesWhitespace(t *testing.T) {
	assert.Equal(t, "Your operations board", trimTitle("  Your\n operations\tboard \n"))
}
