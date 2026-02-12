package sso

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// OpenIDConfiguration represents the OIDC discovery document.
type OpenIDConfiguration struct {
	Issuer      string   `json:"issuer"`
	AuthURL     string   `json:"authorization_endpoint"`
	TokenURL    string   `json:"token_endpoint"`
	UserInfoURL string   `json:"userinfo_endpoint"`
	Scopes      []string `json:"scopes_supported"`
}

// FetchDiscovery fetches the OpenID Connect discovery document from the issuer URL.
func FetchDiscovery(ctx context.Context, issuerURL string) (*OpenIDConfiguration, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	// Clean up the issuer URL
	issuerURL = strings.TrimSuffix(issuerURL, "/")
	discoveryURL := issuerURL + "/.well-known/openid-configuration"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, discoveryURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create discovery request")
	}
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch discovery document")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("discovery endpoint returned status %d: %s", resp.StatusCode, resp.Status)
	}

	var config OpenIDConfiguration
	if err := json.NewDecoder(resp.Body).Decode(&config); err != nil {
		return nil, errors.Wrap(err, "failed to decode discovery document")
	}

	// Validate required fields
	if config.AuthURL == "" {
		return nil, errors.New("authorization_endpoint not found in discovery document")
	}
	if config.TokenURL == "" {
		return nil, errors.New("token_endpoint not found in discovery document")
	}

	return &config, nil
}

// GetDefaultScopes returns the default OAuth2 scopes if not explicitly configured.
func GetDefaultScopes() []string {
	return []string{"openid", "profile", "email"}
}
