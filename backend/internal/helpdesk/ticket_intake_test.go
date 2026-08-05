package helpdesk

// BACKLOG B2. The module has been sending channel, requester_email,
// requester_is_external, requester_name and custom_fields on every intake
// create; nothing persisted them, and because the decode path answered 200 the
// loss was invisible. These tests are the proof that it now round-trips, and
// that a bad channel is refused rather than quietly turned into "agent" --
// a silent default would only move the data loss one layer down.

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/kmuhub/kmuhub/internal/testutil"
)

func strPtr(s string) *string { return &s }

func TestCreateTicket_RejectsUnknownChannel(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()),
		"Subject", TicketPriorityNormal, nil, nil, "", "", nil, nil,
		TicketIntake{Channel: "carrier-pigeon"})

	if !errors.Is(err, ErrInvalidChannel) {
		t.Fatalf("err = %v, want ErrInvalidChannel", err)
	}
}

func TestCreateTicket_EmptyChannelDefaultsToAgent(t *testing.T) {
	h := newTestHarness()

	ticket, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()),
		"Subject", TicketPriorityNormal, nil, nil, "", "", nil, nil, TicketIntake{})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if ticket.Channel != TicketChannelAgent {
		t.Errorf("channel = %q, want %q", ticket.Channel, TicketChannelAgent)
	}
	if ticket.CustomFields == nil {
		t.Error("custom_fields is nil, want an empty map (nil serialises as JSON null)")
	}
}

func TestCreateTicket_RejectsMalformedRequesterEmail(t *testing.T) {
	h := newTestHarness()

	_, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()),
		"Subject", TicketPriorityNormal, nil, nil, "", "", nil, nil,
		TicketIntake{RequesterEmail: strPtr("not-an-address")})

	if !errors.Is(err, ErrInvalidRequesterEmail) {
		t.Fatalf("err = %v, want ErrInvalidRequesterEmail", err)
	}
}

func TestCreateTicket_RejectsNonScalarCustomFields(t *testing.T) {
	cases := map[string]map[string]any{
		"nested object": {"adresse": map[string]any{"stadt": "Bern"}},
		"array":         {"tags": []any{"a", "b"}},
		"null":          {"leer": nil},
		"empty key":     {"": "wert"},
	}

	for name, fields := range cases {
		t.Run(name, func(t *testing.T) {
			h := newTestHarness()
			_, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()),
				"Subject", TicketPriorityNormal, nil, nil, "", "", nil, nil,
				TicketIntake{CustomFields: fields})
			if !errors.Is(err, ErrInvalidCustomFields) {
				t.Fatalf("err = %v, want ErrInvalidCustomFields", err)
			}
		})
	}
}

// An internal requester's name is resolved from their user row. Persisting a
// copy is how it goes stale at the first rename, so CreateTicket must drop it
// rather than store it -- the read-side precedence rule alone would not stop
// the stale copy from existing.
func TestCreateTicket_DropsRequesterNameForInternalRequester(t *testing.T) {
	h := newTestHarness()

	ticket, err := h.svc.CreateTicket(context.Background(), uuid.New(), uuidPtr(uuid.New()),
		"Subject", TicketPriorityNormal, nil, nil, "", "", nil, nil,
		TicketIntake{RequesterName: strPtr("Ines Intern")})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	if ticket.RequesterName != "" {
		t.Errorf("requester_name = %q, want empty for an internal requester", ticket.RequesterName)
	}
}

// The round trip that the whole unit exists for: everything the module sends
// on an intake create has to come back off the database unchanged.
func TestCreateTicket_IntakeFieldsRoundTrip(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk Intake RoundTrip Tenant")

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	created, err := svc.CreateTicket(ctx, tenantID, uuidPtr(uuid.New()),
		"Drucker streikt", TicketPriorityHigh, nil, nil, "Papierstau", "hardware", nil, nil,
		TicketIntake{
			Channel:             TicketChannelExternal,
			RequesterEmail:      strPtr("  erik@extern.example  "),
			RequesterName:       strPtr("Erik Extern"),
			RequesterIsExternal: true,
			CustomFields: map[string]any{
				"standort":     "Bern",
				"seriennummer": float64(4711),
				"garantie":     true,
			},
		})
	if err != nil {
		t.Fatalf("CreateTicket: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", created.ID)

	got, err := repo.GetTicketByID(ctx, created.ID, tenantID)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}

	if got.Channel != TicketChannelExternal {
		t.Errorf("channel = %q, want %q", got.Channel, TicketChannelExternal)
	}
	// Trimmed on the way in: a padded address is not a different address.
	if got.RequesterEmail == nil || *got.RequesterEmail != "erik@extern.example" {
		t.Errorf("requester_email = %v, want %q", got.RequesterEmail, "erik@extern.example")
	}
	if !got.RequesterIsExternal {
		t.Error("requester_is_external = false, want true")
	}
	// External requester has no user row, so the JOIN misses and the stored
	// column is what the read resolves to.
	if got.RequesterName != "Erik Extern" {
		t.Errorf("requester_name = %q, want %q", got.RequesterName, "Erik Extern")
	}
	if len(got.CustomFields) != 3 {
		t.Fatalf("custom_fields = %v, want 3 entries", got.CustomFields)
	}
	if got.CustomFields["standort"] != "Bern" {
		t.Errorf("custom_fields[standort] = %v, want Bern", got.CustomFields["standort"])
	}
	if got.CustomFields["garantie"] != true {
		t.Errorf("custom_fields[garantie] = %v, want true", got.CustomFields["garantie"])
	}
	// JSON numbers come back as float64; the module's union allows numbers, so
	// the value must survive as one rather than as its string rendering.
	if n, ok := got.CustomFields["seriennummer"].(float64); !ok || n != 4711 {
		t.Errorf("custom_fields[seriennummer] = %#v, want float64(4711)", got.CustomFields["seriennummer"])
	}
}

// Tickets converted from the inbox keep channel "agent": how a message reached
// the INBOX is a different question from how the request reached the helpdesk,
// and the column is NOT NULL either way.
func TestCreateTicketFromMessage_SetsAgentChannel(t *testing.T) {
	testutil.SkipIfNoDB(t)
	t.Parallel()

	pool := testutil.PoolFromEnv(t)
	defer pool.Close()

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Helpdesk Intake FromMessage Tenant")

	requesterID := testutil.SeedRow(t, pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("hd-intake-%s@frommessage.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	defer testutil.CleanupRow(t, pool, "users", requesterID)

	messageID := testutil.SeedRow(t, pool, "inbox_messages", map[string]any{
		"tenant_id":   tenantID,
		"user_id":     requesterID,
		"channel":     "email",
		"source_id":   fmt.Sprintf("msg-%s", uuid.New().String()[:8]),
		"subject":     "Rechnung offen",
		"preview":     "Guten Tag ...",
		"received_at": time.Now().UTC(),
	})
	defer testutil.CleanupRow(t, pool, "inbox_messages", messageID)

	repo := NewPostgresRepository(pool)
	svc := NewService(repo, testLogger())
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)

	ticket, _, err := svc.CreateTicketFromMessage(ctx, tenantID, requesterID, messageID,
		"email", "Rechnung offen", "Guten Tag ...")
	if err != nil {
		t.Fatalf("CreateTicketFromMessage: %v", err)
	}
	defer testutil.CleanupRow(t, pool, "tickets", ticket.ID)

	got, err := repo.GetTicketByID(ctx, ticket.ID, tenantID)
	if err != nil {
		t.Fatalf("GetTicketByID: %v", err)
	}
	if got.Channel != TicketChannelAgent {
		t.Errorf("channel = %q, want %q", got.Channel, TicketChannelAgent)
	}
	if got.SourceChannel == nil || *got.SourceChannel != "email" {
		t.Errorf("source_channel = %v, want email (the two columns must not have been conflated)", got.SourceChannel)
	}
}

