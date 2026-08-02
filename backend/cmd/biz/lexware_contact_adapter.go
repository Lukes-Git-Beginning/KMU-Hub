package main

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/kmuhub/kmuhub/internal/biz/bexio"
	"github.com/kmuhub/kmuhub/internal/biz/lexware"
)

// lexwareContactAdapter exposes the CRM contact adapter under the Lexware
// integration's ContactService interface.  Both interfaces are structurally
// identical but use their own package-local result types, so this wrapper only
// translates types — the gRPC calls themselves live in crmContactAdapter and
// are not duplicated here.
type lexwareContactAdapter struct {
	inner *crmContactAdapter
}

// Verify at compile time that lexwareContactAdapter satisfies lexware.ContactService.
var _ lexware.ContactService = (*lexwareContactAdapter)(nil)

func (a *lexwareContactAdapter) GetByID(ctx context.Context, id uuid.UUID) (*lexware.ContactResult, error) {
	res, err := a.inner.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return toLexwareContactResult(res), nil
}

func (a *lexwareContactAdapter) GetByEmail(ctx context.Context, email string) (*lexware.ContactResult, error) {
	res, err := a.inner.GetByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return toLexwareContactResult(res), nil
}

func (a *lexwareContactAdapter) CreateForSync(ctx context.Context, data *lexware.ContactSyncData, createdBy uuid.UUID) (uuid.UUID, error) {
	return a.inner.CreateForSync(ctx, toBexioContactSyncData(data), createdBy)
}

func (a *lexwareContactAdapter) UpdateForSync(ctx context.Context, id uuid.UUID, data *lexware.ContactSyncData) error {
	return a.inner.UpdateForSync(ctx, id, toBexioContactSyncData(data))
}

func (a *lexwareContactAdapter) ListModifiedSince(ctx context.Context, since time.Time) ([]lexware.ContactResult, error) {
	res, err := a.inner.ListModifiedSince(ctx, since)
	if err != nil {
		return nil, err
	}
	out := make([]lexware.ContactResult, 0, len(res))
	for i := range res {
		out = append(out, *toLexwareContactResult(&res[i]))
	}
	return out, nil
}

func toLexwareContactResult(c *bexio.ContactResult) *lexware.ContactResult {
	if c == nil {
		return nil
	}
	return &lexware.ContactResult{
		ID:          c.ID,
		FirstName:   c.FirstName,
		LastName:    c.LastName,
		Email:       c.Email,
		Phone:       c.Phone,
		CompanyName: c.CompanyName,
		Notes:       c.Notes,
		UpdatedAt:   c.UpdatedAt,
	}
}

func toBexioContactSyncData(d *lexware.ContactSyncData) *bexio.ContactSyncData {
	if d == nil {
		return nil
	}
	return &bexio.ContactSyncData{
		FirstName: d.FirstName,
		LastName:  d.LastName,
		Email:     d.Email,
		Phone:     d.Phone,
		Mobile:    d.Mobile,
		Address:   d.Address,
		City:      d.City,
		Zip:       d.Zip,
		Country:   d.Country,
		Notes:     d.Notes,
		IsCompany: d.IsCompany,
	}
}
