package gateway

// Pins the project time-entries wire shape against the desktop ProjectTimeEntry
// type (api/hooks/useProjects.ts, consumed via useProjectTimeEntries): lowercase
// keys, no leaked project field (the route is already scoped by the URL), and
// no billed field (that's a query-side filter here, not a response field).

import (
	"encoding/json"
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
