package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// ============================================================================
// Local DTOs for email send/compose handlers (S4.1 validation boundary)
// ============================================================================

// emailAddressStrings converts a slice of plain email strings to []*emailv1.EmailAddress.
func emailAddressStrings(addrs []string) []*emailv1.EmailAddress {
	out := make([]*emailv1.EmailAddress, 0, len(addrs))
	for _, a := range addrs {
		out = append(out, &emailv1.EmailAddress{Email: a})
	}
	return out
}

type sendEmailDTO struct {
	AccountID string   `json:"account_id"`
	To        []string `json:"to" validate:"required,min=1,dive,email"`
	Cc        []string `json:"cc" validate:"omitempty,dive,email"`
	Bcc       []string `json:"bcc" validate:"omitempty,dive,email"`
	Subject   string   `json:"subject"`
	BodyHtml  string   `json:"body_html"`
	BodyText  string   `json:"body_text"`
	// ContactID, when set, makes the email service enforce marketing consent
	// (UWG §7) for the recipient. Omit for transactional/system mail.
	ContactID string `json:"contact_id" validate:"omitempty,uuid"`
}

type saveDraftDTO struct {
	AccountID string   `json:"account_id"`
	To        []string `json:"to" validate:"omitempty,dive,email"`
	Cc        []string `json:"cc" validate:"omitempty,dive,email"`
	Bcc       []string `json:"bcc" validate:"omitempty,dive,email"`
	Subject   string   `json:"subject"`
	BodyHtml  string   `json:"body_html"`
	BodyText  string   `json:"body_text"`
}

type replyEmailDTO struct {
	AccountID         string `json:"account_id"`
	OriginalMessageID string `json:"original_message_id" validate:"required"`
	BodyHtml          string `json:"body_html"`
	BodyText          string `json:"body_text"`
	ReplyAll          bool   `json:"reply_all"`
}

type forwardEmailDTO struct {
	AccountID         string   `json:"account_id"`
	OriginalMessageID string   `json:"original_message_id" validate:"required"`
	To                []string `json:"to" validate:"required,min=1,dive,email"`
	BodyHtml          string   `json:"body_html"`
	BodyText          string   `json:"body_text"`
}

type setDefaultSignatureDTO struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

type moveToFolderDTO struct {
	TargetFolderID string `json:"target_folder_id" validate:"required,uuid"`
}

// EmailRoutes handles HTTP routes for the Email backend service.
type EmailRoutes struct {
	registry *ServiceRegistry
}

// NewEmailRoutes creates a new EmailRoutes with the given service registry.
func NewEmailRoutes(registry *ServiceRegistry) *EmailRoutes {
	return &EmailRoutes{registry: registry}
}

// ServiceName returns the backend service name.
func (e *EmailRoutes) ServiceName() string { return "email" }

// getEmailClient lazily obtains a gRPC client for the Email service.
func (e *EmailRoutes) getEmailClient() (emailv1.EmailServiceClient, error) {
	conn, err := e.registry.GetConnection("email")
	if err != nil {
		return nil, err
	}
	return emailv1.NewEmailServiceClient(conn), nil
}

// RegisterRoutes registers all Email HTTP routes.
func (e *EmailRoutes) RegisterRoutes(r chi.Router, authMiddleware func(http.Handler) http.Handler) {
	// Email Accounts
	r.Route("/api/v1/email/accounts", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "write")).Post("/", e.HandleCreateAccount)
		r.With(middleware.RequirePermission("email", "read")).Get("/", e.HandleGetAccount)
		r.With(middleware.RequirePermission("email", "write")).Put("/{id}", e.HandleUpdateAccount)
		r.With(middleware.RequirePermission("email", "delete")).Delete("/{id}", e.HandleDeleteAccount)
		r.With(middleware.RequirePermission("email", "write")).Post("/test", e.HandleTestConnection)
	})

	// Folders
	r.Route("/api/v1/email/folders", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "read")).Get("/", e.HandleListFolders)
		r.With(middleware.RequirePermission("email", "read")).Get("/{id}", e.HandleGetFolder)
		r.With(middleware.RequirePermission("email", "write")).Post("/sync", e.HandleSyncFolders)
	})

	// Messages
	r.Route("/api/v1/email/messages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "read")).Get("/", e.HandleListMessages)
		r.With(middleware.RequirePermission("email", "read")).Get("/{id}", e.HandleGetMessage)
		r.With(middleware.RequirePermission("email", "read")).Get("/thread/{threadId}", e.HandleGetThreadMessages)
		r.With(middleware.RequirePermission("email", "write")).Post("/{id}/read", e.HandleMarkRead)
		r.With(middleware.RequirePermission("email", "write")).Post("/{id}/unread", e.HandleMarkUnread)
		r.With(middleware.RequirePermission("email", "write")).Post("/{id}/star", e.HandleToggleStar)
		r.With(middleware.RequirePermission("email", "write")).Post("/{id}/move", e.HandleMoveToFolder)
		r.With(middleware.RequirePermission("email", "delete")).Delete("/{id}", e.HandleDeleteMessage)
	})

	// Send/Compose
	r.Route("/api/v1/email/send", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "write")).Post("/", e.HandleSendEmail)
		r.With(middleware.RequirePermission("email", "write")).Post("/draft", e.HandleSaveDraft)
		r.With(middleware.RequirePermission("email", "write")).Post("/reply", e.HandleReplyEmail)
		r.With(middleware.RequirePermission("email", "write")).Post("/forward", e.HandleForwardEmail)
	})

	// Signatures
	r.Route("/api/v1/email/signatures", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "read")).Get("/", e.HandleListSignatures)
		r.With(middleware.RequirePermission("email", "read")).Get("/{id}", e.HandleGetSignature)
		r.With(middleware.RequirePermission("email", "write")).Post("/", e.HandleCreateSignature)
		r.With(middleware.RequirePermission("email", "write")).Put("/{id}", e.HandleUpdateSignature)
		r.With(middleware.RequirePermission("email", "delete")).Delete("/{id}", e.HandleDeleteSignature)
		r.With(middleware.RequirePermission("email", "write")).Post("/{id}/default", e.HandleSetDefaultSignature)
	})

	// Rules (Regeln & Filter)
	r.Route("/api/v1/email/rules", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "read")).Get("/", e.HandleListEmailRules)
		r.With(middleware.RequirePermission("email", "write")).Post("/", e.HandleCreateEmailRule)
		r.With(middleware.RequirePermission("email", "write")).Post("/apply", e.HandleApplyEmailRules)
		r.With(middleware.RequirePermission("email", "write")).Patch("/{id}", e.HandleUpdateEmailRule)
		r.With(middleware.RequirePermission("email", "delete")).Delete("/{id}", e.HandleDeleteEmailRule)
	})

	// CRM Links
	r.Route("/api/v1/email/links", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "read")).Get("/message/{messageId}", e.HandleGetEmailContactLinks)
		r.With(middleware.RequirePermission("email", "write")).Post("/", e.HandleLinkEmailToContact)
		r.With(middleware.RequirePermission("email", "write")).Delete("/", e.HandleUnlinkEmailFromContact)
		r.With(middleware.RequirePermission("email", "read")).Get("/contact/{contactId}", e.HandleGetContactEmails)
	})

	// Sync
	r.Route("/api/v1/email/sync", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "write")).Post("/trigger", e.HandleTriggerSync)
		r.With(middleware.RequirePermission("email", "read")).Get("/status", e.HandleGetSyncStatus)
		r.With(middleware.RequirePermission("email", "write")).Post("/read-flag", e.HandleSetReadFlag)
	})

	// Attachments
	r.Route("/api/v1/email/attachments", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "write")).Post("/upload", e.HandleUploadAttachment)
		r.With(middleware.RequirePermission("email", "read")).Get("/{id}/download", e.HandleGetAttachmentDownloadURL)
	})

	// Contact Import/Export
	r.Route("/api/v1/email/contacts", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(middleware.RequirePermission("email", "write")).Post("/import/csv", e.HandleImportContactsCSV)
		r.With(middleware.RequirePermission("email", "write")).Post("/import/vcard", e.HandleImportContactsVCard)
		r.With(middleware.RequirePermission("email", "read")).Post("/export/csv", e.HandleExportContactsCSV)
		r.With(middleware.RequirePermission("email", "read")).Post("/export/vcard", e.HandleExportContactsVCard)
	})
}

// ============================================================================
// Account Handlers
// ============================================================================

func (e *EmailRoutes) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.CreateEmailAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateEmailAccount(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleGetAccount(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	userID := r.URL.Query().Get("user_id")
	resp, err := client.GetEmailAccount(r.Context(), &emailv1.GetEmailAccountRequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.UpdateEmailAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Id = chi.URLParam(r, "id")

	resp, err := client.UpdateEmailAccount(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleDeleteAccount(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.DeleteEmailAccount(r.Context(), &emailv1.DeleteEmailAccountRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleTestConnection(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.TestEmailConnectionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.TestEmailConnection(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Folder Handlers
// ============================================================================

func (e *EmailRoutes) HandleListFolders(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	accountID := r.URL.Query().Get("account_id")
	resp, err := client.ListFolders(r.Context(), &emailv1.ListFoldersRequest{AccountId: accountID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetFolder(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.GetFolder(r.Context(), &emailv1.GetFolderRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleSyncFolders(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.SyncFoldersRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.SyncFolders(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Message Handlers
// ============================================================================

func (e *EmailRoutes) HandleListMessages(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	sortDesc, _ := strconv.ParseBool(r.URL.Query().Get("sort_desc"))

	resp, err := client.ListMessages(r.Context(), &emailv1.ListMessagesRequest{
		FolderId: r.URL.Query().Get("folder_id"),
		Page:     int32(page),
		PerPage:  int32(perPage),
		Search:   r.URL.Query().Get("search"),
		SortBy:   r.URL.Query().Get("sort_by"),
		SortDesc: sortDesc,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetMessage(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.GetMessage(r.Context(), &emailv1.GetMessageRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetThreadMessages(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.GetThreadMessages(r.Context(), &emailv1.GetThreadMessagesRequest{
		ThreadId: chi.URLParam(r, "threadId"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.MarkRead(r.Context(), &emailv1.MarkReadRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleMarkUnread(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.MarkUnread(r.Context(), &emailv1.MarkUnreadRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleToggleStar(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.ToggleStar(r.Context(), &emailv1.ToggleStarRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleMoveToFolder(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	body, ok := decodeAndValidate[moveToFolderDTO](w, r)
	if !ok {
		return
	}

	resp, err := client.MoveToFolder(r.Context(), &emailv1.MoveToFolderRequest{
		MessageId:      chi.URLParam(r, "id"),
		TargetFolderId: body.TargetFolderID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.DeleteMessage(r.Context(), &emailv1.DeleteMessageRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Send/Compose Handlers
// ============================================================================

func (e *EmailRoutes) HandleSendEmail(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	dto, ok := decodeAndValidate[sendEmailDTO](w, r)
	if !ok {
		return
	}

	req := &emailv1.SendEmailRequest{
		AccountId: dto.AccountID,
		To:        emailAddressStrings(dto.To),
		Cc:        emailAddressStrings(dto.Cc),
		Bcc:       emailAddressStrings(dto.Bcc),
		Subject:   dto.Subject,
		BodyHtml:  dto.BodyHtml,
		BodyText:  dto.BodyText,
	}
	if dto.ContactID != "" {
		req.ContactId = &dto.ContactID
	}

	resp, err := client.SendEmail(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleSaveDraft(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	dto, ok := decodeAndValidate[saveDraftDTO](w, r)
	if !ok {
		return
	}

	req := &emailv1.SaveDraftRequest{
		AccountId: dto.AccountID,
		To:        emailAddressStrings(dto.To),
		Cc:        emailAddressStrings(dto.Cc),
		Bcc:       emailAddressStrings(dto.Bcc),
		Subject:   dto.Subject,
		BodyHtml:  dto.BodyHtml,
		BodyText:  dto.BodyText,
	}

	resp, err := client.SaveDraft(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleReplyEmail(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	dto, ok := decodeAndValidate[replyEmailDTO](w, r)
	if !ok {
		return
	}

	req := &emailv1.ReplyEmailRequest{
		AccountId:         dto.AccountID,
		OriginalMessageId: dto.OriginalMessageID,
		BodyHtml:          dto.BodyHtml,
		BodyText:          dto.BodyText,
		ReplyAll:          dto.ReplyAll,
	}

	resp, err := client.ReplyEmail(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleForwardEmail(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	dto, ok := decodeAndValidate[forwardEmailDTO](w, r)
	if !ok {
		return
	}

	req := &emailv1.ForwardEmailRequest{
		AccountId:         dto.AccountID,
		OriginalMessageId: dto.OriginalMessageID,
		To:                emailAddressStrings(dto.To),
		BodyHtml:          dto.BodyHtml,
		BodyText:          dto.BodyText,
	}

	resp, err := client.ForwardEmail(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

// ============================================================================
// Signature Handlers
// ============================================================================

func (e *EmailRoutes) HandleListSignatures(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	userID := r.URL.Query().Get("user_id")
	resp, err := client.ListSignatures(r.Context(), &emailv1.ListSignaturesRequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetSignature(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.GetSignature(r.Context(), &emailv1.GetSignatureRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleCreateSignature(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.CreateSignatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateSignature(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleUpdateSignature(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.UpdateSignatureRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Id = chi.URLParam(r, "id")

	resp, err := client.UpdateSignature(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleDeleteSignature(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.DeleteSignature(r.Context(), &emailv1.DeleteSignatureRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleSetDefaultSignature(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	body, ok := decodeAndValidate[setDefaultSignatureDTO](w, r)
	if !ok {
		return
	}

	resp, err := client.SetDefaultSignature(r.Context(), &emailv1.SetDefaultSignatureRequest{
		Id:     chi.URLParam(r, "id"),
		UserId: body.UserID,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// CRM Link Handlers
// ============================================================================

func (e *EmailRoutes) HandleGetEmailContactLinks(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.GetEmailContactLinks(r.Context(), &emailv1.GetEmailContactLinksRequest{
		MessageId: chi.URLParam(r, "messageId"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleLinkEmailToContact(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.LinkEmailToContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.LinkEmailToContact(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleUnlinkEmailFromContact(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.UnlinkEmailFromContactRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.UnlinkEmailFromContact(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetContactEmails(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))

	resp, err := client.GetContactEmails(r.Context(), &emailv1.GetContactEmailsRequest{
		ContactId: chi.URLParam(r, "contactId"),
		Page:      int32(page),
		PerPage:   int32(perPage),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Sync Handlers
// ============================================================================

func (e *EmailRoutes) HandleTriggerSync(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.TriggerSyncRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.TriggerSync(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetSyncStatus(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	accountID := r.URL.Query().Get("account_id")
	resp, err := client.GetSyncStatus(r.Context(), &emailv1.GetSyncStatusRequest{AccountId: accountID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleSetReadFlag(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.SetReadFlagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.SetReadFlag(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Attachment Handlers
// ============================================================================

func (e *EmailRoutes) HandleUploadAttachment(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// Parse multipart form
	if err := r.ParseMultipartForm(50 << 20); err != nil { // 50MB limit
		response.Error(w, http.StatusBadRequest, "invalid multipart form")
		return
	}

	file, header, err := r.FormFile("file")
	if err != nil {
		response.Error(w, http.StatusBadRequest, "file is required")
		return
	}
	defer file.Close()

	content, err := io.ReadAll(file)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "failed to read file")
		return
	}

	resp, err := client.UploadAttachment(r.Context(), &emailv1.UploadAttachmentRequest{
		AccountId:   r.FormValue("account_id"),
		Filename:    header.Filename,
		ContentType: header.Header.Get("Content-Type"),
		Content:     content,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleGetAttachmentDownloadURL(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.GetAttachmentDownloadURL(r.Context(), &emailv1.GetAttachmentDownloadURLRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Contact Import/Export Handlers
// ============================================================================

func (e *EmailRoutes) HandleImportContactsCSV(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.ImportContactsCSVRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ImportContactsCSV(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleImportContactsVCard(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.ImportContactsVCardRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ImportContactsVCard(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleExportContactsCSV(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.ExportContactsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ExportContactsCSV(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleExportContactsVCard(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.ExportContactsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.ExportContactsVCard(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// ============================================================================
// Rule Handlers (Regeln & Filter)
// ============================================================================

func (e *EmailRoutes) HandleListEmailRules(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.ListEmailRules(r.Context(), &emailv1.ListEmailRulesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// Wrapped list via ProtoListWrapped so an empty rule set serialises as
	// {"rules": []} instead of {} -- the frontend types rules as an array.
	response.ProtoListWrapped(w, http.StatusOK, "rules", resp.Rules, nil)
}

func (e *EmailRoutes) HandleCreateEmailRule(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.CreateEmailRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateEmailRule(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleUpdateEmailRule(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	// proto-direct: no local DTO (S4.1 boundary). Every field carries presence,
	// so an omitted key keeps its stored value.
	var req emailv1.UpdateEmailRuleRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Id = chi.URLParam(r, "id")

	resp, err := client.UpdateEmailRule(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleDeleteEmailRule(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.DeleteEmailRule(r.Context(), &emailv1.DeleteEmailRuleRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleApplyEmailRules(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		response.Error(w, http.StatusBadGateway, "email service unavailable")
		return
	}

	resp, err := client.ApplyEmailRules(r.Context(), &emailv1.ApplyEmailRulesRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// Built explicitly rather than passed through response.Proto: protojson
	// omits zero-valued scalars, so a run that changed nothing would answer {}
	// and the frontend's `res.affected` would read undefined instead of 0.
	response.JSON(w, http.StatusOK, map[string]any{
		"affected": int(resp.Affected),
		"scanned":  int(resp.Scanned),
	})
}
