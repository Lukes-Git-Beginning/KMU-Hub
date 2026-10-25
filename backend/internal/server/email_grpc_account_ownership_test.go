package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/middleware"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// Account changes are confined to the caller's own accounts in the email server
// itself, not only at the gateway: a call that reaches the service without the
// coarse provisioning key forwarded touches no one else's account.

func withEmailPerms(ctx context.Context, perms ...string) context.Context {
	return context.WithValue(ctx, middleware.UserPermsKey, perms)
}

func TestCreateEmailAccount_ForSomeoneElseNeedsTheProvisioningKey(t *testing.T) {
	tenantID, caller, other := uuid.New(), uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())
	req := &emailv1.CreateEmailAccountRequest{
		UserId: other.String(), EmailAddress: "other@example.com",
		ImapHost: "imap.example.com", SmtpHost: "smtp.example.com", Username: "o", Password: "p",
	}

	_, err := srv.CreateEmailAccount(ctxWithActorAndTenant(caller, tenantID), req)
	requireGRPCCode(t, err, codes.PermissionDenied)
	require.Empty(t, repo.accounts, "no account may be stored for someone else")

	resp, err := srv.CreateEmailAccount(withEmailPerms(ctxWithActorAndTenant(caller, tenantID), middleware.PermEmailWrite), req)
	requireGRPCOK(t, err)
	require.Equal(t, other.String(), resp.Account.UserId, "admin provisioning keeps the target user")
}

func TestUpdateEmailAccount_SomeoneElsesAccountIsNotFound(t *testing.T) {
	tenantID, caller, owner := uuid.New(), uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	acct := seedEmailAccount(repo, tenantID, owner, true)
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())
	name := "hijacked"

	_, err := srv.UpdateEmailAccount(ctxWithActorAndTenant(caller, tenantID),
		&emailv1.UpdateEmailAccountRequest{Id: acct.ID.String(), DisplayName: &name})
	requireGRPCCode(t, err, codes.NotFound)
	require.Empty(t, repo.accounts[acct.ID].DisplayName)

	// The delete key alone does not open an update.
	_, err = srv.UpdateEmailAccount(withEmailPerms(ctxWithActorAndTenant(caller, tenantID), middleware.PermEmailDelete),
		&emailv1.UpdateEmailAccountRequest{Id: acct.ID.String(), DisplayName: &name})
	requireGRPCCode(t, err, codes.NotFound)

	resp, err := srv.UpdateEmailAccount(withEmailPerms(ctxWithActorAndTenant(caller, tenantID), middleware.PermEmailWrite),
		&emailv1.UpdateEmailAccountRequest{Id: acct.ID.String(), DisplayName: &name})
	requireGRPCOK(t, err)
	require.Equal(t, name, resp.Account.DisplayName)
}

func TestDeleteEmailAccount_SomeoneElsesAccountIsNotFound(t *testing.T) {
	tenantID, caller, owner := uuid.New(), uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	acct := seedEmailAccount(repo, tenantID, owner, true)
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())

	_, err := srv.DeleteEmailAccount(ctxWithActorAndTenant(caller, tenantID),
		&emailv1.DeleteEmailAccountRequest{Id: acct.ID.String()})
	requireGRPCCode(t, err, codes.NotFound)
	require.Contains(t, repo.accounts, acct.ID, "someone else's account must survive")

	_, err = srv.DeleteEmailAccount(withEmailPerms(ctxWithActorAndTenant(caller, tenantID), middleware.PermEmailWrite),
		&emailv1.DeleteEmailAccountRequest{Id: acct.ID.String()})
	requireGRPCCode(t, err, codes.NotFound)

	_, err = srv.DeleteEmailAccount(withEmailPerms(ctxWithActorAndTenant(caller, tenantID), middleware.PermEmailDelete),
		&emailv1.DeleteEmailAccountRequest{Id: acct.ID.String()})
	requireGRPCOK(t, err)
	require.NotContains(t, repo.accounts, acct.ID)
}

func TestEmailAccountChanges_WithoutCaller(t *testing.T) {
	tenantCtx := context.WithValue(context.Background(), middleware.TenantIDKey, uuid.NewString())
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	_, err := srv.CreateEmailAccount(tenantCtx, &emailv1.CreateEmailAccountRequest{UserId: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
	_, err = srv.DeleteEmailAccount(tenantCtx, &emailv1.DeleteEmailAccountRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.Unauthenticated)
}
