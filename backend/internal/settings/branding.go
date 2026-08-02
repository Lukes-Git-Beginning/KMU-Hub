package settings

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	"github.com/google/uuid"
)

const (
	brandingModuleID      = "branding"
	brandingMaxNameLength = 200
	defaultAccentColor    = "#10B981"
)

// allowedAccentColors mirrors desktop/src/renderer/src/lib/swatch-colors.ts
// (SWATCH_COLORS). No codegen link between the two — the palette is small and
// changes rarely; keep both lists in sync by hand.
var allowedAccentColors = map[string]bool{
	"#6B7280": true,
	"#3B82F6": true,
	"#06B6D4": true,
	"#10B981": true,
	"#F59E0B": true,
	"#F97316": true,
	"#EF4444": true,
	"#8B5CF6": true,
	"#EC4899": true,
	"#6366F1": true,
}

// ErrInvalidAccentColor is returned when accent_color is not one of the Cosmi
// swatch colors — a free hex value would break the theme.
var ErrInvalidAccentColor = errors.New("settings: accent_color must be one of the Cosmi swatch colors")

// ErrInvalidBrandingObjectKey is returned when a logo/icon object key does not
// belong to this tenant's branding presign scope. Guards against a client
// wiring in an arbitrary object key (own or another tenant's) instead of one
// it actually received from GetPresignedUploadURL.
var ErrInvalidBrandingObjectKey = errors.New("settings: object key does not belong to this tenant's branding scope")

// ErrBrandingNameTooLong is returned when name exceeds brandingMaxNameLength.
var ErrBrandingNameTooLong = errors.New("settings: name exceeds maximum length")

// Branding is a tenant's workspace branding: display name, logo/icon object
// keys, and an accent color from the fixed swatch palette. It is stored as
// ordinary tenant_settings rows (module_id="branding") — no dedicated table.
type Branding struct {
	Name          string
	LogoObjectKey string
	IconObjectKey string
	AccentColor   string
}

// brandingObjectKeyValid reports whether key is empty (clearing the logo/icon)
// or prefixed with the requesting tenant's branding presign scope
// ({tenant_id}/branding/), the exact shape GetPresignedUploadURL produces.
func brandingObjectKeyValid(tenantID uuid.UUID, key string) bool {
	if key == "" {
		return true
	}
	return strings.HasPrefix(key, tenantID.String()+"/branding/")
}

// GetBranding returns the tenant's branding, defaulting AccentColor to the
// same value the frontend's mock (BrandingAdminHubTab) used before any write.
func (s *Service) GetBranding(ctx context.Context, tenantID uuid.UUID) (*Branding, error) {
	entries, err := s.repo.GetTenantSettings(ctx, tenantID, brandingModuleID)
	if err != nil {
		return nil, err
	}

	b := &Branding{AccentColor: defaultAccentColor}
	for _, e := range entries {
		var v string
		if err := json.Unmarshal(e.Value, &v); err != nil {
			continue
		}
		switch e.Key {
		case "name":
			b.Name = v
		case "logoObjectKey":
			b.LogoObjectKey = v
		case "iconObjectKey":
			b.IconObjectKey = v
		case "accentColor":
			b.AccentColor = v
		}
	}
	return b, nil
}

// PutBranding validates and writes the tenant's branding as a full replace
// (all four fields are always written, so omitting logo/icon on the wire
// clears it). RBAC — admin or module-lead for "branding" — is enforced by
// PutTenantSettings; no module-lead is ever granted for "branding" in
// practice, so this resolves to admin-only.
func (s *Service) PutBranding(ctx context.Context, tenantID, callerID uuid.UUID, b *Branding) (*Branding, error) {
	if len(b.Name) > brandingMaxNameLength {
		return nil, ErrBrandingNameTooLong
	}
	if !allowedAccentColors[b.AccentColor] {
		return nil, ErrInvalidAccentColor
	}
	if !brandingObjectKeyValid(tenantID, b.LogoObjectKey) || !brandingObjectKeyValid(tenantID, b.IconObjectKey) {
		return nil, ErrInvalidBrandingObjectKey
	}

	entries := []*SettingEntry{
		brandingEntry("name", b.Name),
		brandingEntry("logoObjectKey", b.LogoObjectKey),
		brandingEntry("iconObjectKey", b.IconObjectKey),
		brandingEntry("accentColor", b.AccentColor),
	}

	if _, err := s.PutTenantSettings(ctx, tenantID, callerID, brandingModuleID, entries); err != nil {
		return nil, err
	}
	return s.GetBranding(ctx, tenantID)
}

func brandingEntry(key, value string) *SettingEntry {
	// json.Marshal on a string never fails.
	raw, _ := json.Marshal(value)
	return &SettingEntry{Key: key, Value: raw}
}
