/*
 * Copyright (C) 2025 Nethesis S.r.l.
 * http://www.nethesis.it - info@nethesis.it
 *
 * SPDX-License-Identifier: GPL-2.0-only
 *
 * author: Edoardo Spadoni <edoardo.spadoni@nethesis.it>
 */

package configuration

import (
	"os"

	"github.com/nethesis/my/backend/logs"
)

type Configuration struct {
	ListenAddress string `json:"listen_address"`
	LogtoIssuer   string `json:"logto_issuer"`
	LogtoAudience string `json:"logto_audience"`
	JWKSEndpoint  string `json:"jwks_endpoint"`
	// JWT Custom token configuration
	JWTSecret            string `json:"jwt_secret"`
	JWTIssuer            string `json:"jwt_issuer"`
	JWTExpiration        string `json:"jwt_expiration"`
	JWTRefreshExpiration string `json:"jwt_refresh_expiration"`
	// Logto Management API configuration
	LogtoManagementClientID     string `json:"logto_management_client_id"`
	LogtoManagementClientSecret string `json:"logto_management_client_secret"`
	LogtoManagementBaseURL      string `json:"logto_management_base_url"`
}

var Config = Configuration{}

func Init() {
	if os.Getenv("LISTEN_ADDRESS") != "" {
		Config.ListenAddress = os.Getenv("LISTEN_ADDRESS")
	} else {
		Config.ListenAddress = "127.0.0.1:8080"
	}

	if os.Getenv("LOGTO_ISSUER") != "" {
		Config.LogtoIssuer = os.Getenv("LOGTO_ISSUER")
	} else {
		logs.Logs.Println("[CRITICAL][ENV] LOGTO_ISSUER variable is empty")
		os.Exit(1)
	}

	if os.Getenv("LOGTO_AUDIENCE") != "" {
		Config.LogtoAudience = os.Getenv("LOGTO_AUDIENCE")
	} else {
		logs.Logs.Println("[CRITICAL][ENV] LOGTO_AUDIENCE variable is empty")
		os.Exit(1)
	}

	if os.Getenv("JWKS_ENDPOINT") != "" {
		Config.JWKSEndpoint = os.Getenv("JWKS_ENDPOINT")
	} else {
		Config.JWKSEndpoint = Config.LogtoIssuer + "/oidc/jwks"
	}

	// JWT custom token configuration
	if os.Getenv("JWT_SECRET") != "" {
		Config.JWTSecret = os.Getenv("JWT_SECRET")
	} else {
		logs.Logs.Println("[CRITICAL][ENV] JWT_SECRET variable is empty")
		os.Exit(1)
	}

	if os.Getenv("JWT_ISSUER") != "" {
		Config.JWTIssuer = os.Getenv("JWT_ISSUER")
	} else {
		Config.JWTIssuer = "my.nethesis.it"
	}

	if os.Getenv("JWT_EXPIRATION") != "" {
		Config.JWTExpiration = os.Getenv("JWT_EXPIRATION")
	} else {
		Config.JWTExpiration = "24h" // Default: 24 hours
	}

	if os.Getenv("JWT_REFRESH_EXPIRATION") != "" {
		Config.JWTRefreshExpiration = os.Getenv("JWT_REFRESH_EXPIRATION")
	} else {
		Config.JWTRefreshExpiration = "168h" // Default: 7 days
	}

	// Logto Management API configuration
	if os.Getenv("LOGTO_MANAGEMENT_CLIENT_ID") != "" {
		Config.LogtoManagementClientID = os.Getenv("LOGTO_MANAGEMENT_CLIENT_ID")
	} else {
		logs.Logs.Println("[CRITICAL][ENV] LOGTO_MANAGEMENT_CLIENT_ID variable is empty")
		os.Exit(1)
	}

	if os.Getenv("LOGTO_MANAGEMENT_CLIENT_SECRET") != "" {
		Config.LogtoManagementClientSecret = os.Getenv("LOGTO_MANAGEMENT_CLIENT_SECRET")
	} else {
		logs.Logs.Println("[CRITICAL][ENV] LOGTO_MANAGEMENT_CLIENT_SECRET variable is empty")
		os.Exit(1)
	}

	if os.Getenv("LOGTO_MANAGEMENT_BASE_URL") != "" {
		Config.LogtoManagementBaseURL = os.Getenv("LOGTO_MANAGEMENT_BASE_URL")
	} else {
		Config.LogtoManagementBaseURL = Config.LogtoIssuer + "/api"
	}
}
