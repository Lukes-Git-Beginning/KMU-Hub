package server

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/work/recording"
	videov1 "github.com/kmuhub/kmuhub/proto/video/v1"
)

// TestConfirmInitiatorConsent_OverTheWire drives the RPC through a real gRPC
// server registered exactly like cmd/work does (RegisterVideoServiceServer).
// The hand-written predecessor of this RPC was neither a proto.Message nor part
// of the service descriptor, so every call failed with Internal before it left
// the client; calling ts.srv directly could never notice that.
func TestConfirmInitiatorConsent_OverTheWire(t *testing.T) {
	ts := newTestVideoCallServer()
	initiator := uuid.New()
	recID := uuid.New()
	ts.recRepo.recordings[recID] = &recording.Recording{ID: recID, StartedBy: &initiator, CreatedAt: time.Now()}

	client := dialVideoServer(t, ts.srv)

	t.Run("a foreign user stamps nothing", func(t *testing.T) {
		_, err := client.ConfirmInitiatorConsent(context.Background(), &videov1.ConfirmInitiatorConsentRequest{
			RecordingId: recID.String(),
			UserId:      uuid.New().String(),
			TenantId:    uuid.New().String(),
		})
		assert.Equal(t, codes.NotFound, status.Code(err), "got %v", err)
		assert.Nil(t, ts.recRepo.recordings[recID].PreRecordingConsentAt)
	})

	t.Run("the initiator stamps their recording", func(t *testing.T) {
		resp, err := client.ConfirmInitiatorConsent(context.Background(), &videov1.ConfirmInitiatorConsentRequest{
			RecordingId: recID.String(),
			UserId:      initiator.String(),
			TenantId:    uuid.New().String(),
		})
		require.NoError(t, err)
		assert.True(t, resp.GetStamped())
		assert.NotNil(t, ts.recRepo.recordings[recID].PreRecordingConsentAt)
	})

	t.Run("a malformed id is rejected, not stamped", func(t *testing.T) {
		_, err := client.ConfirmInitiatorConsent(context.Background(), &videov1.ConfirmInitiatorConsentRequest{RecordingId: "bogus"})
		assert.Equal(t, codes.InvalidArgument, status.Code(err), "got %v", err)
	})
}
