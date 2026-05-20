/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	"embed"
	"fmt"
	"strings"
)

//go:embed templates/*
var templateFS embed.FS

// ValidTemplateLangs lists supported email template languages.
//
// All supported languages are shipped with every tenant push because the
// merged effective config can mix languages across recipients (one email
// recipient in en, another in it). The renderer picks per-recipient which
// dispatcher to reference.
var ValidTemplateLangs = []string{"en", "it"}

// BuildTemplateFiles returns the complete bundle of Alertmanager template
// files for every supported language plus per-language dispatcher templates
// (alert_<lang>.html / alert_<lang>.txt / alert_<lang>.subject).
//
// appURL is substituted into the ${APP_URL} placeholder inside the
// language-specific templates, used by the "view system" CTA to build a
// link to the frontend.
//
// Output layout:
//
//	firing_en.html / firing_en.txt / firing_en.subject (defined in firing_en.html)
//	resolved_en.html / resolved_en.txt / resolved_en.subject
//	telegram_en.tmpl
//	(same for it)
//	_dispatcher.tmpl  — defines alert_en.{html,txt,subject} and alert_it.{html,txt,subject}
func BuildTemplateFiles(appURL string) (map[string]string, error) {
	files := map[string]string{}
	for _, lang := range ValidTemplateLangs {
		names := []string{
			"firing_" + lang + ".html",
			"resolved_" + lang + ".html",
			"firing_" + lang + ".txt",
			"resolved_" + lang + ".txt",
			"telegram_" + lang + ".tmpl",
		}
		for _, name := range names {
			content, err := templateFS.ReadFile("templates/" + name)
			if err != nil {
				return nil, fmt.Errorf("loading alert template %s: %w", name, err)
			}
			files[name] = strings.ReplaceAll(string(content), "${APP_URL}", appURL)
		}
	}
	files["_dispatcher.tmpl"] = buildDispatcher()
	return files, nil
}

// buildDispatcher generates a per-language dispatcher template file that
// routes firing/resolved notifications to the right language-specific
// template. Names are suffixed with `_<lang>` so multiple languages can
// coexist in the same Mimir-loaded template set without colliding on the
// unqualified `alert.html` / `alert.txt` / `alert.subject` names.
//
// Each email_configs entry in the rendered YAML picks its dispatcher via
// `{{ template "alert_<lang>.html" . }}` (and equivalents for text/subject).
func buildDispatcher() string {
	var sb strings.Builder
	for _, lang := range ValidTemplateLangs {
		fmt.Fprintf(&sb,
			"{{ define \"alert_%s.html\" }}"+
				"{{ if eq .Status \"firing\" }}{{ template \"firing_%s.html\" . }}"+
				"{{ else }}{{ template \"resolved_%s.html\" . }}{{ end }}"+
				"{{ end }}\n",
			lang, lang, lang,
		)
		fmt.Fprintf(&sb,
			"{{ define \"alert_%s.txt\" }}"+
				"{{ if eq .Status \"firing\" }}{{ template \"firing_%s.txt\" . }}"+
				"{{ else }}{{ template \"resolved_%s.txt\" . }}{{ end }}"+
				"{{ end }}\n",
			lang, lang, lang,
		)
		fmt.Fprintf(&sb,
			"{{ define \"alert_%s.subject\" }}"+
				"{{ if eq .Status \"firing\" }}{{ template \"firing_%s.subject\" . }}"+
				"{{ else }}{{ template \"resolved_%s.subject\" . }}{{ end }}"+
				"{{ end }}\n",
			lang, lang, lang,
		)
	}
	return sb.String()
}
