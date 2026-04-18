package consent

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// stubAssertRepo is a minimal in-memory AssertRepo for unit tests.
type stubAssertRepo struct {
	activeConsent  bool
	activeErr      error
	hasEmail       bool
	hasEmailErr    error
}

func (s *stubAssertRepo) ActiveConsent(_ context.Context, _ uuid.UUID, _ Channel) (bool, error) {
	return s.activeConsent, s.activeErr
}

func (s *stubAssertRepo) ContactHasEmail(_ context.Context, _ uuid.UUID) (bool, error) {
	return s.hasEmail, s.hasEmailErr
}

// ---------------------------------------------------------------------------
// Table-driven Asserter tests
// ---------------------------------------------------------------------------

func TestAsserter_Assert(t *testing.T) {
	contactID := uuid.New()
	repoErr := errors.New("db error")

	tests := []struct {
		name          string
		channel       Channel
		hasEmail      bool
		hasEmailErr   error
		activeConsent bool
		activeErr     error
		wantErr       error
	}{
		{
			name:     "email channel — contact has no email, skip check",
			channel:  ChannelEmail,
			hasEmail: false,
			wantErr:  nil,
		},
		{
			name:          "email channel — contact has email, no consent",
			channel:       ChannelEmail,
			hasEmail:      true,
			activeConsent: false,
			wantErr:       ErrNoConsent,
		},
		{
			name:          "email channel — contact has email, active consent",
			channel:       ChannelEmail,
			hasEmail:      true,
			activeConsent: true,
			wantErr:       nil,
		},
		{
			name:          "phone channel — no consent",
			channel:       ChannelPhone,
			activeConsent: false,
			wantErr:       ErrNoConsent,
		},
		{
			name:          "phone channel — active consent",
			channel:       ChannelPhone,
			activeConsent: true,
			wantErr:       nil,
		},
		{
			name:      "repo error on ActiveConsent propagates",
			channel:   ChannelPhone,
			activeErr: repoErr,
			wantErr:   repoErr,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			repo := &stubAssertRepo{
				activeConsent: tc.activeConsent,
				activeErr:     tc.activeErr,
				hasEmail:      tc.hasEmail,
				hasEmailErr:   tc.hasEmailErr,
			}
			a := NewAsserter(repo, nil)

			err := a.Assert(context.Background(), contactID, tc.channel)

			if tc.wantErr == nil {
				require.NoError(t, err)
			} else {
				require.Error(t, err)
				assert.ErrorIs(t, err, tc.wantErr)
			}
		})
	}
}

// TestAsserter_EmailContactHasEmailError verifies that a repo error during the
// ContactHasEmail check is propagated immediately without calling ActiveConsent.
func TestAsserter_EmailContactHasEmailError(t *testing.T) {
	repoErr := errors.New("db timeout")
	repo := &stubAssertRepo{
		hasEmailErr: repoErr,
	}
	a := NewAsserter(repo, nil)

	err := a.Assert(context.Background(), uuid.New(), ChannelEmail)
	require.Error(t, err)
	assert.ErrorIs(t, err, repoErr)
}
