/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/nethesis/my/backend/models"
)

func TestBuildRebrandingUpsert(t *testing.T) {
	t.Run("an uploaded asset is written with its mime and file name", func(t *testing.T) {
		uploads := map[string]models.RebrandingUpload{
			"logo_light_rect": {Data: []byte("svg"), MimeType: "image/svg+xml", Filename: "logo-light.svg"},
		}

		query, args := buildRebrandingUpsert("org1", "nethvoice", nil, uploads, nil)

		assert.Contains(t, query, "logo_light_rect = $4")
		assert.Contains(t, query, "logo_light_rect_mime = $5")
		assert.Contains(t, query, "logo_light_rect_filename = $6")
		assert.Equal(t, []interface{}{
			"org1", "nethvoice", nil,
			[]byte("svg"), "image/svg+xml", "logo-light.svg",
		}, args)
	})

	t.Run("an asset that is neither uploaded nor cleared keeps its value", func(t *testing.T) {
		query, _ := buildRebrandingUpsert("org1", "nethvoice", nil, nil, nil)

		for _, asset := range rebrandingAssetNames {
			assert.NotContains(t, query, asset+" = ")
		}
		assert.Contains(t, query, "updated_at = NOW()")
	})

	t.Run("a cleared asset is nulled, binary mime and file name together", func(t *testing.T) {
		query, args := buildRebrandingUpsert("org1", "nethvoice", nil, nil, []string{"favicon"})

		assert.Contains(t, query, "favicon = NULL")
		assert.Contains(t, query, "favicon_mime = NULL")
		assert.Contains(t, query, "favicon_filename = NULL")
		assert.Len(t, args, 3)
	})

	t.Run("clearing an asset that is also uploaded keeps the upload", func(t *testing.T) {
		uploads := map[string]models.RebrandingUpload{
			"favicon": {Data: []byte("png"), MimeType: "image/png", Filename: "favicon.png"},
		}

		query, _ := buildRebrandingUpsert("org1", "nethvoice", nil, uploads, []string{"favicon"})

		assert.Contains(t, query, "favicon = $4")
		assert.NotContains(t, query, "favicon = NULL")
	})

	t.Run("the brand name is left alone when the form does not send it", func(t *testing.T) {
		query, args := buildRebrandingUpsert("org1", "nethvoice", nil, nil, nil)
		assert.NotContains(t, query, "product_name = $3")
		assert.Nil(t, args[2])

		brand := "UrbanGrid"
		query, args = buildRebrandingUpsert("org1", "nethvoice", &brand, nil, nil)
		assert.Contains(t, query, "product_name = $3")
		assert.Equal(t, brand, args[2])
	})

	t.Run("a brand name sent empty clears the stored one", func(t *testing.T) {
		empty := ""
		query, args := buildRebrandingUpsert("org1", "nethvoice", &empty, nil, nil)

		// The column is written, and what is written is NULL — not an empty
		// string, which would render as a blank custom name instead of falling
		// back to the product's own.
		assert.Contains(t, query, "product_name = $3")
		assert.Nil(t, args[2])
	})
}
