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

// Well-known module IDs
const (
	ModuleChat   = "chat"
	ModuleCRM    = "crm"
	ModuleWork   = "work"
	ModuleSystem = "system"
)

// PostgreSQL LISTEN channel name for the event bus
const PGNotifyChannel = "events"
