package gateway

import (
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"

	"github.com/go-chi/chi/v5"

	"github.com/kmuhub/kmuhub/internal/middleware"
	"github.com/kmuhub/kmuhub/internal/server/response"
	emailv1 "github.com/kmuhub/kmuhub/proto/email/v1"
)

// ============================================================================
// Local DTOs for email send/compose handlers (S4.1 validation boundary)
// ============================================================================

type sendEmailDTO struct {
	AccountID string   `json:"account_id"`
	To        addrList `json:"to" validate:"required,min=1"`
	Cc        addrList `json:"cc"`
	Bcc       addrList `json:"bcc"`
	Subject   string   `json:"subject"`
	BodyHtml  string   `json:"body_html"`
	BodyText  string   `json:"body_text"`
	// ContactID, when set, makes the email service enforce marketing consent
	// (UWG §7) for the recipient. Omit for transactional/system mail.
	ContactID     string   `json:"contact_id" validate:"omitempty,uuid"`
	AttachmentIDs []string `json:"attachment_ids" validate:"omitempty,max=20,dive,uuid"`
}

type saveDraftDTO struct {
	AccountID     string   `json:"account_id"`
	To            addrList `json:"to"`
	Cc            addrList `json:"cc"`
	Bcc           addrList `json:"bcc"`
	Subject       string   `json:"subject"`
	BodyHtml      string   `json:"body_html"`
	BodyText      string   `json:"body_text"`
	AttachmentIDs []string `json:"attachment_ids" validate:"omitempty,max=20,dive,uuid"`
}

type replyEmailDTO struct {
	AccountID         string   `json:"account_id"`
	OriginalMessageID string   `json:"original_message_id" validate:"required"`
	BodyHtml          string   `json:"body_html"`
	BodyText          string   `json:"body_text"`
	ReplyAll          bool     `json:"reply_all"`
	AttachmentIDs     []string `json:"attachment_ids" validate:"omitempty,max=20,dive,uuid"`
}

type forwardEmailDTO struct {
	AccountID         string   `json:"account_id"`
	OriginalMessageID string   `json:"original_message_id" validate:"required"`
	To                addrList `json:"to" validate:"required,min=1"`
	BodyHtml          string   `json:"body_html"`
	BodyText          string   `json:"body_text"`
	AttachmentIDs     []string `json:"attachment_ids" validate:"omitempty,max=20,dive,uuid"`
}

type setDefaultSignatureDTO struct {
	UserID string `json:"user_id" validate:"required,uuid"`
}

type moveToFolderDTO struct {
	TargetFolderID string `json:"target_folder_id" validate:"required,uuid"`
}

// bulkMessageActionDTO backs POST /api/v1/email/messages/bulk. 500 mirrors the
// largest selection a "select all" in the message list can realistically
// produce (per_page caps at 100, so 500 already covers several pages).
type bulkMessageActionDTO struct {
	IDs    []string `json:"ids" validate:"required,min=1,max=500,dive,uuid"`
	Action string   `json:"action" validate:"required"`
	Target string   `json:"target" validate:"omitempty,uuid"`
}

// createEmailTemplateDTO backs POST /api/v1/email/templates. user_id/is_admin
// are resolved server-side from the auth token, not accepted from the body —
// the same IDOR-avoidance as the account-switcher routes (route_email.go
// HandleListAccounts).
type createEmailTemplateDTO struct {
	Name       string `json:"name" validate:"required"`
	Subject    string `json:"subject"`
	BodyHtml   string `json:"body_html"`
	BodyText   string `json:"body_text"`
	Visibility string `json:"visibility" validate:"omitempty,oneof=personal shared"`
}

type updateEmailTemplateDTO struct {
	Name       *string `json:"name"`
	Subject    *string `json:"subject"`
	BodyHtml   *string `json:"body_html"`
	BodyText   *string `json:"body_text"`
	Visibility *string `json:"visibility" validate:"omitempty,oneof=personal shared"`
}

type renderEmailTemplateDTO struct {
	Values map[string]string `json:"values"`
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
	// The email:mailbox keys open what a mailbox owner does with their own
	// mail. The email server confines every route behind them to the caller's
	// own accounts (ownAccount/ownMessage in server/email_grpc.go), so they are
	// safe to hand to every employee. The email:account keys open setting up,
	// changing and removing an account: at scope own the create handler pins
	// user_id to the caller and the email server confines update and delete to
	// the caller's own accounts (the dial guard keeps the hosts public). Rules
	// (tenant-wide, applied to every mailbox), label catalogue writes,
	// shared-template edits, the contact mail history and CRM import/export
	// reach beyond one mailbox and stay on the coarse keys.
	var (
		boxRead    = middleware.RequirePermissionAny([2]string{"email", "read"}, [2]string{"email:mailbox", "read"})
		boxWrite   = middleware.RequirePermissionAny([2]string{"email", "write"}, [2]string{"email:mailbox", "write"})
		boxDelete  = middleware.RequirePermissionAny([2]string{"email", "delete"}, [2]string{"email:mailbox", "delete"})
		acctWrite  = middleware.RequirePermissionAny(emailWriteKey, emailAccountWriteKey)
		acctDelete = middleware.RequirePermissionAny([2]string{"email", "delete"}, [2]string{"email:account", "delete"})
	)

	// Email Accounts
	r.Route("/api/v1/email/accounts", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(acctWrite).Post("/", e.HandleCreateAccount)
		r.With(boxRead).Get("/", e.HandleGetAccount)
		r.With(boxRead).Get("/list", e.HandleListAccounts)
		r.With(acctWrite).Put("/{id}", e.HandleUpdateAccount)
		r.With(acctDelete).Delete("/{id}", e.HandleDeleteAccount)
		r.With(boxWrite).Post("/{id}/default", e.HandleSetDefaultAccount)
		r.With(acctWrite).Post("/test", e.HandleTestConnection)
	})

	// Folders
	r.Route("/api/v1/email/folders", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxRead).Get("/", e.HandleListFolders)
		r.With(boxRead).Get("/{id}", e.HandleGetFolder)
		r.With(boxWrite).Post("/sync", e.HandleSyncFolders)
	})

	// Messages
	r.Route("/api/v1/email/messages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxRead).Get("/", e.HandleListMessages)
		r.With(boxRead).Get("/{id}", e.HandleGetMessage)
		r.With(boxRead).Get("/thread/{threadId}", e.HandleGetThreadMessages)
		r.With(boxWrite).Post("/{id}/read", e.HandleMarkRead)
		r.With(boxWrite).Post("/{id}/unread", e.HandleMarkUnread)
		r.With(boxWrite).Post("/{id}/star", e.HandleToggleStar)
		r.With(boxWrite).Post("/{id}/move", e.HandleMoveToFolder)
		r.With(boxWrite).Post("/{id}/labels", e.HandleAssignMessageLabels)
		r.With(boxDelete).Delete("/{id}", e.HandleDeleteMessage)
		// "write" is the floor; HandleBulkMessageAction itself requires
		// "email:delete" additionally when action == "delete", matching the
		// single-message DELETE route above instead of letting bulk act as a
		// backdoor around it.
		r.With(boxWrite).Post("/bulk", e.HandleBulkMessageAction)
	})

	// Send/Compose
	r.Route("/api/v1/email/send", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxWrite).Post("/", e.HandleSendEmail)
		r.With(boxWrite).Post("/draft", e.HandleSaveDraft)
		r.With(boxWrite).Post("/reply", e.HandleReplyEmail)
		r.With(boxWrite).Post("/forward", e.HandleForwardEmail)
	})

	// Signatures
	r.Route("/api/v1/email/signatures", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxRead).Get("/", e.HandleListSignatures)
		r.With(boxRead).Get("/{id}", e.HandleGetSignature)
		r.With(boxWrite).Post("/", e.HandleCreateSignature)
		r.With(boxWrite).Put("/{id}", e.HandleUpdateSignature)
		r.With(boxDelete).Delete("/{id}", e.HandleDeleteSignature)
		r.With(boxWrite).Post("/{id}/default", e.HandleSetDefaultSignature)
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

	// Labels
	r.Route("/api/v1/email/labels", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxRead).Get("/", e.HandleListEmailLabels)
		r.With(middleware.RequirePermission("email", "write")).Post("/", e.HandleCreateEmailLabel)
		r.With(middleware.RequirePermission("email", "write")).Patch("/{id}", e.HandleUpdateEmailLabel)
		r.With(middleware.RequirePermission("email", "delete")).Delete("/{id}", e.HandleDeleteEmailLabel)
	})

	// Templates (Vorlagen & Quicktexte)
	r.Route("/api/v1/email/templates", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxRead).Get("/", e.HandleListEmailTemplates)
		r.With(boxRead).Get("/{id}", e.HandleGetEmailTemplate)
		r.With(boxWrite).Post("/", e.HandleCreateEmailTemplate)
		r.With(middleware.RequirePermission("email", "write")).Put("/{id}", e.HandleUpdateEmailTemplate)
		r.With(middleware.RequirePermission("email", "delete")).Delete("/{id}", e.HandleDeleteEmailTemplate)
		r.With(boxRead).Post("/{id}/render", e.HandleRenderEmailTemplate)
	})

	// CRM Links
	r.Route("/api/v1/email/links", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxRead).Get("/message/{messageId}", e.HandleGetEmailContactLinks)
		r.With(boxWrite).Post("/", e.HandleLinkEmailToContact)
		r.With(boxWrite).Delete("/", e.HandleUnlinkEmailFromContact)
		r.With(middleware.RequirePermission("email", "read")).Get("/contact/{contactId}", e.HandleGetContactEmails)
	})

	// Sync
	r.Route("/api/v1/email/sync", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxWrite).Post("/trigger", e.HandleTriggerSync)
		r.With(boxRead).Get("/status", e.HandleGetSyncStatus)
		r.With(boxWrite).Post("/read-flag", e.HandleSetReadFlag)
	})

	// Attachments
	r.Route("/api/v1/email/attachments", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(boxWrite).Post("/upload", e.HandleUploadAttachment)
		r.With(boxRead).Get("/{id}/download", e.HandleGetAttachmentDownloadURL)
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

// Setting up and changing a mail account takes the coarse admin key or the
// self-setup key.
var (
	emailWriteKey        = [2]string{"email", "write"}
	emailAccountWriteKey = [2]string{"email:account", "write"}
)

func (e *EmailRoutes) HandleCreateAccount(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.CreateEmailAccountRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	// At scope own the account is the caller's: an empty user_id becomes theirs,
	// someone else's is refused rather than silently rewritten.
	owner, ok := ownerFilterForScopeAny(w, r, emailWriteKey, emailAccountWriteKey)
	if !ok {
		return
	}
	if owner != nil {
		if req.UserId != "" && req.UserId != *owner {
			response.Error(w, http.StatusForbidden, "you can only set up your own email account")
			return
		}
		req.UserId = *owner
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
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	// Only the coarse email:read (admin provisioning) may look up another
	// user's account; everyone else always gets their own, whatever user_id says.
	userID := r.URL.Query().Get("user_id")
	if userID == "" || !callerHasPermission(r, "email:read") {
		userID = middleware.GetUserID(r.Context())
	}
	resp, err := client.GetEmailAccount(r.Context(), &emailv1.GetEmailAccountRequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

// HandleListAccounts lists every email account the calling user holds.
// The user is resolved from the auth token, not a client-supplied query
// param, so a user can only ever list their own accounts.
func (e *EmailRoutes) HandleListAccounts(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	resp, err := client.ListEmailAccounts(r.Context(), &emailv1.ListEmailAccountsRequest{UserId: userID})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleUpdateAccount(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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

func (e *EmailRoutes) HandleSetDefaultAccount(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	resp, err := client.SetDefaultEmailAccount(r.Context(), &emailv1.SetDefaultEmailAccountRequest{
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	page, _ := strconv.Atoi(r.URL.Query().Get("page"))
	perPage, _ := strconv.Atoi(r.URL.Query().Get("per_page"))
	sortDesc, _ := strconv.ParseBool(r.URL.Query().Get("sort_desc"))

	resp, err := client.ListMessages(r.Context(), &emailv1.ListMessagesRequest{
		FolderId:  r.URL.Query().Get("folder_id"),
		Page:      int32(page),
		PerPage:   int32(perPage),
		Search:    r.URL.Query().Get("search"),
		SortBy:    r.URL.Query().Get("sort_by"),
		SortDesc:  sortDesc,
		AccountId: r.URL.Query().Get("account_id"),
		View:      r.URL.Query().Get("view"),
		Filter:    r.URL.Query().Get("filter"),
		LabelId:   r.URL.Query().Get("label_id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	writeEmailMessages(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetMessage(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	resp, err := client.GetMessage(r.Context(), &emailv1.GetMessageRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	writeEmailMessages(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetThreadMessages(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	resp, err := client.GetThreadMessages(r.Context(), &emailv1.GetThreadMessagesRequest{
		ThreadId: chi.URLParam(r, "threadId"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	writeEmailMessages(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleMarkRead(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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

// HandleBulkMessageAction handles POST /api/v1/email/messages/bulk. Unlike the
// per-message routes above, "delete" here needs an explicit second permission
// check: the route guard only requires email:write (the floor every action in
// this handler needs), so without this check a role holding write but not
// delete could delete messages in bulk that the single DELETE /{id} route
// would refuse them — bulk would silently become a wider door than the
// permission model intends.
func (e *EmailRoutes) HandleBulkMessageAction(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	dto, ok := decodeAndValidate[bulkMessageActionDTO](w, r)
	if !ok {
		return
	}

	perms := middleware.GetUserPermissions(r.Context())
	if dto.Action == "delete" && !slices.Contains(perms, "email:delete") && !slices.Contains(perms, "email:mailbox:delete") {
		response.Error(w, http.StatusForbidden, "insufficient permissions")
		return
	}

	resp, err := client.BulkMessageAction(r.Context(), &emailv1.BulkMessageActionRequest{
		Ids:    dto.IDs,
		Action: dto.Action,
		Target: dto.Target,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, map[string]any{"affected": int(resp.Affected)})
}

// ============================================================================
// Send/Compose Handlers
// ============================================================================

func (e *EmailRoutes) HandleSendEmail(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	dto, ok := decodeAndValidate[sendEmailDTO](w, r)
	if !ok || !validateRecipients(w, dto.To, dto.Cc, dto.Bcc) {
		return
	}

	req := &emailv1.SendEmailRequest{
		AccountId:     dto.AccountID,
		To:            dto.To.proto(),
		Cc:            dto.Cc.proto(),
		Bcc:           dto.Bcc.proto(),
		Subject:       dto.Subject,
		BodyHtml:      dto.BodyHtml,
		BodyText:      dto.BodyText,
		AttachmentIds: dto.AttachmentIDs,
	}
	if dto.ContactID != "" {
		req.ContactId = &dto.ContactID
	}

	resp, err := client.SendEmail(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	writeEmailMessages(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleSaveDraft(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	dto, ok := decodeAndValidate[saveDraftDTO](w, r)
	if !ok || !validateRecipients(w, dto.To, dto.Cc, dto.Bcc) {
		return
	}

	req := &emailv1.SaveDraftRequest{
		AccountId:     dto.AccountID,
		To:            dto.To.proto(),
		Cc:            dto.Cc.proto(),
		Bcc:           dto.Bcc.proto(),
		Subject:       dto.Subject,
		BodyHtml:      dto.BodyHtml,
		BodyText:      dto.BodyText,
		AttachmentIds: dto.AttachmentIDs,
	}

	resp, err := client.SaveDraft(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	writeEmailMessages(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleReplyEmail(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
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
		AttachmentIds:     dto.AttachmentIDs,
	}

	resp, err := client.ReplyEmail(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	writeEmailMessages(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleForwardEmail(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	dto, ok := decodeAndValidate[forwardEmailDTO](w, r)
	if !ok || !validateRecipients(w, dto.To, nil, nil) {
		return
	}

	req := &emailv1.ForwardEmailRequest{
		AccountId:         dto.AccountID,
		OriginalMessageId: dto.OriginalMessageID,
		To:                dto.To.proto(),
		BodyHtml:          dto.BodyHtml,
		BodyText:          dto.BodyText,
		AttachmentIds:     dto.AttachmentIDs,
	}

	resp, err := client.ForwardEmail(r.Context(), req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	writeEmailMessages(w, http.StatusCreated, resp)
}

// ============================================================================
// Signature Handlers
// ============================================================================

func (e *EmailRoutes) HandleListSignatures(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
// Template Handlers (Vorlagen & Quicktexte)
// ============================================================================

func (e *EmailRoutes) HandleListEmailTemplates(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	userID := middleware.GetUserID(r.Context())
	isAdmin := middleware.IsAdmin(r.Context())

	resp, err := client.ListEmailTemplates(r.Context(), &emailv1.ListEmailTemplatesRequest{
		UserId:  userID,
		IsAdmin: isAdmin,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleGetEmailTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	resp, err := client.GetEmailTemplate(r.Context(), &emailv1.GetEmailTemplateRequest{
		Id:      chi.URLParam(r, "id"),
		UserId:  middleware.GetUserID(r.Context()),
		IsAdmin: middleware.IsAdmin(r.Context()),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleCreateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	body, ok := decodeAndValidate[createEmailTemplateDTO](w, r)
	if !ok {
		return
	}

	resp, err := client.CreateEmailTemplate(r.Context(), &emailv1.CreateEmailTemplateRequest{
		UserId:     middleware.GetUserID(r.Context()),
		Name:       body.Name,
		Subject:    body.Subject,
		BodyHtml:   body.BodyHtml,
		BodyText:   body.BodyText,
		Visibility: body.Visibility,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleUpdateEmailTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	body, ok := decodeAndValidate[updateEmailTemplateDTO](w, r)
	if !ok {
		return
	}

	resp, err := client.UpdateEmailTemplate(r.Context(), &emailv1.UpdateEmailTemplateRequest{
		Id:         chi.URLParam(r, "id"),
		UserId:     middleware.GetUserID(r.Context()),
		IsAdmin:    middleware.IsAdmin(r.Context()),
		Name:       body.Name,
		Subject:    body.Subject,
		BodyHtml:   body.BodyHtml,
		BodyText:   body.BodyText,
		Visibility: body.Visibility,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleDeleteEmailTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	resp, err := client.DeleteEmailTemplate(r.Context(), &emailv1.DeleteEmailTemplateRequest{
		Id:      chi.URLParam(r, "id"),
		UserId:  middleware.GetUserID(r.Context()),
		IsAdmin: middleware.IsAdmin(r.Context()),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.JSON(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleRenderEmailTemplate(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	body, ok := decodeAndValidate[renderEmailTemplateDTO](w, r)
	if !ok {
		return
	}

	resp, err := client.RenderEmailTemplate(r.Context(), &emailv1.RenderEmailTemplateRequest{
		Id:      chi.URLParam(r, "id"),
		UserId:  middleware.GetUserID(r.Context()),
		IsAdmin: middleware.IsAdmin(r.Context()),
		Values:  body.Values,
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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

	writeEmailMessages(w, http.StatusOK, resp)
}

// ============================================================================
// Sync Handlers
// ============================================================================

func (e *EmailRoutes) HandleTriggerSync(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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
		respondServiceUnavailable(w, e.ServiceName())
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

// ============================================================================
// Label Handlers
// ============================================================================

func (e *EmailRoutes) HandleListEmailLabels(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	resp, err := client.ListEmailLabels(r.Context(), &emailv1.ListEmailLabelsRequest{})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	// Wrapped list via ProtoListWrapped so an empty label set serialises as
	// {"labels": []} instead of {} -- the frontend types labels as an array.
	response.ProtoListWrapped(w, http.StatusOK, "labels", resp.Labels, nil)
}

func (e *EmailRoutes) HandleCreateEmailLabel(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	// proto-direct: no local DTO (S4.1 boundary)
	var req emailv1.CreateEmailLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.CreateEmailLabel(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusCreated, resp)
}

func (e *EmailRoutes) HandleUpdateEmailLabel(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	// proto-direct: no local DTO (S4.1 boundary). Both fields carry presence,
	// so an omitted key keeps its stored value.
	var req emailv1.UpdateEmailLabelRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}
	req.Id = chi.URLParam(r, "id")

	resp, err := client.UpdateEmailLabel(r.Context(), &req)
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

func (e *EmailRoutes) HandleDeleteEmailLabel(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	resp, err := client.DeleteEmailLabel(r.Context(), &emailv1.DeleteEmailLabelRequest{
		Id: chi.URLParam(r, "id"),
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	response.Proto(w, http.StatusOK, resp)
}

type assignMessageLabelsDTO struct {
	LabelIDs []string `json:"label_ids"`
}

func (e *EmailRoutes) HandleAssignMessageLabels(w http.ResponseWriter, r *http.Request) {
	client, err := e.getEmailClient()
	if err != nil {
		respondServiceUnavailable(w, e.ServiceName())
		return
	}

	var dto assignMessageLabelsDTO
	if err := json.NewDecoder(r.Body).Decode(&dto); err != nil {
		response.Error(w, http.StatusBadRequest, "invalid request body")
		return
	}

	resp, err := client.AssignMessageLabels(r.Context(), &emailv1.AssignMessageLabelsRequest{
		MessageId: chi.URLParam(r, "id"),
		LabelIds:  dto.LabelIDs,
	})
	if err != nil {
		respondGRPCError(w, err)
		return
	}

	writeEmailMessages(w, http.StatusOK, resp)
}
