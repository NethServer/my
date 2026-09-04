/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package email

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tempPassword covers every character the generator emits that html/template
// would escape (helpers/password_generation.go)
const tempPassword = "aB3&x+7<Zq>!'"

func TestRenderTemplateEscapesHTMLOnly(t *testing.T) {
	dir := t.TempDir()
	for name, content := range map[string]string{
		"welcome_en.txt":  "Password: {{.TempPassword}}",
		"welcome_en.html": "<p>{{.TempPassword}}</p>",
	} {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o600); err != nil {
			t.Fatalf("write %s: %v", name, err)
		}
	}

	service := &TemplateService{templateDir: dir}
	html, text, err := service.GenerateWelcomeEmail(WelcomeEmailData{TempPassword: tempPassword}, "en")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}

	if want := "Password: " + tempPassword; text != want {
		t.Errorf("text = %q, want %q", text, want)
	}
	if strings.Contains(html, tempPassword) {
		t.Errorf("html = %q, want the password escaped", html)
	}
	if !strings.Contains(html, "&amp;") {
		t.Errorf("html = %q, want HTML entities", html)
	}
}

func TestGenerateWelcomeEmailKeepsPasswordUsableInTextPart(t *testing.T) {
	for _, language := range []string{"en", "it"} {
		t.Run(language, func(t *testing.T) {
			_, text, err := NewTemplateService().GenerateWelcomeEmail(WelcomeEmailData{
				UserName:         "Mario Rossi",
				UserEmail:        "edoardo.spadoni+welcome@nethesis.it",
				OrganizationName: "Società Àcme S.r.l.",
				OrganizationType: "reseller",
				UserRoles:        []string{"Admin"},
				TempPassword:     tempPassword,
				LoginURL:         "https://my.nethesis.it/account?changePassword=true",
				SupportEmail:     "no-reply@nethesis.it",
				CompanyName:      "My Nethesis",
			}, language)
			if err != nil {
				t.Fatalf("generate: %v", err)
			}

			if !strings.Contains(text, tempPassword) {
				t.Errorf("text part does not carry the password verbatim:\n%s", text)
			}
			if !strings.Contains(text, "edoardo.spadoni+welcome@nethesis.it") {
				t.Errorf("text part does not carry the address verbatim:\n%s", text)
			}
			for _, entity := range []string{"&amp;", "&#43;", "&lt;", "&gt;", "&#39;", "&#34;"} {
				if strings.Contains(text, entity) {
					t.Errorf("text part contains HTML entity %s:\n%s", entity, text)
				}
			}
		})
	}
}
