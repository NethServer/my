/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strings"
)

// authzRun executes the requested layers and prints a verdict. It exits non-zero
// on any FAIL so it can gate a merge.
func authzRun(dir string, reg *Registry, flags map[string]string) error {
	a, err := newRunner(dir, reg, flags, false)
	if err != nil {
		return err
	}
	if only := flags["personas"]; only != "" {
		keep := map[string]bool{}
		for _, k := range strings.Split(only, ",") {
			keep[strings.TrimSpace(k)] = true
		}
		var filtered []*persona
		for _, p := range a.personas {
			if keep[p.id] {
				filtered = append(filtered, p)
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("no persona matches --personas=%s", only)
		}
		a.personas = filtered
	}

	fmt.Printf("Minting tokens for %d persona(s)…\n", len(a.personas))
	if err := a.mintTokens(); err != nil {
		return err
	}
	if err := a.preflight(); err != nil {
		return err
	}
	if drift := a.reportDrift(); drift > 0 && flags["ignore-drift"] != "true" {
		return fmt.Errorf("%d persona(s) carry token permissions that disagree with config.yml.\n"+
			"Expectations come from config.yml, so the matrix would be meaningless.\n"+
			"Run `sync push` (or pass --ignore-drift to proceed anyway)", drift)
	}

	layers := flags["layer"]
	if layers == "" {
		layers = "all"
	}
	want := func(name string) bool { return layers == "all" || strings.Contains(layers, name) }
	filter := flags["filter"]

	if want("gate") {
		fmt.Println("\n── gate layer ────────────────────────────────────────────")
		if err := a.runGateLayer(filter); err != nil {
			return err
		}
	}
	if want("scope") {
		fmt.Println("\n── scope layer ───────────────────────────────────────────")
		if err := a.runScopeLayer(filter); err != nil {
			return err
		}
	}
	if want("apps") {
		fmt.Println("\n── apps layer ────────────────────────────────────────────")
		if err := a.runAppsLayer(filter); err != nil {
			return err
		}
	}
	if want("special") {
		fmt.Println("\n── special layer ─────────────────────────────────────────")
		if err := a.runSpecialLayer(filter); err != nil {
			return err
		}
	}

	return a.report(flags["report"])
}

// preflight proves the harness is actually talking to the API before thousands of
// checks are interpreted. A wrong base URL answers every request with the
// router's own 404, which a naive reading takes for "the gate let me through" —
// so this asserts a known-good call succeeds and a known-bad one is recognised as
// unrouted rather than as a real not-found.
func (a *authzRunner) preflight() error {
	owner, ok := a.byKey["owner"]
	if !ok {
		return fmt.Errorf("no owner persona: cannot preflight")
	}
	res, err := a.request(owner.token, "GET", "/api/me", "")
	if err != nil {
		return fmt.Errorf("preflight GET /api/me: %w", err)
	}
	if res.status != 200 {
		return fmt.Errorf("preflight GET /api/me as owner returned %d (%s) via %s — is the backend up and is backend_url right?",
			res.status, res.message, a.baseURL)
	}
	res, err = a.request(owner.token, "GET", "/api/__authz_preflight_no_such_route", "")
	if err != nil {
		return err
	}
	if res.class != outNoRoute {
		return fmt.Errorf("preflight: an unrouted path returned %d %q instead of the router's %q — "+
			"the harness could not tell missing endpoints from real answers",
			res.status, res.message, noRouteMessage)
	}
	fmt.Printf("Preflight OK against %s\n", a.baseURL)
	return nil
}

// reportDrift compares, for each persona, the permissions config.yml grants with
// the ones its real token carries. Any difference means Logto is out of sync, in
// which case a failing check would say nothing about the middleware.
func (a *authzRunner) reportDrift() int {
	drift := 0
	for _, p := range a.personas {
		var missing, extra []string
		for perm := range p.perms {
			if !p.jwtPerms[perm] {
				missing = append(missing, perm)
			}
		}
		for perm := range p.jwtPerms {
			if !p.perms[perm] {
				extra = append(extra, perm)
			}
		}
		if len(missing) == 0 && len(extra) == 0 {
			continue
		}
		sort.Strings(missing)
		sort.Strings(extra)
		drift++
		fmt.Printf("  DRIFT %-22s config-only=%v token-only=%v\n", p.id, missing, extra)
	}
	return drift
}

func (a *authzRunner) report(reportPath string) error {
	fmt.Println("\n═══ summary ══════════════════════════════════════════════")
	sum := a.summary()
	names := make([]string, 0, len(sum))
	for k := range sum {
		names = append(names, k)
	}
	sort.Strings(names)
	totalFail := 0
	fmt.Printf("%-10s %8s %11s %12s %8s %6s\n", "LAYER", "PASS", "FAIL-OPEN", "FAIL-CLOSED", "REVIEW", "SKIP")
	for _, n := range names {
		c := sum[n]
		fmt.Printf("%-10s %8d %11d %12d %8d %6d\n", n, c.pass, c.failOpen, c.failClosed, c.review, c.skip)
		totalFail += c.failOpen + c.failClosed
	}

	a.printPersonaMatrix()

	openCount := a.printFindings(vFailOpen, "FAIL-OPEN — persona got through a gate that should have stopped it")
	closedCount := a.printFindings(vFailClosed, "FAIL-CLOSED — persona was blocked from something it is entitled to")
	reviewCount := a.printFindings(vReview, "REVIEW — needs a human call")

	if reportPath == "" {
		reportPath = "authz-report.json"
	}
	blob, err := json.MarshalIndent(struct {
		Results []checkResult `json:"results"`
	}{a.results}, "", " ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(reportPath, blob, 0o600); err != nil {
		return err
	}
	fmt.Printf("\nFull detail: %s (%d checks)\n", reportPath, len(a.results))

	if totalFail > 0 {
		return fmt.Errorf("authorization suite FAILED: %d fail-open, %d fail-closed, %d to review",
			openCount, closedCount, reviewCount)
	}
	fmt.Println("\nAll authorization expectations hold.")
	return nil
}

// printPersonaMatrix gives the per-persona bird's-eye view.
func (a *authzRunner) printPersonaMatrix() {
	type row struct {
		pass, fail, review int
	}
	rows := map[string]*row{}
	for _, r := range a.results {
		if r.Persona == "-" {
			continue
		}
		x, ok := rows[r.Persona]
		if !ok {
			x = &row{}
			rows[r.Persona] = x
		}
		switch r.Verdict {
		case vPass:
			x.pass++
		case vFailOpen, vFailClosed:
			x.fail++
		case vReview:
			x.review++
		}
	}
	keys := make([]string, 0, len(rows))
	for k := range rows {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	fmt.Printf("\n%-24s %8s %6s %8s\n", "PERSONA", "PASS", "FAIL", "REVIEW")
	for _, k := range keys {
		x := rows[k]
		fmt.Printf("%-24s %8d %6d %8d\n", k, x.pass, x.fail, x.review)
	}
}
