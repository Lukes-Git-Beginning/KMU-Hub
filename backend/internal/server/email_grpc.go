package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"slices"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	crmcompany "github.com/kmuhub/kmuhub/internal/crm/company"
	crmcontact "github.com/kmuhub/kmuhub/internal/crm/contact"
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
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// EmailGRPCServer implements the Email gRPC service.
type EmailGRPCServer struct {
	emailv1.UnimplementedEmailServiceServer
	accountService    *account.Service
	accounts          emailAccountLookup
	messageService    *message.Service
	sendService       *send.Service
	signatureService  *signature.Service
	attachmentService *attachment.Service
	syncEngine        *emailsync.Engine
	linkRepo          EmailContactLinkRepository
	contactService    *crmcontact.Service
	companyService    *crmcompany.Service
	ruleService       *rule.Service
	labelService      *label.Service
	templateService   *template.Service
}

// EmailContactLinkRepository defines the interface for CRM email linking.
type EmailContactLinkRepository interface {
	Create(ctx context.Context, link *models.EmailContactLink) error
	GetByMessageID(ctx context.Context, messageID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailContactLink, error)
	GetByContactID(ctx context.Context, contactID uuid.UUID, tenantID uuid.UUID, page, perPage int) ([]*models.EmailMessage, int, error)
	Delete(ctx context.Context, messageID, contactID uuid.UUID, tenantID uuid.UUID) error
}

// NewEmailGRPCServer creates a new Email gRPC server.
func NewEmailGRPCServer(
	accountService *account.Service,
	messageService *message.Service,
	sendService *send.Service,
	signatureService *signature.Service,
	attachmentService *attachment.Service,
	syncEngine *emailsync.Engine,
	linkRepo EmailContactLinkRepository,
	contactService *crmcontact.Service,
	companyService *crmcompany.Service,
	ruleService *rule.Service,
	labelService *label.Service,
	templateService *template.Service,
) *EmailGRPCServer {
	return &EmailGRPCServer{
		accountService:    accountService,
		accounts:          accountService,
		messageService:    messageService,
		sendService:       sendService,
		signatureService:  signatureService,
		attachmentService: attachmentService,
		syncEngine:        syncEngine,
		linkRepo:          linkRepo,
		contactService:    contactService,
		companyService:    companyService,
		ruleService:       ruleService,
		labelService:      labelService,
		templateService:   templateService,
	}
}

// emailAccountLookup is the part of account.Service the mailbox owner checks
// need; tests swap in a stub.
type emailAccountLookup interface {
	GetByID(ctx context.Context, id uuid.UUID, tenantID uuid.UUID) (*models.EmailAccount, error)
	ListByUserAndTenant(ctx context.Context, userID uuid.UUID, tenantID uuid.UUID) ([]*models.EmailAccount, error)
}

// ownAccount returns the account when the caller holds it. A mailbox belongs to
// exactly one user (email_accounts.user_id; there are no shared mailboxes), so
// every read or write of its folders, messages, attachments, links and sync
// state is confined to that user, admins included. Admins provision accounts
// for others through the account routes, which never touch mailbox content.
// Someone else's account answers NotFound, so an ID does not reveal that it
// exists.
func (s *EmailGRPCServer) ownAccount(ctx context.Context, accountID uuid.UUID) (*models.EmailAccount, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	caller, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	acct, err := s.accounts.GetByID(ctx, accountID, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}
	if acct.UserID != caller {
		return nil, status.Error(codes.NotFound, account.ErrAccountNotFound.Error())
	}
	return acct, nil
}

// ownAccountIDs lists the IDs of the caller's own accounts.
func (s *EmailGRPCServer) ownAccountIDs(ctx context.Context) ([]uuid.UUID, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	caller, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	accounts, err := s.accounts.ListByUserAndTenant(ctx, caller, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}
	ids := make([]uuid.UUID, 0, len(accounts))
	for _, a := range accounts {
		ids = append(ids, a.ID)
	}
	return ids, nil
}

// ownFolder returns the folder when it lies in one of the caller's accounts.
func (s *EmailGRPCServer) ownFolder(ctx context.Context, folderID uuid.UUID) (*models.EmailFolder, error) {
	folder, err := s.messageService.GetFolderByID(ctx, folderID)
	if err != nil {
		return nil, mapEmailError(err)
	}
	if _, err := s.ownAccount(ctx, folder.AccountID); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, message.ErrFolderNotFound.Error())
		}
		return nil, err
	}
	return folder, nil
}

// ownMessage returns the message when it lies in one of the caller's accounts.
func (s *EmailGRPCServer) ownMessage(ctx context.Context, msgID uuid.UUID) (*models.EmailMessage, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	msg, err := s.messageService.GetByID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}
	if _, err := s.ownAccount(ctx, msg.AccountID); err != nil {
		if status.Code(err) == codes.NotFound {
			return nil, status.Error(codes.NotFound, message.ErrMessageNotFound.Error())
		}
		return nil, err
	}
	return msg, nil
}

// ownSignature returns the signature when the caller wrote it.
func (s *EmailGRPCServer) ownSignature(ctx context.Context, id uuid.UUID) (*models.EmailSignature, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	caller, err := callerID(ctx)
	if err != nil {
		return nil, err
	}
	sig, err := s.signatureService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}
	if sig.UserID != caller {
		return nil, status.Error(codes.NotFound, signature.ErrSignatureNotFound.Error())
	}
	return sig, nil
}

// ============================================================================
// Account Management
// ============================================================================

func (s *EmailGRPCServer) CreateEmailAccount(ctx context.Context, req *emailv1.CreateEmailAccountRequest) (*emailv1.CreateEmailAccountResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	input := account.CreateInput{
		UserID:       userID,
		EmailAddress: req.EmailAddress,
		DisplayName:  req.DisplayName,
		IMAPHost:     req.ImapHost,
		IMAPPort:     int(req.ImapPort),
		SMTPHost:     req.SmtpHost,
		SMTPPort:     int(req.SmtpPort),
		Username:     req.Username,
		Password:     req.Password,
		UseSSL:       req.UseSsl,
	}

	acct, err := s.accountService.Create(ctx, tenantID, input)
	if err != nil {
		return nil, mapEmailError(err)
	}

	// Start sync worker for the new account
	if req.SyncEnabled {
		if syncErr := s.syncEngine.StartWorker(ctx, acct.ID); syncErr != nil {
			slog.Warn("failed to start sync worker for new account", "account_id", acct.ID, "error", syncErr)
		}
	}

	return &emailv1.CreateEmailAccountResponse{
		Account: toEmailAccountInfo(acct),
	}, nil
}

func (s *EmailGRPCServer) GetEmailAccount(ctx context.Context, req *emailv1.GetEmailAccountRequest) (*emailv1.GetEmailAccountResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	acct, err := s.accountService.GetByUserIDAndTenant(ctx, userID, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.GetEmailAccountResponse{
		Account: toEmailAccountInfo(acct),
	}, nil
}

func (s *EmailGRPCServer) ListEmailAccounts(ctx context.Context, req *emailv1.ListEmailAccountsRequest) (*emailv1.ListEmailAccountsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	accounts, err := s.accountService.ListByUserAndTenant(ctx, userID, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoAccounts := make([]*emailv1.EmailAccountInfo, 0, len(accounts))
	for _, acct := range accounts {
		protoAccounts = append(protoAccounts, toEmailAccountInfo(acct))
	}

	return &emailv1.ListEmailAccountsResponse{Accounts: protoAccounts}, nil
}

func (s *EmailGRPCServer) UpdateEmailAccount(ctx context.Context, req *emailv1.UpdateEmailAccountRequest) (*emailv1.UpdateEmailAccountResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account id")
	}

	input := account.UpdateInput{}
	if req.DisplayName != nil {
		v := *req.DisplayName
		input.DisplayName = &v
	}
	if req.ImapHost != nil {
		v := *req.ImapHost
		input.IMAPHost = &v
	}
	if req.ImapPort != nil {
		v := int(*req.ImapPort)
		input.IMAPPort = &v
	}
	if req.SmtpHost != nil {
		v := *req.SmtpHost
		input.SMTPHost = &v
	}
	if req.SmtpPort != nil {
		v := int(*req.SmtpPort)
		input.SMTPPort = &v
	}
	if req.Username != nil {
		v := *req.Username
		input.Username = &v
	}
	if req.Password != nil {
		v := *req.Password
		input.Password = &v
	}
	if req.UseSsl != nil {
		v := *req.UseSsl
		input.UseSSL = &v
	}
	if req.SyncEnabled != nil {
		v := *req.SyncEnabled
		input.SyncEnabled = &v
	}

	acct, err := s.accountService.Update(ctx, id, tenantID, input)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.UpdateEmailAccountResponse{
		Account: toEmailAccountInfo(acct),
	}, nil
}

func (s *EmailGRPCServer) DeleteEmailAccount(ctx context.Context, req *emailv1.DeleteEmailAccountRequest) (*emailv1.DeleteEmailAccountResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account id")
	}

	// Stop sync worker before deleting
	s.syncEngine.StopWorker(id)

	if err := s.accountService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.DeleteEmailAccountResponse{}, nil
}

func (s *EmailGRPCServer) SetDefaultEmailAccount(ctx context.Context, req *emailv1.SetDefaultEmailAccountRequest) (*emailv1.SetDefaultEmailAccountResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account id")
	}

	if _, err := s.ownAccount(ctx, id); err != nil {
		return nil, err
	}

	if err := s.accountService.SetDefault(ctx, id, tenantID); err != nil {
		return nil, mapEmailError(err)
	}

	acct, err := s.accountService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.SetDefaultEmailAccountResponse{Account: toEmailAccountInfo(acct)}, nil
}

func (s *EmailGRPCServer) TestEmailConnection(ctx context.Context, req *emailv1.TestEmailConnectionRequest) (*emailv1.TestEmailConnectionResponse, error) {
	input := account.CreateInput{
		IMAPHost: req.ImapHost,
		IMAPPort: int(req.ImapPort),
		SMTPHost: req.SmtpHost,
		SMTPPort: int(req.SmtpPort),
		Username: req.Username,
		Password: req.Password,
		UseSSL:   req.UseSsl,
	}

	resp := &emailv1.TestEmailConnectionResponse{
		ImapOk: true,
		SmtpOk: true,
	}

	if err := s.accountService.TestConnection(ctx, input); err != nil {
		resp.ImapOk = false
		resp.ImapError = err.Error()
	}

	return resp, nil
}

// ============================================================================
// Folder Operations
// ============================================================================

func (s *EmailGRPCServer) ListFolders(ctx context.Context, req *emailv1.ListFoldersRequest) (*emailv1.ListFoldersResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	if _, err := s.ownAccount(ctx, accountID); err != nil {
		return nil, err
	}

	folders, err := s.messageService.ListFolders(ctx, accountID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoFolders := make([]*emailv1.EmailFolderInfo, 0, len(folders))
	for _, f := range folders {
		protoFolders = append(protoFolders, toEmailFolderInfo(f))
	}

	return &emailv1.ListFoldersResponse{Folders: protoFolders}, nil
}

func (s *EmailGRPCServer) GetFolder(ctx context.Context, req *emailv1.GetFolderRequest) (*emailv1.GetFolderResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder id")
	}

	folder, err := s.ownFolder(ctx, id)
	if err != nil {
		return nil, err
	}

	return &emailv1.GetFolderResponse{Folder: toEmailFolderInfo(folder)}, nil
}

func (s *EmailGRPCServer) SyncFolders(ctx context.Context, req *emailv1.SyncFoldersRequest) (*emailv1.SyncFoldersResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	if _, err := s.ownAccount(ctx, accountID); err != nil {
		return nil, err
	}

	// Trigger sync and return current folders
	s.syncEngine.TriggerSync(accountID)

	folders, err := s.messageService.ListFolders(ctx, accountID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoFolders := make([]*emailv1.EmailFolderInfo, 0, len(folders))
	for _, f := range folders {
		protoFolders = append(protoFolders, toEmailFolderInfo(f))
	}

	return &emailv1.SyncFoldersResponse{Folders: protoFolders}, nil
}

// ============================================================================
// Message Operations
// ============================================================================

func (s *EmailGRPCServer) ListMessages(ctx context.Context, req *emailv1.ListMessagesRequest) (*emailv1.ListMessagesResponse, error) {
	if req.FolderId == "" || req.View != "" || req.LabelId != "" || req.Filter != "" || req.AccountId != "" {
		return s.listMessagesFiltered(ctx, req)
	}

	folderID, err := uuid.Parse(req.FolderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder_id")
	}
	folder, err := s.ownFolder(ctx, folderID)
	if err != nil {
		return nil, err
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	perPage := int(req.PerPage)
	if perPage < 1 {
		perPage = 50
	}

	sortBy := req.SortBy
	if sortBy == "" {
		sortBy = "date"
	}
	sortDir := "ASC"
	if req.SortDesc {
		sortDir = "DESC"
	}

	opts := message.ListOpts{
		Page:    page,
		PerPage: perPage,
		SortBy:  sortBy,
		SortDir: sortDir,
	}

	var messages []*models.EmailMessage
	var total int

	if req.Search != "" {
		messages, total, err = s.messageService.Search(ctx, folder.AccountID, req.Search, opts)
	} else {
		messages, total, err = s.messageService.ListByFolder(ctx, folderID, opts)
	}
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoMsgs := make([]*emailv1.EmailMessageInfo, 0, len(messages))
	for _, m := range messages {
		protoMsgs = append(protoMsgs, toEmailMessageInfo(m))
	}

	return &emailv1.ListMessagesResponse{
		Messages: protoMsgs,
		Total:    int32(total),
	}, nil
}

// listMessagesFiltered serves the list forms the plain folder view cannot:
// the unified inbox (view=unified), a label view (label_id) and the
// unread/starred filter. A folder_id, when given, still narrows the result.
// Account-wide views are confined to the calling user's own accounts; an
// account_id from the client only narrows them further.
func (s *EmailGRPCServer) listMessagesFiltered(ctx context.Context, req *emailv1.ListMessagesRequest) (*emailv1.ListMessagesResponse, error) {
	f := message.ListFilter{SearchQuery: req.Search}

	switch req.Filter {
	case "", "all":
	case "unread":
		f.UnreadOnly = true
	case "starred":
		f.StarredOnly = true
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid filter")
	}
	switch req.View {
	case "", "unified":
	default:
		return nil, status.Error(codes.InvalidArgument, "invalid view")
	}

	var err error
	if req.FolderId != "" {
		if f.FolderID, err = uuid.Parse(req.FolderId); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid folder_id")
		}
	}
	if req.LabelId != "" {
		if f.LabelID, err = uuid.Parse(req.LabelId); err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid label_id")
		}
	}
	if f.FolderID == uuid.Nil && f.LabelID == uuid.Nil && req.View != "unified" {
		return nil, status.Error(codes.InvalidArgument, "folder_id, label_id or view=unified required")
	}
	f.InboxOnly = f.FolderID == uuid.Nil && f.LabelID == uuid.Nil
	if f.FolderID != uuid.Nil {
		if _, err := s.ownFolder(ctx, f.FolderID); err != nil {
			return nil, err
		}
	}

	if f.FolderID == uuid.Nil {
		tenantID, tErr := middleware.GetTenantID(ctx)
		if tErr != nil {
			return nil, status.Error(codes.InvalidArgument, "missing tenant context")
		}
		userID, uErr := callerID(ctx)
		if uErr != nil {
			return nil, uErr
		}
		accounts, aErr := s.accountService.ListByUserAndTenant(ctx, userID, tenantID)
		if aErr != nil {
			return nil, mapEmailError(aErr)
		}
		f.AccountIDs = make([]uuid.UUID, 0, len(accounts))
		for _, a := range accounts {
			f.AccountIDs = append(f.AccountIDs, a.ID)
		}
		if req.AccountId != "" {
			wanted, pErr := uuid.Parse(req.AccountId)
			if pErr != nil {
				return nil, status.Error(codes.InvalidArgument, "invalid account_id")
			}
			if !slices.Contains(f.AccountIDs, wanted) {
				return nil, status.Error(codes.NotFound, "account not found")
			}
			f.AccountIDs = []uuid.UUID{wanted}
		}
	}

	page := max(int(req.Page), 1)
	perPage := int(req.PerPage)
	if perPage < 1 {
		perPage = 50
	}
	sortDir := "ASC"
	if req.SortDesc {
		sortDir = "DESC"
	}
	messages, total, err := s.messageService.List(ctx, f, message.ListOpts{Page: page, PerPage: perPage, SortBy: "date", SortDir: sortDir})
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoMsgs := make([]*emailv1.EmailMessageInfo, 0, len(messages))
	for _, m := range messages {
		protoMsgs = append(protoMsgs, toEmailMessageInfo(m))
	}
	return &emailv1.ListMessagesResponse{Messages: protoMsgs, Total: int32(total)}, nil
}

func (s *EmailGRPCServer) GetMessage(ctx context.Context, req *emailv1.GetMessageRequest) (*emailv1.GetMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	msg, err := s.ownMessage(ctx, id)
	if err != nil {
		return nil, err
	}

	protoMsg := toEmailMessageInfo(msg)

	// Fetch attachments
	attachments, err := s.attachmentService.GetByMessage(ctx, id, tenantID)
	if err == nil {
		for _, att := range attachments {
			protoMsg.Attachments = append(protoMsg.Attachments, toEmailAttachmentInfo(att))
		}
	}

	return &emailv1.GetMessageResponse{Message: protoMsg}, nil
}

func (s *EmailGRPCServer) GetThreadMessages(ctx context.Context, req *emailv1.GetThreadMessagesRequest) (*emailv1.GetThreadMessagesResponse, error) {
	threadID, err := uuid.Parse(req.ThreadId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid thread_id")
	}

	messages, err := s.messageService.ListByThread(ctx, threadID)
	if err != nil {
		return nil, mapEmailError(err)
	}
	own, err := s.ownAccountIDs(ctx)
	if err != nil {
		return nil, err
	}
	messages = slices.DeleteFunc(messages, func(m *models.EmailMessage) bool { return !slices.Contains(own, m.AccountID) })

	protoMsgs := make([]*emailv1.EmailMessageInfo, 0, len(messages))
	for _, m := range messages {
		protoMsgs = append(protoMsgs, toEmailMessageInfo(m))
	}

	return &emailv1.GetThreadMessagesResponse{Messages: protoMsgs}, nil
}

func (s *EmailGRPCServer) MarkRead(ctx context.Context, req *emailv1.MarkReadRequest) (*emailv1.MarkReadResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	if _, err := s.ownMessage(ctx, id); err != nil {
		return nil, err
	}

	if err := s.messageService.MarkRead(ctx, id); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.MarkReadResponse{}, nil
}

func (s *EmailGRPCServer) MarkUnread(ctx context.Context, req *emailv1.MarkUnreadRequest) (*emailv1.MarkUnreadResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	if _, err := s.ownMessage(ctx, id); err != nil {
		return nil, err
	}

	if err := s.messageService.MarkUnread(ctx, id); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.MarkUnreadResponse{}, nil
}

func (s *EmailGRPCServer) ToggleStar(ctx context.Context, req *emailv1.ToggleStarRequest) (*emailv1.ToggleStarResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	if _, err := s.ownMessage(ctx, id); err != nil {
		return nil, err
	}

	if err := s.messageService.ToggleStar(ctx, id, tenantID); err != nil {
		return nil, mapEmailError(err)
	}

	// Retrieve updated state
	msg, err := s.messageService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.ToggleStarResponse{IsStarred: msg.IsStarred}, nil
}

func (s *EmailGRPCServer) MoveToFolder(ctx context.Context, req *emailv1.MoveToFolderRequest) (*emailv1.MoveToFolderResponse, error) {
	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	folderID, err := uuid.Parse(req.TargetFolderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid target_folder_id")
	}

	if _, err := s.ownMessage(ctx, msgID); err != nil {
		return nil, err
	}
	if _, err := s.ownFolder(ctx, folderID); err != nil {
		return nil, err
	}

	if err := s.messageService.MoveToFolder(ctx, msgID, folderID); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.MoveToFolderResponse{}, nil
}

func (s *EmailGRPCServer) DeleteMessage(ctx context.Context, req *emailv1.DeleteMessageRequest) (*emailv1.DeleteMessageResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	if _, err := s.ownMessage(ctx, id); err != nil {
		return nil, err
	}

	if err := s.messageService.Delete(ctx, id); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.DeleteMessageResponse{}, nil
}

func (s *EmailGRPCServer) BulkMessageAction(ctx context.Context, req *emailv1.BulkMessageActionRequest) (*emailv1.BulkMessageActionResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	ids := make([]uuid.UUID, 0, len(req.Ids))
	for _, raw := range req.Ids {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid message id: "+raw)
		}
		ids = append(ids, id)
	}

	ids, err = s.ownMessageIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	if req.Action == "move" {
		if target, pErr := uuid.Parse(req.Target); pErr == nil {
			if _, err := s.ownFolder(ctx, target); err != nil {
				return nil, err
			}
		}
	}

	affected, err := s.messageService.BulkAction(ctx, tenantID, ids, req.Action, req.Target)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.BulkMessageActionResponse{Affected: int32(affected)}, nil
}

// ownMessageIDs keeps the ids that lie in the caller's accounts. Missing and
// foreign ids drop out silently, the same way BulkAction already skips an id
// it cannot find, so the affected count only ever reports the caller's mail.
// lean: one lookup per id (<= 500); a single account-filtered query when bulk gets hot
func (s *EmailGRPCServer) ownMessageIDs(ctx context.Context, ids []uuid.UUID) ([]uuid.UUID, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}
	own, err := s.ownAccountIDs(ctx)
	if err != nil {
		return nil, err
	}
	kept := make([]uuid.UUID, 0, len(ids))
	for _, id := range ids {
		msg, err := s.messageService.GetByID(ctx, id, tenantID)
		if err != nil {
			continue
		}
		if slices.Contains(own, msg.AccountID) {
			kept = append(kept, id)
		}
	}
	return kept, nil
}

// ============================================================================
// Send/Compose Operations
// ============================================================================

func (s *EmailGRPCServer) SendEmail(ctx context.Context, req *emailv1.SendEmailRequest) (*emailv1.SendEmailResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	input := send.SendInput{
		AccountID: accountID,
		TenantID:  tenantID,
		To:        toSendAddresses(req.To),
		CC:        toSendAddresses(req.Cc),
		BCC:       toSendAddresses(req.Bcc),
		Subject:   req.Subject,
		BodyHTML:  req.BodyHtml,
		BodyText:  req.BodyText,
	}

	if req.SignatureId != nil {
		sigID, sigErr := uuid.Parse(*req.SignatureId)
		if sigErr == nil {
			input.SignatureID = &sigID
		}
	}

	// contact_id, when supplied, opts this send into the marketing-consent
	// pre-check (UWG §7). An invalid UUID is rejected rather than silently
	// dropped, so a bad client cannot bypass the consent gate.
	if req.ContactId != nil && *req.ContactId != "" {
		contactID, cidErr := uuid.Parse(*req.ContactId)
		if cidErr != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
		}
		input.ContactID = &contactID
	}

	// Automation sends from the account its workflow names and carries no
	// user. Every call through the gateway carries one and may only send from
	// the caller's own mailbox.
	if middleware.GetUserID(ctx) != "" {
		if _, err := s.ownAccount(ctx, accountID); err != nil {
			return nil, err
		}
	}

	msg, err := s.sendService.Send(ctx, input)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.SendEmailResponse{Message: toEmailMessageInfo(msg)}, nil
}

func (s *EmailGRPCServer) SaveDraft(ctx context.Context, req *emailv1.SaveDraftRequest) (*emailv1.SaveDraftResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	input := send.DraftInput{
		AccountID: accountID,
		TenantID:  tenantID,
		Subject:   req.Subject,
		BodyHTML:  req.BodyHtml,
		BodyText:  req.BodyText,
		To:        toSendAddresses(req.To),
		CC:        toSendAddresses(req.Cc),
		BCC:       toSendAddresses(req.Bcc),
	}

	if _, err := s.ownAccount(ctx, accountID); err != nil {
		return nil, err
	}

	msg, err := s.sendService.SaveDraft(ctx, input)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.SaveDraftResponse{Message: toEmailMessageInfo(msg)}, nil
}

func (s *EmailGRPCServer) ReplyEmail(ctx context.Context, req *emailv1.ReplyEmailRequest) (*emailv1.ReplyEmailResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	origID, err := uuid.Parse(req.OriginalMessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid original_message_id")
	}

	if _, err := s.ownAccount(ctx, accountID); err != nil {
		return nil, err
	}
	origMsg, err := s.ownMessage(ctx, origID)
	if err != nil {
		return nil, err
	}

	input := send.ReplyInput{
		AccountID:   accountID,
		TenantID:    tenantID,
		OriginalMsg: origMsg,
		BodyHTML:    req.BodyHtml,
		BodyText:    req.BodyText,
		To: []send.EmailAddress{
			{Name: origMsg.FromName, Email: origMsg.FromEmail},
		},
	}

	var msg *models.EmailMessage
	if req.ReplyAll {
		msg, err = s.sendService.ReplyAll(ctx, input)
	} else {
		msg, err = s.sendService.Reply(ctx, input)
	}
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.ReplyEmailResponse{Message: toEmailMessageInfo(msg)}, nil
}

func (s *EmailGRPCServer) ForwardEmail(ctx context.Context, req *emailv1.ForwardEmailRequest) (*emailv1.ForwardEmailResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	origID, err := uuid.Parse(req.OriginalMessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid original_message_id")
	}

	if _, err := s.ownAccount(ctx, accountID); err != nil {
		return nil, err
	}
	origMsg, err := s.ownMessage(ctx, origID)
	if err != nil {
		return nil, err
	}

	input := send.ForwardInput{
		AccountID:   accountID,
		TenantID:    tenantID,
		OriginalMsg: origMsg,
		To:          toSendAddresses(req.To),
		BodyHTML:    req.BodyHtml,
		BodyText:    req.BodyText,
	}

	msg, err := s.sendService.Forward(ctx, input)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.ForwardEmailResponse{Message: toEmailMessageInfo(msg)}, nil
}

// ============================================================================
// Signature Operations
// ============================================================================

func (s *EmailGRPCServer) CreateSignature(ctx context.Context, req *emailv1.CreateSignatureRequest) (*emailv1.CreateSignatureResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	// Signatures are personal: req.UserId is ignored, the caller owns it.
	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}

	sig, err := s.signatureService.Create(ctx, userID, tenantID, req.Name, req.HtmlContent)
	if err != nil {
		return nil, mapEmailError(err)
	}

	if req.IsDefault {
		if err := s.signatureService.SetDefault(ctx, sig.ID, tenantID); err != nil {
			return nil, mapEmailError(err)
		}
		sig.IsDefault = true
	}

	return &emailv1.CreateSignatureResponse{Signature: toEmailSignatureInfo(sig)}, nil
}

func (s *EmailGRPCServer) GetSignature(ctx context.Context, req *emailv1.GetSignatureRequest) (*emailv1.GetSignatureResponse, error) {
	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid signature id")
	}

	sig, err := s.ownSignature(ctx, id)
	if err != nil {
		return nil, err
	}

	return &emailv1.GetSignatureResponse{Signature: toEmailSignatureInfo(sig)}, nil
}

func (s *EmailGRPCServer) ListSignatures(ctx context.Context, req *emailv1.ListSignaturesRequest) (*emailv1.ListSignaturesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := callerID(ctx)
	if err != nil {
		return nil, err
	}

	sigs, err := s.signatureService.ListByUser(ctx, userID, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoSigs := make([]*emailv1.EmailSignatureInfo, 0, len(sigs))
	for _, sig := range sigs {
		protoSigs = append(protoSigs, toEmailSignatureInfo(sig))
	}

	return &emailv1.ListSignaturesResponse{Signatures: protoSigs}, nil
}

func (s *EmailGRPCServer) UpdateSignature(ctx context.Context, req *emailv1.UpdateSignatureRequest) (*emailv1.UpdateSignatureResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid signature id")
	}

	name := ""
	if req.Name != nil {
		name = *req.Name
	}
	htmlContent := ""
	if req.HtmlContent != nil {
		htmlContent = *req.HtmlContent
	}

	if _, err := s.ownSignature(ctx, id); err != nil {
		return nil, err
	}

	sig, err := s.signatureService.Update(ctx, id, tenantID, name, htmlContent)
	if err != nil {
		return nil, mapEmailError(err)
	}

	if req.IsDefault != nil && *req.IsDefault {
		if err := s.signatureService.SetDefault(ctx, id, tenantID); err != nil {
			return nil, mapEmailError(err)
		}
		sig.IsDefault = true
	}

	return &emailv1.UpdateSignatureResponse{Signature: toEmailSignatureInfo(sig)}, nil
}

func (s *EmailGRPCServer) DeleteSignature(ctx context.Context, req *emailv1.DeleteSignatureRequest) (*emailv1.DeleteSignatureResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid signature id")
	}

	if _, err := s.ownSignature(ctx, id); err != nil {
		return nil, err
	}

	if err := s.signatureService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.DeleteSignatureResponse{}, nil
}

func (s *EmailGRPCServer) SetDefaultSignature(ctx context.Context, req *emailv1.SetDefaultSignatureRequest) (*emailv1.SetDefaultSignatureResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid signature id")
	}

	if _, err := s.ownSignature(ctx, id); err != nil {
		return nil, err
	}

	if err := s.signatureService.SetDefault(ctx, id, tenantID); err != nil {
		return nil, mapEmailError(err)
	}

	sig, err := s.signatureService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.SetDefaultSignatureResponse{Signature: toEmailSignatureInfo(sig)}, nil
}

// ============================================================================
// CRM Linking Operations
// ============================================================================

func (s *EmailGRPCServer) GetEmailContactLinks(ctx context.Context, req *emailv1.GetEmailContactLinksRequest) (*emailv1.GetEmailContactLinksResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if _, err := s.ownMessage(ctx, msgID); err != nil {
		return nil, err
	}

	links, err := s.linkRepo.GetByMessageID(ctx, msgID, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoLinks := make([]*emailv1.EmailContactLinkInfo, 0, len(links))
	for _, link := range links {
		protoLinks = append(protoLinks, toEmailContactLinkInfo(link))
	}

	return &emailv1.GetEmailContactLinksResponse{Links: protoLinks}, nil
}

func (s *EmailGRPCServer) LinkEmailToContact(ctx context.Context, req *emailv1.LinkEmailToContactRequest) (*emailv1.LinkEmailToContactResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	if _, err := s.ownMessage(ctx, msgID); err != nil {
		return nil, err
	}

	link := &models.EmailContactLink{
		ID:        uuid.New(),
		TenantID:  tenantID,
		MessageID: msgID,
		ContactID: contactID,
		LinkType:  req.LinkType,
		CreatedAt: time.Now().UTC(),
	}

	if link.LinkType == "" {
		link.LinkType = models.LinkTypeManual
	}

	if err := s.linkRepo.Create(ctx, link); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.LinkEmailToContactResponse{Link: toEmailContactLinkInfo(link)}, nil
}

func (s *EmailGRPCServer) UnlinkEmailFromContact(ctx context.Context, req *emailv1.UnlinkEmailFromContactRequest) (*emailv1.UnlinkEmailFromContactResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	msgID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	if _, err := s.ownMessage(ctx, msgID); err != nil {
		return nil, err
	}

	if err := s.linkRepo.Delete(ctx, msgID, contactID, tenantID); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.UnlinkEmailFromContactResponse{}, nil
}

func (s *EmailGRPCServer) GetContactEmails(ctx context.Context, req *emailv1.GetContactEmailsRequest) (*emailv1.GetContactEmailsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	contactID, err := uuid.Parse(req.ContactId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid contact_id")
	}

	page := int(req.Page)
	if page < 1 {
		page = 1
	}
	perPage := int(req.PerPage)
	if perPage < 1 {
		perPage = 50
	}

	messages, total, err := s.linkRepo.GetByContactID(ctx, contactID, tenantID, page, perPage)
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoMsgs := make([]*emailv1.EmailMessageInfo, 0, len(messages))
	for _, m := range messages {
		protoMsgs = append(protoMsgs, toEmailMessageInfo(m))
	}

	return &emailv1.GetContactEmailsResponse{
		Messages: protoMsgs,
		Total:    int32(total),
	}, nil
}

// ============================================================================
// Sync Operations
// ============================================================================

func (s *EmailGRPCServer) TriggerSync(ctx context.Context, req *emailv1.TriggerSyncRequest) (*emailv1.TriggerSyncResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	if _, err := s.ownAccount(ctx, accountID); err != nil {
		return nil, err
	}

	s.syncEngine.TriggerSync(accountID)

	return &emailv1.TriggerSyncResponse{Status: "started"}, nil
}

func (s *EmailGRPCServer) GetSyncStatus(ctx context.Context, req *emailv1.GetSyncStatusRequest) (*emailv1.GetSyncStatusResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	if _, err := s.ownAccount(ctx, accountID); err != nil {
		return nil, err
	}

	syncStatus := s.syncEngine.GetStatus(accountID)

	resp := &emailv1.GetSyncStatusResponse{
		Status:       "idle",
		ErrorMessage: syncStatus.Error,
	}

	if syncStatus.IsSyncing {
		resp.Status = "syncing"
	} else if syncStatus.Error != "" {
		resp.Status = "error"
	}

	if syncStatus.LastSync != nil {
		resp.LastSyncAt = syncStatus.LastSync.Format(time.RFC3339)
	}

	return resp, nil
}

func (s *EmailGRPCServer) SetReadFlag(ctx context.Context, req *emailv1.SetReadFlagRequest) (*emailv1.SetReadFlagResponse, error) {
	id, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message_id")
	}

	if _, err := s.ownMessage(ctx, id); err != nil {
		return nil, err
	}

	if req.IsRead {
		err = s.messageService.MarkRead(ctx, id)
	} else {
		err = s.messageService.MarkUnread(ctx, id)
	}
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.SetReadFlagResponse{}, nil
}

// ============================================================================
// Attachment Operations
// ============================================================================

func (s *EmailGRPCServer) UploadAttachment(ctx context.Context, req *emailv1.UploadAttachmentRequest) (*emailv1.UploadAttachmentResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
	}

	if _, err := s.ownAccount(ctx, accountID); err != nil {
		return nil, err
	}

	reader := bytes.NewReader(req.Content)
	att, err := s.attachmentService.CreateFromStream(
		ctx,
		uuid.Nil, // No message yet (pre-send upload)
		accountID,
		0, // No message UID for uploads
		req.Filename,
		req.ContentType,
		int64(len(req.Content)),
		reader,
		"",
		false,
		tenantID,
	)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.UploadAttachmentResponse{
		Id:        att.ID.String(),
		MinioKey:  att.MinIOKey,
		SizeBytes: att.SizeBytes,
	}, nil
}

func (s *EmailGRPCServer) GetAttachmentDownloadURL(ctx context.Context, req *emailv1.GetAttachmentDownloadURLRequest) (*emailv1.GetAttachmentDownloadURLResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid attachment id")
	}

	att, err := s.attachmentService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}
	if _, err := s.ownMessage(ctx, att.MessageID); err != nil {
		return nil, status.Error(codes.NotFound, attachment.ErrAttachmentNotFound.Error())
	}

	url, err := s.attachmentService.GetDownloadURL(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	// Get attachment metadata for the response
	var filename, contentType string
	var sizeBytes int64
	if att, attErr := s.attachmentService.GetByID(ctx, id, tenantID); attErr == nil {
		filename = att.Filename
		contentType = att.ContentType
		sizeBytes = att.SizeBytes
	}

	return &emailv1.GetAttachmentDownloadURLResponse{
		DownloadUrl: url,
		Filename:    filename,
		ContentType: contentType,
		SizeBytes:   sizeBytes,
	}, nil
}

// ============================================================================
// Contact Import/Export
// ============================================================================

func (s *EmailGRPCServer) ImportContactsCSV(ctx context.Context, req *emailv1.ImportContactsCSVRequest) (*emailv1.ImportContactsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	userID, err := uuid.Parse(middleware.GetUserID(ctx))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user_id missing from context")
	}

	// Tenant-scoped adapter built per request, mirroring CRMGRPCServer.ImportContactsCSV —
	// the shared importService singleton is wired with a nil ContactProvider because it has
	// no tenant to bind to outside a request.
	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	importSvc := emailcontact.NewImportService(provider, nil)

	reader := bytes.NewReader(req.FileContent)
	result, err := importSvc.ImportCSV(
		ctx,
		reader,
		req.FieldMapping,
		req.Visibility,
		userID,
		req.MergeByEmail,
	)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return toImportContactsResponse(result), nil
}

func (s *EmailGRPCServer) ImportContactsVCard(ctx context.Context, req *emailv1.ImportContactsVCardRequest) (*emailv1.ImportContactsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}
	userID, err := uuid.Parse(middleware.GetUserID(ctx))
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "user_id missing from context")
	}

	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	importSvc := emailcontact.NewImportService(provider, nil)

	reader := bytes.NewReader(req.FileContent)
	result, err := importSvc.ImportVCard(
		ctx,
		reader,
		req.Visibility,
		userID,
		req.MergeByEmail,
	)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return toImportContactsResponse(result), nil
}

func (s *EmailGRPCServer) ExportContactsCSV(ctx context.Context, req *emailv1.ExportContactsRequest) (*emailv1.ExportContactsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	ids := make([]uuid.UUID, 0, len(req.ContactIds))
	for _, idStr := range req.ContactIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id: "+idStr)
		}
		ids = append(ids, id)
	}

	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	exportSvc := emailcontact.NewExportService(provider, nil)

	data, err := exportSvc.ExportCSV(ctx, ids, req.Fields)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.ExportContactsResponse{
		FileContent: data,
		Filename:    "contacts.csv",
	}, nil
}

func (s *EmailGRPCServer) ExportContactsVCard(ctx context.Context, req *emailv1.ExportContactsRequest) (*emailv1.ExportContactsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.Unauthenticated, "tenant_id missing from context")
	}

	ids := make([]uuid.UUID, 0, len(req.ContactIds))
	for _, idStr := range req.ContactIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id: "+idStr)
		}
		ids = append(ids, id)
	}

	provider := emailcontact.NewTenantScopedAdapter(s.contactService, s.companyService, tenantID)
	exportSvc := emailcontact.NewExportService(provider, nil)

	data, err := exportSvc.ExportVCard(ctx, ids)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.ExportContactsResponse{
		FileContent: data,
		Filename:    "contacts.vcf",
	}, nil
}

// ============================================================================
// Proto Conversion Helpers
// ============================================================================

func toEmailAccountInfo(acct *models.EmailAccount) *emailv1.EmailAccountInfo {
	info := &emailv1.EmailAccountInfo{
		Id:           acct.ID.String(),
		UserId:       acct.UserID.String(),
		EmailAddress: acct.EmailAddress,
		DisplayName:  acct.DisplayName,
		ImapHost:     acct.IMAPHost,
		ImapPort:     int32(acct.IMAPPort),
		SmtpHost:     acct.SMTPHost,
		SmtpPort:     int32(acct.SMTPPort),
		Username:     acct.Username,
		UseSsl:       acct.UseSSL,
		SyncEnabled:  acct.SyncEnabled,
		IsDefault:    acct.IsDefault,
		CreatedAt:    acct.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    acct.UpdatedAt.Format(time.RFC3339),
	}
	if acct.LastSyncAt != nil {
		info.LastSyncAt = acct.LastSyncAt.Format(time.RFC3339)
	}
	return info
}

func toEmailFolderInfo(f *models.EmailFolder) *emailv1.EmailFolderInfo {
	return &emailv1.EmailFolderInfo{
		Id:           f.ID.String(),
		AccountId:    f.AccountID.String(),
		Name:         f.Name,
		ImapName:     f.IMAPName,
		FolderType:   f.FolderType,
		UidValidity:  f.UIDValidity,
		MessageCount: int32(f.MessageCount),
		UnreadCount:  int32(f.UnreadCount),
		SortOrder:    int32(f.SortOrder),
		CreatedAt:    f.CreatedAt.Format(time.RFC3339),
		UpdatedAt:    f.UpdatedAt.Format(time.RFC3339),
	}
}

func toEmailMessageInfo(m *models.EmailMessage) *emailv1.EmailMessageInfo {
	info := &emailv1.EmailMessageInfo{
		Id:              m.ID.String(),
		AccountId:       m.AccountID.String(),
		FolderId:        m.FolderID.String(),
		Uid:             m.UID,
		MessageIdHeader: m.MessageID,
		InReplyTo:       m.InReplyTo,
		References:      m.References,
		From:            &emailv1.EmailAddress{Name: m.FromName, Email: m.FromEmail},
		Subject:         m.Subject,
		Preview:         m.Preview,
		BodyText:        m.BodyText,
		BodyHtml:        m.BodyHTML,
		IsRead:          m.IsRead,
		IsStarred:       m.IsStarred,
		IsDraft:         m.IsDraft,
		HasAttachments:  m.HasAttachments,
		Date:            m.Date.Format(time.RFC3339),
		SizeBytes:       m.SizeBytes,
		CreatedAt:       m.CreatedAt.Format(time.RFC3339),
		UpdatedAt:       m.UpdatedAt.Format(time.RFC3339),
	}

	for _, id := range m.LabelIDs {
		info.LabelIds = append(info.LabelIds, id.String())
	}

	if m.ThreadID != nil {
		info.ThreadId = m.ThreadID.String()
	}

	for _, addr := range m.ToAddresses {
		info.To = append(info.To, &emailv1.EmailAddress{Name: addr.Name, Email: addr.Email})
	}
	for _, addr := range m.CcAddresses {
		info.Cc = append(info.Cc, &emailv1.EmailAddress{Name: addr.Name, Email: addr.Email})
	}
	for _, addr := range m.BccAddresses {
		info.Bcc = append(info.Bcc, &emailv1.EmailAddress{Name: addr.Name, Email: addr.Email})
	}

	return info
}

func toEmailAttachmentInfo(att *models.EmailAttachment) *emailv1.EmailAttachmentInfo {
	return &emailv1.EmailAttachmentInfo{
		Id:          att.ID.String(),
		Filename:    att.Filename,
		ContentType: att.ContentType,
		SizeBytes:   att.SizeBytes,
		MinioKey:    att.MinIOKey,
		ContentId:   att.ContentID,
		IsInline:    att.IsInline,
	}
}

func toEmailSignatureInfo(sig *models.EmailSignature) *emailv1.EmailSignatureInfo {
	return &emailv1.EmailSignatureInfo{
		Id:          sig.ID.String(),
		UserId:      sig.UserID.String(),
		Name:        sig.Name,
		HtmlContent: sig.HTMLContent,
		IsDefault:   sig.IsDefault,
		CreatedAt:   sig.CreatedAt.Format(time.RFC3339),
		UpdatedAt:   sig.UpdatedAt.Format(time.RFC3339),
	}
}

func toEmailContactLinkInfo(link *models.EmailContactLink) *emailv1.EmailContactLinkInfo {
	return &emailv1.EmailContactLinkInfo{
		Id:        link.ID.String(),
		MessageId: link.MessageID.String(),
		ContactId: link.ContactID.String(),
		LinkType:  link.LinkType,
		CreatedAt: link.CreatedAt.Format(time.RFC3339),
	}
}

func toSendAddresses(addrs []*emailv1.EmailAddress) []send.EmailAddress {
	if len(addrs) == 0 {
		return nil
	}
	result := make([]send.EmailAddress, 0, len(addrs))
	for _, a := range addrs {
		result = append(result, send.EmailAddress{Name: a.Name, Email: a.Email})
	}
	return result
}

func toImportContactsResponse(result *emailcontact.ImportResult) *emailv1.ImportContactsResponse {
	resp := &emailv1.ImportContactsResponse{
		ImportedCount: int32(result.ImportedCount),
		MergedCount:   int32(result.MergedCount),
		SkippedCount:  int32(result.SkippedCount),
	}
	for _, e := range result.Errors {
		resp.Errors = append(resp.Errors, &emailv1.ImportError{
			Row:   int32(e.Row),
			Error: e.Reason,
		})
	}
	return resp
}

// ============================================================================
// Error Mapping
// ============================================================================

func mapEmailError(err error) error {
	if err == nil {
		return nil
	}

	switch {
	// Account errors
	case errors.Is(err, account.ErrAccountNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, account.ErrInvalidCredentials):
		// Third-party credentials rejected: FailedPrecondition (409), not Unauthenticated -
		// the FE treats every 401 as an expired session (refresh + retry + logout).
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, account.ErrConnectionFailed):
		return status.Error(codes.Unavailable, err.Error())

	// Message errors
	case errors.Is(err, message.ErrMessageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, message.ErrFolderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, message.ErrThreadNotFound):
		return status.Error(codes.NotFound, err.Error())
	// OutOfRange -> 422: a semantically valid request with an unprocessable
	// value, same convention as the RBAC capability-key check.
	case errors.Is(err, message.ErrUnknownBulkAction):
		return status.Error(codes.OutOfRange, err.Error())
	case errors.Is(err, message.ErrBulkTargetRequired):
		return status.Error(codes.InvalidArgument, err.Error())

	// Send errors
	case errors.Is(err, send.ErrSendFailed):
		return status.Error(codes.Internal, err.Error())
	case errors.Is(err, send.ErrSMTPAuthFailed):
		// Third-party credentials rejected: FailedPrecondition (409), not Unauthenticated -
		// the FE treats every 401 as an expired session (refresh + retry + logout).
		return status.Error(codes.FailedPrecondition, err.Error())
	case errors.Is(err, send.ErrInvalidRecipient):
		return status.Error(codes.InvalidArgument, err.Error())

	// Signature errors
	case errors.Is(err, signature.ErrSignatureNotFound):
		return status.Error(codes.NotFound, err.Error())

	// Attachment errors
	case errors.Is(err, attachment.ErrAttachmentNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, attachment.ErrAttachmentTooLarge):
		return status.Error(codes.InvalidArgument, err.Error())

	// Sync errors
	case errors.Is(err, emailsync.ErrSyncInProgress):
		return status.Error(codes.AlreadyExists, err.Error())

	// Rule errors
	case errors.Is(err, rule.ErrRuleNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rule.ErrTargetNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, rule.ErrInvalidField),
		errors.Is(err, rule.ErrInvalidOperator),
		errors.Is(err, rule.ErrInvalidActionType),
		errors.Is(err, rule.ErrNameRequired),
		errors.Is(err, rule.ErrValueRequired),
		errors.Is(err, rule.ErrTargetRequired):
		return status.Error(codes.InvalidArgument, err.Error())

	// Label errors
	case errors.Is(err, label.ErrNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, label.ErrMessageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, label.ErrDuplicateName):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, label.ErrNameRequired),
		errors.Is(err, label.ErrColorInvalid),
		errors.Is(err, label.ErrTargetInvalid):
		return status.Error(codes.InvalidArgument, err.Error())

	// Template errors
	case errors.Is(err, template.ErrTemplateNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, template.ErrNameRequired),
		errors.Is(err, template.ErrInvalidVisibility):
		return status.Error(codes.InvalidArgument, err.Error())

	// Import errors
	case errors.Is(err, emailcontact.ErrImportFailed):
		return status.Error(codes.Internal, err.Error())
	case errors.Is(err, emailcontact.ErrInvalidCSV):
		return status.Error(codes.InvalidArgument, err.Error())
	case errors.Is(err, emailcontact.ErrInvalidVCard):
		return status.Error(codes.InvalidArgument, err.Error())

	default:
		slog.Error("unhandled email service error", "error", err)
		return status.Error(codes.Internal, "internal server error")
	}
}

// ============================================================================
// Rules (Regeln & Filter)
// ============================================================================

func (s *EmailGRPCServer) ListEmailRules(ctx context.Context, _ *emailv1.ListEmailRulesRequest) (*emailv1.ListEmailRulesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	rules, err := s.ruleService.List(ctx, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	out := make([]*emailv1.EmailRuleInfo, 0, len(rules))
	for _, r := range rules {
		out = append(out, toEmailRuleInfo(r))
	}
	return &emailv1.ListEmailRulesResponse{Rules: out}, nil
}

func (s *EmailGRPCServer) CreateEmailRule(ctx context.Context, req *emailv1.CreateEmailRuleRequest) (*emailv1.CreateEmailRuleResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	r, err := s.ruleService.Create(ctx, tenantID, rule.RuleInput{
		Name:         &req.Name,
		Field:        &req.Field,
		Op:           &req.Op,
		Value:        &req.Value,
		ActionType:   &req.ActionType,
		ActionTarget: &req.ActionTarget,
	})
	if err != nil {
		return nil, mapEmailError(err)
	}
	return &emailv1.CreateEmailRuleResponse{Rule: toEmailRuleInfo(r)}, nil
}

func (s *EmailGRPCServer) UpdateEmailRule(ctx context.Context, req *emailv1.UpdateEmailRuleRequest) (*emailv1.UpdateEmailRuleResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule id")
	}

	r, err := s.ruleService.Update(ctx, id, tenantID, rule.RuleInput{
		Name:         req.Name,
		Field:        req.Field,
		Op:           req.Op,
		Value:        req.Value,
		ActionType:   req.ActionType,
		ActionTarget: req.ActionTarget,
	})
	if err != nil {
		return nil, mapEmailError(err)
	}
	return &emailv1.UpdateEmailRuleResponse{Rule: toEmailRuleInfo(r)}, nil
}

func (s *EmailGRPCServer) DeleteEmailRule(ctx context.Context, req *emailv1.DeleteEmailRuleRequest) (*emailv1.DeleteEmailRuleResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid rule id")
	}

	if err := s.ruleService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapEmailError(err)
	}
	return &emailv1.DeleteEmailRuleResponse{Success: true}, nil
}

func (s *EmailGRPCServer) ApplyEmailRules(ctx context.Context, _ *emailv1.ApplyEmailRulesRequest) (*emailv1.ApplyEmailRulesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	res, err := s.ruleService.ApplyAll(ctx, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}
	return &emailv1.ApplyEmailRulesResponse{
		Affected: int32(res.Affected),
		Scanned:  int32(res.Scanned),
	}, nil
}

func toEmailRuleInfo(r *models.EmailRule) *emailv1.EmailRuleInfo {
	return &emailv1.EmailRuleInfo{
		Id:           r.ID.String(),
		Name:         r.Name,
		Field:        r.Field,
		Op:           r.Op,
		Value:        r.Value,
		ActionType:   r.ActionType,
		ActionTarget: r.ActionTarget.String(),
	}
}

// ============================================================================
// Labels
// ============================================================================

func (s *EmailGRPCServer) ListEmailLabels(ctx context.Context, _ *emailv1.ListEmailLabelsRequest) (*emailv1.ListEmailLabelsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	labels, err := s.labelService.List(ctx, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	out := make([]*emailv1.EmailLabelInfo, 0, len(labels))
	for _, l := range labels {
		out = append(out, toEmailLabelInfo(l))
	}
	return &emailv1.ListEmailLabelsResponse{Labels: out}, nil
}

func (s *EmailGRPCServer) CreateEmailLabel(ctx context.Context, req *emailv1.CreateEmailLabelRequest) (*emailv1.CreateEmailLabelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	l, err := s.labelService.Create(ctx, tenantID, req.Name, req.Color)
	if err != nil {
		return nil, mapEmailError(err)
	}
	return &emailv1.CreateEmailLabelResponse{Label: toEmailLabelInfo(l)}, nil
}

func (s *EmailGRPCServer) UpdateEmailLabel(ctx context.Context, req *emailv1.UpdateEmailLabelRequest) (*emailv1.UpdateEmailLabelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid label id")
	}

	l, err := s.labelService.Update(ctx, id, tenantID, req.Name, req.Color)
	if err != nil {
		return nil, mapEmailError(err)
	}
	return &emailv1.UpdateEmailLabelResponse{Label: toEmailLabelInfo(l)}, nil
}

func (s *EmailGRPCServer) DeleteEmailLabel(ctx context.Context, req *emailv1.DeleteEmailLabelRequest) (*emailv1.DeleteEmailLabelResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid label id")
	}

	if err := s.labelService.Delete(ctx, id, tenantID); err != nil {
		return nil, mapEmailError(err)
	}
	return &emailv1.DeleteEmailLabelResponse{Success: true}, nil
}

func (s *EmailGRPCServer) AssignMessageLabels(ctx context.Context, req *emailv1.AssignMessageLabelsRequest) (*emailv1.AssignMessageLabelsResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	messageID, err := uuid.Parse(req.MessageId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	labelIDs := make([]uuid.UUID, 0, len(req.LabelIds))
	for _, raw := range req.LabelIds {
		id, err := uuid.Parse(raw)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid label id")
		}
		labelIDs = append(labelIDs, id)
	}

	if _, err := s.ownMessage(ctx, messageID); err != nil {
		return nil, err
	}

	msg, err := s.labelService.AssignToMessage(ctx, tenantID, messageID, labelIDs)
	if err != nil {
		return nil, mapEmailError(err)
	}
	return &emailv1.AssignMessageLabelsResponse{Message: toEmailMessageInfo(msg)}, nil
}

func toEmailLabelInfo(l *models.EmailLabel) *emailv1.EmailLabelInfo {
	return &emailv1.EmailLabelInfo{
		Id:    l.ID.String(),
		Name:  l.Name,
		Color: l.Color,
	}
}

// ============================================================================
// Template Operations (Vorlagen & Quicktexte)
// ============================================================================

func (s *EmailGRPCServer) ListEmailTemplates(ctx context.Context, req *emailv1.ListEmailTemplatesRequest) (*emailv1.ListEmailTemplatesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	templates, err := s.templateService.List(ctx, tenantID, userID, req.IsAdmin)
	if err != nil {
		return nil, mapEmailError(err)
	}

	protoTemplates := make([]*emailv1.EmailTemplateInfo, 0, len(templates))
	for _, tpl := range templates {
		protoTemplates = append(protoTemplates, toEmailTemplateInfo(tpl))
	}

	return &emailv1.ListEmailTemplatesResponse{Templates: protoTemplates}, nil
}

func (s *EmailGRPCServer) GetEmailTemplate(ctx context.Context, req *emailv1.GetEmailTemplateRequest) (*emailv1.GetEmailTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid template id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	tpl, err := s.templateService.GetByID(ctx, id, tenantID, userID, req.IsAdmin)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.GetEmailTemplateResponse{Template: toEmailTemplateInfo(tpl)}, nil
}

func (s *EmailGRPCServer) CreateEmailTemplate(ctx context.Context, req *emailv1.CreateEmailTemplateRequest) (*emailv1.CreateEmailTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	tpl, err := s.templateService.Create(ctx, tenantID, userID, req.Name, req.Subject, req.BodyHtml, req.BodyText, req.Visibility)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.CreateEmailTemplateResponse{Template: toEmailTemplateInfo(tpl)}, nil
}

func (s *EmailGRPCServer) UpdateEmailTemplate(ctx context.Context, req *emailv1.UpdateEmailTemplateRequest) (*emailv1.UpdateEmailTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid template id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	tpl, err := s.templateService.Update(ctx, id, tenantID, userID, req.IsAdmin,
		req.Name, req.Subject, req.BodyHtml, req.BodyText, req.Visibility)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.UpdateEmailTemplateResponse{Template: toEmailTemplateInfo(tpl)}, nil
}

func (s *EmailGRPCServer) DeleteEmailTemplate(ctx context.Context, req *emailv1.DeleteEmailTemplateRequest) (*emailv1.DeleteEmailTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid template id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	if err := s.templateService.Delete(ctx, id, tenantID, userID, req.IsAdmin); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.DeleteEmailTemplateResponse{}, nil
}

func (s *EmailGRPCServer) RenderEmailTemplate(ctx context.Context, req *emailv1.RenderEmailTemplateRequest) (*emailv1.RenderEmailTemplateResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid template id")
	}
	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
	}

	subject, bodyHTML, bodyText, err := s.templateService.Render(ctx, id, tenantID, userID, req.IsAdmin, req.Values)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.RenderEmailTemplateResponse{Subject: subject, BodyHtml: bodyHTML, BodyText: bodyText}, nil
}

func toEmailTemplateInfo(tpl *models.EmailTemplate) *emailv1.EmailTemplateInfo {
	ownerID := ""
	if tpl.OwnerID != nil {
		ownerID = tpl.OwnerID.String()
	}
	return &emailv1.EmailTemplateInfo{
		Id:         tpl.ID.String(),
		OwnerId:    ownerID,
		Visibility: tpl.Visibility,
		Name:       tpl.Name,
		Subject:    tpl.Subject,
		BodyHtml:   tpl.BodyHTML,
		BodyText:   tpl.BodyText,
		CreatedAt:  tpl.CreatedAt.Format(time.RFC3339),
		UpdatedAt:  tpl.UpdatedAt.Format(time.RFC3339),
	}
}
