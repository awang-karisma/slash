package sso

import (
	"context"

	"github.com/pkg/errors"
	"github.com/yourselfhosted/slash/plugin/idp/oauth2"
	storepb "github.com/yourselfhosted/slash/proto/gen/store"
)

// BuildEnvIdentityProvider creates an identity provider from environment config.
func BuildEnvIdentityProvider(ctx context.Context, config *SSOConfig) (*storepb.IdentityProvider, error) {
	// Fetch discovery document
	discovery, err := FetchDiscovery(ctx, config.IssuerURL)
	if err != nil {
		return nil, errors.Wrap(err, "failed to fetch OpenID discovery")
	}

	// Determine scopes - use env var if set, otherwise use discovery defaults
	scopes := config.Scopes
	if len(scopes) == 0 || (len(scopes) == 1 && scopes[0] == "") {
		scopes = GetDefaultScopes()
	}

	return &storepb.IdentityProvider{
		Id:    "env:0",
		Title: config.Title,
		Type:  storepb.IdentityProvider_OAUTH2,
		Config: &storepb.IdentityProviderConfig{
			Config: &storepb.IdentityProviderConfig_Oauth2{
				Oauth2: &storepb.IdentityProviderConfig_OAuth2Config{
					ClientId:     config.ClientID,
					ClientSecret: config.ClientSecret,
					AuthUrl:      discovery.AuthURL,
					TokenUrl:     discovery.TokenURL,
					UserInfoUrl:  discovery.UserInfoURL,
					Scopes:       scopes,
					FieldMapping: &storepb.IdentityProviderConfig_FieldMapping{
						Identifier:  config.IdentifierField,
						DisplayName: config.DisplayNameField,
					},
				},
			},
		},
	}, nil
}

// GetEnvIdentityProvider loads and builds an environment-based identity provider.
func GetEnvIdentityProvider(ctx context.Context) (*storepb.IdentityProvider, error) {
	config, err := LoadSSOConfig()
	if err != nil {
		return nil, errors.Wrap(err, "failed to load env config")
	}
	if config == nil {
		return nil, errors.New("environment SSO not configured")
	}

	return BuildEnvIdentityProvider(ctx, config)
}

// IsEnvIdentityProviderID checks if the given ID is an environment-based identity provider.
func IsEnvIdentityProviderID(idpID string) bool {
	return idpID == "env:0"
}

// NewEnvOAuth2Provider creates an OAuth2 identity provider for authentication.
func NewEnvOAuth2Provider(ctx context.Context) (*oauth2.IdentityProvider, error) {
	identityProvider, err := GetEnvIdentityProvider(ctx)
	if err != nil {
		return nil, err
	}

	oauth2Config := identityProvider.GetConfig().GetOauth2()
	if oauth2Config == nil {
		return nil, errors.New("identity provider is not OAuth2 type")
	}

	return oauth2.NewIdentityProvider(oauth2Config)
}
