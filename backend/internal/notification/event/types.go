package event

// Well-known event type keys for the notification system.
// Modules register their event types at startup; these constants
// provide compile-time references for built-in types.
const (
	// Chat events
	EventChatMention        = "chat.mention"
	EventChatDMNew          = "chat.dm.new"
	EventChatChannelMessage = "chat.channel.message"

	// CRM events
	EventCRMDealStageChanged = "crm.deal.stage_changed"
	EventCRMDealAssigned     = "crm.deal.assigned"
	EventCRMContactAssigned  = "crm.contact.assigned"

	// Work events
	EventWorkTaskCreated       = "work.task.created"
	EventWorkTaskAssigned      = "work.task.assigned"
	EventWorkTaskStatusChanged = "work.task.status_changed"
	EventWorkTaskCommented     = "work.task.commented"
	EventWorkTaskMentioned     = "work.task.mentioned"

	// System events
	EventSystemAlert = "system.alert"
)

// Email events
const (
	EventEmailReceived = "email.message.received"
	EventEmailSent     = "email.message.sent"
	EventEmailDeleted  = "email.message.deleted"
)

// Document events
const (
	EventDocumentUploaded  = "document.file.uploaded"
	EventDocumentShared    = "document.file.shared"
	EventDocumentVersioned = "document.file.versioned"
)

// Finance (Biz) events
const (
	EventInvoiceCreated  = "biz.invoice.created"
	EventInvoiceSent     = "biz.invoice.sent"
	EventInvoiceOverdue  = "biz.invoice.overdue"
	EventPaymentReceived = "biz.payment.received"
	EventQuoteCreated    = "biz.quote.created"
	EventDunningCreated  = "biz.dunning.created"
)

// HR events
const (
	EventLeaveRequested = "hr.leave.requested"
	EventLeaveApproved  = "hr.leave.approved"
	EventLeaveRejected  = "hr.leave.rejected"
	EventShiftStarted   = "hr.shift.started"
	EventShiftEnded     = "hr.shift.ended"
)

// Inbox events
const (
	EventInboxItemCreated  = "inbox.item.created"
	EventInboxItemAssigned = "inbox.item.assigned"
)

// Automation events
const (
	EventAutomationExecuted = "automation.workflow.executed"
	EventAutomationFailed   = "automation.workflow.failed"
)

// Guest chat events
const (
	EventChatGuestNew     = "chat.guest.new"     // new guest session started
	EventChatGuestMessage = "chat.guest.message"  // guest sent a message
	EventChatGuestEnded   = "chat.guest.ended"    // guest session deactivated
)

// Well-known module IDs
const (
	ModuleChat     = "chat"
	ModuleCRM      = "crm"
	ModuleWork     = "work"
	ModuleSystem   = "system"
	ModuleEmail    = "email"
	ModuleDocument = "document"
	ModuleBiz      = "biz"
	ModuleHR       = "hr"
	ModuleInbox        = "inbox"
	ModuleAutomation   = "automation"
	ModuleIntegration  = "integration"
	ModuleGuest        = "guest"
)

// PostgreSQL LISTEN channel name for the event bus
const PGNotifyChannel = "events"
