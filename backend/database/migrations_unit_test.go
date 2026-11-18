/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package database

import (
	"os"
	"regexp"
	"strings"
	"testing"
)

var migrationFile = regexp.MustCompile(`^(\d{3})_(.+?)(_rollback)?\.sql$`)

func TestMigrationFiles(t *testing.T) {
	entries, err := os.ReadDir("migrations")
	if err != nil {
		t.Fatalf("read migrations: %v", err)
	}

	forward := map[string]string{}
	rollback := map[string]string{}
	for _, entry := range entries {
		name := entry.Name()
		if !strings.HasSuffix(name, ".sql") {
			continue
		}
		m := migrationFile.FindStringSubmatch(name)
		if m == nil {
			t.Errorf("%s: name must be NNN_description.sql or NNN_description_rollback.sql", name)
			continue
		}
		number, description := m[1], m[2]
		files := forward
		if m[3] != "" {
			files = rollback
		}
		if other, taken := files[number]; taken {
			t.Errorf("migration number %s used twice: %s and %s", number, other, name)
			continue
		}
		files[number] = description
	}

	for number, description := range forward {
		if rollback[number] != description {
			t.Errorf("migration %s_%s.sql has no %s_%s_rollback.sql", number, description, number, description)
		}
	}
	for number, description := range rollback {
		if _, ok := forward[number]; !ok {
			t.Errorf("rollback %s_%s_rollback.sql has no forward migration", number, description)
		}
	}
}
