package helpdesk

// CsatConfig is a tenant's CSAT (customer satisfaction) survey configuration:
// whether surveys go out at all, how long after ticket close to wait before
// sending one, and the question text shown to the customer. Stored via the
// generic tenant_settings mechanism (module_id CsatConfigModuleID, single
// entry key CsatConfigEntryKey) -- no dedicated table, same recipe as
// internal/settings.Branding.
//
// The 1..5 rating scale itself is NOT a field here and is not configurable:
// it is fixed by the ticket_csat_responses CHECK constraint and enforced
// again in SubmitCsat (ErrInvalidCsatRating).
type CsatConfig struct {
	Enabled            bool
	SurveyDelayMinutes int32
	SurveyQuestion     string
}

const (
	// CsatConfigModuleID is the tenant_settings module_id under which the
	// CSAT survey configuration is stored.
	CsatConfigModuleID = "helpdesk_csat"
	// CsatConfigEntryKey is the single tenant_settings entry key holding the
	// whole config as one JSON object (a full-replace form, not sparse --
	// there is exactly one CSAT config per tenant, not one row per locale).
	CsatConfigEntryKey = "config"

	csatSurveyQuestionMaxLength = 500
	// csatSurveyDelayMaxMinutes caps the delay at 14 days: long enough for any
	// reasonable follow-up cadence, short enough that a fat-fingered value
	// doesn't leave a survey token dangling for months.
	csatSurveyDelayMaxMinutes = 14 * 24 * 60
)

// DefaultCsatConfig is what a tenant gets before it has ever written a row.
//
// Surveys are OFF by default -- opt-in, not opt-out. Two reasons, and both
// have to go away before this may flip:
//
//  1. The invitation carries a link built from CSAT_SURVEY_BASE_URL (default
//     https://app.zentria.tech/csat), and nothing is served there: the domain
//     proxies wholesale to the gateway, and Cosmi has no public web surface.
//     A survey that goes out today lands its recipient on a bare 404. The
//     recipient is resolved as ticket.requester_email or, failing that, the
//     requesting user's own address -- so that includes staff, not just
//     external customers.
//  2. There is no gateway route for this config. Get/UpdateCsatConfig exist as
//     RPCs but are not exposed, and the desktop panel's toggle is local store
//     state with no backend call -- meaning a tenant handed surveys by default
//     would have no way to switch them off.
//
// The delay and question below are the values a tenant starts from once it
// does switch surveys on, not a dormant schedule: IssueCsatSurveyToken parks
// no token at all while Enabled is false.
func DefaultCsatConfig() CsatConfig {
	return CsatConfig{
		Enabled:            false,
		SurveyDelayMinutes: 24 * 60,
		SurveyQuestion:     "Wie zufrieden waren Sie mit der Bearbeitung Ihres Anliegens?",
	}
}

// ValidateCsatConfig rejects a delay outside [0, csatSurveyDelayMaxMinutes]
// and a survey question over csatSurveyQuestionMaxLength characters. Enabled
// has no invalid value.
func ValidateCsatConfig(cfg CsatConfig) error {
	if cfg.SurveyDelayMinutes < 0 || cfg.SurveyDelayMinutes > csatSurveyDelayMaxMinutes {
		return ErrInvalidCsatDelay
	}
	if len(cfg.SurveyQuestion) > csatSurveyQuestionMaxLength {
		return ErrCsatQuestionTooLong
	}
	return nil
}
