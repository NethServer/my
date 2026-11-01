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
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/nethesis/my/services/ssh-gateway/configuration"
)

// The backend's internal SSH routes, authenticated by the gateway's secret
const (
	gatewayPath         = "/api/support-ssh/gateway"
	gatewaySecretHeader = "X-SSH-Gateway-Secret"
	grantHeader         = "X-SSH-Grant"
)

var (
	errNotFound = errors.New("not found")
	errPending  = errors.New("pending")
	errGone     = errors.New("gone")
)

// the key lookup checks the operator on Logto: leave it room
var backendClient = &http.Client{Timeout: 30 * time.Second}

type accessRequest struct {
	Code string `json:"code"`
	URL  string `json:"url"`
}

type accessResult struct {
	Status     string `json:"status"`
	GrantID    string `json:"grant_id"`
	Operator   string `json:"operator"`
	SystemName string `json:"system_name"`
}

type knownKey struct {
	GrantID    string `json:"grant_id"`
	Operator   string `json:"operator"`
	SystemName string `json:"system_name"`
}

// callBackend runs one internal call; out receives the "data" of a 2xx answer
func callBackend(ctx context.Context, method, path string, body, out interface{}) (int, error) {
	var reader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return 0, err
		}
		reader = bytes.NewReader(data)
	}
	req, err := http.NewRequestWithContext(ctx, method, configuration.Config.BackendURL+gatewayPath+path, reader)
	if err != nil {
		return 0, err
	}
	req.Header.Set(gatewaySecretHeader, configuration.Config.SSHGatewaySecret)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := backendClient.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	if out != nil && resp.StatusCode/100 == 2 {
		envelope := struct {
			Data json.RawMessage `json:"data"`
		}{}
		if err := json.NewDecoder(io.LimitReader(resp.Body, 64*1024)).Decode(&envelope); err != nil {
			return resp.StatusCode, err
		}
		if len(envelope.Data) > 0 {
			if err := json.Unmarshal(envelope.Data, out); err != nil {
				return resp.StatusCode, err
			}
		}
	}
	return resp.StatusCode, nil
}

func createRequest(ctx context.Context, systemKey, nodeID, token, clientIP, fingerprint string) (*accessRequest, error) {
	var out accessRequest
	status, err := callBackend(ctx, http.MethodPost, "/requests", map[string]string{
		"system_key":  systemKey,
		"node_id":     nodeID,
		"token":       token,
		"client_ip":   clientIP,
		"fingerprint": fingerprint,
	}, &out)
	switch {
	case err != nil:
		return nil, err
	case status == http.StatusNotFound:
		return nil, errNotFound
	case status != http.StatusCreated || out.Code == "":
		return nil, fmt.Errorf("create request: status %d", status)
	}
	return &out, nil
}

func requestResult(ctx context.Context, code string) (*accessResult, error) {
	var out accessResult
	status, err := callBackend(ctx, http.MethodGet, "/requests/"+url.PathEscape(code), nil, &out)
	switch {
	case err != nil:
		return nil, err
	case status == http.StatusAccepted:
		return nil, errPending
	case status == http.StatusGone:
		return nil, errGone
	case status != http.StatusOK:
		return nil, fmt.Errorf("request result: status %d", status)
	}
	return &out, nil
}

func cancelRequest(code string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, _ = callBackend(ctx, http.MethodDelete, "/requests/"+url.PathEscape(code), nil, nil)
}

func lookupKey(ctx context.Context, target, fingerprint string) (*knownKey, error) {
	var out knownKey
	q := url.Values{"target": {target}, "fingerprint": {fingerprint}}
	status, err := callBackend(ctx, http.MethodGet, "/keys?"+q.Encode(), nil, &out)
	switch {
	case err != nil:
		return nil, err
	case status != http.StatusOK || out.GrantID == "":
		return nil, errNotFound
	}
	return &out, nil
}
