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

	// System events
	EventSystemAlert = "system.alert"
)

// Well-known module IDs
const (
	ModuleChat   = "chat"
	ModuleCRM    = "crm"
	ModuleSystem = "system"
)

// PostgreSQL LISTEN channel name for the event bus
const PGNotifyChannel = "events"
