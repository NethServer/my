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

	"github.com/nethesis/my/backend/cache"
	"github.com/nethesis/my/backend/database"
	"github.com/nethesis/my/backend/helpers"
	"github.com/nethesis/my/backend/models"
	"github.com/nethesis/my/backend/services/logto"
)

const (
	apiKeyDefaultTTLDays = 90
	apiKeyMaxTTLDays     = 365
	// APIKeyMaxPerUser is the maximum number of active keys a user may hold. It
	// is surfaced in the limit-reached error so the UI can show the exact number.
	APIKeyMaxPerUser = 5

	// ownerSuspendedCacheTTL bounds how long a Logto suspension can take to
	// propagate to owner keys. Kept short because the owner account is managed
	// directly in Logto (no local users row), so no lifecycle hook invalidates
	// this cache — TTL expiry is the only propagation path. Revocation
	// (revoked_at) is always instant and is the intended emergency kill.
	ownerSuspendedCacheTTL = 1 * time.Minute
)

// Sentinel errors returned by AuthenticateAPIKey. The middleware maps all of
// them to a single opaque 401 so a caller cannot tell why a key was rejected.
var (
	ErrAPIKeyInvalid      = errors.New("invalid api key")
	ErrAPIKeyRevoked      = errors.New("api key has been revoked")
	ErrAPIKeyExpired      = errors.New("api key has expired")
	ErrAPIKeyUserInactive = errors.New("api key owner is not active")

	// ErrAPIKeyNoLocalUser is returned when the caller has no row in the local
	// users table and is not an owner account. Regular keys are anchored to a
	// users row so the live suspend check can resolve the owner; owner accounts
	// have no local row by design and anchor on the Logto ID instead (myo_).
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

// isOwnerAccount reports whether the user is an owner-account caller: keys for
// these are anchored on the Logto ID (myo_ prefix) because the Owner
// organization has no local users rows. Requires a Logto ID to anchor on.
func isOwnerAccount(user *models.User) bool {
	return strings.EqualFold(user.OrgRole, "owner") && user.LogtoID != nil && *user.LogtoID != ""
}

// APIKeyAnchor returns the identifier key and audit rows are keyed on for this
// user: the local users id for regular accounts, the Logto ID for owner
// accounts (which have no local row).
func (s *APIKeysService) APIKeyAnchor(user *models.User) string {
	if isOwnerAccount(user) {
		return *user.LogtoID
	}
	return user.ID
}

// CreateAPIKey mints a new key for the user and returns the full plaintext
// token exactly once. The caller must surface it and never persist it.
// Owner accounts get a myo_-prefixed key anchored to their Logto ID; everyone
// else gets a myk_ key anchored to their local users row.
func (s *APIKeysService) CreateAPIKey(user *models.User, name, mode string, expiresInDays int) (*models.APIKey, string, error) {
	if mode != "read" && mode != "write" {
		return nil, "", fmt.Errorf("invalid mode: must be read or write")
	}

	owner := isOwnerAccount(user)
	if !owner {
		exists, err := s.userExists(user.ID)
		if err != nil {
			return nil, "", fmt.Errorf("failed to verify user: %w", err)
		}
		if !exists {
			return nil, "", ErrAPIKeyNoLocalUser
		}
	}

	if expiresInDays <= 0 {
		expiresInDays = apiKeyDefaultTTLDays
	}
	if expiresInDays > apiKeyMaxTTLDays {
		expiresInDays = apiKeyMaxTTLDays
	}

	generate := helpers.GenerateAPIKey
	if owner {
		generate = helpers.GenerateOwnerAPIKey
	}
	fullToken, public, secret, err := generate()
	if err != nil {
		return nil, "", err
	}
	secretHash, err := helpers.HashSystemSecretSHA256(secret)
	if err != nil {
		return nil, "", err
	}

	// Owner keys anchor exclusively on the Logto ID: user_id stays NULL even
	// though the auth context may carry the Logto ID as a fallback user.ID
	// (there is no users row it could reference).
	localUserID := user.ID
	if owner {
		localUserID = ""
	}

	key := &models.APIKey{
		ID:             uuid.New().String(),
		UserID:         localUserID,
		OrganizationID: user.OrganizationID,
		Name:           name,
		KeyPublic:      public,
		Mode:           mode,
		ExpiresAt:      time.Now().Add(time.Duration(expiresInDays) * 24 * time.Hour),
		CreatedAt:      time.Now(),
	}

	// Enforce the per-user limit and insert atomically. Concurrent creates for
	// the same account are serialized — via the users row lock for regular
	// accounts, via an advisory lock on the Logto ID for owner accounts (no
	// users row to lock) — so the count-then-insert cannot race past the cap.
	tx, err := database.DB.Begin()
	if err != nil {
		return nil, "", fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer func() { _ = tx.Rollback() }()

	var active int
	if owner {
		if _, err := tx.Exec(`SELECT pg_advisory_xact_lock(hashtext($1))`, *user.LogtoID); err != nil {
			return nil, "", fmt.Errorf("failed to lock owner account: %w", err)
		}
		err = tx.QueryRow(`
			SELECT COUNT(*) FROM user_api_keys
			WHERE logto_id = $1 AND user_id IS NULL AND revoked_at IS NULL AND expires_at > NOW()
		`, *user.LogtoID).Scan(&active)
	} else {
		if _, err := tx.Exec(`SELECT 1 FROM users WHERE id = $1 FOR UPDATE`, user.ID); err != nil {
			return nil, "", fmt.Errorf("failed to lock user: %w", err)
		}
		err = tx.QueryRow(`
			SELECT COUNT(*) FROM user_api_keys
			WHERE user_id = $1 AND revoked_at IS NULL AND expires_at > NOW()
		`, user.ID).Scan(&active)
	}
	if err != nil {
		return nil, "", fmt.Errorf("failed to count active api keys: %w", err)
	}
	if active >= APIKeyMaxPerUser {
		return nil, "", ErrAPIKeyLimitReached
	}

	var logtoID string
	if owner {
		logtoID = *user.LogtoID
	}
	_, err = tx.Exec(`
		INSERT INTO user_api_keys (id, user_id, logto_id, organization_id, name, key_public, key_secret_sha256, mode, expires_at, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`,
		key.ID, nullString(key.UserID), nullString(logtoID), nullString(key.OrganizationID), key.Name,
		key.KeyPublic, secretHash, key.Mode, key.ExpiresAt, key.CreatedAt)
	if err != nil {
		return nil, "", fmt.Errorf("failed to create api key: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, "", fmt.Errorf("failed to commit api key: %w", err)
	}

	return key, fullToken, nil
}

// apiKeyScope builds the WHERE fragment selecting keys owned by the user, on
// the anchor appropriate for the account type.
func apiKeyScope(user *models.User) (clause string, arg string) {
	if isOwnerAccount(user) {
		return "logto_id = $1 AND user_id IS NULL", *user.LogtoID
	}
	return "user_id = $1", user.ID
}

// ListAPIKeys returns all keys owned by the user, newest first, without secrets.
func (s *APIKeysService) ListAPIKeys(user *models.User) ([]models.APIKey, error) {
	scope, anchor := apiKeyScope(user)
	rows, err := database.DB.Query(`
		SELECT id, COALESCE(user_id, ''), COALESCE(organization_id, ''), name, key_public, mode,
		       expires_at, last_used_at, last_used_ip, revoked_at, created_at
		FROM user_api_keys
		WHERE `+scope+`
		ORDER BY created_at DESC
	`, anchor)
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

// RevokeAPIKey marks a key revoked. It only touches keys owned by the user, so
// a user can never revoke someone else's key. Returns sql.ErrNoRows when the
// key does not exist, is not owned by the user, or is already revoked.
func (s *APIKeysService) RevokeAPIKey(user *models.User, keyID string) error {
	scope, anchor := apiKeyScope(user)
	res, err := database.DB.Exec(`
		UPDATE user_api_keys
		SET revoked_at = NOW()
		WHERE id = $2 AND `+scope+` AND revoked_at IS NULL
	`, anchor, keyID)
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
// authoritatively — against the local users row (suspended_at / deleted_at) for
// regular keys, against the Logto profile (isSuspended, briefly cached) for
// owner keys — so suspending the owner kills every key at once and reactivating
// restores them.
func (s *APIKeysService) AuthenticateAPIKey(token string) (*APIKeyAuthResult, error) {
	public, secret, ownerKey, err := helpers.ParseAPIKey(token)
	if err != nil {
		return nil, ErrAPIKeyInvalid
	}
	if ownerKey {
		return s.authenticateOwnerAPIKey(public, secret)
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

// authenticateOwnerAPIKey resolves a myo_ key. Owner keys have no local users
// row: the row is matched on user_id IS NULL (so a myo_ token can never resolve
// a regular key and vice versa), the suspend check runs against the Logto
// profile, and the resolved account must still hold the owner org role.
//
// Suspend and role-downgrade take effect only once their respective caches
// expire (ownerSuspendedCacheTTL, and the user_profile cache behind
// ResolveUserByLogtoID which carries OrgRole): there is no lifecycle hook to
// invalidate them for the Logto-managed owner account, so a suspended or
// downgraded owner key can survive up to those TTLs. To kill a key
// immediately, revoke it (revoked_at is read fresh on every request, below).
func (s *APIKeysService) authenticateOwnerAPIKey(public, secret string) (*APIKeyAuthResult, error) {
	var keyID, logtoID, mode, name, secretHash string
	var orgID sql.NullString
	var expiresAt time.Time
	var revokedAt sql.NullTime

	err := database.DB.QueryRow(`
		SELECT k.id, k.key_secret_sha256, k.mode, k.name, k.organization_id, k.expires_at, k.revoked_at, k.logto_id
		FROM user_api_keys k
		WHERE k.key_public = $1 AND k.user_id IS NULL AND k.logto_id IS NOT NULL
	`, public).Scan(&keyID, &secretHash, &mode, &name, &orgID, &expiresAt, &revokedAt, &logtoID)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, ErrAPIKeyInvalid
	}
	if err != nil {
		return nil, fmt.Errorf("failed to look up owner api key: %w", err)
	}

	// Attribution carried back even on failure; owner keys are keyed on the
	// Logto ID, which is what audit rows store as user_id for them.
	res := &APIKeyAuthResult{
		KeyID:          keyID,
		UserID:         logtoID,
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

	suspended, err := s.ownerAccountSuspended(logtoID)
	if err != nil {
		// Fail closed: without an authoritative answer the key must not work.
		return nil, fmt.Errorf("failed to verify owner account state: %w", err)
	}
	if suspended {
		return res, ErrAPIKeyUserInactive
	}

	user, err := ResolveUserByLogtoID(logtoID)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(user.OrgRole, "owner") {
		return res, ErrAPIKeyUserInactive
	}
	user.UserPermissions = maskAPIKeyPermissions(user.UserPermissions, mode)
	user.OrgPermissions = maskAPIKeyPermissions(user.OrgPermissions, mode)
	res.User = user

	return res, nil
}

// ownerAccountSuspended checks the Logto profile's isSuspended flag, cached
// briefly so a busy integration does not hit the Logto Management API on every
// request. The TTL bounds suspension propagation; revocation stays instant.
func (s *APIKeysService) ownerAccountSuspended(logtoID string) (bool, error) {
	cacheKey := "api_key_owner_suspended:" + logtoID
	rc := cache.GetRedisClient()
	if rc != nil {
		var suspended bool
		if err := rc.Get(cacheKey, &suspended); err == nil {
			return suspended, nil
		}
	}

	profile, err := logto.GetUserProfileFromLogto(logtoID)
	if err != nil {
		return false, err
	}
	if rc != nil {
		_ = rc.Set(cacheKey, profile.IsSuspended, ownerSuspendedCacheTTL)
	}
	return profile.IsSuspended, nil
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
