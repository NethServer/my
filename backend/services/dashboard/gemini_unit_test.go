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
	"google.golang.org/genai"
)

// The pure functions only: nothing here builds an SDK client or reaches the
// network, so these run in CI with no key.

func TestBuildResponseSchemaConstrainsIDsToTheCallersCatalog(t *testing.T) {
	allowed := customerCatalog()
	schema := buildResponseSchema(allowedIDs(allowed))

	widgets := schema.Properties["widgets"]
	require.NotNil(t, widgets)
	require.NotNil(t, widgets.Items)

	enum := widgets.Items.Properties["id"].Enum
	assert.ElementsMatch(t, allowedIDs(allowed), enum)

	// The point of the enum: an id this caller may not see is not even
	// expressible in the answer.
	for _, denied := range []string{"distributors_counter", "resellers_counter", "customers_counter"} {
		assert.NotContains(t, enum, denied)
	}

	assert.Equal(t, int64(MinWidgets), *widgets.MinItems)
	assert.Equal(t, int64(MaxWidgets), *widgets.MaxItems)
	assert.ElementsMatch(t,
		[]string{"sm", "md", "lg"},
		widgets.Items.Properties["size"].Enum)
}

// The thinking level is worth pinning: the model's default costs several
// seconds on a task that does not need it, and the request deadline turns that
// straight into a fallback board.
func TestBuildConfigAsksForMinimalThinking(t *testing.T) {
	config := buildConfig(promptInput{Allowed: customerCatalog()})

	require.NotNil(t, config.ThinkingConfig)
	assert.Equal(t, genai.ThinkingLevelMinimal, config.ThinkingConfig.ThinkingLevel)
	// A zero budget is the other way to ask for this, and Gemini 3 rejects it.
	assert.Nil(t, config.ThinkingConfig.ThinkingBudget)
}

func TestBuildPromptListsOnlyTheAllowedWidgets(t *testing.T) {
	allowed := customerCatalog()
	prompt := buildPrompt(promptInput{
		Allowed: allowed,
		Topics:  []Topic{TopicAlerts, TopicSystems},
		Locale:  "en",
	})

	for _, widget := range allowed {
		assert.Contains(t, prompt, widget.ID)
	}
	for _, denied := range []string{"distributors_counter", "resellers_counter", "customers_counter"} {
		assert.NotContains(t, prompt, denied)
	}
	assert.Contains(t, prompt, "alerts, systems")
}

func TestBuildPromptFencesAndTruncatesTheHint(t *testing.T) {
	hint := strings.Repeat("à", maxHintRunes+50)
	prompt := buildPrompt(promptInput{
		Allowed: customerCatalog(),
		Hint:    hint,
		Locale:  "en",
	})

	assert.Contains(t, prompt, "<<<user-text")
	assert.Contains(t, prompt, "user-text>>>")
	assert.Contains(t, prompt, strings.Repeat("à", maxHintRunes))
	assert.NotContains(t, prompt, strings.Repeat("à", maxHintRunes+1))
}

func TestBuildPromptOmitsTheHintBlockWhenThereIsNoHint(t *testing.T) {
	prompt := buildPrompt(promptInput{Allowed: customerCatalog(), Locale: "en"})

	assert.NotContains(t, prompt, "<<<user-text")
}

func TestSanitizeHintStripsControlCharacters(t *testing.T) {
	assert.Equal(t, "one   two", sanitizeHint("one \x00 two\n"),
		"a control character becomes a space rather than vanishing, so words stay separated")
}

func TestBuildSystemInstructionFollowsTheLocale(t *testing.T) {
	assert.Contains(t, buildSystemInstruction("it"), "Italian")
	assert.Contains(t, buildSystemInstruction("en"), "English")
	assert.Contains(t, buildSystemInstruction(""), "English")
}
