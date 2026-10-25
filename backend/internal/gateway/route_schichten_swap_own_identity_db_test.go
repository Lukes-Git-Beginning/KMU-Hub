package gateway

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"

	"github.com/kmuhub/kmuhub/internal/schichten"
	"github.com/kmuhub/kmuhub/internal/testutil"
	schichtenv1 "github.com/kmuhub/kmuhub/proto/schichten/v1"
)

// The member preset holds schichten:swap:create and schichten:swap:read at
// scope "own". shift_assignments.employee_id carries either the user id
// (API) or the HR profile id (the desktop roster), and the desktop files a
// swap with the assignment's employee_id as requested_by_employee_id. Both
// identities therefore count as the caller — when filing and in the list —
// and a colleague's identity does not. Same production chain as
// TestSchichtenAssignments_MemberSeesOwnOnly.

var swapOwnIdentityTenant = uuid.MustParse("e4fec700-0000-4000-8000-0000000000e6")

func TestSchichtenSwapRequests_MemberActsUnderUserAndProfileID(t *testing.T) {
	h := newSchichtenSeedHarness(t, swapOwnIdentityTenant)

	memberID, memberToken := h.token("member")
	colleagueID, _ := h.token("member")
	thirdID, _ := h.token("member")
	_, managerToken := h.token("manager")
	memberProfile := h.profile(memberID)
	colleagueProfile := h.profile(colleagueID)

	rosterShift, rosterSeats := h.shift(memberProfile, colleagueProfile)
	apiShift, apiSeats := h.shift(memberID, colleagueID)

	file := func(token string, shiftID, assignmentID, requestedBy, swapWith uuid.UUID) (int, string) {
		body := `{"assignment_id":"` + assignmentID.String() + `","shift_id":"` + shiftID.String() +
			`","requested_by_employee_id":"` + requestedBy.String() + `","swap_with_employee_id":"` + swapWith.String() + `"}`
		rec := h.do(http.MethodPost, token, "/api/v1/schichten/swap-requests", body)
		if rec.Code != http.StatusCreated {
			return rec.Code, ""
		}
		var resp schichtenv1.SwapRequestResponse
		if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode swap: %v", err)
		}
		return rec.Code, resp.GetSwapRequest().GetId()
	}

	var byProfile, byUser string
	t.Run("member files a swap for the roster seat under their HR profile id", func(t *testing.T) {
		code, id := file(memberToken, rosterShift, rosterSeats[memberProfile], memberProfile, colleagueProfile)
		if code != http.StatusCreated {
			t.Fatalf("member swap by profile id = %d, want 201", code)
		}
		byProfile = id
	})

	t.Run("member files a swap for the API seat under their user id", func(t *testing.T) {
		code, id := file(memberToken, apiShift, apiSeats[memberID], memberID, colleagueID)
		if code != http.StatusCreated {
			t.Fatalf("member swap by user id = %d, want 201", code)
		}
		byUser = id
	})

	t.Run("member cannot file in a colleague's name, by profile or user id", func(t *testing.T) {
		if code, _ := file(memberToken, rosterShift, rosterSeats[colleagueProfile], colleagueProfile, memberProfile); code != http.StatusForbidden {
			t.Errorf("member swap as colleague's profile = %d, want 403", code)
		}
		if code, _ := file(memberToken, apiShift, apiSeats[colleagueID], colleagueID, memberID); code != http.StatusForbidden {
			t.Errorf("member swap as colleague's user id = %d, want 403", code)
		}
	})

	// Seeded behind the API: the colleague asks the member's profile to swap
	// (member is the partner), and the colleague asks a third person (foreign).
	ctx := testutil.WithTenantCtx(context.Background(), swapOwnIdentityTenant)
	seedSwap := func(assignmentID, shiftID, requestedBy, swapWith uuid.UUID) string {
		now := time.Now().UTC()
		req := &schichten.SwapRequest{
			ID: uuid.New(), TenantID: swapOwnIdentityTenant, AssignmentID: assignmentID, ShiftID: shiftID,
			RequestedByEmployeeID: requestedBy, SwapWithEmployeeID: swapWith, Status: schichten.SwapRequestStatusPending,
			IdempotencyKey: uuid.NewString(), CreatedAt: now, UpdatedAt: now,
		}
		if err := h.repo.CreateSwapRequest(ctx, req); err != nil {
			t.Fatalf("CreateSwapRequest: %v", err)
		}
		return req.ID.String()
	}
	asPartner := seedSwap(rosterSeats[colleagueProfile], rosterShift, colleagueProfile, memberProfile)
	foreign := seedSwap(apiSeats[colleagueID], apiShift, colleagueID, thirdID)

	list := func(t *testing.T, token string) map[string]bool {
		t.Helper()
		rec := h.do(http.MethodGet, token, "/api/v1/schichten/swap-requests", "")
		if rec.Code != http.StatusOK {
			t.Fatalf("list swaps = %d, body %s", rec.Code, rec.Body.String())
		}
		var resp schichtenv1.ListSwapRequestsResponse
		if err := protojson.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("decode: %v", err)
		}
		seen := map[string]bool{}
		for _, s := range resp.GetSwapRequests() {
			seen[s.GetId()] = true
		}
		if int(resp.GetTotal()) != len(seen) {
			t.Errorf("total = %d, items = %d", resp.GetTotal(), len(seen))
		}
		return seen
	}

	t.Run("member's list holds their swaps under both identities, not the foreign one", func(t *testing.T) {
		seen := list(t, memberToken)
		for name, id := range map[string]string{"by profile": byProfile, "by user": byUser, "as partner by profile": asPartner} {
			if !seen[id] {
				t.Errorf("member list misses the swap %s (%s): %v", name, id, seen)
			}
		}
		if seen[foreign] || len(seen) != 3 {
			t.Errorf("member list = %v, want exactly the three own swaps (foreign %s)", seen, foreign)
		}
	})

	t.Run("manager sees every swap", func(t *testing.T) {
		if seen := list(t, managerToken); len(seen) != 4 {
			t.Errorf("manager list = %v, want all four", seen)
		}
	})
}
