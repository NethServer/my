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
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	gossh "golang.org/x/crypto/ssh"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
	"github.com/nethesis/my/services/ssh-gateway/logger"
)

const pollInterval = time.Second

// Permissions extensions carried from the authentication to the session
const (
	extGrant    = "grant"
	extOperator = "operator"
	extSystem   = "system"
)

// The SSH user names the system by its key, optionally a cluster node, and
// carries the one-time token of the copied command: NETH-XXXX.2+<token>
var (
	systemKeyPattern = regexp.MustCompile(`^[A-Za-z0-9-]{1,128}$`)
	nodeIDPattern    = regexp.MustCompile(`^[0-9]{1,10}$`)
	tokenPattern     = regexp.MustCompile(`^[0-9a-f]{32}$`)
)

const copyHint = "Copia il comando dal pulsante SSH della sessione in my: contiene il tuo token personale.\n"

// Target is what the SSH user asks for
type Target struct {
	SystemKey string
	NodeID    string
	Token     string
}

// Name is the target without the token: approved keys are remembered per name
func (t Target) Name() string {
	if t.NodeID == "" {
		return t.SystemKey
	}
	return t.SystemKey + "." + t.NodeID
}

// ParseTarget splits the SSH user into system key, node ID and token
func ParseTarget(user string) (Target, error) {
	var t Target
	rest := user
	if i := strings.LastIndex(rest, "+"); i >= 0 {
		t.Token, rest = rest[i+1:], rest[:i]
		if !tokenPattern.MatchString(t.Token) {
			return Target{}, errors.New("invalid token")
		}
	}
	t.SystemKey = rest
	if i := strings.LastIndex(rest, "."); i > 0 && nodeIDPattern.MatchString(rest[i+1:]) {
		t.SystemKey, t.NodeID = rest[:i], rest[i+1:]
	}
	if !systemKeyPattern.MatchString(t.SystemKey) {
		return Target{}, errors.New("invalid system key")
	}
	return t, nil
}

// connState follows one client connection through its authentication
type connState struct {
	clientIP string
	// done closes when the client goes away: a pending approval stops there
	done <-chan struct{}
	// authStarted gives the connection the time to wait for an approval
	authStarted func()
}

// Authenticator runs the authentication: an SSH key approved in the last
// hours opens the session at once; otherwise the operator who copied the
// command approves it in the browser.
type Authenticator struct {
	limiter *rateLimiter
}

func NewAuthenticator() *Authenticator {
	return &Authenticator{limiter: newRateLimiter(configuration.Config.MaxRequestsPerMinute, time.Minute)}
}

// ServerConfig builds the per-connection SSH configuration
func (a *Authenticator) ServerConfig(st *connState, hostKey gossh.Signer) *gossh.ServerConfig {
	cfg := &gossh.ServerConfig{
		MaxAuthTries:  6,
		ServerVersion: "SSH-2.0-MyNethesis",
		// any key is worth checking: what counts is that the client owns it
		PublicKeyCallback: func(conn gossh.ConnMetadata, key gossh.PublicKey) (*gossh.Permissions, error) {
			return &gossh.Permissions{}, nil
		},
		VerifiedPublicKeyCallback: func(conn gossh.ConnMetadata, key gossh.PublicKey, _ *gossh.Permissions, _ string) (*gossh.Permissions, error) {
			fingerprint := gossh.FingerprintSHA256(key)
			st.authStarted()
			if target, err := ParseTarget(conn.User()); err == nil {
				if perms := a.knownKey(st, target, fingerprint); perms != nil {
					return perms, nil
				}
			}
			return nil, &gossh.PartialSuccessError{
				Next: gossh.ServerAuthCallbacks{KeyboardInteractiveCallback: a.browserApproval(st, fingerprint)},
			}
		},
		// clients without a key go through the browser every time
		KeyboardInteractiveCallback: a.browserApproval(st, ""),
	}
	cfg.AddHostKey(hostKey)
	return cfg
}

func (a *Authenticator) knownKey(st *connState, target Target, fingerprint string) *gossh.Permissions {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	known, err := lookupKey(ctx, target.Name(), fingerprint)
	if err != nil {
		return nil
	}
	logger.ComponentLogger("auth").Info().Str("target", target.Name()).Str("operator", known.Operator).Str("client_ip", st.clientIP).Msg("ssh access with an approved key")
	return &gossh.Permissions{Extensions: map[string]string{
		extGrant:    known.GrantID,
		extOperator: known.Operator,
		extSystem:   known.SystemName,
	}}
}

// browserApproval turns the token of the copied command into an access
// request and waits for its operator to approve it in my
func (a *Authenticator) browserApproval(st *connState, fingerprint string) func(gossh.ConnMetadata, gossh.KeyboardInteractiveChallenge) (*gossh.Permissions, error) {
	return func(conn gossh.ConnMetadata, challenge gossh.KeyboardInteractiveChallenge) (*gossh.Permissions, error) {
		log := logger.ComponentLogger("auth")
		target, err := ParseTarget(conn.User())
		if err != nil || target.Token == "" {
			_, _ = challenge("", "\nAccesso non valido. "+copyHint, nil, nil)
			return nil, errors.New("missing or invalid token")
		}
		if !a.limiter.Allow(st.clientIP) {
			log.Warn().Str("client_ip", st.clientIP).Msg("ssh access request rate limit exceeded")
			return nil, errors.New("rate limited")
		}
		st.authStarted()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		go func() {
			select {
			case <-st.done:
				cancel()
			case <-ctx.Done():
			}
		}()

		req, err := createRequest(ctx, target.SystemKey, target.NodeID, target.Token, st.clientIP, fingerprint)
		if errors.Is(err, errNotFound) {
			_, _ = challenge("", "\nComando scaduto o già usato (vale 10 minuti, una volta sola). "+copyHint, nil, nil)
			return nil, err
		}
		if err != nil {
			log.Error().Err(err).Msg("cannot create the ssh access request")
			_, _ = challenge("", "\nServizio non disponibile, riprova tra poco.\n", nil, nil)
			return nil, err
		}

		target.Token = ""
		_, _ = challenge("", fmt.Sprintf(
			"\nMy Nethesis - accesso SSH a %s\n\n"+
				"Apri questo link e autorizza l'accesso (solo tu puoi farlo):\n\n  %s\n\n"+
				"In attesa dell'autorizzazione...\n",
			target.Name(), req.URL,
		), nil, nil)
		log.Info().Str("target", target.Name()).Str("client_ip", st.clientIP).Msg("ssh access request waiting for approval")

		result, err := waitResult(ctx, req.Code)
		if err != nil {
			// a client that went away can no longer be approved
			cancelRequest(req.Code)
			if ctx.Err() == nil {
				_, _ = challenge("", "\nTempo scaduto: nessuna autorizzazione ricevuta.\n", nil, nil)
			}
			return nil, err
		}

		switch {
		case result.Status == "approved" && result.GrantID != "":
		case result.Status == "no_session":
			_, _ = challenge("", "\nNessuna sessione di supporto attiva su questo sistema.\n", nil, nil)
			return nil, errors.New("no active session")
		default:
			_, _ = challenge("", "\nAccesso rifiutato.\n", nil, nil)
			return nil, errors.New("access denied")
		}

		_, _ = challenge("", fmt.Sprintf("\nAutorizzato. Connessione a %s...\n", result.SystemName), nil, nil)
		log.Info().Str("target", target.Name()).Str("operator", result.Operator).Bool("key", fingerprint != "").Msg("ssh access approved")
		return &gossh.Permissions{Extensions: map[string]string{
			extGrant:    result.GrantID,
			extOperator: result.Operator,
			extSystem:   result.SystemName,
		}}, nil
	}
}

func waitResult(ctx context.Context, code string) (*accessResult, error) {
	ctx, cancel := context.WithTimeout(ctx, configuration.Config.AuthTimeout)
	defer cancel()
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			result, err := requestResult(ctx, code)
			switch {
			case err == nil:
				return result, nil
			case errors.Is(err, errGone):
				return nil, err
			case errors.Is(err, errPending):
			default:
				logger.ComponentLogger("auth").Warn().Err(err).Msg("cannot check the ssh access request")
			}
		}
	}
}
