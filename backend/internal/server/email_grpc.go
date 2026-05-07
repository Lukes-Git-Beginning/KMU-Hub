package server

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"

	"github.com/kmuhub/kmuhub/internal/email/account"
	"github.com/kmuhub/kmuhub/internal/email/attachment"
	emailcontact "github.com/kmuhub/kmuhub/internal/email/contact"
	"github.com/kmuhub/kmuhub/internal/email/message"
	"github.com/kmuhub/kmuhub/internal/email/send"
	"github.com/kmuhub/kmuhub/internal/email/signature"
	emailsync "github.com/kmuhub/kmuhub/internal/email/sync"
	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/models"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// EmailGRPCServer implements the Email gRPC service.
type EmailGRPCServer struct {
	emailv1.UnimplementedEmailServiceServer
	accountService    *account.Service
	messageService    *message.Service
	sendService       *send.Service
	signatureService  *signature.Service
	attachmentService *attachment.Service
	syncEngine        *emailsync.Engine
	linkRepo          EmailContactLinkRepository
	importService     *emailcontact.ImportService
	exportService     *emailcontact.ExportService
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
	importService *emailcontact.ImportService,
	exportService *emailcontact.ExportService,
) *EmailGRPCServer {
	return &EmailGRPCServer{
		accountService:    accountService,
		messageService:    messageService,
		sendService:       sendService,
		signatureService:  signatureService,
		attachmentService: attachmentService,
		syncEngine:        syncEngine,
		linkRepo:          linkRepo,
		importService:     importService,
		exportService:     exportService,
	}
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

	folder, err := s.messageService.GetFolderByID(ctx, id)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.GetFolderResponse{Folder: toEmailFolderInfo(folder)}, nil
}

func (s *EmailGRPCServer) SyncFolders(ctx context.Context, req *emailv1.SyncFoldersRequest) (*emailv1.SyncFoldersResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
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
	folderID, err := uuid.Parse(req.FolderId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid folder_id")
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
		// Get folder to determine account_id
		folder, folderErr := s.messageService.GetFolderByID(ctx, folderID)
		if folderErr != nil {
			return nil, mapEmailError(folderErr)
		}
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

func (s *EmailGRPCServer) GetMessage(ctx context.Context, req *emailv1.GetMessageRequest) (*emailv1.GetMessageResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid message id")
	}

	msg, err := s.messageService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
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

	if err := s.messageService.Delete(ctx, id); err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.DeleteMessageResponse{}, nil
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

	origMsg, err := s.messageService.GetByID(ctx, origID, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
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

	origMsg, err := s.messageService.GetByID(ctx, origID, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
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

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
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
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	id, err := uuid.Parse(req.Id)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid signature id")
	}

	sig, err := s.signatureService.GetByID(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.GetSignatureResponse{Signature: toEmailSignatureInfo(sig)}, nil
}

func (s *EmailGRPCServer) ListSignatures(ctx context.Context, req *emailv1.ListSignaturesRequest) (*emailv1.ListSignaturesResponse, error) {
	tenantID, err := middleware.GetTenantID(ctx)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "missing tenant context")
	}

	userID, err := uuid.Parse(req.UserId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid user_id")
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

	s.syncEngine.TriggerSync(accountID)

	return &emailv1.TriggerSyncResponse{Status: "started"}, nil
}

func (s *EmailGRPCServer) GetSyncStatus(ctx context.Context, req *emailv1.GetSyncStatusRequest) (*emailv1.GetSyncStatusResponse, error) {
	accountID, err := uuid.Parse(req.AccountId)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid account_id")
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

	url, err := s.attachmentService.GetDownloadURL(ctx, id, tenantID)
	if err != nil {
		return nil, mapEmailError(err)
	}

	// Get attachment metadata for the response
	attachments, _ := s.attachmentService.GetByMessage(ctx, uuid.Nil, tenantID)
	var filename, contentType string
	var sizeBytes int64
	for _, att := range attachments {
		if att.ID == id {
			filename = att.Filename
			contentType = att.ContentType
			sizeBytes = att.SizeBytes
			break
		}
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
	reader := bytes.NewReader(req.FileContent)
	result, err := s.importService.ImportCSV(
		ctx,
		reader,
		req.FieldMapping,
		req.Visibility,
		uuid.Nil, // Owner determined by auth context
		req.MergeByEmail,
	)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return toImportContactsResponse(result), nil
}

func (s *EmailGRPCServer) ImportContactsVCard(ctx context.Context, req *emailv1.ImportContactsVCardRequest) (*emailv1.ImportContactsResponse, error) {
	reader := bytes.NewReader(req.FileContent)
	result, err := s.importService.ImportVCard(
		ctx,
		reader,
		req.Visibility,
		uuid.Nil,
		req.MergeByEmail,
	)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return toImportContactsResponse(result), nil
}

func (s *EmailGRPCServer) ExportContactsCSV(ctx context.Context, req *emailv1.ExportContactsRequest) (*emailv1.ExportContactsResponse, error) {
	ids := make([]uuid.UUID, 0, len(req.ContactIds))
	for _, idStr := range req.ContactIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id: "+idStr)
		}
		ids = append(ids, id)
	}

	data, err := s.exportService.ExportCSV(ctx, ids, req.Fields)
	if err != nil {
		return nil, mapEmailError(err)
	}

	return &emailv1.ExportContactsResponse{
		FileContent: data,
		Filename:    "contacts.csv",
	}, nil
}

func (s *EmailGRPCServer) ExportContactsVCard(ctx context.Context, req *emailv1.ExportContactsRequest) (*emailv1.ExportContactsResponse, error) {
	ids := make([]uuid.UUID, 0, len(req.ContactIds))
	for _, idStr := range req.ContactIds {
		id, err := uuid.Parse(idStr)
		if err != nil {
			return nil, status.Error(codes.InvalidArgument, "invalid contact_id: "+idStr)
		}
		ids = append(ids, id)
	}

	data, err := s.exportService.ExportVCard(ctx, ids)
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
	case errors.Is(err, account.ErrAccountExists):
		return status.Error(codes.AlreadyExists, err.Error())
	case errors.Is(err, account.ErrInvalidCredentials):
		return status.Error(codes.Unauthenticated, err.Error())
	case errors.Is(err, account.ErrConnectionFailed):
		return status.Error(codes.Unavailable, err.Error())

	// Message errors
	case errors.Is(err, message.ErrMessageNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, message.ErrFolderNotFound):
		return status.Error(codes.NotFound, err.Error())
	case errors.Is(err, message.ErrThreadNotFound):
		return status.Error(codes.NotFound, err.Error())

	// Send errors
	case errors.Is(err, send.ErrSendFailed):
		return status.Error(codes.Internal, err.Error())
	case errors.Is(err, send.ErrSMTPAuthFailed):
		return status.Error(codes.Unauthenticated, err.Error())
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

