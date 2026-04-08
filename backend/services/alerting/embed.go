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
var ValidTemplateLangs = []string{"en", "it"}

// BuildTemplateFiles returns all Alertmanager template file contents for
// the given language, plus a generated dispatcher template that routes
// firing/resolved notifications to the correct language-specific template.
// lang defaults to "en" when empty.
func BuildTemplateFiles(lang string) (map[string]string, error) {
	if lang == "" {
		lang = "en"
	}

	names := []string{
		"firing_" + lang + ".html",
		"resolved_" + lang + ".html",
		"firing_" + lang + ".txt",
		"resolved_" + lang + ".txt",
	}

	files := make(map[string]string, len(names)+1)
	for _, name := range names {
		content, err := templateFS.ReadFile("templates/" + name)
		if err != nil {
			return nil, fmt.Errorf("loading alert template %s: %w", name, err)
		}
		files[name] = string(content)
	}

	// Dispatcher routes firing/resolved to the correct language template.
	files["_dispatcher.tmpl"] = buildDispatcher(lang)

	return files, nil
}

// buildDispatcher generates a small Alertmanager template file that dispatches
// to the correct firing/resolved template based on .Status.
func buildDispatcher(lang string) string {
	var sb strings.Builder
	fmt.Fprintf(&sb,
		"{{ define \"alert.html\" }}"+
			"{{ if eq .Status \"firing\" }}{{ template \"firing_%s.html\" . }}"+
			"{{ else }}{{ template \"resolved_%s.html\" . }}{{ end }}"+
			"{{ end }}\n",
		lang, lang,
	)
	fmt.Fprintf(&sb,
		"{{ define \"alert.txt\" }}"+
			"{{ if eq .Status \"firing\" }}{{ template \"firing_%s.txt\" . }}"+
			"{{ else }}{{ template \"resolved_%s.txt\" . }}{{ end }}"+
			"{{ end }}\n",
		lang, lang,
	)
	fmt.Fprintf(&sb,
		"{{ define \"alert.subject\" }}"+
			"{{ if eq .Status \"firing\" }}{{ template \"firing_%s.subject\" . }}"+
			"{{ else }}{{ template \"resolved_%s.subject\" . }}{{ end }}"+
			"{{ end }}\n",
		lang, lang,
	)
	return sb.String()
}
