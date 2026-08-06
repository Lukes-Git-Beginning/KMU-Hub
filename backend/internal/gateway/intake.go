package gateway

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// ============================================================================
// Intake engine — turning a form submission into a module record
// ============================================================================
//
// A form schema can declare which module record it feeds (`intake_target_id`,
// BACKLOG B6) and which of its fields fill that record's first-class
// properties (`FormField.role`). This file is the server-side half of the
// shared intake engine the FE builds on
// (desktop/src/renderer/src/components/shared/intake): the pure mapping plus a
// registry of targets. The Helpdesk target lives in intake_helpdesk.go and is
// the only entry today -- the engine itself knows nothing about tickets, which
// is the whole point of the split.
//
// Why the gateway and not the formulare service: the gateway holds the gRPC
// connections to every module. A cross-module create from inside
// internal/formulare would mean giving one service a client for another, a
// dependency direction this codebase does not have anywhere else.

const (
	// intakePageBreakLabel is the form builder's page separator. It is a field
	// row in `fields` but never an answer, so the mapper skips it (mirrors
	// PAGE_BREAK_LABEL in shared/intake/engine.ts).
	intakePageBreakLabel = "__page_break__"

	// intakeRoleExtra is the reserved role meaning "no role -- treat as an
	// extra", written by the builder's role dropdown when the author picks the
	// blank option.
	intakeRoleExtra = "extra"
)

// intakeField is the slice of a form field the mapper needs: which answer it
// owns (ID), what to call it as a custom field (Label) and which target
// property it fills, if any (Role).
type intakeField struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Role  string `json:"role"`
}

// intakeOrigin says where a submission came from and who sent it. It is the
// server-side counterpart of IntakeBuildContext in shared/intake/types.ts.
type intakeOrigin struct {
	// Channel is the intake origin ("agent" | "selfservice" | "external").
	// Empty is left to the target, which defaults it.
	Channel string
	// RequesterID is the submitting user, empty for unauthenticated intake.
	RequesterID string
	// RequesterName / RequesterEmail carry an external requester's identity
	// when the form did not collect one through a role.
	RequesterName  string
	RequesterEmail string
	// RequesterIsExternal, when set, overrides the channel-derived default
	// (everything but "selfservice" counts as external).
	RequesterIsExternal *bool
}

// isExternal resolves the flag the way the FE target does:
// `context.requester?.isExternal ?? channel !== 'selfservice'`.
func (o intakeOrigin) isExternal() bool {
	if o.RequesterIsExternal != nil {
		return *o.RequesterIsExternal
	}
	return o.Channel != intakeChannelSelfService
}

// intakeSubmission is the mapped form of one submission, handed to a target's
// create: role values in Mapped, everything unmarked in Extras.
type intakeSubmission struct {
	TenantID uuid.UUID
	Mapped   map[string]any
	Extras   map[string]any
	Origin   intakeOrigin
}

// intakeTarget is one module's registration in the intake registry: the roles
// its records expose plus how to create one. Targets are values, not
// interfaces, because a target is exactly this pair and nothing else.
type intakeTarget struct {
	id string
	// roles is the set of role keys this target understands. A field carrying
	// any other role is treated as unmarked, so a role left over from a
	// re-bound form degrades into a custom field instead of vanishing.
	roles map[string]struct{}
	// create turns a mapped submission into the module record and returns the
	// new record's id.
	create func(ctx context.Context, reg *ServiceRegistry, in intakeSubmission) (string, error)
}

// intakeTargets is the registry. Explicit map literal rather than an init()
// registration hook: the service registry in this codebase is wired by hand
// for the same reason (no hidden registration order).
var intakeTargets = map[string]*intakeTarget{
	intakeTargetHelpdeskTicket: helpdeskTicketIntakeTarget(),
}

// lookupIntakeTarget returns the registered target, or nil when a schema names
// one this build does not have (a form bound before a module was removed, or a
// row written by a newer version).
func lookupIntakeTarget(id string) *intakeTarget {
	return intakeTargets[strings.TrimSpace(id)]
}

// parseIntakeFields decodes a schema's `fields` JSONB array. An empty or null
// array is not an error -- it just maps to nothing.
func parseIntakeFields(raw []byte) ([]intakeField, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" || trimmed == "[]" {
		return nil, nil
	}
	var fields []intakeField
	if err := json.Unmarshal(raw, &fields); err != nil {
		return nil, fmt.Errorf("decode form fields: %w", err)
	}
	return fields, nil
}

// mapIntakeAnswers splits a submission's answers into role values and extras,
// the exact split mapSubmissionToRecord performs in shared/intake/engine.ts:
// answers are keyed by field id, extras by a slug of the field label.
func mapIntakeAnswers(target *intakeTarget, fields []intakeField, answers map[string]any) (mapped, extras map[string]any) {
	mapped = make(map[string]any)
	extras = make(map[string]any)

	for _, f := range fields {
		if f.Label == intakePageBreakLabel {
			continue
		}
		answer := answers[f.ID]

		role := strings.TrimSpace(f.Role)
		if role != "" && role != intakeRoleExtra {
			if _, known := target.roles[role]; known {
				mapped[role] = answer
				continue
			}
		}

		// Unmarked -> extra. Empty answers are dropped so the record's custom
		// fields stay lean; false and 0 are answers and are kept.
		value := intakeExtraValue(answer)
		if value == "" {
			continue
		}
		extras[slugifyIntakeKey(f.Label, f.ID)] = value
	}
	return mapped, extras
}

// intakeExtraValue coerces an answer into a scalar a module custom-field map
// accepts (helpdesk.normalizeCustomFields rejects nested values outright).
// Objects and arrays are re-encoded as JSON text rather than dropped: the
// answer is the user's, and a lossy string beats losing it.
func intakeExtraValue(v any) any {
	switch t := v.(type) {
	case nil:
		return ""
	case bool, float64:
		return t
	case string:
		return t
	default:
		encoded, err := json.Marshal(t)
		if err != nil {
			return fmt.Sprintf("%v", t)
		}
		return string(encoded)
	}
}

// slugifyIntakeKey turns a field label into a stable custom-field key
// ("Bestell-Nr." -> "bestell_nr"), falling back to the field id when the label
// slugifies to nothing. Byte-for-byte the same rules as slugifyKey in
// shared/intake/engine.ts, so a record created here and one created by the FE
// preview carry identical keys.
func slugifyIntakeKey(label, fallback string) string {
	var b strings.Builder
	for _, r := range strings.ToLower(strings.TrimSpace(label)) {
		switch {
		case r == 'ä':
			b.WriteString("ae")
		case r == 'ö':
			b.WriteString("oe")
		case r == 'ü':
			b.WriteString("ue")
		case r == 'ß':
			b.WriteString("ss")
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
		default:
			b.WriteByte('_')
		}
	}
	slug := strings.Trim(collapseUnderscores(b.String()), "_")
	if slug == "" {
		return fallback
	}
	return slug
}

// collapseUnderscores folds runs of "_" into one, matching the JS regex
// `[^a-z0-9]+` replacing a whole run with a single underscore.
func collapseUnderscores(s string) string {
	var b strings.Builder
	prev := false
	for _, r := range s {
		if r == '_' {
			if prev {
				continue
			}
			prev = true
		} else {
			prev = false
		}
		b.WriteRune(r)
	}
	return b.String()
}

// intakeSchemaLookup resolves the bound target and field list of the schema a
// submission belongs to. A function rather than a client call so the dispatch
// can be tested without a live formulare service.
type intakeSchemaLookup func(ctx context.Context) (targetID string, fields []byte, err error)

// runIntakeDispatch creates the module record a submitted form is bound to.
//
// It runs AFTER the submission is stored and has no way to touch it: a target
// that rejects the record (invalid channel, missing requester, module down)
// costs the caller a ticket, never their answers. Callers log the error and
// still answer 201 -- an intake form that swallowed the submission because the
// Helpdesk was briefly unreachable would be the worse failure by far.
//
// Authorisation note: the caller was authorised to submit the form
// (formulare:submissions:write), NOT to create the target record. That is the
// point of an intake form -- someone filing a support request holds no
// helpdesk:ticket:create grant. What keeps this from being a privilege
// escalation is that the shape of the record is fixed by the schema its author
// bound, and the tenant travels with the outbound call
// (TenantOutboundUnaryInterceptor), so nothing crosses a tenant boundary.
//
// Returns the new record's id, or "" when the schema has no intake binding.
func runIntakeDispatch(
	ctx context.Context,
	reg *ServiceRegistry,
	lookup intakeSchemaLookup,
	tenantID uuid.UUID,
	answers []byte,
	origin intakeOrigin,
) (string, error) {
	targetID, rawFields, err := lookup(ctx)
	if err != nil {
		return "", fmt.Errorf("load intake schema: %w", err)
	}
	if strings.TrimSpace(targetID) == "" {
		return "", nil // plain form, nothing to dispatch
	}

	target := lookupIntakeTarget(targetID)
	if target == nil {
		return "", fmt.Errorf("no intake target registered for %q", targetID)
	}

	fields, err := parseIntakeFields(rawFields)
	if err != nil {
		return "", err
	}

	decoded := map[string]any{}
	if trimmed := strings.TrimSpace(string(answers)); trimmed != "" && trimmed != "null" {
		if err := json.Unmarshal(answers, &decoded); err != nil {
			return "", fmt.Errorf("decode submission answers: %w", err)
		}
	}

	mapped, extras := mapIntakeAnswers(target, fields, decoded)

	recordID, err := target.create(ctx, reg, intakeSubmission{
		TenantID: tenantID,
		Mapped:   mapped,
		Extras:   extras,
		Origin:   origin,
	})
	if err != nil {
		return "", fmt.Errorf("create %s record: %w", target.id, err)
	}
	return recordID, nil
}
