/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/lib/pq"

	"github.com/nethesis/my/backend/database"
)

// LocalLegacySystemsByOrgRepository serves the per-organization count of
// systems that still live on the old my (legacy.my.nethesis.it): NethServer 7
// and earlier, plus anything the import never carried over.
//
// The rows are PUSHED by the proxy_sync cron running on the legacy host — this
// process never opens a connection to the old MySQL. A legacy outage therefore
// makes the number stale, never the dashboard slow or broken, and the whole
// feature disappears by dropping the table at decommission.
//
// Counts sit on the reseller/distributor organization the legacy VAT maps to.
// They are deliberately not attributable to a customer: most of these systems
// belong to clients that were never created here.
type LocalLegacySystemsByOrgRepository struct {
	db *sql.DB
}

// NewLocalLegacySystemsByOrgRepository creates a new repository instance.
func NewLocalLegacySystemsByOrgRepository() *LocalLegacySystemsByOrgRepository {
	return &LocalLegacySystemsByOrgRepository{db: database.DB}
}

// legacySystemsCount builds the SQL expression summing the legacy systems of
// the organizations selected by orgSet (any expression valid inside IN (...):
// a column, a placeholder or a subquery), for the org-list and org-stats
// queries that report the count alongside their own.
//
// The table holds one pre-aggregated row per organization keyed by primary key,
// so this stays an index probe even when evaluated per row. Passing a single
// reseller rather than its subtree is not a shortcut: legacy counts are keyed
// on the reseller organization the legacy VAT maps to, never on a customer.
func legacySystemsCount(orgSet string) string {
	return "(SELECT COALESCE(SUM(l.total), 0) FROM legacy_systems_by_org l WHERE l.organization_id IN (" + orgSet + "))"
}

// LegacySystemsSum is the aggregated answer for one scope. OldestUpdate is the
// MIN(updated_at) across the queried rows, so a caller can tell a genuine zero
// from a sync that stopped running. It is the zero time when no rows matched.
type LegacySystemsSum struct {
	Total        int
	OldestUpdate time.Time
}

// SumByOrgIDs returns the legacy systems count summed across the given
// organization IDs. Callers pass their whole RBAC scope, which is what makes a
// distributor pick up its resellers without any hierarchy logic here. An empty
// input returns a zero-valued struct with no error, and organizations the sync
// has never pushed are simply absent and contribute zero.
func (r *LocalLegacySystemsByOrgRepository) SumByOrgIDs(orgIDs []string) (LegacySystemsSum, error) {
	var out LegacySystemsSum
	if len(orgIDs) == 0 {
		return out, nil
	}

	var oldest sql.NullTime
	err := r.db.QueryRow(`
		SELECT
			COALESCE(SUM(total), 0),
			MIN(updated_at)
		FROM legacy_systems_by_org
		WHERE organization_id = ANY($1)
	`, pq.Array(orgIDs)).Scan(&out.Total, &oldest)
	if err != nil {
		return LegacySystemsSum{}, fmt.Errorf("sum legacy_systems_by_org: %w", err)
	}
	if oldest.Valid {
		out.OldestUpdate = oldest.Time
	}
	return out, nil
}

// SumAll returns the legacy systems count across every organization. It backs
// the owner view, where the service passes a nil scope meaning "no RBAC filter"
// rather than an explicit list of organizations.
func (r *LocalLegacySystemsByOrgRepository) SumAll() (LegacySystemsSum, error) {
	var out LegacySystemsSum
	var oldest sql.NullTime
	err := r.db.QueryRow(`
		SELECT COALESCE(SUM(total), 0), MIN(updated_at) FROM legacy_systems_by_org
	`).Scan(&out.Total, &oldest)
	if err != nil {
		return LegacySystemsSum{}, fmt.Errorf("sum all legacy_systems_by_org: %w", err)
	}
	if oldest.Valid {
		out.OldestUpdate = oldest.Time
	}
	return out, nil
}

// ReplaceAll swaps the whole table for the counts of one sync run, inside a
// transaction. The legacy side always sends the complete picture, so replacing
// is what keeps the table honest: an organization that drops to zero legacy
// systems — every machine migrated, or the reseller retired — disappears from
// the payload, and an upsert-only write would leave its last non-zero count
// there for good. Organizations whose VAT matches nothing here are simply not
// in the payload.
func (r *LocalLegacySystemsByOrgRepository) ReplaceAll(counts map[string]int) error {
	tx, err := r.db.Begin()
	if err != nil {
		return fmt.Errorf("begin legacy_systems_by_org replace: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`DELETE FROM legacy_systems_by_org`); err != nil {
		return fmt.Errorf("clear legacy_systems_by_org: %w", err)
	}

	stmt, err := tx.Prepare(`
		INSERT INTO legacy_systems_by_org (organization_id, total, updated_at)
		VALUES ($1, $2, NOW())
	`)
	if err != nil {
		return fmt.Errorf("prepare legacy_systems_by_org insert: %w", err)
	}
	defer func() { _ = stmt.Close() }()

	for orgID, total := range counts {
		if orgID == "" {
			continue
		}
		if _, err := stmt.Exec(orgID, total); err != nil {
			return fmt.Errorf("insert legacy count for %s: %w", orgID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit legacy_systems_by_org replace: %w", err)
	}
	return nil
}
