/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package local

import (
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"

	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/models"
)

const (
	apiKeyDefaultTTLDays = 90
	apiKeyMaxTTLDays     = 365
	// APIKeyMaxPerUser is the maximum number of active keys a user may hold. It
	// is surfaced in the limit-reached error so the UI can show the exact number.
	APIKeyMaxPerUser = 5
)

// Sentinel errors returned by AuthenticateAPIKey. The middleware maps all of
// them to a single opaque 401 so a caller cannot tell why a key was rejected.
var (
	ErrAPIKeyInvalid      = errors.New("invalid api key")
	ErrAPIKeyRevoked      = errors.New("api key has been revoked")
	ErrAPIKeyExpired      = errors.New("api key has expired")
	ErrAPIKeyUserInactive = errors.New("api key owner is not active")

	// ErrAPIKeyNoLocalUser is returned when the caller has no row in the local
	// users table (e.g. the Nethesis owner super-admin). Keys are anchored to a
	// users row so the live suspend check can resolve the owner, so such an
	// account cannot hold keys.
	ErrAPIKeyNoLocalUser = errors.New("api keys are not available for this account")

	// ErrAPIKeyLimitReached is returned when the user already holds the maximum
	// number of active keys.
	ErrAPIKeyLimitReached = errors.New("api key limit reached")
)

// apiKeyDeniedPermissions are stripped from every key regardless of mode. The
// verb mask already excludes non read:/manage: permissions; this is a second,
// explicit layer for sensitive permissions that must never reach an integration
// (impersonation, alerting config visibility, and any destructive action).
var apiKeyDeniedPermissions = map[string]bool{
	"impersonate:users":    true,
	"config:alerts":        true,
	"destroy:users":        true,
	"destroy:systems":      true,
	"destroy:customers":    true,
	"destroy:resellers":    true,
	"destroy:distributors": true,
}

// APIKeysService owns the lifecycle and authentication of personal API keys.
type APIKeysService struct{}

func NewAPIKeysService() *APIKeysService {
	return &APIKeysService{}
}

// APIKeyAuthResult is the outcome of a key authentication. On success User holds
// the resolved owner (permissions masked to the key's mode). The attribution
// fields are filled whenever the key row is found — even on failure — so the
// caller can write an audit entry.
type APIKeyAuthResult struct {
	User           *models.User
	KeyID          string
	UserID         string
	OrganizationID string
	Name           string
	Mode           string
}

// CreateAPIKey mints a new key for the user and returns the full plaintext
// token exactly once. The caller must surface it and never persist it.
func (s *APIKeysService) CreateAPIKey(userID, organizationID, name, mode string, expiresInDays int) (*models.APIKey, string, error) {
	if mode != "read" && mode != "write" {
		return nil, "", fmt.Errorf("invalid mode: must be read or write")
	}

	exists, err := s.userExists(userID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to verify user: %w", err)
	}
	if !exists {
		return nil, "", ErrAPIKeyNoLocalUser
	}

	if expiresInDays <= 0 {
		expiresInDays = apiKeyDefaultTTLDays
	}
	if expiresInDays > apiKeyMaxTTLDays {
		expiresInDays = apiKeyMaxTTLDays
	}

	fullToken, public, secret, err := helpers.GenerateAPIKey()
	if err != nil {
		return nil, "", err
	}
	secretHash, err := helpers.HashSystemSecretSHA256(secret)
	if err != nil {
		return nil, "", err
	}

	key := &models.APIKey{
		ID:             uuid.New().String(),
		UserID:         userID,
		OrganizationID: organizationID,
		Name:           name,
		KeyPublic:      public,
		Mode:           mode,
		ExpiresAt:      time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour),
		CreatedAt:      time.Now(),
	}

	// Enforce the per-user limit and insert atomically. Locking the owner row
	// serializes concurrent creates for the same user, so the count-then-insert
	// cannot race past the cap.
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	if _, err := tx.Exec(`SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, userID); err != nil {
		return nil, "", fmt.Errorf("failed to lock user: %w", err)
	}

	var active int
	if err := tx.QueryRow(`
		SELECT COUNT(*) FROM user_api_keys
		WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
	`, userID).Scan(&active); err != nil {
		return nil, "", fmt.Errorf("failed to count active api keys: %w", err)
	}
	if active >= APIKeyMaxPerUser {
		return nil, "", ErrAPIKeyLimitReached
	}

	_, err = tx.Exec(`
		INSERT INTO user_api_keys (id, user_id, organization_id, name, key_public, key_secret_sha256, mode, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`,
		key.ID, key.UserID, nullString(organizationID), key.Name,
		key.KeyPublic, secretHash, key.Mode, key.ExpiresAt, key.CreatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create api key: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("failed to commit api key: %w", err)
	}

	return key, fullToken, nil
}

// ListAPIKeys returns all keys owned by the user, newest first, without secrets.
func (s *APIKeysService) ListAPIKeys(userID string) ([]models.APIKey, error) {
	rows, err := database.DB.Query(`
		SELECT id, user_id, COALESCE(organization_id, ''), name, key_public, mode,
		       expires_at, last_used_at, last_used_ip, revoked_at, created_at
		FROM user_api_keys
		WHERE user_id = $1
		ORDER BY created_at DESC
	`, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to list api keys: %w", err)
	}
	defer func() { _ = rows.Close() }()

	keys := make([]models.APIKey, 0)
	for rows.Next() {
		var k models.APIKey
		var lastUsedAt, revokedAt sql.NullTime
		var lastUsedIP sql.NullString
		if err := rows.Scan(&k.ID, &k.UserID, &k.OrganizationID, &k.Name, &k.KeyPublic, &k.Mode,
			&k.ExpiresAt, &lastUsedAt, &lastUsedIP, &revokedAt, &k.CreatedAt); err != nil {
			return nil, fmt.Errorf("failed to scan api key: %w", err)
		}
		if lastUsedAt.Valid {
			k.LastUsedAt = &lastUsedAt.Time
		}
		if lastUsedIP.Valid {
			k.LastUsedIP = &lastUsedIP.String
		}
		if revokedAt.Valid {
			k.RevokedAt = &revokedAt.Time
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

// RevokeAPIKey marks a key revoked. It only touches keys owned by userID, so a
// user can never revoke someone else's key. Returns sql.ErrNoRows when the key
// does not exist, is not owned by the user, or is already revoked.
func (s *APIKeysService) RevokeAPIKey(userID, keyID string) error {
	res, err := database.DB.Exec(`
		UPDATE user_api_keys
		SET revoked_at = NOW()
		WHERE id = $1 AND user_id = $2 AND revoked_at IS NULL
	`, keyID, userID)
	if err != nil {
		return fmt.Errorf("failed to revoke api key: %w", err)
	}
	n, err := res.RowsAffected()
	if err != nil {
		return err
	}
	if n == 0 {
		return sql.ErrNoRows
	}
	return nil
}

// AuthenticateAPIKey validates a plaintext token and returns the resolved owner
// with permissions masked to the key's mode. The owner's active state is checked
// authoritatively against the DB (suspended_at / deleted_at), so suspending the
// owner kills every key at once and reactivating restores them.
func (s *APIKeysService) AuthenticateAPIKey(token string) (*APIKeyAuthResult, error) {
	public, secret, err := helpers.ParseAPIKey(token)
	if err != nil {
		return nil, ErrAPIKeyInvalid
	}

	var keyID, userLocalID, mode, name, secretHash string
	var logtoID, orgID sql.NullString
	var expiresAt time.Time
	var revokedAt, suspendedAt, deletedAt sql.NullTime

	err = database.DB.QueryRow(`
		SELECT k.id, k.key_secret_sha256, k.mode, k.name, k.organization_id, k.expires_at, k.revoked_at,
		       u.id, u.logto_id, u.suspended_at, u.deleted_at
		FROM user_api_keys k
		JOIN users u ON u.id = k.user_id
		WHERE k.key_public = $1
	`, public).Scan(&keyID, &secretHash, &mode, &name, &orgID, &expiresAt, &revokedAt,
		&userLocalID, &logtoID, &suspendedAt, &deletedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAPIKeyInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to look up api key: %w", err)
	}

	// Attribution carried back even on failure so the caller can audit it.
	res := &APIKeyAuthResult{
		KeyID:          keyID,
		UserID:         userLocalID,
		OrganizationID: orgID.String,
		Name:           name,
		Mode:           mode,
	}

	valid, err := helpers.VerifySystemSecretSHA256(secret, secretHash)
	if err != nil || !valid {
		return res, ErrAPIKeyInvalid
	}
	if revokedAt.Valid {
		return res, ErrAPIKeyRevoked
	}
	if time.Now().After(expiresAt) {
		return res, ErrAPIKeyExpired
	}
	if suspendedAt.Valid || deletedAt.Valid {
		return res, ErrAPIKeyUserInactive
	}
	if !logtoID.Valid || logtoID.String == "" {
		return res, ErrAPIKeyUserInactive
	}

	user, err := ResolveUserByLogtoID(logtoID.String)
	if err != nil {
		return nil, err
	}
	user.ID = userLocalID
	user.UserPermissions = maskAPIKeyPermissions(user.UserPermissions, mode)
	user.OrgPermissions = maskAPIKeyPermissions(user.OrgPermissions, mode)
	res.User = user

	return res, nil
}

// TouchLastUsed records the key's last usage. It is throttled to at most one
// write per minute per key, so a busy integration does not write on every call.
// Best-effort: errors are ignored.
func (s *APIKeysService) TouchLastUsed(keyID, ip string) {
	_, _ = database.DB.Exec(`
		UPDATE user_api_keys
		SET last_used_at = NOW(), last_used_ip = $2
		WHERE id = $1 AND (last_used_at IS NULL OR last_used_at < NOW() - INTERVAL '1 minute')
	`, keyID, nullString(ip))
}

func (s *APIKeysService) userExists(userID string) (bool, error) {
	if userID == "" {
		return false, nil
	}
	var exists bool
	err := database.DB.QueryRow(
		`SELECT EXISTS(SELECT 1 FROM users WHERE id = $1 AND deleted_at IS NULL)`,
		userID,
	).Scan(&exists)
	return exists, err
}

// maskAPIKeyPermissions keeps only the permissions a key of the given mode may
// use: read:* always, manage:* additionally for write. Anything in the denylist
// is dropped first, so destructive/sensitive permissions never pass.
func maskAPIKeyPermissions(perms []string, mode string) []string {
	out := make([]string, 0, len(perms))
	for _, p := range perms {
		if apiKeyDeniedPermissions[p] {
			continue
		}
		if strings.HasPrefix(p, "read:") {
			out = append(out, p)
			continue
		}
		if mode == "write" && strings.HasPrefix(p, "manage:") {
			out = append(out, p)
		}
	}
	return out
}

func nullString(s string) sql.NullString {
	return sql.NullString{String: s, Valid: s != ""}
}
