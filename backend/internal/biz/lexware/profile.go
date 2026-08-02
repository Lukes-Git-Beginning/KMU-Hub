package lexware

import (
	"context"
	"fmt"
	"net/http"

	"github.com/google/uuid"
)

// LexwareProfile is the response of GET /v1/profile — the organisation the
// stored API key belongs to. It is the cheapest authenticated endpoint of the
// Lexware Office API and therefore what TestConnection probes.
type LexwareProfile struct {
	OrganizationID string `json:"organizationId,omitempty"`
	CompanyName    string `json:"companyName,omitempty"`
	Created        string `json:"created,omitempty"`
	ConnectionID   string `json:"connectionId,omitempty"`
	TaxType        string `json:"taxType,omitempty"`
}

// GetProfile fetches the organisation profile for the tenant's stored API key.
// A successful response proves the key is present, valid and accepted by the
// Lexware API — unlike a "the key field is not empty" check, which proves
// nothing about the credential itself.
func (c *Client) GetProfile(ctx context.Context, tenantID uuid.UUID) (*LexwareProfile, error) {
	var result LexwareProfile
	if err := c.do(ctx, tenantID, http.MethodGet, "v1/profile", nil, &result); err != nil {
		return nil, fmt.Errorf("lexware: get profile: %w", err)
	}
	return &result, nil
}
