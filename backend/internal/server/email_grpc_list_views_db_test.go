package server

// DB-backed proof for the mail list views (fix-email-list-views-unified-label-filter):
// the unified inbox (view=unified), the label view (label_id) and the
// unread/starred filter. Before, folder_id="" answered 400 and the other
// parameters never reached the backend, so "Alle Eingänge" and every label
// view were empty for each user.
//
// Real postgres repositories under kmuhub_app, so RLS is active: the tests also
// pin that a user only sees the messages of their own accounts.

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/require"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/email/attachment"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	"github.com/kmuhub/kmuhub/internal/testutil"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

type listViewsFixture struct {
	pool     *pgxpool.Pool
	srv      *EmailGRPCServer
	repo     *message.PostgresRepository
	tenantID uuid.UUID
}

func newListViewsFixture(t *testing.T) *listViewsFixture {
	t.Helper()
	testutil.SkipIfNoDB(t)
	pool := testutil.PoolFromEnv(t)
	t.Cleanup(pool.Close)

	tenantID := uuid.New()
	testutil.EnsureTenant(t, pool, tenantID, "Mail List Views Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, pool, "tenants", tenantID) })

	repo := message.NewPostgresRepository(pool)
	msgSvc := message.NewService(repo, message.NewPostgresFolderRepository(pool))
	accSvc := account.NewService(account.NewPostgresRepository(pool), nil)
	return &listViewsFixture{
		pool:     pool,
		srv:      NewEmailGRPCServer(accSvc, msgSvc, nil, nil, attachment.NewService(attachment.NewPostgresRepository(pool), nil), nil, nil, nil, nil, nil, nil, nil),
		repo:     repo,
		tenantID: tenantID,
	}
}

func (f *listViewsFixture) ctx(tenantID, userID uuid.UUID) context.Context {
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	if userID != uuid.Nil {
		ctx = context.WithValue(ctx, middleware.UserIDKey, userID.String())
	}
	return ctx
}

func (f *listViewsFixture) user(t *testing.T, tenantID uuid.UUID) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, f.pool, "users", map[string]any{
		"tenant_id":     tenantID,
		"email":         fmt.Sprintf("mail-views-%s@tenant.local", uuid.New().String()[:8]),
		"password_hash": "$argon2id$v=19$test",
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "users", id) })
	return id
}

// account seeds a mail account; a user may hold only one default account.
func (f *listViewsFixture) account(t *testing.T, tenantID, userID uuid.UUID, isDefault bool) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, f.pool, "email_accounts", map[string]any{
		"tenant_id":          tenantID,
		"user_id":            userID,
		"email_address":      fmt.Sprintf("acct-%s@tenant.local", uuid.New().String()[:8]),
		"imap_host":          "imap.tenant.local",
		"smtp_host":          "smtp.tenant.local",
		"username":           "acct",
		"display_name":       "Postfach",
		"password_encrypted": "vault-key",
		"is_default":         isDefault,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "email_accounts", id) })
	return id
}

func (f *listViewsFixture) folder(t *testing.T, tenantID, accountID uuid.UUID, folderType string) uuid.UUID {
	t.Helper()
	id := testutil.SeedRow(t, f.pool, "email_folders", map[string]any{
		"tenant_id":   tenantID,
		"account_id":  accountID,
		"name":        folderType,
		"imap_name":   folderType,
		"folder_type": folderType,
	})
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "email_folders", id) })
	return id
}

// msg stores one message; offset orders the dates (higher = newer).
func (f *listViewsFixture) msg(t *testing.T, tenantID, accountID, folderID uuid.UUID, offset int, read, starred bool, labels ...uuid.UUID) uuid.UUID {
	t.Helper()
	ctx := testutil.WithTenantCtx(context.Background(), tenantID)
	now := time.Now().UTC()
	m := &models.EmailMessage{
		ID: uuid.New(), TenantID: tenantID, AccountID: accountID, FolderID: folderID,
		UID: int64(offset), MessageID: fmt.Sprintf("<%s@tenant.local>", uuid.New()),
		FromEmail: "absender@example.com", Subject: "Betreff",
		ToAddresses: []models.EmailAddress{}, CcAddresses: []models.EmailAddress{}, BccAddresses: []models.EmailAddress{},
		IsRead: read, IsStarred: starred,
		Date: now.Add(time.Duration(offset) * time.Minute), CreatedAt: now, UpdatedAt: now,
	}
	require.NoError(t, f.repo.Create(ctx, m))
	if len(labels) > 0 {
		ids := make([]string, len(labels))
		for i, l := range labels {
			ids[i] = l.String()
		}
		_, err := f.pool.Exec(ctx, `UPDATE email_messages SET label_ids = $1::uuid[] WHERE id = $2`, ids, m.ID)
		require.NoError(t, err)
	}
	return m.ID
}

func TestEmailGRPC_ListMessages_UnifiedLabelAndFilterViews(t *testing.T) {
	t.Parallel()
	f := newListViewsFixture(t)

	userA, userB := f.user(t, f.tenantID), f.user(t, f.tenantID)
	a1, a2, b1 := f.account(t, f.tenantID, userA, true), f.account(t, f.tenantID, userA, false), f.account(t, f.tenantID, userB, true)
	a1Inbox, a1Sent := f.folder(t, f.tenantID, a1, models.FolderTypeInbox), f.folder(t, f.tenantID, a1, models.FolderTypeSent)
	a2Inbox, b1Inbox := f.folder(t, f.tenantID, a2, models.FolderTypeInbox), f.folder(t, f.tenantID, b1, models.FolderTypeInbox)

	label := uuid.New()
	m1 := f.msg(t, f.tenantID, a1, a1Inbox, 1, false, false)
	m2 := f.msg(t, f.tenantID, a1, a1Inbox, 2, true, true, label)
	f.msg(t, f.tenantID, a1, a1Sent, 3, false, false)               // sent mail: never in the unified inbox
	m4 := f.msg(t, f.tenantID, a2, a2Inbox, 4, false, false, label) // second account of user A
	f.msg(t, f.tenantID, b1, b1Inbox, 5, false, false, label)       // user B: never visible to user A

	// Another tenant carrying the same label id must stay invisible.
	otherTenant := uuid.New()
	testutil.EnsureTenant(t, f.pool, otherTenant, "Mail List Views Other Tenant")
	t.Cleanup(func() { testutil.CleanupRow(t, f.pool, "tenants", otherTenant) })
	userC := f.user(t, otherTenant)
	c1 := f.account(t, otherTenant, userC, true)
	c1Inbox := f.folder(t, otherTenant, c1, models.FolderTypeInbox)
	m6 := f.msg(t, otherTenant, c1, c1Inbox, 6, false, false, label)

	ctxA := f.ctx(f.tenantID, userA)
	idStrings := func(ids ...uuid.UUID) []string {
		out := make([]string, len(ids))
		for i, id := range ids {
			out[i] = id.String()
		}
		return out
	}
	list := func(ctx context.Context, req *emailv1.ListMessagesRequest) ([]string, error) {
		resp, err := f.srv.ListMessages(ctx, req)
		if err != nil {
			return nil, err
		}
		out := make([]string, 0, len(resp.GetMessages()))
		for _, m := range resp.GetMessages() {
			out = append(out, m.GetId())
		}
		require.EqualValues(t, len(out), resp.GetTotal())
		return out, nil
	}

	t.Run("unified lists the inboxes of the caller's accounts only", func(t *testing.T) {
		got, err := list(ctxA, &emailv1.ListMessagesRequest{View: "unified", SortDesc: true})
		require.NoError(t, err)
		require.Equal(t, idStrings(m4, m2, m1), got)
	})

	t.Run("unified + unread", func(t *testing.T) {
		got, err := list(ctxA, &emailv1.ListMessagesRequest{View: "unified", Filter: "unread", SortDesc: true})
		require.NoError(t, err)
		require.Equal(t, idStrings(m4, m1), got)
	})

	t.Run("unified + starred", func(t *testing.T) {
		got, err := list(ctxA, &emailv1.ListMessagesRequest{View: "unified", Filter: "starred"})
		require.NoError(t, err)
		require.Equal(t, idStrings(m2), got)
	})

	t.Run("unified narrowed to one own account", func(t *testing.T) {
		got, err := list(ctxA, &emailv1.ListMessagesRequest{View: "unified", AccountId: a2.String()})
		require.NoError(t, err)
		require.Equal(t, idStrings(m4), got)
	})

	t.Run("account of another user is rejected, not listed", func(t *testing.T) {
		_, err := list(ctxA, &emailv1.ListMessagesRequest{View: "unified", AccountId: b1.String()})
		require.Equal(t, codes.NotFound, status.Code(err))
	})

	t.Run("label view spans folders and accounts, but not other users or tenants", func(t *testing.T) {
		got, err := list(ctxA, &emailv1.ListMessagesRequest{LabelId: label.String(), SortDesc: true})
		require.NoError(t, err)
		require.Equal(t, idStrings(m4, m2), got)
	})

	t.Run("label view + unread + account", func(t *testing.T) {
		got, err := list(ctxA, &emailv1.ListMessagesRequest{LabelId: label.String(), Filter: "unread", AccountId: a1.String()})
		require.NoError(t, err)
		require.Empty(t, got) // m2 is read, m4 belongs to a2
	})

	t.Run("filter works inside a folder", func(t *testing.T) {
		got, err := list(ctxA, &emailv1.ListMessagesRequest{FolderId: a1Inbox.String(), Filter: "unread"})
		require.NoError(t, err)
		require.Equal(t, idStrings(m1), got)
	})

	t.Run("plain folder view is unchanged", func(t *testing.T) {
		got, err := list(ctxA, &emailv1.ListMessagesRequest{FolderId: a1Inbox.String()})
		require.NoError(t, err)
		require.Equal(t, idStrings(m1, m2), got)
	})

	t.Run("the other tenant's user sees only their own mail", func(t *testing.T) {
		got, err := list(f.ctx(otherTenant, userC), &emailv1.ListMessagesRequest{View: "unified"})
		require.NoError(t, err)
		require.Equal(t, idStrings(m6), got)
	})

	t.Run("a user without accounts gets an empty list, not an error", func(t *testing.T) {
		resp, err := f.srv.ListMessages(f.ctx(f.tenantID, f.user(t, f.tenantID)), &emailv1.ListMessagesRequest{View: "unified"})
		require.NoError(t, err)
		require.NotNil(t, resp.GetMessages())
		require.Empty(t, resp.GetMessages())
	})

	t.Run("invalid input", func(t *testing.T) {
		for name, req := range map[string]*emailv1.ListMessagesRequest{
			"nothing to list":  {},
			"unknown filter":   {View: "unified", Filter: "spam"},
			"unknown view":     {View: "everything"},
			"bad label id":     {LabelId: "nope"},
			"bad account id":   {View: "unified", AccountId: "nope"},
			"bad folder id":    {FolderId: "nope"},
			"bad folder+label": {FolderId: "nope", LabelId: label.String()},
		} {
			_, err := f.srv.ListMessages(ctxA, req)
			require.Equal(t, codes.InvalidArgument, status.Code(err), name)
		}
	})

	t.Run("account-wide view needs a caller", func(t *testing.T) {
		_, err := f.srv.ListMessages(f.ctx(f.tenantID, uuid.Nil), &emailv1.ListMessagesRequest{View: "unified"})
		require.Equal(t, codes.Unauthenticated, status.Code(err))
	})
}

// TestMailbox_DB_SecondUserSeesNothingOfTheFirst runs the mailbox owner check of
// fix-email-module-for-members against Postgres as kmuhub_app: two employees of one
// tenant, RLS lets both rows through, the owner check must not. B gets NotFound for
// A's account, folders and messages and cannot send from A's account; A still can.
func TestMailbox_DB_SecondUserSeesNothingOfTheFirst(t *testing.T) {
	f := newListViewsFixture(t)
	a, b := f.user(t, f.tenantID), f.user(t, f.tenantID)
	acctA := f.account(t, f.tenantID, a, true)
	inboxA := f.folder(t, f.tenantID, acctA, models.FolderTypeInbox)
	msgA := f.msg(t, f.tenantID, acctA, inboxA, 1, false, false)
	asA, asB := f.ctx(f.tenantID, a), f.ctx(f.tenantID, b)

	_, err := f.srv.GetMessage(asA, &emailv1.GetMessageRequest{Id: msgA.String()})
	require.NoError(t, err, "the owner reads their own mail")
	list, err := f.srv.ListFolders(asA, &emailv1.ListFoldersRequest{AccountId: acctA.String()})
	require.NoError(t, err)
	require.Len(t, list.Folders, 1)

	for name, call := range map[string]func() error{
		"get message": func() error { _, e := f.srv.GetMessage(asB, &emailv1.GetMessageRequest{Id: msgA.String()}); return e },
		"list folders": func() error {
			_, e := f.srv.ListFolders(asB, &emailv1.ListFoldersRequest{AccountId: acctA.String()})
			return e
		},
		"get folder": func() error { _, e := f.srv.GetFolder(asB, &emailv1.GetFolderRequest{Id: inboxA.String()}); return e },
		"list folder messages": func() error {
			_, e := f.srv.ListMessages(asB, &emailv1.ListMessagesRequest{FolderId: inboxA.String()})
			return e
		},
		"mark read": func() error { _, e := f.srv.MarkRead(asB, &emailv1.MarkReadRequest{Id: msgA.String()}); return e },
		"set default": func() error {
			_, e := f.srv.SetDefaultEmailAccount(asB, &emailv1.SetDefaultEmailAccountRequest{Id: acctA.String()})
			return e
		},
		"send from A": func() error {
			_, e := f.srv.SendEmail(asB, &emailv1.SendEmailRequest{AccountId: acctA.String(), To: []*emailv1.EmailAddress{{Email: "x@example.com"}}})
			return e
		},
	} {
		t.Run(name, func(t *testing.T) {
			require.Equal(t, codes.NotFound, status.Code(call()))
		})
	}

	unified, err := f.srv.ListMessages(asB, &emailv1.ListMessagesRequest{View: "unified"})
	require.NoError(t, err)
	require.Empty(t, unified.Messages, "B's unified inbox holds none of A's mail")

	var read bool
	require.NoError(t, f.pool.QueryRow(testutil.WithTenantCtx(context.Background(), f.tenantID),
		"SELECT is_read FROM email_messages WHERE id = $1", msgA).Scan(&read))
	require.False(t, read, "B's mark-read must not have touched A's message")
}
