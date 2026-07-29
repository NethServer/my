/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package alerting

import (
	htmltpl "html/template"
	"regexp"
	"strings"
	"testing"
	texttpl "text/template"
	"time"
	"unicode/utf8"
)

// The alert notification templates are executed by Mimir's alertmanager, not
// by this codebase, so nothing here would otherwise notice a template that
// fails to render — a broken one is only visible as a notification that never
// arrives. These tests execute the shipped bundle against the same data shape
// and function map alertmanager uses, so a bad expression fails the build
// instead of silently dropping mail.

// amFuncs mirrors the subset of alertmanager's DefaultFuncs the templates use.
// Keep in sync with the vendored fork (grafana/prometheus-alertmanager): note
// it exposes only date/tz, NOT since/now/humanizeDuration.
func amFuncs() map[string]any {
	return map[string]any{
		"toUpper": strings.ToUpper,
		"toLower": strings.ToLower,
		"date":    func(format string, t time.Time) string { return t.Format(format) },
		"tz": func(name string, t time.Time) (time.Time, error) {
			loc, err := time.LoadLocation(name)
			if err != nil {
				return time.Time{}, err
			}
			return t.In(loc), nil
		},
	}
}

// amAlert / amData mirror alertmanager's template data. .Alerts.Firing and
// .Alerts.Resolved are methods on the real type and plain fields here; the
// templates reach them identically either way.
type amAlert struct {
	Labels      map[string]string
	Annotations map[string]string
	StartsAt    time.Time
	EndsAt      time.Time
}

type amData struct {
	Status string
	Alerts struct {
		Firing   []amAlert
		Resolved []amAlert
	}
	CommonLabels map[string]string
}

func sampleAlert(startsAt, endsAt time.Time) amAlert {
	return amAlert{
		Labels: map[string]string{
			"alertname":         "SystemDown",
			"severity":          "critical",
			"service":           "node",
			"organization_name": "Rossi Informatica Srl",
			"organization_type": "customer",
			"organization_vat":  "01234567890",
			"system_name":       "srv-milano-01",
			"system_key":        "a1b2c3d4",
			"system_fqdn":       "srv-milano-01.rossi.local",
			"system_id":         "42",
		},
		Annotations: map[string]string{
			"summary_it":     "Il sistema non risponde",
			"summary_en":     "System is unreachable",
			"description_it": "Nessun dato ricevuto dal sistema negli ultimi 10 minuti.",
			"description_en": "No data received from the system in the last 10 minutes.",
		},
		StartsAt: startsAt,
		EndsAt:   endsAt,
	}
}

func sampleData(status string, a amAlert) amData {
	d := amData{Status: status, CommonLabels: a.Labels}
	if status == "firing" {
		d.Alerts.Firing = []amAlert{a}
	} else {
		d.Alerts.Resolved = []amAlert{a}
	}
	return d
}

// renderBundle executes one named template out of the BuildTemplateFiles
// bundle. htmlSet picks html/template over text/template, matching how
// alertmanager renders the `html` field versus `text`/`message`/`Subject`.
func renderBundle(t *testing.T, files map[string]string, name string, htmlSet bool, data amData) string {
	t.Helper()
	var (
		out strings.Builder
		err error
	)
	if htmlSet {
		tmpl := htmltpl.New("root").Funcs(amFuncs())
		for _, src := range files {
			if tmpl, err = tmpl.Parse(src); err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
		}
		err = tmpl.ExecuteTemplate(&out, name, data)
	} else {
		tmpl := texttpl.New("root").Funcs(amFuncs())
		for _, src := range files {
			if tmpl, err = tmpl.Parse(src); err != nil {
				t.Fatalf("parse %s: %v", name, err)
			}
		}
		err = tmpl.ExecuteTemplate(&out, name, data)
	}
	if err != nil {
		t.Fatalf("execute %s: %v", name, err)
	}
	return out.String()
}

// TestTemplates_RenderWithoutError executes every entry point alertmanager
// references — the per-language dispatchers for both statuses, plus the
// Telegram message — and checks nothing is left unsubstituted.
func TestTemplates_RenderWithoutError(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "Europe/Rome")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}
	alert := sampleAlert(
		time.Date(2026, 7, 3, 11, 40, 13, 0, time.UTC),
		time.Date(2026, 7, 3, 12, 12, 41, 0, time.UTC),
	)

	for _, lang := range ValidTemplateLangs {
		for _, status := range []string{"firing", "resolved"} {
			data := sampleData(status, alert)
			for _, tc := range []struct {
				name    string
				htmlSet bool
			}{
				{"alert_" + lang + ".html", true},
				{"alert_" + lang + ".txt", false},
				{"alert_" + lang + ".subject", false},
				{"telegram_" + lang + ".message", false},
			} {
				out := renderBundle(t, files, tc.name, tc.htmlSet, data)
				if out == "" {
					t.Errorf("%s (%s): rendered empty", tc.name, status)
				}
				if strings.Contains(out, "${") {
					t.Errorf("%s (%s): unsubstituted placeholder in output", tc.name, status)
				}
			}
		}
	}
}

// TestTemplates_TimestampsUseConfiguredTimezone pins the rendered timestamp
// format, including the DST-dependent zone abbreviation. The offset and the
// abbreviation both come from tzdata, so this also fails if the timezone is
// unresolvable in the environment running the test.
func TestTemplates_TimestampsUseConfiguredTimezone(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "Europe/Rome")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}

	cases := []struct {
		name     string
		startsAt time.Time
		lang     string
		want     string
	}{
		{"it summer CEST", time.Date(2026, 7, 3, 11, 40, 13, 0, time.UTC), "it", "03 lug 2026, 13:40 CEST"},
		{"it winter CET", time.Date(2026, 1, 15, 11, 40, 13, 0, time.UTC), "it", "15 gen 2026, 12:40 CET"},
		{"en summer CEST", time.Date(2026, 7, 3, 11, 40, 13, 0, time.UTC), "en", "Jul 03, 2026, 01:40 PM CEST"},
		{"en winter CET", time.Date(2026, 1, 15, 11, 40, 13, 0, time.UTC), "en", "Jan 15, 2026, 12:40 PM CET"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			data := sampleData("firing", sampleAlert(tc.startsAt, tc.startsAt.Add(30*time.Minute)))
			for _, target := range []struct {
				name    string
				htmlSet bool
			}{
				{"alert_" + tc.lang + ".html", true},
				{"alert_" + tc.lang + ".txt", false},
				{"telegram_" + tc.lang + ".message", false},
			} {
				out := renderBundle(t, files, target.name, target.htmlSet, data)
				if !strings.Contains(out, tc.want) {
					t.Errorf("%s: missing %q in output", target.name, tc.want)
				}
				// The UTC duplicate this format replaced carried seconds.
				if strings.Contains(out, "11:40:13") {
					t.Errorf("%s: unexpected seconds-precision timestamp in output", target.name)
				}
			}
		})
	}
}

// TestBuildTemplateFiles_TimezoneIsSubstituted guards the placeholder itself:
// a template that hardcodes a zone instead of using ${ALERT_TZ} would silently
// ignore ALERTING_TIMEZONE.
func TestBuildTemplateFiles_TimezoneIsSubstituted(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "America/New_York")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}
	for name, src := range files {
		if strings.Contains(src, "${ALERT_TZ}") {
			t.Errorf("%s: ${ALERT_TZ} left unsubstituted", name)
		}
		if strings.Contains(src, `tz "Europe/`) {
			t.Errorf("%s: hardcoded timezone instead of ${ALERT_TZ}", name)
		}
		// Only the shared helper may call tz; a template formatting a
		// timestamp inline would drift from the rest.
		if name != "_datetime.tmpl" && strings.Contains(src, "tz ") {
			t.Errorf("%s: calls tz directly, use the ts_it/ts_en templates", name)
		}
	}

	data := sampleData("firing", sampleAlert(
		time.Date(2026, 7, 3, 11, 40, 13, 0, time.UTC),
		time.Date(2026, 7, 3, 12, 12, 41, 0, time.UTC),
	))
	out := renderBundle(t, files, "alert_it.txt", false, data)
	if want := "03 lug 2026, 07:40 EDT"; !strings.Contains(out, want) {
		t.Errorf("missing %q in output rendered with America/New_York", want)
	}
}

// TestTemplates_UTCLineNotInTelegram pins where the secondary UTC timestamp
// appears: both email bodies carry it as a second line under the local time —
// muted grey in the HTML, indented to the value column in the plain text. The
// Telegram message does not: it is a chat notification where a second line per
// timestamp costs more than the correlation with server logs is worth there.
func TestTemplates_UTCLineNotInTelegram(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "Europe/Rome")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}
	data := sampleData("firing", sampleAlert(
		time.Date(2026, 7, 3, 11, 40, 13, 0, time.UTC),
		time.Date(2026, 7, 3, 12, 12, 41, 0, time.UTC),
	))
	const utcStamp = "2026-07-03 11:40 UTC"

	for _, lang := range ValidTemplateLangs {
		if out := renderBundle(t, files, "alert_"+lang+".html", true, data); !strings.Contains(out, utcStamp) {
			t.Errorf("alert_%s.html: missing the secondary UTC line %q", lang, utcStamp)
		}
		if out := renderBundle(t, files, "alert_"+lang+".txt", false, data); !strings.Contains(out, utcStamp) {
			t.Errorf("alert_%s.txt: missing the secondary UTC line %q", lang, utcStamp)
		}
		if out := renderBundle(t, files, "telegram_"+lang+".message", false, data); strings.Contains(out, utcStamp) {
			t.Errorf("telegram_%s.message: UTC line does not belong in the Telegram body", lang)
		}
	}
}

// TestPlainTextUTCLine_AlignedToValueColumn keeps the secondary UTC line under
// the value it belongs to rather than under the label — the indentation is the
// only thing marking it as secondary in a body with no styling.
func TestPlainTextUTCLine_AlignedToValueColumn(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "Europe/Rome")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}
	data := sampleData("resolved", sampleAlert(
		time.Date(2026, 7, 3, 11, 40, 13, 0, time.UTC),
		time.Date(2026, 7, 3, 12, 12, 41, 0, time.UTC),
	))
	widths := map[string]int{"it": 16, "en": 14}

	for _, lang := range ValidTemplateLangs {
		out := renderBundle(t, files, "alert_"+lang+".txt", false, data)
		found := 0
		for _, line := range strings.Split(out, "\n") {
			if !strings.Contains(line, "UTC") || strings.TrimSpace(line) == "" {
				continue
			}
			trimmed := strings.TrimLeft(line, " ")
			indent := utf8.RuneCountInString(line) - utf8.RuneCountInString(trimmed)
			if indent != widths[lang] {
				t.Errorf("alert_%s.txt: UTC line %q indented %d, want %d", lang, trimmed, indent, widths[lang])
			}
			found++
		}
		if found != 2 {
			t.Errorf("alert_%s.txt: found %d UTC lines in a resolved body, want 2", lang, found)
		}
	}
}

// TestLabels_SameWordingAcrossChannels pins the wording of every field label.
// The three media style labels differently on purpose — the plain text uses an
// aligned uppercase column, the HTML a styled uppercase cell with no colon, the
// Telegram message inline sentence case — but the words themselves must match,
// so the same alert reads the same way wherever it lands.
func TestLabels_SameWordingAcrossChannels(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "Europe/Rome")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}

	cases := []struct {
		kind     string // firing | resolved
		lang     string
		text     string // aligned column label, colon included
		html     string // styled label cell
		telegram string // inline label, colon included
	}{
		{"firing", "it", "ATTIVO DAL:", ">ATTIVO DAL<", "Attivo dal:"},
		{"resolved", "it", "INIZIO:", ">INIZIO<", "Inizio:"},
		{"resolved", "it", "RISOLTO:", ">RISOLTO<", "Risolto:"},
		{"firing", "en", "FIRING SINCE:", ">FIRING SINCE<", "Firing since:"},
		{"resolved", "en", "STARTED:", ">STARTED<", "Started:"},
		{"resolved", "en", "RESOLVED:", ">RESOLVED<", "Resolved:"},
		{"firing", "it", "DETTAGLI:", ">DETTAGLI<", "Dettagli:"},
		{"firing", "en", "DETAILS:", ">DETAILS<", "Details:"},
		// One label per organization_type, including the fallback branch. The
		// three narrow ones used to be abbreviated in the plain text column
		// only, which is exactly the divergence this pins down.
		{"firing", "it", "CLIENTE:", "}}CLIENTE{{", "}}Cliente{{"},
		{"firing", "it", "RIVENDITORE:", "}}RIVENDITORE{{", "}}Rivenditore{{"},
		{"firing", "it", "DISTRIBUTORE:", "}}DISTRIBUTORE{{", "}}Distributore{{"},
		{"firing", "it", "ORGANIZZAZIONE:", "}}ORGANIZZAZIONE{{", "}}Organizzazione{{"},
		{"firing", "en", "CUSTOMER:", "}}CUSTOMER{{", "}}Customer{{"},
		{"firing", "en", "RESELLER:", "}}RESELLER{{", "}}Reseller{{"},
		{"firing", "en", "DISTRIBUTOR:", "}}DISTRIBUTOR{{", "}}Distributor{{"},
		{"firing", "en", "ORGANIZATION:", "}}ORGANIZATION{{", "}}Organization{{"},
	}

	for _, tc := range cases {
		t.Run(tc.kind+"/"+tc.lang+"/"+tc.text, func(t *testing.T) {
			if got := files[tc.kind+"_"+tc.lang+".txt"]; !strings.Contains(got, tc.text) {
				t.Errorf("%s_%s.txt: missing label %q", tc.kind, tc.lang, tc.text)
			}
			if got := files[tc.kind+"_"+tc.lang+".html"]; !strings.Contains(got, tc.html) {
				t.Errorf("%s_%s.html: missing label %q", tc.kind, tc.lang, tc.html)
			}
			if got := files["telegram_"+tc.lang+".tmpl"]; !strings.Contains(got, tc.telegram) {
				t.Errorf("telegram_%s.tmpl: missing label %q", tc.lang, tc.telegram)
			}
		})
	}
}

// TestPlainTextLabels_Aligned checks the fixed-width label column in the plain
// text bodies. Every branch has to pad to the same column, including the org
// fallback that only fires when organization_type is unrecognised — the case
// that is easiest to leave behind when a label is renamed.
func TestPlainTextLabels_Aligned(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "Europe/Rome")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}
	// Widths fit the longest label each language can emit: ORGANIZZAZIONE: and
	// ORGANIZATION:, both from the fallback branch.
	widths := map[string]int{"it": 16, "en": 14}
	label := regexp.MustCompile(`(?m)(?:^|\}\})([A-ZÀ-Ù][A-ZÀ-Ù'. ]*?:)( +)`)

	for _, name := range []string{"firing_it.txt", "resolved_it.txt", "firing_en.txt", "resolved_en.txt"} {
		lang := "en"
		if strings.Contains(name, "_it") {
			lang = "it"
		}
		want := widths[lang]
		for _, m := range label.FindAllStringSubmatch(files[name], -1) {
			// Runes, not bytes: SEVERITÀ: is 9 columns wide but 10 bytes long.
			got := utf8.RuneCountInString(m[1]) + utf8.RuneCountInString(m[2])
			if got != want {
				t.Errorf("%s: label %q occupies %d columns, want %d", name, m[1], got, want)
			}
		}
	}
}

// TestLabels_NoRetiredWording fails if a superseded label creeps back in.
func TestLabels_NoRetiredWording(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "Europe/Rome")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}
	retired := []string{
		"DESCRIZIONE", "DESCRIPTION", // superseded by DETTAGLI / DETAILS
		"SEVERITA'",           // superseded by SEVERITÀ
		"Attivo da:",          // superseded by Attivo dal: / Inizio:
		"RIVEND.", "DISTRIB.", // abbreviations dropped for the full wording
		"ORG.:", "ORGANIZATION:", // ORG.: dropped; ORGANIZATION: only in the txt column
	}
	for name, src := range files {
		for _, word := range retired {
			// ORGANIZATION:/ORGANIZZAZIONE: are legitimate in the plain text column.
			if strings.HasSuffix(word, ":") && strings.HasSuffix(name, ".txt") {
				continue
			}
			if strings.Contains(src, word) {
				t.Errorf("%s: retired label %q is back", name, word)
			}
		}
	}
}

// TestMonthIT_AllMonths covers the hand-written Italian month abbreviations —
// a table Go's time package cannot provide, so nothing else would catch a typo
// or a missing branch.
func TestMonthIT_AllMonths(t *testing.T) {
	files, err := BuildTemplateFiles("https://my.nethesis.it", "UTC")
	if err != nil {
		t.Fatalf("BuildTemplateFiles: %v", err)
	}
	want := []string{"gen", "feb", "mar", "apr", "mag", "giu", "lug", "ago", "set", "ott", "nov", "dic"}
	for i, abbr := range want {
		month := time.Month(i + 1)
		data := sampleData("firing", sampleAlert(
			time.Date(2026, month, 9, 8, 30, 0, 0, time.UTC),
			time.Date(2026, month, 9, 9, 0, 0, 0, time.UTC),
		))
		out := renderBundle(t, files, "alert_it.txt", false, data)
		if expect := "09 " + abbr + " 2026, 08:30 UTC"; !strings.Contains(out, expect) {
			t.Errorf("month %d: missing %q", i+1, expect)
		}
	}
}
