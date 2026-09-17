/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package models

import "testing"

// Anything that is not an explicit opt-in must resolve to no counters: the
// default is what the integrations and the exports get, and it has to be the
// cheap one.
func TestParseCountsMode(t *testing.T) {
	cases := map[string]CountsMode{
		"":      CountsNone,
		"false": CountsNone,
		"1":     CountsNone,
		"basic": CountsNone,
		"True":  CountsNone,
		"ALL":   CountsNone,
		"true":  CountsBasic,
		"all":   CountsAll,
	}

	for in, want := range cases {
		if got := ParseCountsMode(in); got != want {
			t.Errorf("ParseCountsMode(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCountsModePredicates(t *testing.T) {
	if CountsNone.WantsCounters() || CountsNone.WantsApplications() {
		t.Error("CountsNone must want nothing")
	}
	if !CountsBasic.WantsCounters() || CountsBasic.WantsApplications() {
		t.Error("CountsBasic must want counters but not applications")
	}
	if !CountsAll.WantsCounters() || !CountsAll.WantsApplications() {
		t.Error("CountsAll must want everything")
	}
}
