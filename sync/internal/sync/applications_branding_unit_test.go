/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package sync

import (
	"encoding/base64"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/nethesis/my/sync/internal/config"
)

// newBrandingEngine returns an engine whose config file sits in a temporary
// directory holding an apps/ subdirectory with a single SVG icon pair
func newBrandingEngine(t *testing.T) *Engine {
	t.Helper()

	dir := t.TempDir()
	appsDir := filepath.Join(dir, "apps")
	if err := os.MkdirAll(appsDir, 0o755); err != nil {
		t.Fatalf("failed to create apps directory: %v", err)
	}

	for name, content := range map[string]string{
		"forum.svg":      `<svg id="light"/>`,
		"forum-dark.svg": `<svg id="dark"/>`,
	} {
		if err := os.WriteFile(filepath.Join(appsDir, name), []byte(content), 0o644); err != nil {
			t.Fatalf("failed to write %s: %v", name, err)
		}
	}

	return NewEngine(nil, &Options{ConfigFile: filepath.Join(dir, "config.yml")})
}

func TestBuildAppBranding(t *testing.T) {
	engine := newBrandingEngine(t)

	t.Run("no branding block leaves the uploaded icons alone", func(t *testing.T) {
		branding, err := engine.buildAppBranding(config.Application{Name: "partner.nethesis.it"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if branding != nil {
			t.Errorf("expected no branding payload, got %+v", branding)
		}
	})

	t.Run("empty branding block clears the icons", func(t *testing.T) {
		branding, err := engine.buildAppBranding(config.Application{
			Name:     "partner.nethesis.it",
			Branding: &config.ApplicationBranding{},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if branding == nil {
			t.Fatal("expected an empty branding payload, got nil")
		}
		if branding.LogoURL != "" || branding.DarkLogoURL != "" {
			t.Errorf("expected empty icons, got %+v", branding)
		}
	})

	t.Run("icons are loaded as data URLs", func(t *testing.T) {
		branding, err := engine.buildAppBranding(config.Application{
			Name: "partner.nethesis.it",
			Branding: &config.ApplicationBranding{
				LogoPath:     "apps/forum.svg",
				LogoDarkPath: "apps/forum-dark.svg",
			},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, tc := range []struct {
			name     string
			dataURL  string
			expected string
		}{
			{"logo", branding.LogoURL, `<svg id="light"/>`},
			{"dark logo", branding.DarkLogoURL, `<svg id="dark"/>`},
		} {
			if !strings.HasPrefix(tc.dataURL, "data:image/svg+xml;base64,") {
				t.Errorf("%s: expected an SVG data URL, got %q", tc.name, tc.dataURL)
			}
			if !strings.Contains(tc.dataURL, base64Of(t, tc.expected)) {
				t.Errorf("%s: data URL does not carry the file content", tc.name)
			}
		}
	})

	t.Run("a single icon is enough", func(t *testing.T) {
		branding, err := engine.buildAppBranding(config.Application{
			Name:     "partner.nethesis.it",
			Branding: &config.ApplicationBranding{LogoPath: "apps/forum.svg"},
		})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if branding.LogoURL == "" {
			t.Error("expected the logo to be loaded")
		}
		if branding.DarkLogoURL != "" {
			t.Errorf("expected no dark logo, got %q", branding.DarkLogoURL)
		}
	})

	t.Run("a missing icon fails the application", func(t *testing.T) {
		_, err := engine.buildAppBranding(config.Application{
			Name:     "partner.nethesis.it",
			Branding: &config.ApplicationBranding{LogoPath: "apps/missing.svg"},
		})
		if err == nil {
			t.Fatal("expected an error for a missing icon")
		}
		if !strings.Contains(err.Error(), "missing.svg") {
			t.Errorf("expected the error to name the file, got %v", err)
		}
	})
}

func base64Of(t *testing.T, content string) string {
	t.Helper()
	return base64.StdEncoding.EncodeToString([]byte(content))
}
