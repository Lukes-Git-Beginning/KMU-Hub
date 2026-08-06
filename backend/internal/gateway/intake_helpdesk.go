package gateway

import (
	"context"
	"fmt"
	"strings"

	"google.golang.org/protobuf/types/known/structpb"

	helpdeskv1 "github.com/kmuhub/kmuhub/proto/helpdesk/v1"
)

// ============================================================================
// Helpdesk ticket — the first (and today only) intake target
// ============================================================================
//
// Mirrors desktop/src/renderer/src/modules/helpdesk/intake/helpdesk-ticket-target.ts
// field for field: same role keys, same priority coercion, same fallbacks. A
// form submitted through the FE preview and the same form submitted through
// POST /formulare/schemas/{id}/submissions must produce the same ticket, so
// the two mappings are deliberately kept identical rather than "equivalent".

// intakeTargetHelpdeskTicket is the registry key stored in
// form_schemas.intake_target_id (HELPDESK_TICKET_TARGET_ID on the FE side).
const intakeTargetHelpdeskTicket = "helpdesk_ticket"

// Role keys the Helpdesk target understands. Identical set to
// formulare.validIntakeRoles, which is what the schema service validates
// against on write -- this map is the read side of the same contract.
const (
	intakeRoleSubject        = "subject"
	intakeRoleDescription    = "description"
	intakeRolePriority       = "priority"
	intakeRoleCategory       = "category"
	intakeRoleRequesterName  = "requester_name"
	intakeRoleRequesterEmail = "requester_email"
)

// intakeChannelSelfService is the one channel that counts as internal intake;
// mirrors helpdesk.TicketChannelSelfService.
const intakeChannelSelfService = "selfservice"

// intakeChannelExternal is the channel a submission arriving without any login
// carries -- the public share-link route. Both values are members of
// helpdesk.ValidTicketChannels; a channel outside that set is refused by
// TicketIntake.normalize, which is where the list is authoritative.
const intakeChannelExternal = "external"

// intakeDefaultTicketSubject is used when the form has no subject role or the
// submitter left it blank -- a ticket without a subject is rejected by the
// service, and losing the submission over an empty field would be absurd.
const intakeDefaultTicketSubject = "Anfrage über Formular"

// helpdeskTicketIntakeTarget builds the registry entry. A constructor rather
// than a package-level literal so the roles map is built in one place next to
// the create that consumes it.
func helpdeskTicketIntakeTarget() *intakeTarget {
	return &intakeTarget{
		id: intakeTargetHelpdeskTicket,
		roles: map[string]struct{}{
			intakeRoleSubject:        {},
			intakeRoleDescription:    {},
			intakeRolePriority:       {},
			intakeRoleCategory:       {},
			intakeRoleRequesterName:  {},
			intakeRoleRequesterEmail: {},
		},
		create: createHelpdeskTicketFromIntake,
	}
}

// createHelpdeskTicketFromIntake turns a mapped submission into a ticket over
// the helpdesk gRPC client. Like every other gateway path it goes through the
// client, never a directly held service instance: a direct call would run
// outside the service's tenant context and read past RLS.
func createHelpdeskTicketFromIntake(ctx context.Context, reg *ServiceRegistry, in intakeSubmission) (string, error) {
	conn, err := reg.GetConnection("helpdesk")
	if err != nil {
		return "", fmt.Errorf("helpdesk connection: %w", err)
	}
	client := helpdeskv1.NewHelpdeskServiceClient(conn)

	subject := intakeString(in.Mapped[intakeRoleSubject])
	if subject == "" {
		subject = intakeDefaultTicketSubject
	}

	requesterName := intakeString(in.Mapped[intakeRoleRequesterName])
	if requesterName == "" {
		requesterName = strings.TrimSpace(in.Origin.RequesterName)
	}
	requesterEmail := intakeString(in.Mapped[intakeRoleRequesterEmail])
	if requesterEmail == "" {
		requesterEmail = strings.TrimSpace(in.Origin.RequesterEmail)
	}

	req := &helpdeskv1.CreateTicketRequest{
		TenantId:            in.TenantID.String(),
		RequesterId:         strings.TrimSpace(in.Origin.RequesterID),
		Subject:             subject,
		Priority:            coerceIntakePriority(in.Mapped[intakeRolePriority]),
		Channel:             optionalIntakeString(in.Origin.Channel),
		Description:         optionalIntakeString(intakeString(in.Mapped[intakeRoleDescription])),
		Category:            optionalIntakeString(intakeString(in.Mapped[intakeRoleCategory])),
		RequesterName:       optionalIntakeString(requesterName),
		RequesterEmail:      optionalIntakeString(requesterEmail),
		RequesterIsExternal: in.Origin.isExternal(),
	}

	if len(in.Extras) > 0 {
		custom, cfErr := structpb.NewStruct(in.Extras)
		if cfErr != nil {
			return "", fmt.Errorf("build custom fields: %w", cfErr)
		}
		req.CustomFields = custom
	}

	resp, err := client.CreateTicket(ctx, req)
	if err != nil {
		return "", err
	}
	return resp.GetId(), nil
}

// intakePriorities maps what a form answer may say -- wire value or German
// label from a select -- onto a ticket priority. Same table as coercePriority
// in helpdesk-ticket-target.ts.
var intakePriorities = map[string]string{
	"low":      "low",
	"niedrig":  "low",
	"normal":   "normal",
	"mittel":   "normal",
	"medium":   "normal",
	"high":     "high",
	"hoch":     "high",
	"urgent":   "urgent",
	"dringend": "urgent",
	"kritisch": "urgent",
}

// coerceIntakePriority returns "" for anything unrecognised, which the helpdesk
// service reads as "normal". Guessing at an unknown label would be worse: a
// mistyped option silently raising a ticket to urgent is a support queue
// distorted by a typo.
func coerceIntakePriority(v any) string {
	return intakePriorities[strings.ToLower(intakeString(v))]
}

// intakeString renders an answer as trimmed text. Numbers arrive as float64
// from encoding/json; %v keeps whole numbers whole (5 -> "5", not "5e+00").
func intakeString(v any) string {
	switch t := v.(type) {
	case nil:
		return ""
	case string:
		return strings.TrimSpace(t)
	case float64:
		if t == float64(int64(t)) {
			return fmt.Sprintf("%d", int64(t))
		}
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	default:
		return strings.TrimSpace(fmt.Sprintf("%v", t))
	}
}

// optionalIntakeString maps "" to nil so an unanswered field stays absent on
// the wire instead of overwriting a column with an empty string.
func optionalIntakeString(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}
