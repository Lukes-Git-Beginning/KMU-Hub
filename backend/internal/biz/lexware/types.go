package lexware

import (
	"context"

	"github.com/google/uuid"
)

type ClientConfig struct {
	BaseURL string
}

func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		BaseURL: "https://api.lexware.io",
	}
}

type VaultService interface {
	GetSecret(ctx context.Context, keyName string) (string, error)
	SetSecret(ctx context.Context, keyName, plaintext, description string, createdBy uuid.UUID) error
}
