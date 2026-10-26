package livekit

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	lkauth "github.com/livekit/protocol/auth"
	lkproto "github.com/livekit/protocol/livekit"
	"google.golang.org/protobuf/proto"

	"github.com/kmuhub/kmuhub/internal/work/recording"
)

const (
	testAPIKey    = "test-api-key"
	testAPISecret = "test-api-secret-that-is-long-enough"
)

// fakeLiveKit is a Twirp server standing in for the LiveKit control plane. The
// real SDK clients run against it, so request mapping, the error wrapping and
// the auth header are exercised end to end; only the network peer is faked.
type fakeLiveKit struct {
	t        *testing.T
	mu       sync.Mutex
	calls    []string
	handlers map[string]func(body []byte) (proto.Message, *twirpErr)
	auth     map[string]string
}

type twirpErr struct {
	status int
	code   string
	msg    string
}

func newFakeLiveKit(t *testing.T) (*fakeLiveKit, string) {
	t.Helper()
	f := &fakeLiveKit{
		t:        t,
		handlers: map[string]func([]byte) (proto.Message, *twirpErr){},
		auth:     map[string]string{},
	}
	srv := httptest.NewServer(http.HandlerFunc(f.serve))
	t.Cleanup(srv.Close)
	return f, srv.URL
}

func (f *fakeLiveKit) on(method string, h func(body []byte) (proto.Message, *twirpErr)) {
	f.handlers[method] = h
}

func (f *fakeLiveKit) serve(w http.ResponseWriter, r *http.Request) {
	// Path: /twirp/livekit.RoomService/CreateRoom
	method := r.URL.Path[strings.LastIndex(r.URL.Path, "/")+1:]
	body, _ := io.ReadAll(r.Body)

	f.mu.Lock()
	f.calls = append(f.calls, method)
	f.auth[method] = r.Header.Get("Authorization")
	f.mu.Unlock()

	h, ok := f.handlers[method]
	if !ok {
		writeTwirpErr(w, &twirpErr{status: http.StatusNotFound, code: "bad_route", msg: "no handler for " + method})
		return
	}
	msg, terr := h(body)
	if terr != nil {
		writeTwirpErr(w, terr)
		return
	}
	out, err := proto.Marshal(msg)
	if err != nil {
		f.t.Errorf("marshal fake response: %v", err)
	}
	w.Header().Set("Content-Type", "application/protobuf")
	_, _ = w.Write(out)
}

func writeTwirpErr(w http.ResponseWriter, e *twirpErr) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(e.status)
	_ = json.NewEncoder(w).Encode(map[string]string{"code": e.code, "msg": e.msg})
}

func (f *fakeLiveKit) called() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.calls...)
}

func decode[T proto.Message](t *testing.T, body []byte, into T) T {
	t.Helper()
	if err := proto.Unmarshal(body, into); err != nil {
		t.Fatalf("decode request: %v", err)
	}
	return into
}

var errUnavailable = &twirpErr{status: http.StatusServiceUnavailable, code: "unavailable", msg: "livekit down"}

func TestNewRoomManager_RequiresAllParams(t *testing.T) {
	cases := map[string][3]string{
		"no key":    {"", "s", "wss://x"},
		"no secret": {"k", "", "wss://x"},
		"no url":    {"k", "s", ""},
	}
	for name, c := range cases {
		if rm := NewRoomManager(c[0], c[1], c[2]); rm != nil {
			t.Errorf("%s: expected nil RoomManager, got %+v", name, rm)
		}
		if rm := NewRoomManagerWithTURN(c[0], c[1], c[2], "turn-secret", "turn.example.com"); rm != nil {
			t.Errorf("%s (TURN): expected nil RoomManager, got %+v", name, rm)
		}
	}
}

func TestNewEgressManager_RequiresAllParams(t *testing.T) {
	if em := NewEgressManager("", "s", "wss://x"); em != nil {
		t.Error("expected nil EgressManager without key")
	}
	if em := NewEgressManager("k", "", "wss://x"); em != nil {
		t.Error("expected nil EgressManager without secret")
	}
	if em := NewEgressManager("k", "s", ""); em != nil {
		t.Error("expected nil EgressManager without url")
	}
	if em := NewEgressManager("k", "s", "wss://x"); em == nil {
		t.Error("expected EgressManager with full config")
	}
}

func TestRoomManager_CreateRoom(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	var got *lkproto.CreateRoomRequest
	fake.on("CreateRoom", func(body []byte) (proto.Message, *twirpErr) {
		got = decode(t, body, &lkproto.CreateRoomRequest{})
		return &lkproto.Room{Name: got.Name}, nil
	})
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	if err := rm.CreateRoom(context.Background(), "cal-abc", 25); err != nil {
		t.Fatalf("CreateRoom: %v", err)
	}
	if got.Name != "cal-abc" || got.MaxParticipants != 25 {
		t.Errorf("request = name %q max %d, want cal-abc / 25", got.Name, got.MaxParticipants)
	}
	if !strings.HasPrefix(fake.auth["CreateRoom"], "Bearer ") {
		t.Errorf("Authorization = %q, want a bearer token", fake.auth["CreateRoom"])
	}
	claims := verifyBearer(t, fake.auth["CreateRoom"])
	if claims.Video == nil || !claims.Video.RoomCreate {
		t.Errorf("CreateRoom must carry a roomCreate grant, got %+v", claims.Video)
	}
}

func TestRoomManager_CreateRoom_Error(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("CreateRoom", func([]byte) (proto.Message, *twirpErr) { return nil, errUnavailable })
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	err := rm.CreateRoom(context.Background(), "cal-abc", 5)
	if err == nil {
		t.Fatal("expected an error from a failing LiveKit")
	}
	if !strings.Contains(err.Error(), "livekit create room") || !strings.Contains(err.Error(), "livekit down") {
		t.Errorf("error = %v, want wrapped create-room context with the server message", err)
	}
}

func TestRoomManager_DeleteRoom(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	var got *lkproto.DeleteRoomRequest
	fake.on("DeleteRoom", func(body []byte) (proto.Message, *twirpErr) {
		got = decode(t, body, &lkproto.DeleteRoomRequest{})
		return &lkproto.DeleteRoomResponse{}, nil
	})
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	if err := rm.DeleteRoom(context.Background(), "cal-abc"); err != nil {
		t.Fatalf("DeleteRoom: %v", err)
	}
	if got.Room != "cal-abc" {
		t.Errorf("request room = %q, want cal-abc", got.Room)
	}

	fake.on("DeleteRoom", func([]byte) (proto.Message, *twirpErr) { return nil, errUnavailable })
	err := rm.DeleteRoom(context.Background(), "cal-abc")
	if err == nil || !strings.Contains(err.Error(), "livekit delete room") {
		t.Errorf("error = %v, want wrapped delete-room error", err)
	}
}

func TestRoomManager_ListParticipants(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	var gotRoom string
	fake.on("ListParticipants", func(body []byte) (proto.Message, *twirpErr) {
		gotRoom = decode(t, body, &lkproto.ListParticipantsRequest{}).Room
		return &lkproto.ListParticipantsResponse{Participants: []*lkproto.ParticipantInfo{
			{Identity: "alice"}, {Identity: "bob"},
		}}, nil
	})
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	ids, err := rm.ListParticipants(context.Background(), "cal-abc")
	if err != nil {
		t.Fatalf("ListParticipants: %v", err)
	}
	if gotRoom != "cal-abc" {
		t.Errorf("request room = %q, want cal-abc", gotRoom)
	}
	if len(ids) != 2 || ids[0] != "alice" || ids[1] != "bob" {
		t.Errorf("identities = %v, want [alice bob] in server order", ids)
	}
}

func TestRoomManager_ListParticipants_EmptyIsNonNilSlice(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("ListParticipants", func([]byte) (proto.Message, *twirpErr) {
		return &lkproto.ListParticipantsResponse{}, nil
	})
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	ids, err := rm.ListParticipants(context.Background(), "cal-abc")
	if err != nil {
		t.Fatalf("ListParticipants: %v", err)
	}
	if ids == nil || len(ids) != 0 {
		t.Errorf("identities = %#v, want empty non-nil slice (renders as [] not null)", ids)
	}
}

func TestRoomManager_ListParticipants_Error(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("ListParticipants", func([]byte) (proto.Message, *twirpErr) { return nil, errUnavailable })
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	ids, err := rm.ListParticipants(context.Background(), "cal-abc")
	if err == nil || !strings.Contains(err.Error(), "livekit list participants") {
		t.Fatalf("error = %v, want wrapped list-participants error", err)
	}
	if ids != nil {
		t.Errorf("identities = %v, want nil on error", ids)
	}
}

func TestRoomManager_MuteParticipant_MutesEveryTrackOfThatParticipantOnly(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("ListParticipants", func([]byte) (proto.Message, *twirpErr) {
		return &lkproto.ListParticipantsResponse{Participants: []*lkproto.ParticipantInfo{
			{Identity: "alice", Tracks: []*lkproto.TrackInfo{{Sid: "TR_a1"}, {Sid: "TR_a2"}}},
			{Identity: "bob", Tracks: []*lkproto.TrackInfo{{Sid: "TR_b1"}}},
		}}, nil
	})
	var muted []*lkproto.MuteRoomTrackRequest
	fake.on("MutePublishedTrack", func(body []byte) (proto.Message, *twirpErr) {
		req := decode(t, body, &lkproto.MuteRoomTrackRequest{})
		muted = append(muted, req)
		return &lkproto.MuteRoomTrackResponse{}, nil
	})
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	if err := rm.MuteParticipant(context.Background(), "cal-abc", "alice"); err != nil {
		t.Fatalf("MuteParticipant: %v", err)
	}
	if len(muted) != 2 {
		t.Fatalf("muted %d tracks, want 2 (alice only)", len(muted))
	}
	for i, want := range []string{"TR_a1", "TR_a2"} {
		m := muted[i]
		if m.TrackSid != want || m.Identity != "alice" || m.Room != "cal-abc" || !m.Muted {
			t.Errorf("mute %d = %+v, want room cal-abc identity alice track %s muted=true", i, m, want)
		}
	}
}

func TestRoomManager_MuteParticipant_UnknownParticipantIsNoop(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("ListParticipants", func([]byte) (proto.Message, *twirpErr) {
		return &lkproto.ListParticipantsResponse{Participants: []*lkproto.ParticipantInfo{
			{Identity: "bob", Tracks: []*lkproto.TrackInfo{{Sid: "TR_b1"}}},
		}}, nil
	})
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	if err := rm.MuteParticipant(context.Background(), "cal-abc", "ghost"); err != nil {
		t.Fatalf("MuteParticipant for absent identity: %v", err)
	}
	for _, c := range fake.called() {
		if c == "MutePublishedTrack" {
			t.Error("must not mute anything when the participant is not in the room")
		}
	}
}

func TestRoomManager_MuteParticipant_ListError(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("ListParticipants", func([]byte) (proto.Message, *twirpErr) { return nil, errUnavailable })
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	err := rm.MuteParticipant(context.Background(), "cal-abc", "alice")
	if err == nil || !strings.Contains(err.Error(), "mute participant") {
		t.Errorf("error = %v, want wrapped list error", err)
	}
}

func TestRoomManager_MuteParticipant_TrackErrorStops(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("ListParticipants", func([]byte) (proto.Message, *twirpErr) {
		return &lkproto.ListParticipantsResponse{Participants: []*lkproto.ParticipantInfo{
			{Identity: "alice", Tracks: []*lkproto.TrackInfo{{Sid: "TR_a1"}, {Sid: "TR_a2"}}},
		}}, nil
	})
	fake.on("MutePublishedTrack", func([]byte) (proto.Message, *twirpErr) { return nil, errUnavailable })
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	err := rm.MuteParticipant(context.Background(), "cal-abc", "alice")
	if err == nil || !strings.Contains(err.Error(), "livekit mute track TR_a1") {
		t.Fatalf("error = %v, want the failing track sid in the message", err)
	}
	var mutes int
	for _, c := range fake.called() {
		if c == "MutePublishedTrack" {
			mutes++
		}
	}
	if mutes != 1 {
		t.Errorf("mute calls = %d, want 1 (stop at the first failure)", mutes)
	}
}

func TestRoomManager_RemoveParticipant(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	var got *lkproto.RoomParticipantIdentity
	fake.on("RemoveParticipant", func(body []byte) (proto.Message, *twirpErr) {
		got = decode(t, body, &lkproto.RoomParticipantIdentity{})
		return &lkproto.RemoveParticipantResponse{}, nil
	})
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	if err := rm.RemoveParticipant(context.Background(), "cal-abc", "alice"); err != nil {
		t.Fatalf("RemoveParticipant: %v", err)
	}
	if got.Room != "cal-abc" || got.Identity != "alice" {
		t.Errorf("request = %+v, want room cal-abc identity alice", got)
	}

	fake.on("RemoveParticipant", func([]byte) (proto.Message, *twirpErr) { return nil, errUnavailable })
	err := rm.RemoveParticipant(context.Background(), "cal-abc", "alice")
	if err == nil || !strings.Contains(err.Error(), "livekit remove participant") {
		t.Errorf("error = %v, want wrapped remove-participant error", err)
	}
}

func TestRoomManager_APICallsHonourCancelledContext(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("CreateRoom", func([]byte) (proto.Message, *twirpErr) { return &lkproto.Room{}, nil })
	rm := NewRoomManager(testAPIKey, testAPISecret, url)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := rm.CreateRoom(ctx, "cal-abc", 5); err == nil {
		t.Fatal("expected an error with an already cancelled context")
	}
	if len(fake.called()) != 0 {
		t.Errorf("no request may reach LiveKit with a cancelled context, got %v", fake.called())
	}
}

func verifyBearer(t *testing.T, header string) *lkauth.ClaimGrants {
	t.Helper()
	tok, err := lkauth.ParseAPIToken(strings.TrimPrefix(header, "Bearer "))
	if err != nil {
		t.Fatalf("parse bearer: %v", err)
	}
	if tok.APIKey() != testAPIKey {
		t.Errorf("bearer issuer = %q, want %q", tok.APIKey(), testAPIKey)
	}
	_, claims, err := tok.Verify(testAPISecret)
	if err != nil {
		t.Fatalf("verify bearer with the shared secret: %v", err)
	}
	return claims
}

func tokenClaims(t *testing.T, jwt string) *lkauth.ClaimGrants {
	t.Helper()
	tok, err := lkauth.ParseAPIToken(jwt)
	if err != nil {
		t.Fatalf("parse token: %v", err)
	}
	_, claims, err := tok.Verify(testAPISecret)
	if err != nil {
		t.Fatalf("verify token: %v", err)
	}
	return claims
}

func TestRoomManager_GenerateToken(t *testing.T) {
	rm := NewRoomManager(testAPIKey, testAPISecret, "wss://livekit.example.com")

	jwt, err := rm.GenerateToken("cal-abc", "user-1", "Max Mustermann")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	c := tokenClaims(t, jwt)
	if c.Identity != "user-1" || c.Name != "Max Mustermann" {
		t.Errorf("identity/name = %q/%q, want user-1/Max Mustermann", c.Identity, c.Name)
	}
	if c.Video == nil || !c.Video.RoomJoin || c.Video.Room != "cal-abc" {
		t.Errorf("video grant = %+v, want roomJoin on cal-abc", c.Video)
	}
	if c.Video.RoomAdmin || c.Video.RoomCreate {
		t.Errorf("join token must not carry admin/create rights: %+v", c.Video)
	}
	if c.Metadata != "" {
		t.Errorf("metadata = %q, want empty when TURN is off", c.Metadata)
	}
}

func TestRoomManager_GenerateToken_EmbedsTURNMetadata(t *testing.T) {
	rm := NewRoomManagerWithTURN(testAPIKey, testAPISecret, "wss://livekit.example.com", "turn-secret", "turn.zentria.tech")

	jwt, err := rm.GenerateToken("cal-abc", "user-1", "Max")
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	c := tokenClaims(t, jwt)
	var meta turnMetadata
	if err := json.Unmarshal([]byte(c.Metadata), &meta); err != nil {
		t.Fatalf("metadata %q is not the turn JSON: %v", c.Metadata, err)
	}
	if len(meta.IceServers) != 2 {
		t.Fatalf("iceServers = %d, want 2", len(meta.IceServers))
	}
	for _, s := range meta.IceServers {
		if !strings.HasSuffix(s.Username, ":user-1") || s.Credential == "" {
			t.Errorf("ice server %+v: username must end in the user id and carry a credential", s)
		}
	}
}

func TestRoomManager_GenerateToken_TURNNeedsBothSecretAndHost(t *testing.T) {
	for name, rm := range map[string]*RoomManager{
		"secret only": NewRoomManagerWithTURN(testAPIKey, testAPISecret, "wss://x", "turn-secret", ""),
		"host only":   NewRoomManagerWithTURN(testAPIKey, testAPISecret, "wss://x", "", "turn.zentria.tech"),
	} {
		jwt, err := rm.GenerateToken("cal-abc", "user-1", "Max")
		if err != nil {
			t.Fatalf("%s: GenerateToken: %v", name, err)
		}
		if m := tokenClaims(t, jwt).Metadata; m != "" {
			t.Errorf("%s: metadata = %q, want none", name, m)
		}
		if s := rm.TURNIceServers("user-1"); s != nil {
			t.Errorf("%s: TURNIceServers = %v, want nil", name, s)
		}
	}
}

func TestRoomManager_TURNIceServers_MapsEveryField(t *testing.T) {
	rm := NewRoomManagerWithTURN(testAPIKey, testAPISecret, "wss://x", "turn-secret", "turn.zentria.tech")

	got := rm.TURNIceServers("user-1")
	want := (&Service{turnSecret: "turn-secret", coturnHost: "turn.zentria.tech"}).TURNIceServers("user-1")
	if len(got) != len(want) {
		t.Fatalf("got %d servers, want %d", len(got), len(want))
	}
	// The credential embeds an expiry timestamp, so compare structure: URLs
	// verbatim, username suffix, and a non-empty credential per server.
	for i := range got {
		if strings.Join(got[i].URLs, ",") != strings.Join(want[i].URLs, ",") {
			t.Errorf("server %d URLs = %v, want %v", i, got[i].URLs, want[i].URLs)
		}
		if !strings.HasSuffix(got[i].Username, ":user-1") {
			t.Errorf("server %d username = %q", i, got[i].Username)
		}
		if got[i].Credential == "" {
			t.Errorf("server %d has no credential", i)
		}
	}
}

func TestEgressManager_StartRoomCompositeEgress(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	var got *lkproto.RoomCompositeEgressRequest
	fake.on("StartRoomCompositeEgress", func(body []byte) (proto.Message, *twirpErr) {
		got = decode(t, body, &lkproto.RoomCompositeEgressRequest{})
		return &lkproto.EgressInfo{EgressId: "EG_123"}, nil
	})
	em := NewEgressManager(testAPIKey, testAPISecret, url)

	id, err := em.StartRoomCompositeEgress(context.Background(), "cal-abc", "grid", recording.S3Config{
		Endpoint:  "minio:9000",
		AccessKey: "ak",
		Secret:    "sk",
		Bucket:    "recordings",
	})
	if err != nil {
		t.Fatalf("StartRoomCompositeEgress: %v", err)
	}
	if id != "EG_123" {
		t.Errorf("egress id = %q, want EG_123", id)
	}
	if got.RoomName != "cal-abc" || got.Layout != "grid" {
		t.Errorf("request room/layout = %q/%q, want cal-abc/grid", got.RoomName, got.Layout)
	}
	// The production code still uses the deprecated single-file output, so the
	// test reads the very same field.
	out, ok := got.Output.(*lkproto.RoomCompositeEgressRequest_File) //nolint:staticcheck // mirrors egress_manager.go
	if !ok {
		t.Fatalf("output = %T, want the file output", got.Output)
	}
	file := out.File //nolint:staticcheck // mirrors egress_manager.go
	if file == nil || file.FileType != lkproto.EncodedFileType_MP4 {
		t.Fatalf("output = %+v, want an MP4 file output", got.Output)
	}
	s3 := file.GetS3()
	if s3 == nil {
		t.Fatalf("file output has no S3 target: %+v", file)
	}
	if s3.AccessKey != "ak" || s3.Secret != "sk" || s3.Endpoint != "minio:9000" || s3.Bucket != "recordings" {
		t.Errorf("s3 = %+v, want the credentials/endpoint/bucket passed through", s3)
	}
	if !s3.ForcePathStyle {
		t.Error("MinIO needs path-style addressing")
	}
}

func TestEgressManager_StartRoomCompositeEgress_Error(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	fake.on("StartRoomCompositeEgress", func([]byte) (proto.Message, *twirpErr) { return nil, errUnavailable })
	em := NewEgressManager(testAPIKey, testAPISecret, url)

	id, err := em.StartRoomCompositeEgress(context.Background(), "cal-abc", "grid", recording.S3Config{})
	if err == nil || !strings.Contains(err.Error(), "livekit start egress") {
		t.Fatalf("error = %v, want wrapped start-egress error", err)
	}
	if id != "" {
		t.Errorf("egress id = %q, want empty on error", id)
	}
}

func TestEgressManager_StopEgress(t *testing.T) {
	fake, url := newFakeLiveKit(t)
	var got *lkproto.StopEgressRequest
	fake.on("StopEgress", func(body []byte) (proto.Message, *twirpErr) {
		got = decode(t, body, &lkproto.StopEgressRequest{})
		return &lkproto.EgressInfo{EgressId: got.EgressId}, nil
	})
	em := NewEgressManager(testAPIKey, testAPISecret, url)

	if err := em.StopEgress(context.Background(), "EG_123"); err != nil {
		t.Fatalf("StopEgress: %v", err)
	}
	if got.EgressId != "EG_123" {
		t.Errorf("request egress id = %q, want EG_123", got.EgressId)
	}

	fake.on("StopEgress", func([]byte) (proto.Message, *twirpErr) { return nil, errUnavailable })
	err := em.StopEgress(context.Background(), "EG_123")
	if err == nil || !strings.Contains(err.Error(), "livekit stop egress") {
		t.Errorf("error = %v, want wrapped stop-egress error", err)
	}
	var terr interface{ Msg() string }
	if !errors.As(err, &terr) {
		t.Errorf("error chain must keep the twirp error (%%w wrapping), got %T", err)
	}
}
