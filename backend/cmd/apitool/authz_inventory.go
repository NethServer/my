/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// The inventory exists so a new endpoint cannot be added without an
// authorization intent: `apitool authz coverage` fails when main.go registers a
// route that routes.yml does not mention.
//
// It deliberately reads ONLY method and path. The middleware chain is not
// interpreted anywhere in this suite: expectations are hand-authored, so that a
// wrong middleware shows up as a failing check instead of being copied into the
// spec as if it were the intent.

type inventoryRoute struct {
	Method string
	Path   string
	Line   int
}

var (
	groupRe = regexp.MustCompile(`^\s*(\w+)\s*:?=\s*(\w+)\.Group\("([^"]*)"`)
	routeRe = regexp.MustCompile(`^\s*(\w+)\.(GET|POST|PUT|PATCH|DELETE|HEAD|OPTIONS)\("([^"]*)"`)
)

// routeInventory parses gin route registrations out of backend/main.go,
// resolving group prefixes so paths are absolute.
func routeInventory(mainGoOverride string) ([]inventoryRoute, error) {
	path, err := findMainGo(mainGoOverride)
	if err != nil {
		return nil, err
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer func() { _ = f.Close() }()

	prefixes := map[string]string{"router": ""}
	var routes []inventoryRoute

	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 0, 64*1024), 1024*1024)
	line := 0
	for sc.Scan() {
		line++
		text := sc.Text()

		if m := groupRe.FindStringSubmatch(text); m != nil {
			varName, parent, prefix := m[1], m[2], m[3]
			base, ok := prefixes[parent]
			if !ok {
				continue // not a gin group we track
			}
			prefixes[varName] = joinRoutePath(base, prefix)
			continue
		}
		if m := routeRe.FindStringSubmatch(text); m != nil {
			varName, method, p := m[1], m[2], m[3]
			base, ok := prefixes[varName]
			if !ok {
				continue
			}
			routes = append(routes, inventoryRoute{Method: method, Path: joinRoutePath(base, p), Line: line})
		}
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	if len(routes) == 0 {
		return nil, fmt.Errorf("no routes found in %s — did the registration style change?", path)
	}
	return routes, nil
}

// joinRoutePath mirrors how gin normalizes group prefixes: duplicate slashes
// collapse and a trailing slash is dropped (groups are declared as Group("/")).
func joinRoutePath(base, part string) string {
	full := base + part
	for strings.Contains(full, "//") {
		full = strings.ReplaceAll(full, "//", "/")
	}
	if len(full) > 1 {
		full = strings.TrimSuffix(full, "/")
	}
	return full
}

func findMainGo(override string) (string, error) {
	cands := []string{override, "main.go", filepath.Join("backend", "main.go")}
	for _, c := range cands {
		if c == "" {
			continue
		}
		if _, err := os.Stat(c); err == nil {
			return c, nil
		}
	}
	return "", fmt.Errorf("main.go not found (run from backend/ or repo root, or pass --main-go=<path>)")
}
