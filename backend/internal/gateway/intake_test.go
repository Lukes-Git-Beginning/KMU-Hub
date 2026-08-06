package gateway

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
)

// testIntakeTarget builds a throwaway target whose create records what it was
// handed, so the mapping can be asserted without a live module behind it.
func testIntakeTarget(t *testing.T, roles []string, err error) (*intakeTarget, *intakeSubmission, *int) {
	t.Helper()
	var got intakeSubmission
	calls := 0
	roleSet := make(map[string]struct{}, len(roles))
	for _, r := range roles {
		roleSet[r] = struct{}{}
	}
	return &intakeTarget{
		id:    "test_target",
		roles: roleSet,
		create: func(_ context.Context, _ *ServiceRegistry, in intakeSubmission) (string, error) {
			calls++
			got = in
			if err != nil {
				return "", err
			}
			return "record-1", nil
		},
	}, &got, &calls
}

// withTestTarget registers a target under an id for the duration of a test.
func withTestTarget(t *testing.T, id string, target *intakeTarget) {
	t.Helper()
	previous, existed := intakeTargets[id]
	intakeTargets[id] = target
	t.Cleanup(func() {
		if existed {
			intakeTargets[id] = previous
			return
		}
		delete(intakeTargets, id)
	})
}

func staticLookup(targetID string, fields []byte) intakeSchemaLookup {
	return func(context.Context) (string, []byte, error) {
		return targetID, fields, nil
	}
}

// ============================================================================
// Mapping
// ============================================================================

func TestMapIntakeAnswers_RolesAndExtras(t *testing.T) {
	target, _, _ := testIntakeTarget(t, []string{"subject", "priority"}, nil)
	fields := []intakeField{
		{ID: "f1", Label: "Betreff", Role: "subject"},
		{ID: "f2", Label: "Dringlichkeit", Role: "priority"},
		{ID: "f3", Label: "Bestell-Nr."},
		{ID: "f4", Label: "Rückruf gewünscht", Role: intakeRoleExtra},
	}
	answers := map[string]any{
		"f1": "Drucker kaputt",
		"f2": "hoch",
		"f3": float64(4711),
		"f4": true,
	}

	mapped, extras := mapIntakeAnswers(target, fields, answers)

	if mapped["subject"] != "Drucker kaputt" || mapped["priority"] != "hoch" {
		t.Fatalf("role values not mapped: %#v", mapped)
	}
	if len(mapped) != 2 {
		t.Fatalf("expected exactly the two role values, got %#v", mapped)
	}
	if extras["bestell_nr"] != float64(4711) {
		t.Errorf("unmarked field missing from extras: %#v", extras)
	}
	// The reserved "extra" role means "no role" -- it must land in extras, not
	// in mapped under the literal key "extra".
	if extras["rueckruf_gewuenscht"] != true {
		t.Errorf("explicit extra role not treated as extra: %#v", extras)
	}
}

func TestMapIntakeAnswers_SkipsPageBreaksAndEmptyAnswers(t *testing.T) {
	target, _, _ := testIntakeTarget(t, []string{"subject"}, nil)
	fields := []intakeField{
		{ID: "f1", Label: intakePageBreakLabel},
		{ID: "f2", Label: "Kommentar"},
		{ID: "f3", Label: "Nicht beantwortet"},
		{ID: "f4", Label: "Null-Antwort"},
		{ID: "f5", Label: "Nullwert"},
	}
	answers := map[string]any{
		"f1": "sollte nie ankommen",
		"f2": "",
		"f4": nil,
		"f5": float64(0),
	}

	_, extras := mapIntakeAnswers(target, fields, answers)

	for _, key := range []string{"__page_break__", "kommentar", "nicht_beantwortet", "null_antwort"} {
		if _, ok := extras[key]; ok {
			t.Errorf("expected %q to be dropped, extras = %#v", key, extras)
		}
	}
	// 0 and false are answers, not absences.
	if extras["nullwert"] != float64(0) {
		t.Errorf("numeric zero must survive as an answer: %#v", extras)
	}
}

func TestMapIntakeAnswers_UnknownRoleDegradesToExtra(t *testing.T) {
	// A form re-bound to another target keeps roles the new target never
	// declared. Those answers must still reach the record as custom fields
	// rather than disappear between two valid states.
	target, _, _ := testIntakeTarget(t, []string{"subject"}, nil)
	fields := []intakeField{{ID: "f1", Label: "Antragsteller", Role: "applicant"}}

	mapped, extras := mapIntakeAnswers(target, fields, map[string]any{"f1": "Meier"})

	if len(mapped) != 0 {
		t.Fatalf("unknown role must not be mapped: %#v", mapped)
	}
	if extras["antragsteller"] != "Meier" {
		t.Fatalf("unknown role must degrade to an extra: %#v", extras)
	}
}

func TestSlugifyIntakeKey(t *testing.T) {
	cases := []struct {
		label, fallback, want string
	}{
		{"Bestell-Nr.", "f1", "bestell_nr"},
		{"Größe / Länge", "f2", "groesse_laenge"},
		{"  Mehrere   Wörter  ", "f3", "mehrere_woerter"},
		{"straße", "f4", "strasse"},
		{"???", "f5", "f5"},
		{"", "f6", "f6"},
	}
	for _, c := range cases {
		if got := slugifyIntakeKey(c.label, c.fallback); got != c.want {
			t.Errorf("slugifyIntakeKey(%q) = %q, want %q", c.label, got, c.want)
		}
	}
}

func TestIntakeExtraValue_NestedAnswerIsFlattened(t *testing.T) {
	// helpdesk.normalizeCustomFields rejects nested values outright, so a
	// multi-select answer has to arrive as text or the whole ticket is lost.
	got := intakeExtraValue([]any{"a", "b"})
	if got != `["a","b"]` {
		t.Fatalf("nested answer = %#v, want JSON text", got)
	}
}

// ============================================================================
// Dispatch
// ============================================================================

func TestRunIntakeDispatch_UnboundSchemaDoesNothing(t *testing.T) {
	target, _, calls := testIntakeTarget(t, []string{"subject"}, nil)
	withTestTarget(t, "test_target", target)

	id, err := runIntakeDispatch(context.Background(), nil, staticLookup("", nil), uuid.New(), []byte(`{}`), intakeOrigin{})
	if err != nil {
		t.Fatalf("unbound schema must not error: %v", err)
	}
	if id != "" {
		t.Errorf("unbound schema must not produce a record, got %q", id)
	}
	if *calls != 0 {
		t.Errorf("target was called for an unbound schema")
	}
}

func TestRunIntakeDispatch_MapsAndCreates(t *testing.T) {
	target, got, calls := testIntakeTarget(t, []string{"subject"}, nil)
	withTestTarget(t, "test_target", target)

	tenantID := uuid.New()
	fields := []byte(`[{"id":"f1","label":"Betreff","role":"subject"},{"id":"f2","label":"Notiz"}]`)
	answers := []byte(`{"f1":"Kaputt","f2":"seit gestern"}`)

	id, err := runIntakeDispatch(context.Background(), nil, staticLookup("test_target", fields), tenantID, answers,
		intakeOrigin{Channel: intakeChannelSelfService, RequesterID: "user-1"})
	if err != nil {
		t.Fatalf("dispatch: %v", err)
	}
	if id != "record-1" || *calls != 1 {
		t.Fatalf("expected exactly one created record, got id=%q calls=%d", id, *calls)
	}
	if got.TenantID != tenantID {
		t.Errorf("tenant not carried into the target: %v", got.TenantID)
	}
	if got.Mapped["subject"] != "Kaputt" || got.Extras["notiz"] != "seit gestern" {
		t.Errorf("submission mapped wrong: mapped=%#v extras=%#v", got.Mapped, got.Extras)
	}
	if got.Origin.isExternal() {
		t.Errorf("selfservice submission must not count as external intake")
	}
}

func TestRunIntakeDispatch_TargetFailureIsReportedNotSwallowed(t *testing.T) {
	// The submission is already stored when this runs; the caller logs the
	// error and still answers 201. What must not happen is the error being
	// dropped here, leaving a form that quietly stops producing tickets.
	boom := errors.New("helpdesk unavailable")
	target, _, calls := testIntakeTarget(t, []string{"subject"}, boom)
	withTestTarget(t, "test_target", target)

	fields := []byte(`[{"id":"f1","label":"Betreff","role":"subject"}]`)
	id, err := runIntakeDispatch(context.Background(), nil, staticLookup("test_target", fields), uuid.New(),
		[]byte(`{"f1":"Kaputt"}`), intakeOrigin{})
	if !errors.Is(err, boom) {
		t.Fatalf("target error must reach the caller, got %v", err)
	}
	if id != "" {
		t.Errorf("failed create must not report a record id, got %q", id)
	}
	if *calls != 1 {
		t.Errorf("target should have been called exactly once, got %d", *calls)
	}
}

func TestRunIntakeDispatch_UnknownTargetIsAnError(t *testing.T) {
	id, err := runIntakeDispatch(context.Background(), nil, staticLookup("no_such_target", nil), uuid.New(), []byte(`{}`), intakeOrigin{})
	if err == nil {
		t.Fatal("a schema bound to an unregistered target must report an error")
	}
	if id != "" {
		t.Errorf("got record id %q for an unknown target", id)
	}
}

func TestRunIntakeDispatch_BadPayloadsDoNotPanic(t *testing.T) {
	target, _, calls := testIntakeTarget(t, []string{"subject"}, nil)
	withTestTarget(t, "test_target", target)

	t.Run("broken fields", func(t *testing.T) {
		if _, err := runIntakeDispatch(context.Background(), nil, staticLookup("test_target", []byte(`{"not":"an array"}`)),
			uuid.New(), []byte(`{}`), intakeOrigin{}); err == nil {
			t.Fatal("expected an error for a non-array fields payload")
		}
	})

	t.Run("broken answers", func(t *testing.T) {
		if _, err := runIntakeDispatch(context.Background(), nil, staticLookup("test_target", []byte(`[]`)),
			uuid.New(), []byte(`not json`), intakeOrigin{}); err == nil {
			t.Fatal("expected an error for a non-object answers payload")
		}
	})

	t.Run("empty payloads", func(t *testing.T) {
		before := *calls
		if _, err := runIntakeDispatch(context.Background(), nil, staticLookup("test_target", nil),
			uuid.New(), nil, intakeOrigin{}); err != nil {
			t.Fatalf("empty fields/answers must map to an empty record, got %v", err)
		}
		if *calls != before+1 {
			t.Errorf("target should still be called for an empty submission")
		}
	})
}

func TestRunIntakeDispatch_LookupFailureStopsBeforeCreate(t *testing.T) {
	target, _, calls := testIntakeTarget(t, []string{"subject"}, nil)
	withTestTarget(t, "test_target", target)

	failing := func(context.Context) (string, []byte, error) {
		return "", nil, errors.New("schema gone")
	}
	if _, err := runIntakeDispatch(context.Background(), nil, failing, uuid.New(), []byte(`{}`), intakeOrigin{}); err == nil {
		t.Fatal("expected the lookup error to surface")
	}
	if *calls != 0 {
		t.Errorf("no record may be created when the schema could not be read")
	}
}

// ============================================================================
// Helpdesk target
// ============================================================================

func TestHelpdeskTicketTarget_IsRegisteredWithTheFERoles(t *testing.T) {
	target := lookupIntakeTarget(intakeTargetHelpdeskTicket)
	if target == nil {
		t.Fatal("helpdesk_ticket target is not registered")
		return // unreachable; keeps the nil-deref analysis happy
	}
	// The list the FE role dropdown offers (helpdesk-ticket-target.ts) and the
	// list formulare.validIntakeRoles validates against on write.
	want := []string{"subject", "description", "priority", "category", "requester_name", "requester_email"}
	if len(target.roles) != len(want) {
		t.Fatalf("role count drifted from the FE contract: %#v", target.roles)
	}
	for _, role := range want {
		if _, ok := target.roles[role]; !ok {
			t.Errorf("role %q missing from the helpdesk target", role)
		}
	}
}

func TestCoerceIntakePriority(t *testing.T) {
	cases := map[any]string{
		"hoch":     "high",
		"Dringend": "urgent",
		"normal":   "normal",
		"mittel":   "normal",
		"low":      "low",
		"":         "",
		nil:        "",
		"vielleicht wichtig": "",
	}
	for in, want := range cases {
		if got := coerceIntakePriority(in); got != want {
			t.Errorf("coerceIntakePriority(%#v) = %q, want %q", in, got, want)
		}
	}
}

func TestIntakeString_WholeNumbersStayWhole(t *testing.T) {
	if got := intakeString(float64(5)); got != "5" {
		t.Errorf("intakeString(5) = %q, want \"5\"", got)
	}
	if got := intakeString(float64(2.5)); got != "2.5" {
		t.Errorf("intakeString(2.5) = %q, want \"2.5\"", got)
	}
}

func TestIntakeOrigin_ExternalDefaultsFromChannel(t *testing.T) {
	if (intakeOrigin{Channel: intakeChannelSelfService}).isExternal() {
		t.Error("selfservice must default to internal")
	}
	if !(intakeOrigin{Channel: "external"}).isExternal() {
		t.Error("external channel must default to external")
	}
	internal := false
	if (intakeOrigin{Channel: "external", RequesterIsExternal: &internal}).isExternal() {
		t.Error("an explicit flag must win over the channel default")
	}
}
