/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package gateway

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"regexp"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	gossh "golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/logger"
)

// Keys shared with the backend, which approves the requests
const (
	requestPrefix = "ssh_gw:request:"
	resultPrefix  = "ssh_gw:result:"
	keyPrefix     = "ssh_gw:key:"
	pollInterval  = time.Second
)

// Permissions extensions carried from the authentication to the session
const (
	extGrant       = "grant"
	extKeyCacheKey = "key_cache_key"
	extOperator    = "operator"
	extSystem      = "system"
)

// The SSH user names the system by its key, optionally a cluster node:
// NETH-XXXX or NETH-XXXX.2
var (
	systemKeyPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)
	nodeIDPattern    = regexp.MustCompile(`^[0-9]{1,4}$`)
)

// user codes avoid characters that are easy to confuse (0/O, 1/I)
const userCodeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

type accessRequest struct {
	SystemKey string    `json:"system_key"`
	NodeID    string    `json:"node_id"`
	ClientIP  string    `json:"client_ip"`
	UserCode  string    `json:"user_code"`
	CreatedAt time.Time `json:"created_at"`
}

type accessResult struct {
	Status     string `json:"status"`
	GrantID    string `json:"grant_id"`
	Operator   string `json:"operator"`
	SystemName string `json:"system_name"`
	NodeID     string `json:"node_id"`
}

type cachedKey struct {
	GrantID    string `json:"grant_id"`
	Operator   string `json:"operator"`
	SystemName string `json:"system_name"`
}

// ParseTarget splits the SSH user into system key and node ID
func ParseTarget(user string) (systemKey, nodeID string, err error) {
	systemKey = user
	if i := strings.LastIndex(user, "."); i > 0 && nodeIDPattern.MatchString(user[i+1:]) {
		systemKey, nodeID = user[:i], user[i+1:]
	}
	if !systemKeyPattern.MatchString(systemKey) {
		return "", "", errors.New("invalid system key")
	}
	return systemKey, nodeID, nil
}

// Authenticator runs the two-step authentication: an SSH key whose ownership
// the client proves, then the operator's approval in the browser. An approved
// key skips the browser until its grant expires.
type Authenticator struct {
	redis   *redis.Client
	limiter *rateLimiter
}

func NewAuthenticator(rdb *redis.Client) *Authenticator {
	return &Authenticator{redis: rdb, limiter: newRateLimiter(configuration.Config.MaxRequestsPerMinute, time.Minute)}
}

// ServerConfig builds the per-connection SSH configuration
func (a *Authenticator) ServerConfig(clientIP string, hostKey gossh.Signer) *gossh.ServerConfig {
	cfg := &gossh.ServerConfig{
		MaxAuthTries:  6,
		ServerVersion: "SSH-2.0-MyNethesis",
		// any key is worth checking: what counts is that the client owns it
		PublicKeyCallback: func(conn gossh.ConnMetadata, key gossh.PublicKey) (*gossh.Permissions, error) {
			return &gossh.Permissions{}, nil
		},
		VerifiedPublicKeyCallback: func(conn gossh.ConnMetadata, key gossh.PublicKey, _ *gossh.Permissions, _ string) (*gossh.Permissions, error) {
			cacheKey := keyPrefix + conn.User() + ":" + gossh.FingerprintSHA256(key)
			if perms := a.cachedPermissions(cacheKey); perms != nil {
				return perms, nil
			}
			return nil, &gossh.PartialSuccessError{
				Next: gossh.ServerAuthCallbacks{KeyboardInteractiveCallback: a.browserApproval(clientIP, cacheKey)},
			}
		},
		// clients without a key go through the browser every time
		KeyboardInteractiveCallback: a.browserApproval(clientIP, ""),
	}
	cfg.AddHostKey(hostKey)
	return cfg
}

func (a *Authenticator) cachedPermissions(cacheKey string) *gossh.Permissions {
	val, err := a.redis.Get(context.Background(), cacheKey).Result()
	if err != nil {
		return nil
	}
	var cached cachedKey
	if json.Unmarshal([]byte(val), &cached) != nil || cached.GrantID == "" {
		return nil
	}
	return &gossh.Permissions{Extensions: map[string]string{
		extGrant:       cached.GrantID,
		extKeyCacheKey: cacheKey,
		extOperator:    cached.Operator,
		extSystem:      cached.SystemName,
	}}
}

// browserApproval parks an access request and waits for an operator to
// approve it in my; keyCacheKey, when set, remembers the verified key
func (a *Authenticator) browserApproval(clientIP, keyCacheKey string) func(gossh.ConnMetadata, gossh.KeyboardInteractiveChallenge) (*gossh.Permissions, error) {
	return func(conn gossh.ConnMetadata, challenge gossh.KeyboardInteractiveChallenge) (*gossh.Permissions, error) {
		log := logger.ComponentLogger("auth")
		systemKey, nodeID, err := ParseTarget(conn.User())
		if err != nil {
			_, _ = challenge("", "Utente non valido: usa la chiave del sistema, es. ssh NETH-XXXX@my oppure NETH-XXXX.2@my per il nodo 2.\n", nil, nil)
			return nil, err
		}
		if !a.limiter.Allow(clientIP) {
			log.Warn().Str("client_ip", clientIP).Msg("ssh access request rate limit exceeded")
			return nil, errors.New("rate limited")
		}

		code, err := randomHex(32)
		if err != nil {
			return nil, err
		}
		userCode, err := randomUserCode()
		if err != nil {
			return nil, err
		}
		ctx := context.Background()
		data, _ := json.Marshal(accessRequest{
			SystemKey: systemKey,
			NodeID:    nodeID,
			ClientIP:  clientIP,
			UserCode:  userCode,
			CreatedAt: time.Now(),
		})
		timeout := configuration.Config.AuthTimeout
		if err := a.redis.Set(ctx, requestPrefix+code, data, timeout).Err(); err != nil {
			log.Error().Err(err).Msg("failed to store ssh access request")
			return nil, err
		}

		target := systemKey
		if nodeID != "" {
			target += " (nodo " + nodeID + ")"
		}
		_, _ = challenge("", fmt.Sprintf(
			"\nMy Nethesis - accesso SSH a %s\n\n"+
				"Apri questo link e autorizza l'accesso:\n\n  %s/ssh-auth?code=%s\n\n"+
				"Codice di verifica: %s\n\nIn attesa dell'autorizzazione...\n",
			target, configuration.Config.AppURL, code, userCode,
		), nil, nil)

		log.Info().Str("system_key", systemKey).Str("node_id", nodeID).Str("client_ip", clientIP).Msg("ssh access request waiting for approval")

		result, err := a.waitResult(ctx, code, timeout)
		if err != nil {
			a.redis.Del(ctx, requestPrefix+code) //nolint:errcheck
			_, _ = challenge("", "\nTempo scaduto: nessuna autorizzazione ricevuta.\n", nil, nil)
			return nil, err
		}

		switch result.Status {
		case "approved":
		case "no_session":
			_, _ = challenge("", "\nNessuna sessione di supporto attiva su questo sistema.\n", nil, nil)
			return nil, errors.New("no active session")
		default:
			_, _ = challenge("", "\nAccesso rifiutato.\n", nil, nil)
			return nil, errors.New("access denied")
		}

		if keyCacheKey != "" {
			cached, _ := json.Marshal(cachedKey{GrantID: result.GrantID, Operator: result.Operator, SystemName: result.SystemName})
			a.redis.Set(ctx, keyCacheKey, cached, configuration.Config.KeyAuthTTL) //nolint:errcheck
		}
		_, _ = challenge("", fmt.Sprintf("\nAutorizzato da %s. Connessione a %s...\n", result.Operator, result.SystemName), nil, nil)

		log.Info().Str("system_key", systemKey).Str("node_id", nodeID).Str("operator", result.Operator).Bool("key_cached", keyCacheKey != "").Msg("ssh access approved")
		return &gossh.Permissions{Extensions: map[string]string{
			extGrant:       result.GrantID,
			extKeyCacheKey: keyCacheKey,
			extOperator:    result.Operator,
			extSystem:      result.SystemName,
		}}, nil
	}
}

func (a *Authenticator) waitResult(ctx context.Context, code string, timeout time.Duration) (*accessResult, error) {
	deadline := time.Now().Add(timeout)
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for range ticker.C {
		if time.Now().After(deadline) {
			return nil, errors.New("approval timed out")
		}
		val, err := a.redis.GetDel(ctx, resultPrefix+code).Result()
		if err != nil {
			continue
		}
		var result accessResult
		if err := json.Unmarshal([]byte(val), &result); err != nil {
			return nil, err
		}
		return &result, nil
	}
	return nil, errors.New("approval timed out")
}

// ForgetKey drops a cached key whose grant is no longer valid
func (a *Authenticator) ForgetKey(cacheKey string) {
	if cacheKey != "" {
		a.redis.Del(context.Background(), cacheKey) //nolint:errcheck
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func randomUserCode() (string, error) {
	var sb strings.Builder
	for i := 0; i < 8; i++ {
		if i == 4 {
			sb.WriteByte('-')
		}
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(userCodeAlphabet))))
		if err != nil {
			return "", err
		}
		sb.WriteByte(userCodeAlphabet[n.Int64()])
	}
	return sb.String(), nil
}
