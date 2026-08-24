package server

import (
	"context"
	"errors"
	"strings"
	"testing"

	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/notification/integration"
	notificationv1 "github.com/kmuhub/kmuhub/proto/notification/v1"
)

// stubIntegrationRepo answers only the config lookup TestIntegrationConfig
// performs; every other method of the interface stays unimplemented on purpose,
// so a future caller that reaches for one fails loudly instead of silently.
type stubIntegrationRepo struct {
	integration.Repository
	cfg *integration.IntegrationConfig
	err error
}

func (r *stubIntegrationRepo) GetConfigByPlatform(_ context.Context, _ string) (*integration.IntegrationConfig, error) {
	if r.err != nil {
		return nil, r.err
	}
	return r.cfg, nil
}

type stubProber struct {
	detail string
	err    error
	calls  int
}

func (p *stubProber) ProbeConnection(context.Context) (*integration.ProbeResult, error) {
	p.calls++
	if p.err != nil {
		return nil, p.err
	}
	return &integration.ProbeResult{Detail: p.detail}, nil
}

func newTestConfigServer(repo integration.Repository, probers map[string]integration.ConnectionProber) *NotificationGRPCServer {
	return NewNotificationGRPCServer(nil, nil, nil,
		WithIntegration(repo, nil),
		WithConnectionProbers(probers),
	)
}

func TestTestIntegrationConfigReportsProbedIdentity(t *testing.T) {
	prober := &stubProber{detail: "authenticated as cosmi-bot in workspace Zentria"}
	srv := newTestConfigServer(
		&stubIntegrationRepo{cfg: &integration.IntegrationConfig{Platform: integration.PlatformSlack}},
		map[string]integration.ConnectionProber{integration.PlatformSlack: prober},
	)

	resp, err := srv.TestIntegrationConfig(context.Background(), &notificationv1.TestIntegrationConfigRequest{
		Platform: "Slack",
	})
	if err != nil {
		t.Fatalf("expected success, got %v", err)
	}
	if prober.calls != 1 {
		t.Fatalf("expected the platform to be contacted once, got %d calls", prober.calls)
	}
	if !resp.Success {
		t.Fatal("expected success=true")
	}
	if resp.Message != prober.detail {
		t.Fatalf("expected the probe detail in the message, got %q", resp.Message)
	}
}

// The whole point of the unit: no green answer unless something answered.
func TestTestIntegrationConfigNeverSucceedsWithoutContact(t *testing.T) {
	cfg := &integration.IntegrationConfig{Platform: integration.PlatformSlack}

	tests := []struct {
		name     string
		repo     integration.Repository
		probers  map[string]integration.ConnectionProber
		platform string
		wantCode codes.Code
	}{
		{
			name:     "unknown platform",
			repo:     &stubIntegrationRepo{cfg: cfg},
			probers:  map[string]integration.ConnectionProber{integration.PlatformSlack: &stubProber{}},
			platform: "discord",
			wantCode: codes.InvalidArgument,
		},
		{
			name:     "no config for this tenant",
			repo:     &stubIntegrationRepo{err: integration.ErrConfigNotFound},
			probers:  map[string]integration.ConnectionProber{integration.PlatformSlack: &stubProber{}},
			platform: integration.PlatformSlack,
			wantCode: codes.NotFound,
		},
		{
			name:     "server holds no client for the platform",
			repo:     &stubIntegrationRepo{cfg: cfg},
			probers:  map[string]integration.ConnectionProber{},
			platform: integration.PlatformSlack,
			wantCode: codes.FailedPrecondition,
		},
		{
			name: "platform refuses the credentials",
			repo: &stubIntegrationRepo{cfg: cfg},
			probers: map[string]integration.ConnectionProber{
				integration.PlatformSlack: &stubProber{err: errors.New("invalid_auth")},
			},
			platform: integration.PlatformSlack,
			wantCode: codes.FailedPrecondition,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := newTestConfigServer(tt.repo, tt.probers)

			resp, err := srv.TestIntegrationConfig(context.Background(), &notificationv1.TestIntegrationConfigRequest{
				Platform: tt.platform,
			})
			if err == nil {
				t.Fatalf("expected an error, got response %+v", resp)
			}
			if resp != nil {
				t.Fatalf("expected no response alongside the error, got %+v", resp)
			}
			if got := status.Code(err); got != tt.wantCode {
				t.Fatalf("expected code %v, got %v (%v)", tt.wantCode, got, err)
			}
		})
	}
}

// TestTestIntegrationConfigDoesNotLeakProbeErrorDetail pins the fix for the
// fourth occurrence of a leak class fixed three times already this run
// (bexio, DATEV, lexware): a prober's raw error can carry the external
// platform's response body verbatim (Teams: up to 2KB of the AAD error page).
// The client must see only the platform name it sent, never that text.
func TestTestIntegrationConfigDoesNotLeakProbeErrorDetail(t *testing.T) {
	cfg := &integration.IntegrationConfig{Platform: integration.PlatformTeams}
	sensitive := "invalid_client: AADSTS7000215: Invalid client secret provided. Trace ID: deadbeef\r\nSet-Cookie: x=1"
	srv := newTestConfigServer(
		&stubIntegrationRepo{cfg: cfg},
		map[string]integration.ConnectionProber{
			integration.PlatformTeams: &stubProber{err: errors.New(sensitive)},
		},
	)

	resp, err := srv.TestIntegrationConfig(context.Background(), &notificationv1.TestIntegrationConfigRequest{
		Platform: integration.PlatformTeams,
	})
	if err == nil {
		t.Fatalf("expected an error, got response %+v", resp)
	}
	if status.Code(err) != codes.FailedPrecondition {
		t.Fatalf("expected FailedPrecondition, got %v", err)
	}
	if strings.Contains(status.Convert(err).Message(), sensitive) || strings.Contains(status.Convert(err).Message(), "AADSTS") {
		t.Fatalf("error message leaks the platform's raw response: %q", status.Convert(err).Message())
	}
}

func TestTestIntegrationConfigWithoutIntegrationStorage(t *testing.T) {
	srv := NewNotificationGRPCServer(nil, nil, nil)

	_, err := srv.TestIntegrationConfig(context.Background(), &notificationv1.TestIntegrationConfigRequest{
		Platform: integration.PlatformSlack,
	})
	if status.Code(err) != codes.Unavailable {
		t.Fatalf("expected Unavailable, got %v", err)
	}
}
