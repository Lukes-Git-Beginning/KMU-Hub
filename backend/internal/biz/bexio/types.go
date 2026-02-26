package bexio

import (
	"context"

	"github.com/google/uuid"
)

// ClientConfig holds the configuration for the Bexio API client.
type ClientConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	AuthURL      string
	TokenURL     string
	RedirectURL  string
	Scopes       []string
}

// DefaultClientConfig returns a ClientConfig with Bexio default endpoints.
func DefaultClientConfig(clientID, clientSecret, redirectURL string) ClientConfig {
	return ClientConfig{
		BaseURL:      "https://api.bexio.com/2.0/",
		ClientID:     clientID,
		ClientSecret: clientSecret,
		AuthURL:      "https://auth.bexio.com/realms/bexio/protocol/openid-connect/auth",
		TokenURL:     "https://auth.bexio.com/realms/bexio/protocol/openid-connect/token",
		RedirectURL:  redirectURL,
		Scopes:       []string{"openid", "profile", "offline_access", "contact_edit", "kb_invoice_edit", "kb_offer_edit"},
	}
}

// VaultService defines the vault operations needed by the Bexio token manager.
type VaultService interface {
	GetSecret(ctx context.Context, keyName string) (string, error)
	SetSecret(ctx context.Context, keyName, plaintext, description string, createdBy uuid.UUID) error
}
