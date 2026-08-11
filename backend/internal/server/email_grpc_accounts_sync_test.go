package server

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"

	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/email/attachment"
	emailcontact "github.com/kmuhub/kmuhub/internal/email/contact"
	"github.com/kmuhub/kmuhub/internal/email/label"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/email/rule"
	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/email/signature"
	emailsync "github.com/kmuhub/kmuhub/internal/email/sync"
	"github.com/kmuhub/kmuhub/internal/email/template"
	"github.com/kmuhub/kmuhub/internal/models"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// ---------------------------------------------------------------------------
// stubAccountRepo implements account.Repository over an in-memory map.
// ---------------------------------------------------------------------------

type stubAccountRepo struct {
	accounts map[uuid.UUID]*models.EmailAccount
}

func newStubAccountRepo() *stubAccountRepo {
	return &stubAccountRepo{accounts: make(map[uuid.UUID]*models.EmailAccount)}
}

func (r *stubAccountRepo) Create(_ context.Context, a *models.EmailAccount) error {
	r.accounts[a.ID] = a
	return nil
}

func (r *stubAccountRepo) GetByID(_ context.Context, id, tenantID uuid.UUID) (*models.EmailAccount, error) {
	a, ok := r.accounts[id]
	if !ok || a.TenantID != tenantID {
		return nil, account.ErrAccountNotFound
	}
	cp := *a
	return &cp, nil
}

func (r *stubAccountRepo) GetByUserIDAndTenant(_ context.Context, userID, tenantID uuid.UUID) (*models.EmailAccount, error) {
	for _, a := range r.accounts {
		if a.UserID == userID && a.TenantID == tenantID && a.IsDefault {
			cp := *a
			return &cp, nil
		}
	}
	return nil, account.ErrAccountNotFound
}

func (r *stubAccountRepo) ListByUserAndTenant(_ context.Context, userID, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	out := make([]*models.EmailAccount, 0)
	for _, a := range r.accounts {
		if a.UserID == userID && a.TenantID == tenantID {
			cp := *a
			out = append(out, &cp)
		}
	}
	return out, nil
}

func (r *stubAccountRepo) Update(_ context.Context, a *models.EmailAccount) error {
	if _, ok := r.accounts[a.ID]; !ok {
		return account.ErrAccountNotFound
	}
	r.accounts[a.ID] = a
	return nil
}

func (r *stubAccountRepo) Delete(_ context.Context, id, tenantID uuid.UUID) error {
	a, ok := r.accounts[id]
	if !ok || a.TenantID != tenantID {
		return account.ErrAccountNotFound
	}
	delete(r.accounts, id)
	return nil
}

func (r *stubAccountRepo) SetDefault(_ context.Context, id, userID, tenantID uuid.UUID) error {
	for _, a := range r.accounts {
		if a.UserID == userID && a.TenantID == tenantID {
			a.IsDefault = a.ID == id
		}
	}
	return nil
}

func (r *stubAccountRepo) ListActive(_ context.Context, tenantID uuid.UUID) ([]*models.EmailAccount, error) {
	out := make([]*models.EmailAccount, 0)
	for _, a := range r.accounts {
		if a.TenantID == tenantID && a.SyncEnabled {
			out = append(out, a)
		}
	}
	return out, nil
}

func (r *stubAccountRepo) ListAllActive(_ context.Context) ([]*models.EmailAccount, error) {
	out := make([]*models.EmailAccount, 0)
	for _, a := range r.accounts {
		if a.SyncEnabled {
			out = append(out, a)
		}
	}
	return out, nil
}

// fakeVaultEncryptor is a reversible, non-cryptographic stand-in for the real
// vault so account.Service.Create/Update never dial an actual vault service.
type fakeVaultEncryptor struct{}

func (fakeVaultEncryptor) Encrypt(_ context.Context, plaintext []byte) (string, error) {
	return "enc:" + string(plaintext), nil
}

func (fakeVaultEncryptor) Decrypt(_ context.Context, encrypted string) ([]byte, error) {
	return []byte(encrypted[len("enc:"):]), nil
}

// ---------------------------------------------------------------------------
// stubFolderRepo implements message.FolderRepository over an in-memory map.
// ---------------------------------------------------------------------------

type stubFolderRepo struct {
	folders map[uuid.UUID]*models.EmailFolder
}

func newStubFolderRepo() *stubFolderRepo {
	return &stubFolderRepo{folders: make(map[uuid.UUID]*models.EmailFolder)}
}

func (r *stubFolderRepo) Create(_ context.Context, f *models.EmailFolder) error {
	r.folders[f.ID] = f
	return nil
}

func (r *stubFolderRepo) GetByID(_ context.Context, id uuid.UUID) (*models.EmailFolder, error) {
	f, ok := r.folders[id]
	if !ok {
		return nil, message.ErrFolderNotFound
	}
	return f, nil
}

func (r *stubFolderRepo) GetByIMAPName(_ context.Context, accountID uuid.UUID, imapName string) (*models.EmailFolder, error) {
	for _, f := range r.folders {
		if f.AccountID == accountID && f.IMAPName == imapName {
			return f, nil
		}
	}
	return nil, message.ErrFolderNotFound
}

func (r *stubFolderRepo) ListByAccount(_ context.Context, accountID uuid.UUID) ([]*models.EmailFolder, error) {
	out := make([]*models.EmailFolder, 0)
	for _, f := range r.folders {
		if f.AccountID == accountID {
			out = append(out, f)
		}
	}
	return out, nil
}

func (r *stubFolderRepo) GetByAccountAndType(_ context.Context, accountID uuid.UUID, folderType string) (*models.EmailFolder, error) {
	for _, f := range r.folders {
		if f.AccountID == accountID && f.FolderType == folderType {
			return f, nil
		}
	}
	return nil, message.ErrFolderNotFound
}

func (r *stubFolderRepo) UpdateCounts(_ context.Context, folderID uuid.UUID, messageCount, unreadCount int) error {
	f, ok := r.folders[folderID]
	if !ok {
		return message.ErrFolderNotFound
	}
	f.MessageCount = messageCount
	f.UnreadCount = unreadCount
	return nil
}

func (r *stubFolderRepo) UpdateUIDValidity(_ context.Context, folderID uuid.UUID, uidValidity int64) error {
	f, ok := r.folders[folderID]
	if !ok {
		return message.ErrFolderNotFound
	}
	f.UIDValidity = uidValidity
	return nil
}

func (r *stubFolderRepo) DeleteMessagesByFolder(_ context.Context, _ uuid.UUID) error {
	return nil
}

// newTestEmailAccountsServer wires an EmailGRPCServer for the account/folder/sync
// RPCs only. The sync engine's MessageSyncer/FolderSyncer/AttachmentStorer are nil
// because none of TriggerSync/GetStatus/StopWorker (the only engine methods these
// RPCs reach) touch them -- only StartWorker's spawned worker goroutine would, and
// tests here never enable sync on account creation, so that path is never taken.
// messageService's own Repository is nil for the same reason: ListFolders and
// GetFolderByID only read through folderRepo.
func newTestEmailAccountsServer(accountRepo account.Repository, folderRepo message.FolderRepository) *EmailGRPCServer {
	acctSvc := account.NewService(accountRepo, fakeVaultEncryptor{})
	msgSvc := message.NewService(nil, folderRepo)
	engine := emailsync.NewEngine(acctSvc, nil, nil, nil)
	return NewEmailGRPCServer(acctSvc, msgSvc, nil, nil, nil, engine, nil, nil, nil, nil, nil, nil)
}

func seedEmailAccount(repo *stubAccountRepo, tenantID, userID uuid.UUID, isDefault bool) *models.EmailAccount {
	a := &models.EmailAccount{
		ID:           uuid.New(),
		TenantID:     tenantID,
		UserID:       userID,
		EmailAddress: "seed@example.com",
		IMAPHost:     "imap.example.com",
		IMAPPort:     993,
		SMTPHost:     "smtp.example.com",
		SMTPPort:     587,
		Username:     "seed",
		SyncEnabled:  false,
		IsDefault:    isDefault,
	}
	repo.accounts[a.ID] = a
	return a
}

// ---------------------------------------------------------------------------
// mapEmailError -- every sentinel against its expected gRPC code.
// ---------------------------------------------------------------------------

func TestMapEmailError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want codes.Code
	}{
		{"account not found", account.ErrAccountNotFound, codes.NotFound},
		{"invalid credentials", account.ErrInvalidCredentials, codes.Unauthenticated},
		{"connection failed", account.ErrConnectionFailed, codes.Unavailable},
		{"message not found", message.ErrMessageNotFound, codes.NotFound},
		{"folder not found", message.ErrFolderNotFound, codes.NotFound},
		{"thread not found", message.ErrThreadNotFound, codes.NotFound},
		{"unknown bulk action", message.ErrUnknownBulkAction, codes.OutOfRange},
		{"bulk target required", message.ErrBulkTargetRequired, codes.InvalidArgument},
		{"send failed", send.ErrSendFailed, codes.Internal},
		{"smtp auth failed", send.ErrSMTPAuthFailed, codes.Unauthenticated},
		{"invalid recipient", send.ErrInvalidRecipient, codes.InvalidArgument},
		{"signature not found", signature.ErrSignatureNotFound, codes.NotFound},
		{"attachment not found", attachment.ErrAttachmentNotFound, codes.NotFound},
		{"attachment too large", attachment.ErrAttachmentTooLarge, codes.InvalidArgument},
		{"sync in progress", emailsync.ErrSyncInProgress, codes.AlreadyExists},
		{"rule not found", rule.ErrRuleNotFound, codes.NotFound},
		{"rule target not found", rule.ErrTargetNotFound, codes.NotFound},
		{"rule invalid field", rule.ErrInvalidField, codes.InvalidArgument},
		{"rule invalid operator", rule.ErrInvalidOperator, codes.InvalidArgument},
		{"rule invalid action type", rule.ErrInvalidActionType, codes.InvalidArgument},
		{"rule name required", rule.ErrNameRequired, codes.InvalidArgument},
		{"rule value required", rule.ErrValueRequired, codes.InvalidArgument},
		{"rule target required", rule.ErrTargetRequired, codes.InvalidArgument},
		{"label not found", label.ErrNotFound, codes.NotFound},
		{"label message not found", label.ErrMessageNotFound, codes.NotFound},
		{"label duplicate name", label.ErrDuplicateName, codes.AlreadyExists},
		{"label name required", label.ErrNameRequired, codes.InvalidArgument},
		{"label color invalid", label.ErrColorInvalid, codes.InvalidArgument},
		{"label target invalid", label.ErrTargetInvalid, codes.InvalidArgument},
		{"template not found", template.ErrTemplateNotFound, codes.NotFound},
		{"template name required", template.ErrNameRequired, codes.InvalidArgument},
		{"template invalid visibility", template.ErrInvalidVisibility, codes.InvalidArgument},
		{"import failed", emailcontact.ErrImportFailed, codes.Internal},
		{"invalid csv", emailcontact.ErrInvalidCSV, codes.InvalidArgument},
		{"invalid vcard", emailcontact.ErrInvalidVCard, codes.InvalidArgument},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			requireGRPCCode(t, mapEmailError(tc.err), tc.want)
		})
	}
}

func TestMapEmailError_Nil(t *testing.T) {
	require.NoError(t, mapEmailError(nil))
}

func TestMapEmailError_Unknown(t *testing.T) {
	requireGRPCCode(t, mapEmailError(context.Canceled), codes.Internal)
}

// ---------------------------------------------------------------------------
// Account management RPCs
// ---------------------------------------------------------------------------

func TestCreateEmailAccount(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	resp, err := srv.CreateEmailAccount(ctx, &emailv1.CreateEmailAccountRequest{
		UserId:       userID.String(),
		EmailAddress: "new@example.com",
		ImapHost:     "imap.example.com",
		SmtpHost:     "smtp.example.com",
		Username:     "new",
		Password:     "s3cret",
	})
	requireGRPCOK(t, err)
	require.True(t, resp.Account.IsDefault, "first account for a user must become the default")
	require.Equal(t, "new@example.com", resp.Account.EmailAddress)
}

func TestCreateEmailAccount_InvalidUserID(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())
	ctx := ctxWithActorAndTenant(uuid.New(), uuid.New())

	_, err := srv.CreateEmailAccount(ctx, &emailv1.CreateEmailAccountRequest{UserId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetEmailAccount(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	seedEmailAccount(repo, tenantID, userID, true)
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	resp, err := srv.GetEmailAccount(ctx, &emailv1.GetEmailAccountRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.Equal(t, "seed@example.com", resp.Account.EmailAddress)
}

func TestGetEmailAccount_NotFound(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	_, err := srv.GetEmailAccount(ctx, &emailv1.GetEmailAccountRequest{UserId: userID.String()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestListEmailAccounts(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	seedEmailAccount(repo, tenantID, userID, true)
	seedEmailAccount(repo, tenantID, userID, false)
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	resp, err := srv.ListEmailAccounts(ctx, &emailv1.ListEmailAccountsRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Accounts, 2)
}

func TestListEmailAccounts_Empty(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	resp, err := srv.ListEmailAccounts(ctx, &emailv1.ListEmailAccountsRequest{UserId: userID.String()})
	requireGRPCOK(t, err)
	require.NotNil(t, resp.Accounts, "wire shape must be [] not null for an empty list")
	require.Empty(t, resp.Accounts)
}

func TestListEmailAccounts_InvalidUserID(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())
	ctx := ctxWithActorAndTenant(uuid.New(), uuid.New())

	_, err := srv.ListEmailAccounts(ctx, &emailv1.ListEmailAccountsRequest{UserId: "garbage"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestUpdateEmailAccount(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	acct := seedEmailAccount(repo, tenantID, userID, true)
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	newName := "Renamed"
	resp, err := srv.UpdateEmailAccount(ctx, &emailv1.UpdateEmailAccountRequest{
		Id:          acct.ID.String(),
		DisplayName: &newName,
	})
	requireGRPCOK(t, err)
	require.Equal(t, "Renamed", resp.Account.DisplayName)
}

func TestUpdateEmailAccount_InvalidID(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())
	ctx := ctxWithActorAndTenant(uuid.New(), uuid.New())

	_, err := srv.UpdateEmailAccount(ctx, &emailv1.UpdateEmailAccountRequest{Id: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestDeleteEmailAccount(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	acct := seedEmailAccount(repo, tenantID, userID, true)
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	_, err := srv.DeleteEmailAccount(ctx, &emailv1.DeleteEmailAccountRequest{Id: acct.ID.String()})
	requireGRPCOK(t, err)

	_, getErr := srv.GetEmailAccount(ctx, &emailv1.GetEmailAccountRequest{UserId: userID.String()})
	requireGRPCCode(t, getErr, codes.NotFound)
}

func TestDeleteEmailAccount_NotFound(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	_, err := srv.DeleteEmailAccount(ctx, &emailv1.DeleteEmailAccountRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSetDefaultEmailAccount(t *testing.T) {
	tenantID, userID := uuid.New(), uuid.New()
	repo := newStubAccountRepo()
	seedEmailAccount(repo, tenantID, userID, true)
	second := seedEmailAccount(repo, tenantID, userID, false)
	srv := newTestEmailAccountsServer(repo, newStubFolderRepo())
	ctx := ctxWithActorAndTenant(userID, tenantID)

	resp, err := srv.SetDefaultEmailAccount(ctx, &emailv1.SetDefaultEmailAccountRequest{Id: second.ID.String()})
	requireGRPCOK(t, err)
	require.True(t, resp.Account.IsDefault)

	for id, a := range repo.accounts {
		if id != second.ID {
			require.False(t, a.IsDefault, "only the newly set account may stay default")
		}
	}
}

func TestSetDefaultEmailAccount_NotFound(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())
	ctx := ctxWithActorAndTenant(uuid.New(), uuid.New())

	_, err := srv.SetDefaultEmailAccount(ctx, &emailv1.SetDefaultEmailAccountRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestTestEmailConnection_Unreachable(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	resp, err := srv.TestEmailConnection(context.Background(), &emailv1.TestEmailConnectionRequest{
		ImapHost: "127.0.0.1",
		ImapPort: 1, // nothing listens here; dial must fail fast, not hang
		Username: "x",
		Password: "x",
	})
	requireGRPCOK(t, err) // a connection failure is reported in the response, not as a gRPC error
	require.False(t, resp.ImapOk)
	require.NotEmpty(t, resp.ImapError)
}

// ---------------------------------------------------------------------------
// Folder + sync RPCs
// ---------------------------------------------------------------------------

func seedEmailFolder(repo *stubFolderRepo, accountID uuid.UUID) *models.EmailFolder {
	f := &models.EmailFolder{
		ID:        uuid.New(),
		AccountID: accountID,
		Name:      "INBOX",
		IMAPName:  "INBOX",
	}
	repo.folders[f.ID] = f
	return f
}

func TestListFolders(t *testing.T) {
	accountID := uuid.New()
	folderRepo := newStubFolderRepo()
	seedEmailFolder(folderRepo, accountID)
	srv := newTestEmailAccountsServer(newStubAccountRepo(), folderRepo)

	resp, err := srv.ListFolders(context.Background(), &emailv1.ListFoldersRequest{AccountId: accountID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Folders, 1)
	require.Equal(t, "INBOX", resp.Folders[0].Name)
}

func TestListFolders_InvalidAccountID(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	_, err := srv.ListFolders(context.Background(), &emailv1.ListFoldersRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetFolder(t *testing.T) {
	folderRepo := newStubFolderRepo()
	f := seedEmailFolder(folderRepo, uuid.New())
	srv := newTestEmailAccountsServer(newStubAccountRepo(), folderRepo)

	resp, err := srv.GetFolder(context.Background(), &emailv1.GetFolderRequest{Id: f.ID.String()})
	requireGRPCOK(t, err)
	require.Equal(t, f.ID.String(), resp.Folder.Id)
}

func TestGetFolder_NotFound(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	_, err := srv.GetFolder(context.Background(), &emailv1.GetFolderRequest{Id: uuid.NewString()})
	requireGRPCCode(t, err, codes.NotFound)
}

func TestSyncFolders(t *testing.T) {
	accountID := uuid.New()
	folderRepo := newStubFolderRepo()
	seedEmailFolder(folderRepo, accountID)
	srv := newTestEmailAccountsServer(newStubAccountRepo(), folderRepo)

	// No worker is registered for accountID, so the engine's TriggerSync call
	// inside SyncFolders is a documented no-op -- it must not error or panic.
	resp, err := srv.SyncFolders(context.Background(), &emailv1.SyncFoldersRequest{AccountId: accountID.String()})
	requireGRPCOK(t, err)
	require.Len(t, resp.Folders, 1)
}

func TestSyncFolders_InvalidAccountID(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	_, err := srv.SyncFolders(context.Background(), &emailv1.SyncFoldersRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestTriggerSync(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	resp, err := srv.TriggerSync(context.Background(), &emailv1.TriggerSyncRequest{AccountId: uuid.NewString()})
	requireGRPCOK(t, err)
	require.Equal(t, "started", resp.Status)
}

func TestTriggerSync_InvalidAccountID(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	_, err := srv.TriggerSync(context.Background(), &emailv1.TriggerSyncRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}

func TestGetSyncStatus_Idle(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	resp, err := srv.GetSyncStatus(context.Background(), &emailv1.GetSyncStatusRequest{AccountId: uuid.NewString()})
	requireGRPCOK(t, err)
	require.Equal(t, "idle", resp.Status)
}

func TestGetSyncStatus_InvalidAccountID(t *testing.T) {
	srv := newTestEmailAccountsServer(newStubAccountRepo(), newStubFolderRepo())

	_, err := srv.GetSyncStatus(context.Background(), &emailv1.GetSyncStatusRequest{AccountId: "not-a-uuid"})
	requireGRPCCode(t, err, codes.InvalidArgument)
}
