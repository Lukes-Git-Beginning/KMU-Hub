package gateway

// Pins the project time-entries wire shape against the desktop ProjectTimeEntry
// type (api/hooks/useProjects.ts, consumed via useProjectTimeEntries): lowercase
// keys, no leaked project field (the route is already scoped by the URL), and
// no billed field (that's a query-side filter here, not a response field).

import (
	"encoding/json"
	"strings"
	"testing"

	workv1 "github.com/kmuhub/kmuhub/proto/work/v1"
)

func TestToProjectTimeEntryWire_Keys(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(toProjectTimeEntryWire(&workv1.ProjectTimeEntryProto{
		Id:          "55555555-5555-5555-5555-555555555555",
		Date:        "2026-08-01",
		Task:        "Frontend-Entwicklung",
		Person:      "Anna Mueller",
		Hours:       2.5,
		Description: "React-Komponenten implementiert",
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	for _, key := range []string{"id", "date", "task", "person", "hours", "description"} {
		if _, ok := decoded[key]; !ok {
			t.Errorf("%s missing; got keys %v", key, keysOf(decoded))
		}
	}
	for _, key := range []string{"project", "billed", "employee"} {
		if _, ok := decoded[key]; ok {
			t.Errorf("%s present — not part of the project-scoped wire shape", key)
		}
	}
	if hours, ok := decoded["hours"].(float64); !ok || hours != 2.5 {
		t.Errorf("hours = %v (%T), want 2.5 as a number", decoded["hours"], decoded["hours"])
	}
}

// Pins the team-utilization wire shape against the desktop MemberUtilization
// type (api/hooks/useProjects.ts, consumed via useProjectTeamUtilization).
// Crucially: no "rate" key anywhere -- that field drives the FE's
// personnel-cost calculation and this route sits behind project-read, not
// team:salary:view, so no cost/salary figure may leave the backend here.
func TestToMemberUtilizationWire_Keys(t *testing.T) {
	t.Parallel()

	got, err := json.Marshal(toMemberUtilizationWire(&workv1.ProjectMemberUtilizationProto{
		UserId:       "55555555-5555-5555-5555-555555555555",
		Name:         "Anna Mueller",
		Role:         "member",
		WeeklyTarget: 40,
		WeeklyData:   []*workv1.UtilizationPointProto{{Label: "KW 31", Hours: 12.5}},
		MonthlyData:  []*workv1.UtilizationPointProto{{Label: "Aug 2026", Hours: 40}},
	}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	if strings.Contains(string(got), `"rate"`) {
		t.Fatalf("wire shape leaks a rate/cost field: %s", got)
	}

	var decoded map[string]any
	if err := json.Unmarshal(got, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	member, ok := decoded["member"].(map[string]any)
	if !ok {
		t.Fatalf("member missing or not an object; got %v", decoded)
	}
	for _, key := range []string{"id", "name", "role", "avatarInitial", "weeklyTarget"} {
		if _, ok := member[key]; !ok {
			t.Errorf("member.%s missing; got keys %v", key, keysOf(member))
		}
	}
	if member["avatarInitial"] != "A" {
		t.Errorf("avatarInitial = %v, want \"A\"", member["avatarInitial"])
	}

	for _, key := range []string{"weeklyData", "monthlyData"} {
		points, ok := decoded[key].([]any)
		if !ok || len(points) != 1 {
			t.Fatalf("%s missing or wrong length; got %v", key, decoded[key])
		}
		point, ok := points[0].(map[string]any)
		if !ok {
			t.Fatalf("%s[0] not an object", key)
		}
		if _, ok := point["label"]; !ok {
			t.Errorf("%s[0].label missing", key)
		}
		if _, ok := point["hours"]; !ok {
			t.Errorf("%s[0].hours missing", key)
		}
	}
}
