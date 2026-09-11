/*
 * Copyright (C) 2026 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: AGPL-3.0-or-later
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package logto

import (
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rsa"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/http"
	"slices"
	"sync"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"github.com/nethesis/my/backend/configuration"
	"github.com/nethesis/my/backend/logger"
)

// ErrAccessTokenInvalid is the only error the exchange endpoint surfaces to a
// caller: the reason (audience, client, signature, expiry) stays in the logs.
var ErrAccessTokenInvalid = errors.New("invalid access token")

// ErrExchangeNotConfigured is returned when the audience or the SPA client id
// is not configured: with nothing to bind the token to, every exchange is
// refused rather than falling back to an unbound check.
var ErrExchangeNotConfigured = errors.New("token exchange is not configured")

// spaAccessTokenClaims are the claims of a Logto JWT access token issued to
// the my SPA for the my API resource.
type spaAccessTokenClaims struct {
	ClientID string `json:"client_id"`
	Scope    string `json:"scope"`
	jwt.RegisteredClaims
}

// AccessTokenValidator validates Logto JWT access tokens against the tenant
// JWKS and binds them to the my SPA: issuer, audience (the API resource
// indicator), client_id and expiry are all required. An opaque token (the kind
// Logto issues without a resource, and the kind every third-party app in the
// tenant obtains at login) is not a JWT and is refused outright — which is the
// point: only a token minted FOR the my API TO the my SPA can be exchanged.
type AccessTokenValidator struct {
	jwksURL  string
	issuers  []string
	audience string
	clientID string

	mu        sync.RWMutex
	keys      map[string]interface{}
	fetchedAt time.Time
}

const (
	jwksMinRefreshInterval = time.Minute
	jwksMaxAge             = time.Hour
	accessTokenLeeway      = 30 * time.Second
)

// validSigningMethods lists the asymmetric algorithms Logto signs with
// (ES384 by default). HMAC is excluded on purpose: a JWKS holds public keys.
var validSigningMethods = []string{"ES256", "ES384", "ES512", "RS256", "RS384", "RS512"}

// NewAccessTokenValidator builds a validator for tokens signed by the keys at
// jwksURL, issued by one of issuers, for audience, to clientID.
func NewAccessTokenValidator(jwksURL string, issuers []string, audience, clientID string) *AccessTokenValidator {
	return &AccessTokenValidator{
		jwksURL:  jwksURL,
		issuers:  issuers,
		audience: audience,
		clientID: clientID,
		keys:     map[string]interface{}{},
	}
}

var (
	spaValidator     *AccessTokenValidator
	spaValidatorOnce sync.Once
)

// SPAAccessTokenValidator returns the validator bound to the configured my
// SPA. Tokens may carry either the tenant issuer or the custom-domain issuer:
// Logto signs both with the same key set, served from the tenant JWKS.
func SPAAccessTokenValidator() *AccessTokenValidator {
	spaValidatorOnce.Do(func() {
		cfg := configuration.Config
		issuers := []string{cfg.LogtoIssuer + "/oidc"}
		if cfg.TenantDomain != "" {
			issuers = append(issuers, "https://"+cfg.TenantDomain+"/oidc")
		}
		spaValidator = NewAccessTokenValidator(cfg.LogtoIssuer+"/oidc/jwks", issuers, cfg.LogtoAPIResource, cfg.LogtoFrontendAppID)
	})
	return spaValidator
}

// Validate checks the token and returns the Logto user id (sub) it was issued
// for. Every failure is reported as ErrAccessTokenInvalid; details are logged.
func (v *AccessTokenValidator) Validate(tokenString string) (string, error) {
	if v.audience == "" || v.clientID == "" {
		logger.ComponentLogger("logto").Error().
			Str("operation", "validate_access_token").
			Msg("Token exchange refused: LOGTO_API_RESOURCE / LOGTO_FRONTEND_APP_ID not configured")
		return "", ErrExchangeNotConfigured
	}

	parser := jwt.NewParser(
		jwt.WithValidMethods(validSigningMethods),
		jwt.WithExpirationRequired(),
		jwt.WithAudience(v.audience),
		jwt.WithLeeway(accessTokenLeeway),
	)
	claims := &spaAccessTokenClaims{}
	token, err := parser.ParseWithClaims(tokenString, claims, v.keyFunc)
	if err != nil || !token.Valid {
		v.reject("signature or standard claims", err)
		return "", ErrAccessTokenInvalid
	}
	if !slices.Contains(v.issuers, claims.Issuer) {
		v.reject("issuer "+claims.Issuer, nil)
		return "", ErrAccessTokenInvalid
	}
	if claims.ClientID != v.clientID {
		v.reject("client_id "+claims.ClientID, nil)
		return "", ErrAccessTokenInvalid
	}
	if claims.Subject == "" {
		v.reject("empty subject", nil)
		return "", ErrAccessTokenInvalid
	}
	return claims.Subject, nil
}

func (v *AccessTokenValidator) reject(reason string, err error) {
	logger.ComponentLogger("logto").Warn().
		Err(err).
		Str("operation", "validate_access_token").
		Str("reason", reason).
		Msg("Logto access token rejected")
}

// keyFunc resolves the signing key by kid, refreshing the JWKS once when the
// kid is unknown (key rotation) and at most once a minute.
func (v *AccessTokenValidator) keyFunc(token *jwt.Token) (interface{}, error) {
	kid, _ := token.Header["kid"].(string)
	if kid == "" {
		return nil, errors.New("token has no kid")
	}
	if key, ok := v.cachedKey(kid, false); ok {
		return key, nil
	}
	if err := v.refresh(); err != nil {
		return nil, err
	}
	if key, ok := v.cachedKey(kid, true); ok {
		return key, nil
	}
	return nil, fmt.Errorf("unknown signing key %q", kid)
}

// cachedKey returns the key for kid when the cache is fresh enough. With
// afterRefresh the age check is skipped: a refresh that just happened is by
// definition as fresh as the JWKS gets.
func (v *AccessTokenValidator) cachedKey(kid string, afterRefresh bool) (interface{}, bool) {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if !afterRefresh && time.Since(v.fetchedAt) > jwksMaxAge {
		return nil, false
	}
	key, ok := v.keys[kid]
	return key, ok
}

func (v *AccessTokenValidator) refresh() error {
	v.mu.Lock()
	defer v.mu.Unlock()
	if time.Since(v.fetchedAt) < jwksMinRefreshInterval && len(v.keys) > 0 {
		return nil
	}
	keys, err := fetchJWKS(v.jwksURL)
	if err != nil {
		return err
	}
	v.keys = keys
	v.fetchedAt = time.Now()
	return nil
}

type jwk struct {
	Kty string `json:"kty"`
	Kid string `json:"kid"`
	Use string `json:"use"`
	Crv string `json:"crv"`
	X   string `json:"x"`
	Y   string `json:"y"`
	N   string `json:"n"`
	E   string `json:"e"`
}

func fetchJWKS(url string) (map[string]interface{}, error) {
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to build jwks request: %w", err)
	}
	resp, err := sharedHTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch jwks: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("jwks request failed with status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, fmt.Errorf("failed to read jwks: %w", err)
	}
	var set struct {
		Keys []jwk `json:"keys"`
	}
	if err := json.Unmarshal(body, &set); err != nil {
		return nil, fmt.Errorf("failed to decode jwks: %w", err)
	}
	keys := make(map[string]interface{}, len(set.Keys))
	for _, k := range set.Keys {
		if k.Use != "" && k.Use != "sig" {
			continue
		}
		pub, err := k.publicKey()
		if err != nil {
			logger.ComponentLogger("logto").Warn().Err(err).Str("kid", k.Kid).Msg("Skipping unusable JWK")
			continue
		}
		keys[k.Kid] = pub
	}
	if len(keys) == 0 {
		return nil, errors.New("jwks holds no usable signing key")
	}
	return keys, nil
}

func (k jwk) publicKey() (interface{}, error) {
	switch k.Kty {
	case "EC":
		var curve elliptic.Curve
		switch k.Crv {
		case "P-256":
			curve = elliptic.P256()
		case "P-384":
			curve = elliptic.P384()
		case "P-521":
			curve = elliptic.P521()
		default:
			return nil, fmt.Errorf("unsupported curve %q", k.Crv)
		}
		size := (curve.Params().BitSize + 7) / 8
		x, err := decodeCoordinate(k.X, size)
		if err != nil {
			return nil, err
		}
		y, err := decodeCoordinate(k.Y, size)
		if err != nil {
			return nil, err
		}
		// Uncompressed point encoding (0x04 || X || Y); the parser performs the
		// on-curve check.
		point := append(append([]byte{0x04}, x...), y...)
		pub, err := ecdsa.ParseUncompressedPublicKey(curve, point)
		if err != nil {
			return nil, fmt.Errorf("invalid EC public key: %w", err)
		}
		return pub, nil
	case "RSA":
		n, err := decodeBigInt(k.N)
		if err != nil {
			return nil, err
		}
		e, err := decodeBigInt(k.E)
		if err != nil {
			return nil, err
		}
		if !e.IsInt64() || e.Int64() <= 0 {
			return nil, errors.New("invalid RSA exponent")
		}
		return &rsa.PublicKey{N: n, E: int(e.Int64())}, nil
	default:
		return nil, fmt.Errorf("unsupported key type %q", k.Kty)
	}
}

// decodeCoordinate decodes a base64url EC coordinate and left-pads it to the
// curve's byte size, as required by the uncompressed point encoding.
func decodeCoordinate(s string, size int) ([]byte, error) {
	if s == "" {
		return nil, errors.New("missing key component")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid key component: %w", err)
	}
	if len(b) > size {
		return nil, errors.New("EC coordinate longer than the curve size")
	}
	out := make([]byte, size)
	copy(out[size-len(b):], b)
	return out, nil
}

func decodeBigInt(s string) (*big.Int, error) {
	if s == "" {
		return nil, errors.New("missing key component")
	}
	b, err := base64.RawURLEncoding.DecodeString(s)
	if err != nil {
		return nil, fmt.Errorf("invalid key component: %w", err)
	}
	return new(big.Int).SetBytes(b), nil
}
