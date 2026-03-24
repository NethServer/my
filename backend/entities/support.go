/*
Copyright (C) 2026 Nethesis S.r.l.
SPDX-License-Identifier: AGPL-3.0-or-later
*/

package entities

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/models"
)

// SupportRepository handles support session database operations
type SupportRepository struct {
	db *sql.DB
}

// NewSupportRepository creates a new support repository
func NewSupportRepository() *SupportRepository {
	return &SupportRepository{db: database.DB}
}

// buildRBACFilter returns a WHERE condition and args for RBAC scope filtering.
// The condition filters systems by organization_id based on the user's org role.
func buildRBACFilter(userOrgRole, userOrgID string, argIdx int) (string, []interface{}, int) {
	switch strings.ToLower(userOrgRole) {
	case "owner":
		return "", nil, argIdx
	case "distributor":
		condition := fmt.Sprintf(`s.organization_id IN (
			SELECT $%d
			UNION
			SELECT logto_id FROM resellers
			WHERE custom_data->>'createdBy' = $%d AND deleted_at IS NULL
			UNION
			SELECT logto_id FROM customers
			WHERE deleted_at IS NULL AND (
				custom_data->>'createdBy' = $%d OR
				custom_data->>'createdBy' IN (
					SELECT logto_id FROM resellers
					WHERE custom_data->>'createdBy' = $%d AND deleted_at IS NULL
				)
			)
		)`, argIdx, argIdx, argIdx, argIdx)
		return condition, []interface{}{userOrgID}, argIdx + 1
	case "reseller":
		condition := fmt.Sprintf(`s.organization_id IN (
			SELECT $%d
			UNION
			SELECT logto_id FROM customers
			WHERE custom_data->>'createdBy' = $%d AND deleted_at IS NULL
		)`, argIdx, argIdx)
		return condition, []interface{}{userOrgID}, argIdx + 1
	case "customer":
		condition := fmt.Sprintf("s.organization_id = $%d", argIdx)
		return condition, []interface{}{userOrgID}, argIdx + 1
	default:
		// Unknown role: deny access
		return "1=0", nil, argIdx
	}
}

// GetSystemSessions returns support sessions grouped by system, with server-side
// pagination based on distinct systems (not individual sessions).
func (r *SupportRepository) GetSystemSessions(
	userOrgRole, userOrgID string,
	page, pageSize int,
	status, systemID string,
	sortBy, sortDirection string,
) ([]models.SystemSessionGroup, int, error) {
	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	// RBAC scope filter
	rbacCondition, rbacArgs, newArgIdx := buildRBACFilter(userOrgRole, userOrgID, argIdx)
	if rbacCondition != "" {
		conditions = append(conditions, rbacCondition)
		args = append(args, rbacArgs...)
		argIdx = newArgIdx
	}

	// Optional status filter: show systems that have at least one session with this status
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("ss.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if systemID != "" {
		conditions = append(conditions, fmt.Sprintf("ss.system_id = $%d", argIdx))
		args = append(args, systemID)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Count distinct systems
	countQuery := fmt.Sprintf(
		`SELECT COUNT(DISTINCT ss.system_id)
		 FROM support_sessions ss
		 JOIN systems s ON ss.system_id = s.id
		 WHERE %s`, whereClause)

	var totalCount int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count system groups: %w", err)
	}

	if totalCount == 0 {
		return nil, 0, nil
	}

	// Validate sort column (mapped to aggregate expressions)
	allowedSortColumns := map[string]string{
		"started_at": "MIN(ss.started_at)",
		"expires_at": "MAX(ss.expires_at)",
		"created_at": "MIN(ss.created_at)",
		"status": `CASE
			WHEN bool_or(ss.status = 'active') THEN 0
			WHEN bool_or(ss.status = 'pending') THEN 1
			WHEN bool_or(ss.status = 'expired') THEN 2
			ELSE 3
		END`,
	}
	sortColumn, ok := allowedSortColumns[sortBy]
	if !ok {
		sortColumn = "MIN(ss.created_at)"
	}
	if sortDirection != "asc" && sortDirection != "desc" {
		sortDirection = "desc"
	}

	// Get paginated system groups with aggregate data
	offset := (page - 1) * pageSize
	groupQuery := fmt.Sprintf(
		`SELECT
			ss.system_id,
			MIN(ss.started_at) AS started_at,
			MAX(ss.expires_at) AS expires_at,
			CASE
				WHEN bool_or(ss.status = 'active') THEN 'active'
				WHEN bool_or(ss.status = 'pending') THEN 'pending'
				WHEN bool_or(ss.status = 'expired') THEN 'expired'
				ELSE 'closed'
			END AS best_status,
			COUNT(*) AS session_count,
			COUNT(DISTINCT ss.node_id) FILTER (WHERE ss.node_id IS NOT NULL) AS node_count,
			s.name, s.type, s.system_key, s.organization_id,
			COALESCE(uo.name, '') AS org_name,
			COALESCE(uo.db_id, '') AS org_db_id,
			COALESCE(uo.org_type, '') AS org_type
		 FROM support_sessions ss
		 JOIN systems s ON ss.system_id = s.id
		 LEFT JOIN unified_organizations uo ON s.organization_id = uo.logto_id
		 WHERE %s
		 GROUP BY ss.system_id, s.name, s.type, s.system_key, s.organization_id, uo.name, uo.db_id, uo.org_type
		 ORDER BY %s %s
		 LIMIT $%d OFFSET $%d`,
		whereClause, sortColumn, sortDirection, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.Query(groupQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query system groups: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var groups []models.SystemSessionGroup
	var systemIDs []string
	systemMap := make(map[string]int) // system_id → index in groups

	for rows.Next() {
		var g models.SystemSessionGroup
		var systemType sql.NullString
		var orgID, orgName, orgDBID, orgType string

		err := rows.Scan(
			&g.SystemID, &g.StartedAt, &g.ExpiresAt, &g.Status,
			&g.SessionCount, &g.NodeCount,
			&g.SystemName, &systemType, &g.SystemKey,
			&orgID, &orgName, &orgDBID, &orgType,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan system group: %w", err)
		}

		if systemType.Valid {
			g.SystemType = &systemType.String
		}
		g.Organization = &models.Organization{
			LogtoID: orgID,
			ID:      orgDBID,
			Name:    orgName,
			Type:    orgType,
		}
		g.Sessions = []models.SessionRef{}

		systemMap[g.SystemID] = len(groups)
		systemIDs = append(systemIDs, g.SystemID)
		groups = append(groups, g)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, fmt.Errorf("failed to iterate system groups: %w", err)
	}

	if len(systemIDs) == 0 {
		return groups, totalCount, nil
	}

	// Fetch individual sessions for the returned systems
	placeholders := make([]string, len(systemIDs))
	sessionArgs := make([]interface{}, len(systemIDs))
	for i, id := range systemIDs {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		sessionArgs[i] = id
	}
	sessionQuery := fmt.Sprintf(
		`SELECT id, system_id, node_id, status, started_at, expires_at
		 FROM support_sessions
		 WHERE system_id IN (%s) AND status IN ('active', 'pending')
		 ORDER BY system_id, node_id NULLS FIRST, started_at DESC`,
		strings.Join(placeholders, ","))

	sessionRows, err := r.db.Query(sessionQuery, sessionArgs...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sessions for groups: %w", err)
	}
	defer func() { _ = sessionRows.Close() }()

	for sessionRows.Next() {
		var ref models.SessionRef
		var sysID string
		var nodeID sql.NullString

		err := sessionRows.Scan(&ref.ID, &sysID, &nodeID, &ref.Status, &ref.StartedAt, &ref.ExpiresAt)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan session ref: %w", err)
		}
		if nodeID.Valid {
			ref.NodeID = &nodeID.String
		}

		if idx, ok := systemMap[sysID]; ok {
			groups[idx].Sessions = append(groups[idx].Sessions, ref)
		}
	}

	return groups, totalCount, sessionRows.Err()
}

// GetSessions returns paginated support sessions filtered by RBAC scope
func (r *SupportRepository) GetSessions(
	userOrgRole, userOrgID string,
	page, pageSize int,
	status, systemID string,
	sortBy, sortDirection string,
) ([]models.SupportSession, int, error) {
	conditions := []string{"1=1"}
	args := []interface{}{}
	argIdx := 1

	// RBAC scope filter
	rbacCondition, rbacArgs, newArgIdx := buildRBACFilter(userOrgRole, userOrgID, argIdx)
	if rbacCondition != "" {
		conditions = append(conditions, rbacCondition)
		args = append(args, rbacArgs...)
		argIdx = newArgIdx
	}

	// Optional filters
	if status != "" {
		conditions = append(conditions, fmt.Sprintf("ss.status = $%d", argIdx))
		args = append(args, status)
		argIdx++
	}
	if systemID != "" {
		conditions = append(conditions, fmt.Sprintf("ss.system_id = $%d", argIdx))
		args = append(args, systemID)
		argIdx++
	}

	whereClause := strings.Join(conditions, " AND ")

	// Validate sort column
	allowedSortColumns := map[string]string{
		"started_at": "ss.started_at",
		"expires_at": "ss.expires_at",
		"status":     "ss.status",
		"created_at": "ss.created_at",
	}
	sortColumn, ok := allowedSortColumns[sortBy]
	if !ok {
		sortColumn = "ss.created_at"
	}
	if sortDirection != "asc" && sortDirection != "desc" {
		sortDirection = "desc"
	}

	// Count query
	countQuery := fmt.Sprintf(
		`SELECT COUNT(*)
		 FROM support_sessions ss
		 JOIN systems s ON ss.system_id = s.id
		 WHERE %s`, whereClause)

	var totalCount int
	err := r.db.QueryRow(countQuery, args...).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count sessions: %w", err)
	}

	// Data query
	offset := (page - 1) * pageSize
	dataQuery := fmt.Sprintf(
		`SELECT ss.id, ss.system_id, ss.node_id, ss.session_token, ss.started_at, ss.expires_at,
		        ss.status, ss.closed_at, ss.closed_by, ss.created_at, ss.updated_at,
		        s.name, s.type, s.system_key, s.organization_id,
		        COALESCE(uo.name, '') AS org_name,
		        COALESCE(uo.db_id, '') AS org_db_id,
		        COALESCE(uo.org_type, '') AS org_type
		 FROM support_sessions ss
		 JOIN systems s ON ss.system_id = s.id
		 LEFT JOIN unified_organizations uo ON s.organization_id = uo.logto_id
		 WHERE %s
		 ORDER BY %s %s
		 LIMIT $%d OFFSET $%d`,
		whereClause, sortColumn, sortDirection, argIdx, argIdx+1)

	args = append(args, pageSize, offset)

	rows, err := r.db.Query(dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query sessions: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var sessions []models.SupportSession
	for rows.Next() {
		session, err := scanSession(rows)
		if err != nil {
			return nil, 0, err
		}
		// Do not expose session_token in list
		session.SessionToken = ""
		sessions = append(sessions, session)
	}

	return sessions, totalCount, rows.Err()
}

// scannable is an interface for *sql.Row and *sql.Rows
type scannable interface {
	Scan(dest ...interface{}) error
}

// scanSession scans a session row into a SupportSession model
func scanSession(row scannable) (models.SupportSession, error) {
	var session models.SupportSession
	var nodeID sql.NullString
	var closedAt sql.NullTime
	var closedBy sql.NullString
	var systemType sql.NullString
	var orgID, orgName, orgDBID, orgType string

	err := row.Scan(
		&session.ID, &session.SystemID, &nodeID, &session.SessionToken,
		&session.StartedAt, &session.ExpiresAt,
		&session.Status, &closedAt, &closedBy,
		&session.CreatedAt, &session.UpdatedAt,
		&session.SystemName, &systemType, &session.SystemKey,
		&orgID, &orgName, &orgDBID, &orgType,
	)
	if err != nil {
		return session, fmt.Errorf("failed to scan session: %w", err)
	}

	if nodeID.Valid {
		session.NodeID = &nodeID.String
	}
	if closedAt.Valid {
		session.ClosedAt = &closedAt.Time
	}
	if closedBy.Valid {
		session.ClosedBy = &closedBy.String
	}
	if systemType.Valid {
		session.SystemType = &systemType.String
	}

	session.Organization = &models.Organization{
		LogtoID: orgID,
		ID:      orgDBID,
		Name:    orgName,
		Type:    orgType,
	}

	return session, nil
}

// GetSessionByID returns a single session with system info, filtered by RBAC scope
func (r *SupportRepository) GetSessionByID(sessionID, userOrgRole, userOrgID string) (*models.SupportSession, error) {
	conditions := []string{"ss.id = $1"}
	args := []interface{}{sessionID}
	argIdx := 2

	// RBAC scope filter
	rbacCondition, rbacArgs, _ := buildRBACFilter(userOrgRole, userOrgID, argIdx)
	if rbacCondition != "" {
		conditions = append(conditions, rbacCondition)
		args = append(args, rbacArgs...)
	}

	query := fmt.Sprintf(`SELECT ss.id, ss.system_id, ss.node_id, ss.session_token, ss.started_at, ss.expires_at,
	                 ss.status, ss.closed_at, ss.closed_by, ss.created_at, ss.updated_at,
	                 s.name, s.type, s.system_key, s.organization_id,
	                 COALESCE(uo.name, '') AS org_name,
	                 COALESCE(uo.db_id, '') AS org_db_id,
	                 COALESCE(uo.org_type, '') AS org_type
	          FROM support_sessions ss
	          JOIN systems s ON ss.system_id = s.id
	          LEFT JOIN unified_organizations uo ON s.organization_id = uo.logto_id
	          WHERE %s`, strings.Join(conditions, " AND "))

	session, err := scanSession(r.db.QueryRow(query, args...))
	if err != nil {
		if err.Error() == "failed to scan session: sql: no rows in result set" {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get session: %w", err)
	}

	// Do not expose session_token in API
	session.SessionToken = ""

	return &session, nil
}

// GetDiagnostics returns the diagnostics data for a session, if available and accessible.
// Returns nil, nil, nil if diagnostics have not been received yet.
func (r *SupportRepository) GetDiagnostics(sessionID, userOrgRole, userOrgID string) (map[string]interface{}, *time.Time, error) {
	conditions := []string{"ss.id = $1"}
	args := []interface{}{sessionID}
	argIdx := 2

	// RBAC scope filter
	rbacCondition, rbacArgs, _ := buildRBACFilter(userOrgRole, userOrgID, argIdx)
	if rbacCondition != "" {
		conditions = append(conditions, rbacCondition)
		args = append(args, rbacArgs...)
	}

	query := fmt.Sprintf(`SELECT ss.diagnostics, ss.diagnostics_at
	          FROM support_sessions ss
	          JOIN systems s ON ss.system_id = s.id
	          WHERE %s`, strings.Join(conditions, " AND "))

	var rawDiagnostics []byte
	var diagnosticsAt sql.NullTime

	err := r.db.QueryRow(query, args...).Scan(&rawDiagnostics, &diagnosticsAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get diagnostics: %w", err)
	}

	if rawDiagnostics == nil {
		return nil, nil, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rawDiagnostics, &data); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal diagnostics: %w", err)
	}

	var at *time.Time
	if diagnosticsAt.Valid {
		t := diagnosticsAt.Time
		at = &t
	}

	return data, at, nil
}

// GetUsers returns the ephemeral support users for a session, if available and accessible.
// Returns nil, nil, nil if users have not been provisioned yet.
func (r *SupportRepository) GetUsers(sessionID, userOrgRole, userOrgID string) (map[string]interface{}, *time.Time, error) {
	conditions := []string{"ss.id = $1"}
	args := []interface{}{sessionID}
	argIdx := 2

	// RBAC scope filter
	rbacCondition, rbacArgs, _ := buildRBACFilter(userOrgRole, userOrgID, argIdx)
	if rbacCondition != "" {
		conditions = append(conditions, rbacCondition)
		args = append(args, rbacArgs...)
	}

	query := fmt.Sprintf(`SELECT ss.users, ss.users_at
	          FROM support_sessions ss
	          JOIN systems s ON ss.system_id = s.id
	          WHERE %s`, strings.Join(conditions, " AND "))

	var rawUsers []byte
	var usersAt sql.NullTime

	err := r.db.QueryRow(query, args...).Scan(&rawUsers, &usersAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil, nil
		}
		return nil, nil, fmt.Errorf("failed to get users: %w", err)
	}

	if rawUsers == nil {
		return nil, nil, nil
	}

	var data map[string]interface{}
	if err := json.Unmarshal(rawUsers, &data); err != nil {
		return nil, nil, fmt.Errorf("failed to unmarshal users: %w", err)
	}

	var at *time.Time
	if usersAt.Valid {
		t := usersAt.Time
		at = &t
	}

	return data, at, nil
}

// statusSeverity maps a diagnostic status to a numeric severity for comparison.
// Higher values indicate worse status.
var statusSeverity = map[string]int{
	"ok":       0,
	"warning":  1,
	"critical": 2,
	"error":    3,
	"timeout":  4,
}

// worstStatus returns the most severe status among the given statuses.
func worstStatus(statuses []string) string {
	worst := "ok"
	for _, s := range statuses {
		if statusSeverity[s] > statusSeverity[worst] {
			worst = s
		}
	}
	return worst
}

// GetSystemDiagnostics returns diagnostics for all active sessions of a system,
// grouped by node, with an overall status reflecting the worst across all nodes.
func (r *SupportRepository) GetSystemDiagnostics(systemID, userOrgRole, userOrgID string) (*models.SystemDiagnostics, error) {
	conditions := []string{"ss.system_id = $1", "ss.status IN ('active', 'pending')"}
	args := []interface{}{systemID}
	argIdx := 2

	// RBAC scope filter
	rbacCondition, rbacArgs, _ := buildRBACFilter(userOrgRole, userOrgID, argIdx)
	if rbacCondition != "" {
		conditions = append(conditions, rbacCondition)
		args = append(args, rbacArgs...)
	}

	query := fmt.Sprintf(`SELECT ss.id, ss.node_id, ss.diagnostics, ss.diagnostics_at
		FROM support_sessions ss
		JOIN systems s ON ss.system_id = s.id
		WHERE %s
		ORDER BY ss.node_id NULLS FIRST, ss.started_at DESC`,
		strings.Join(conditions, " AND "))

	rows, err := r.db.Query(query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query system diagnostics: %w", err)
	}
	defer func() { _ = rows.Close() }()

	result := &models.SystemDiagnostics{
		SystemID: systemID,
		Nodes:    []models.NodeDiagnostics{},
	}

	var statuses []string
	for rows.Next() {
		var nd models.NodeDiagnostics
		var nodeID sql.NullString
		var rawDiagnostics []byte
		var diagnosticsAt sql.NullTime

		if err := rows.Scan(&nd.SessionID, &nodeID, &rawDiagnostics, &diagnosticsAt); err != nil {
			return nil, fmt.Errorf("failed to scan system diagnostics: %w", err)
		}

		if nodeID.Valid {
			nd.NodeID = &nodeID.String
		}
		if diagnosticsAt.Valid {
			t := diagnosticsAt.Time
			nd.DiagnosticsAt = &t
		}
		if rawDiagnostics != nil {
			var data map[string]interface{}
			if err := json.Unmarshal(rawDiagnostics, &data); err != nil {
				return nil, fmt.Errorf("failed to unmarshal diagnostics: %w", err)
			}
			nd.Diagnostics = data
			if os, ok := data["overall_status"].(string); ok {
				statuses = append(statuses, os)
			}
		}

		result.Nodes = append(result.Nodes, nd)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("failed to iterate system diagnostics: %w", err)
	}

	if len(statuses) > 0 {
		result.OverallStatus = worstStatus(statuses)
	}

	return result, nil
}

// maxSessionDuration is the maximum total duration a session can have from its start time (30 days)
const maxSessionDuration = 30 * 24 // hours

// ExtendSession extends the expiration of a session atomically.
// Rejects extensions that would push the total session duration beyond 30 days.
func (r *SupportRepository) ExtendSession(sessionID string, hours int) error {
	result, err := r.db.Exec(
		`UPDATE support_sessions
		 SET expires_at = expires_at + $2 * INTERVAL '1 hour', updated_at = NOW()
		 WHERE id = $1 AND status IN ('pending', 'active')
		   AND (expires_at + $2 * INTERVAL '1 hour') - started_at <= $3 * INTERVAL '1 hour'`,
		sessionID, hours, maxSessionDuration,
	)
	if err != nil {
		return fmt.Errorf("failed to extend session: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		// Distinguish between "not found" and "would exceed max duration"
		var exists bool
		_ = r.db.QueryRow(
			`SELECT EXISTS(SELECT 1 FROM support_sessions WHERE id = $1 AND status IN ('pending', 'active'))`,
			sessionID,
		).Scan(&exists)
		if exists {
			return fmt.Errorf("extension would exceed maximum session duration of %d days", maxSessionDuration/24)
		}
		return fmt.Errorf("session not found or not extendable")
	}
	return nil
}

// CloseSession force-closes a session
func (r *SupportRepository) CloseSession(sessionID string) error {
	result, err := r.db.Exec(
		`UPDATE support_sessions
		 SET status = 'closed', closed_at = NOW(), closed_by = 'operator', updated_at = NOW()
		 WHERE id = $1 AND status IN ('pending', 'active')`,
		sessionID,
	)
	if err != nil {
		return fmt.Errorf("failed to close session: %w", err)
	}

	rows, _ := result.RowsAffected()
	if rows == 0 {
		return fmt.Errorf("session not found or already closed")
	}
	return nil
}

// InsertAccessLog inserts a new access log entry and returns its ID
func (r *SupportRepository) InsertAccessLog(sessionID, operatorID, operatorName, accessType, metadata string) (string, error) {
	var logID string
	err := r.db.QueryRow(
		`INSERT INTO support_access_logs (session_id, operator_id, operator_name, access_type, connected_at, metadata)
		 VALUES ($1, $2, $3, $4, NOW(), $5) RETURNING id`,
		sessionID, operatorID, operatorName, accessType, metadata,
	).Scan(&logID)
	if err != nil {
		return "", fmt.Errorf("failed to insert access log: %w", err)
	}
	return logID, nil
}

// DisconnectAccessLog sets disconnected_at on an access log entry
func (r *SupportRepository) DisconnectAccessLog(logID string) error {
	_, err := r.db.Exec(
		`UPDATE support_access_logs SET disconnected_at = NOW() WHERE id = $1 AND disconnected_at IS NULL`,
		logID,
	)
	if err != nil {
		return fmt.Errorf("failed to update access log disconnect: %w", err)
	}
	return nil
}

// GetSessionTokenByID returns the session_token for internal service communication.
// Unlike GetSessionByID, this does NOT strip the token.
func (r *SupportRepository) GetSessionTokenByID(sessionID string) (string, error) {
	var token string
	err := r.db.QueryRow(
		`SELECT session_token FROM support_sessions WHERE id = $1 AND status IN ('pending', 'active')`,
		sessionID,
	).Scan(&token)
	if err != nil {
		return "", fmt.Errorf("session not found or not active: %w", err)
	}
	return token, nil
}

// GetAccessLogs returns access logs for a session
func (r *SupportRepository) GetAccessLogs(sessionID string, page, pageSize int) ([]models.SupportAccessLog, int, error) {
	var totalCount int
	err := r.db.QueryRow(
		`SELECT COUNT(*) FROM support_access_logs WHERE session_id = $1`,
		sessionID,
	).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count access logs: %w", err)
	}

	offset := (page - 1) * pageSize
	rows, err := r.db.Query(
		`SELECT id, session_id, operator_id, operator_name, access_type,
		        connected_at, disconnected_at, metadata
		 FROM support_access_logs
		 WHERE session_id = $1
		 ORDER BY connected_at DESC
		 LIMIT $2 OFFSET $3`,
		sessionID, pageSize, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to query access logs: %w", err)
	}
	defer func() { _ = rows.Close() }()

	var logs []models.SupportAccessLog
	for rows.Next() {
		var log models.SupportAccessLog
		var operatorName sql.NullString
		var disconnectedAt sql.NullTime
		var metadata sql.NullString

		err := rows.Scan(
			&log.ID, &log.SessionID, &log.OperatorID, &operatorName,
			&log.AccessType, &log.ConnectedAt, &disconnectedAt, &metadata,
		)
		if err != nil {
			return nil, 0, fmt.Errorf("failed to scan access log: %w", err)
		}

		if operatorName.Valid {
			log.OperatorName = &operatorName.String
		}
		if disconnectedAt.Valid {
			log.DisconnectedAt = &disconnectedAt.Time
		}
		if metadata.Valid {
			log.Metadata = &metadata.String
		}

		logs = append(logs, log)
	}

	return logs, totalCount, rows.Err()
}
