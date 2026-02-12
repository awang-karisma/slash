package sso

import (
	"strings"

	"github.com/spf13/viper"
)

// SSOConfig represents SSO configuration loaded from environment variables.
type SSOConfig struct {
	Title            string
	ClientID         string
	ClientSecret     string
	IssuerURL        string
	IdentifierField  string
	DisplayNameField string
	Scopes           []string
}

// LoadSSOConfig loads SSO configuration from environment variables.
// Returns nil if mandatory config (CLIENT_ID, CLIENT_SECRET, ISSUER_URL) is incomplete.
func LoadSSOConfig() (*SSOConfig, error) {
	// Use viper's built-in prefixing with AutomaticEnv()
	// viper.SetEnvPrefix("slash") and viper.AutomaticEnv() are set in main.go
	// So viper.GetString("sso.client_id") will look for SLASH_SSO_CLIENT_ID
	clientID := viper.GetString("sso.client_id")
	clientSecret := viper.GetString("sso.client_secret")
	issuerURL := viper.GetString("sso.issuer_url")

	// Check mandatory fields
	if clientID == "" || clientSecret == "" || issuerURL == "" {
		return nil, nil // Incomplete config, fall back to database
	}

	return &SSOConfig{
		Title:            viper.GetString("sso.title"),
		ClientID:         clientID,
		ClientSecret:     clientSecret,
		IssuerURL:        issuerURL,
		IdentifierField:  viper.GetString("sso.identifier_field"),
		DisplayNameField: viper.GetString("sso.display_name_field"),
		Scopes:           strings.Split(viper.GetString("sso.scopes"), " "),
	}, nil
}

// HasSSO checks if SSO is configured via environment variables.
func HasSSO() bool {
	config, err := LoadSSOConfig()
	if err != nil {
		return false
	}
	return config != nil
}
