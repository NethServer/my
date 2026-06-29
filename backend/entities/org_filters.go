/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package entities

import (
	"fmt"
	"regexp"
	"strings"
)

// createdByIDPattern matches the Logto ID charset used for user and organization
// IDs, so matching values can be inlined into SQL without risk of injection.
var createdByIDPattern = regexp.MustCompile(`^[A-Za-z0-9_-]{1,64}$`)

// createdByFilterClause builds a SQL fragment restricting organizations to those
// whose creator snapshot (custom_data.createdByUser) matches any of the given
// user or organization logto IDs, mirroring the systems created_by filter which
// matches either the creating user or their organization. Values are strictly
// validated against the Logto ID charset so they can be inlined safely; invalid
// or duplicate values are ignored. Returns "" when no usable value is provided.
//
// The fragment uses a bare custom_data reference so it works in both the
// single-table COUNT query and the aliased list query.
func createdByFilterClause(createdBy []string) string {
	seen := make(map[string]bool)
	quoted := make([]string, 0, len(createdBy))
	for _, v := range createdBy {
		if v == "" || seen[v] || !createdByIDPattern.MatchString(v) {
			continue
		}
		seen[v] = true
		quoted = append(quoted, "'"+v+"'")
	}
	if len(quoted) == 0 {
		return ""
	}
	list := strings.Join(quoted, ", ")
	return fmt.Sprintf(" AND (custom_data->'createdByUser'->>'user_id' IN (%s) OR custom_data->'createdByUser'->>'organization_id' IN (%s))", list, list)
}
