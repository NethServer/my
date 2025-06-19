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
}
