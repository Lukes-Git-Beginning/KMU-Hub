package dialer

import (
	"fmt"

	"github.com/kmuhub/kmuhub/internal/dachfmt"
)

// NormalizePhoneE164 normalises raw to E.164 format using DACH country codes.
//
// It delegates to internal/dachfmt, the single source of truth for DACH phone
// formatting, which is shared with the input-validation layer. See
// dachfmt.NormalizePhoneE164 for the accepted input forms.
func NormalizePhoneE164(raw string, defaultCountry string) (string, error) {
	return dachfmt.NormalizePhoneE164(raw, defaultCountry)
}

// FormatDuration formats a call duration given in seconds as "M:SS" for
// display in CRM activity subjects and UI labels (e.g. 90 → "1:30").
func FormatDuration(seconds int) string {
	m := seconds / 60
	s := seconds % 60
	return fmt.Sprintf("%d:%02d", m, s)
}
