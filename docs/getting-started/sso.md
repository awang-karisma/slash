# Single Sign-On(SSO)

> **Note**: This feature is only available in the **Team** plan.

**Single Sign-On (SSO)** is an authentication method that enables users to securely authenticate with multiple applications and websites by using just one set of credentials.

Slash supports SSO integration with **OAuth 2.0** standard.

## Create a new SSO provider

As an Admin user, you can create a new SSO provider in Setting > Workspace settings > SSO.

![sso-setting](../assets/getting-started/sso-setting.png)

For example, to integrate with GitHub, you might need to fill in the following fields:

![github-sso](../assets/getting-started/github-sso.png)

### Identity provider information

The information is the base concept of OAuth 2.0 and comes from your provider.

- **Client ID** is a public identifier of the custom provider;
- **Client Secret** is the OAuth2 client secret from identity provider;
- **Authorization endpoint** is the custom provider's OAuth2 login page address;
- **Token endpoint** is the API address for obtaining access token;
- **User endpoint** URL is the API address for obtaining user information by access token;
- **Scopes** is the scope parameter carried when accessing the OAuth2 URL, which is filled in according to the custom provider;

### User information mapping

For different providers, the structures returned by their user information API are usually not the same. In order to know how to map the user information from an provider into user fields, you need to fill the user information mapping form.

Slash will use the mapping to import the user profile fields when creating new accounts. The most important user field mapping is the identifier which is used to identify the Slash account associated with the OAuth 2.0 login.

- **Identifier** is the field name of primary email in 3rd-party user info;
- **Display name** is the field name of display name in 3rd-party user info (optional);

## Configure SSO via Environment Variables

You can configure SSO providers using environment variables instead of the admin UI. This is useful for containerized deployments or when you want to manage SSO configuration as code.

### Quick Setup

For OAuth2 providers that support OpenID Connect (OIDC), you only need three environment variables:

```bash
SLASH_SSO_0_CLIENT_ID=your-client-id
SLASH_SSO_0_CLIENT_SECRET=your-client-secret
SLASH_SSO_0_ISSUER_URL=https://your-idp.com
```

Slash will automatically discover the OAuth endpoints from the issuer URL's `.well-known/openid-configuration`.

### Environment Variables

| Variable | Required | Description |
|----------|----------|-------------|
| `SLASH_SSO_{INDEX}_CLIENT_ID` | Yes | OAuth2 client ID |
| `SLASH_SSO_{INDEX}_CLIENT_SECRET` | Yes | OAuth2 client secret |
| `SLASH_SSO_{INDEX}_ISSUER_URL` | Yes | OIDC issuer URL (e.g., `https://accounts.google.com`) |
| `SLASH_SSO_{INDEX}_TITLE` | No | Display name (default: `SSO {INDEX}`) |
| `SLASH_SSO_{INDEX}_IDENTIFIER_FIELD` | No | User info field for email (default: `email`) |
| `SLASH_SSO_{INDEX}_DISPLAY_NAME_FIELD` | No | User info field for name (default: `name`) |
| `SLASH_SSO_{INDEX}_SCOPES` | No | OAuth scopes, space-separated (default: `openid profile email`) |

**Note:** `{INDEX}` is the provider index starting from 0. Use `0` for the first provider, `1` for the second, etc.

### Auto-discovery

When `SLASH_SSO_{INDEX}_ISSUER_URL` is set, Slash automatically fetches the following from `{ISSUER_URL}/.well-known/openid-configuration`:

- Authorization endpoint
- Token endpoint
- User info endpoint

You don't need to configure these manually unless your provider requires non-standard endpoints.

### Docker Compose Example

```yaml
services:
  slash:
    image: yourselfhosted/slash:latest
    environment:
      # SSO via Google
      - SLASH_SSO_0_CLIENT_ID=your-google-client-id.apps.googleusercontent.com
      - SLASH_SSO_0_CLIENT_SECRET=your-google-client-secret
      - SLASH_SSO_0_ISSUER_URL=https://accounts.google.com
      - SLASH_SSO_0_TITLE=Google SSO
```

### Multiple Providers

You can configure multiple SSO providers by incrementing the index:

```bash
# First provider (Google)
SLASH_SSO_0_CLIENT_ID=google-client-id
SLASH_SSO_0_CLIENT_SECRET=google-client-secret
SLASH_SSO_0_ISSUER_URL=https://accounts.google.com

# Second provider (Okta)
SLASH_SSO_1_CLIENT_ID=okta-client-id
SLASH_SSO_1_CLIENT_SECRET=okta-client-secret
SLASH_SSO_1_ISSUER_URL=https://your-org.okta.com
SLASH_SSO_1_TITLE=Okta SSO
```

### Priority

When SSO providers are configured via environment variables, they take precedence over database providers for faster authentication. If any required environment variable is missing, Slash will fall back to using database-configured SSO providers.
