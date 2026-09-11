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
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

const (
	testIssuer   = "https://tenant.logto.app/oidc"
	testAudience = "https://my.test/api/permissions"
	testClientID = "spa-client-id"
	testKid      = "kid-1"
)

func newJWKSServer(t *testing.T, key *ecdsa.PrivateKey) *httptest.Server {
	t.Helper()
	// Uncompressed point: 0x04 || X || Y, each coordinate curve-size bytes.
	size := (key.Curve.Params().BitSize + 7) / 8
	point, err := key.PublicKey.Bytes()
	require.NoError(t, err)
	require.Len(t, point, 1+2*size)
	set := map[string]interface{}{"keys": []map[string]string{{
		"kty": "EC", "kid": testKid, "use": "sig", "alg": "ES384", "crv": "P-384",
		"x": base64.RawURLEncoding.EncodeToString(point[1 : 1+size]),
		"y": base64.RawURLEncoding.EncodeToString(point[1+size:]),
	}}}
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_ = json.NewEncoder(w).Encode(set)
	}))
}

func signed(t *testing.T, key *ecdsa.PrivateKey, kid string, claims spaAccessTokenClaims) string {
	t.Helper()
	tok := jwt.NewWithClaims(jwt.SigningMethodES384, claims)
	tok.Header["kid"] = kid
	s, err := tok.SignedString(key)
	require.NoError(t, err)
	return s
}

func goodClaims() spaAccessTokenClaims {
	return spaAccessTokenClaims{
		ClientID: testClientID,
		Scope:    "read:systems",
		RegisteredClaims: jwt.RegisteredClaims{
			Issuer:    testIssuer,
			Subject:   "user-123",
			Audience:  jwt.ClaimStrings{testAudience},
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
		},
	}
}

func TestAccessTokenValidator(t *testing.T) {
	key, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
	require.NoError(t, err)
	srv := newJWKSServer(t, key)
	defer srv.Close()

	v := NewAccessTokenValidator(srv.URL, []string{testIssuer}, testAudience, testClientID)

	t.Run("token for the SPA and the API resource is accepted", func(t *testing.T) {
		sub, err := v.Validate(signed(t, key, testKid, goodClaims()))
		require.NoError(t, err)
		assert.Equal(t, "user-123", sub)
	})

	t.Run("custom-domain issuer is accepted when listed", func(t *testing.T) {
		v2 := NewAccessTokenValidator(srv.URL, []string{testIssuer, "https://id.example.com/oidc"}, testAudience, testClientID)
		c := goodClaims()
		c.Issuer = "https://id.example.com/oidc"
		_, err := v2.Validate(signed(t, key, testKid, c))
		assert.NoError(t, err)
	})

	t.Run("a third-party app's token is refused by client_id", func(t *testing.T) {
		c := goodClaims()
		c.ClientID = "nethshop-app"
		_, err := v.Validate(signed(t, key, testKid, c))
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("a token for another resource is refused by audience", func(t *testing.T) {
		c := goodClaims()
		c.Audience = jwt.ClaimStrings{"https://other.test/api"}
		_, err := v.Validate(signed(t, key, testKid, c))
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("a token without audience is refused", func(t *testing.T) {
		c := goodClaims()
		c.Audience = nil
		_, err := v.Validate(signed(t, key, testKid, c))
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("foreign issuer is refused", func(t *testing.T) {
		c := goodClaims()
		c.Issuer = "https://evil.logto.app/oidc"
		_, err := v.Validate(signed(t, key, testKid, c))
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("expired token is refused", func(t *testing.T) {
		c := goodClaims()
		c.ExpiresAt = jwt.NewNumericDate(time.Now().Add(-2 * time.Minute))
		_, err := v.Validate(signed(t, key, testKid, c))
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("token without exp is refused", func(t *testing.T) {
		c := goodClaims()
		c.ExpiresAt = nil
		_, err := v.Validate(signed(t, key, testKid, c))
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("unknown kid is refused", func(t *testing.T) {
		_, err := v.Validate(signed(t, key, "rotated-away", goodClaims()))
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("HMAC-signed token is refused even with a matching kid", func(t *testing.T) {
		tok := jwt.NewWithClaims(jwt.SigningMethodHS256, goodClaims())
		tok.Header["kid"] = testKid
		s, err := tok.SignedString([]byte("secret"))
		require.NoError(t, err)
		_, err = v.Validate(s)
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("opaque token is refused", func(t *testing.T) {
		_, err := v.Validate("Vj3q1Ff3R8n0uYwGz2sX7kPq")
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("wrong key is refused", func(t *testing.T) {
		other, err := ecdsa.GenerateKey(elliptic.P384(), rand.Reader)
		require.NoError(t, err)
		_, err = v.Validate(signed(t, other, testKid, goodClaims()))
		assert.ErrorIs(t, err, ErrAccessTokenInvalid)
	})

	t.Run("unconfigured audience or client refuses everything", func(t *testing.T) {
		_, err := NewAccessTokenValidator(srv.URL, []string{testIssuer}, "", testClientID).Validate(signed(t, key, testKid, goodClaims()))
		assert.ErrorIs(t, err, ErrExchangeNotConfigured)
		_, err = NewAccessTokenValidator(srv.URL, []string{testIssuer}, testAudience, "").Validate(signed(t, key, testKid, goodClaims()))
		assert.ErrorIs(t, err, ErrExchangeNotConfigured)
	})
}
