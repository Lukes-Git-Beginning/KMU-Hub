# Phase 17: Integration - Teams & Slack - Research

**Researched:** 2026-02-20
**Domain:** External platform integration (Microsoft Teams Bot Framework, Slack API, notification forwarding, interactive messaging)
**Confidence:** MEDIUM

## Summary

Phase 17 integrates KMU Hub's existing notification infrastructure (Phase 4) with Microsoft Teams and Slack. Notifications flow outbound as rich platform-native cards (Adaptive Cards for Teams, Block Kit for Slack), and users interact back via buttons on those cards (acknowledge, reply, approve/reject). The integration requires two distinct approaches: Teams needs a full Bot Framework app (Azure AD registration, messaging endpoint) because Office 365 Connectors / incoming webhooks are being retired (deadline extended to ~April 2026); Slack needs a proper Slack App with OAuth install flow and bot token (not incoming webhooks) because interactive messages require `chat.postMessage`, not webhook-based posting.

The existing notification system uses `pg_notify` event bus with `EventPayload` dispatched to handlers, and a `DeliveryCallback` pattern in the dispatcher. The Teams/Slack forwarder will register as an additional delivery callback (or wildcard event handler) that formats and forwards notifications to configured external channels. Inbound interactions from Teams/Slack will arrive via HTTPS webhooks to the gateway, which verifies signatures, resolves linked KMU Hub users, and performs the requested action (mark read, reply, approve/reject) via existing gRPC service calls.

**Primary recommendation:** Build a platform-agnostic integration layer in the notification service with platform-specific adapters for Teams and Slack, using `infracloudio/msbotbuilder-go` for Teams Bot Framework and `slack-go/slack` for Slack API, co-hosted in the notification service binary.

<user_constraints>
## User Constraints (from CONTEXT.md)

### Locked Decisions
- All notification types are forwardable -- admin controls which modules forward via module-level toggles per channel
- Rich cards using native platform formats: Teams Adaptive Cards, Slack Block Kit
- Cards include title, body, colored sidebar, action buttons, and module icon
- Module-level toggle per channel mapping (CRM on/off, HR on/off, Finance on/off, etc.) -- not individual notification categories
- Multi-channel mapping: admin maps module groups to different Teams/Slack channels (e.g., CRM -> #sales, HR -> #hr)
- Both Teams AND Slack can be configured simultaneously for the same org
- "Send test notification" button when setting up a channel mapping to verify webhook connectivity
- Most-specific-wins routing: if a notification matches multiple channel mappings, only the most specific one fires (avoids duplicates)
- Full context-dependent action set on notification cards: Acknowledge, Reply, Approve/Reject
- Buttons shown are context-dependent based on notification type
- Explicit account linking: user runs a `/link` command in Teams/Slack once to connect their KMU Hub account
- Card updates in-place after action (e.g., "Approved by Max" with checkmark) -- no reply clutter
- Unlinked users get an ephemeral (private) prompt to link their account when they try to interact
- Teams: Full Teams App via Bot Framework (Azure AD app registration) -- handles both outbound posting and inbound interactions in one setup
- Slack: Slack App with OAuth install flow -- bot posting, interactive messages, slash commands
- Configuration lives in Admin Settings > Integrations -- dedicated section with cards for each platform
- Step-by-step wizard for setup: 1) Choose platform, 2) Authenticate/install app, 3) Select channels + module mappings, 4) Send test notification

### Claude's Discretion
- Whether forwarded notifications include original KMU Hub user identity (name/avatar) vs bot-only identity -- decide based on DSGVO considerations and platform conventions
- Card color scheme and icon mapping per module
- Exact Adaptive Card / Block Kit template structure
- Error handling for failed webhook deliveries (retry strategy, admin alerts)
- Rate limiting on outbound notifications to avoid Teams/Slack throttling

### Deferred Ideas (OUT OF SCOPE)
None -- discussion stayed within phase scope
</user_constraints>

<phase_requirements>
## Phase Requirements

| ID | Description | Research Support |
|----|-------------|-----------------|
| INT-04 | Admin can configure Microsoft Teams webhook for notification forwarding to a Teams channel | Teams Bot Framework app registration with Azure AD, channel mapping config stored in DB, outbound posting via Bot connector API, setup wizard in Admin > Integrations |
| INT-05 | Admin can configure Slack webhook for notification forwarding to a Slack channel | Slack App with OAuth install flow, bot token stored encrypted in vault, `chat.postMessage` for outbound posting, channel mapping config in DB |
| INT-06 | Users can perform basic interactions (acknowledge, respond) from Teams/Slack back to KMU Hub | Inbound HTTPS webhook endpoints in gateway for both platforms, user account linking via `/link` command, card updates via `updateActivity` (Teams) and `chat.update` (Slack), action routing to existing gRPC services |
</phase_requirements>

## Standard Stack

### Core
| Library | Version | Purpose | Why Standard |
|---------|---------|---------|--------------|
| `infracloudio/msbotbuilder-go` | latest | Teams Bot Framework SDK for Go | Only maintained Go SDK for Bot Framework; supports proactive messaging, activity updates, Adaptive Card attachments |
| `slack-go/slack` | latest (pre-v1) | Slack API client for Go | De-facto standard Go Slack library; supports chat.postMessage, chat.update, interactive messages, OAuth, slash commands, Socket Mode |
| `atc0005/go-teams-notify/v2` | v2.9+ | Adaptive Card builder for Teams | Rich Go types for constructing Adaptive Cards with actions, containers, columns |

### Supporting
| Library | Version | Purpose | When to Use |
|---------|---------|---------|-------------|
| existing `internal/notification/event` | n/a | Event bus and notification dispatch | Register Teams/Slack forwarder as wildcard handler or additional delivery callback |
| existing `security.v1.VaultService` | n/a | Encrypted secret storage | Store Teams app credentials and Slack bot tokens encrypted at rest |
| existing `chi/v5` | n/a | HTTP router | Inbound webhook endpoints for Teams/Slack interactive message callbacks |

### Alternatives Considered
| Instead of | Could Use | Tradeoff |
|------------|-----------|----------|
| `msbotbuilder-go` | Raw REST calls to Bot Framework API | More control but must handle auth token refresh, conversation references, activity schema manually. SDK handles this. |
| `slack-go/slack` | Raw HTTP to Slack API | Possible but loses type safety, rate limit handling, OAuth helpers. Library is mature and widely used. |
| `go-teams-notify` for outbound | `msbotbuilder-go` only | go-teams-notify has better Adaptive Card builder types; msbotbuilder-go handles bot lifecycle. Use both: card building + bot delivery. |

**Installation:**
```bash
go get github.com/infracloudio/msbotbuilder-go
go get github.com/slack-go/slack
go get github.com/atc0005/go-teams-notify/v2
```

## Architecture Patterns

### Recommended Project Structure
```
backend/
  internal/
    notification/
      integration/               # NEW - external platform integration
        types.go                 # PlatformConfig, ChannelMapping, AccountLink models
        repository.go            # Repository interface
        postgres_repository.go   # DB operations for configs, mappings, links
        forwarder.go             # Core forwarder: event -> platform dispatch
        teams/
          client.go              # Teams Bot Framework client wrapper
          card_builder.go        # Adaptive Card template builder
          webhook_handler.go     # Inbound Action.Execute handler
        slack/
          client.go              # Slack API client wrapper
          block_builder.go       # Block Kit message builder
          webhook_handler.go     # Inbound interactive message handler
          oauth.go               # OAuth install flow handler
        account_link.go          # /link command handler, user resolution
  internal/gateway/
    route_integration.go         # NEW - HTTP routes for admin config + inbound webhooks
  proto/notification/v1/
    notification.proto           # EXTENDED - add integration RPCs
  migrations/
    000053_create_integration_tables.up.sql
    000053_create_integration_tables.down.sql
desktop/
  src/renderer/src/
    modules/settings/tabs/
      IntegrationsSettingsTab.tsx # NEW - Admin > Integrations page
    modules/settings/integrations/
      TeamsSetupWizard.tsx       # NEW - 4-step Teams setup wizard
      SlackSetupWizard.tsx       # NEW - 4-step Slack setup wizard
      ChannelMappingEditor.tsx   # NEW - module-to-channel mapping config
      IntegrationCard.tsx        # NEW - platform card component (reusable for Phase 18-19)
    api/hooks/
      useIntegration.ts          # NEW - TanStack Query hooks for integration endpoints
    api/
      integration-types.ts       # NEW - TypeScript types
```

### Pattern 1: Platform-Agnostic Forwarder with Adapter Pattern
**What:** A `Forwarder` struct receives notification events and dispatches to platform-specific adapters based on channel mapping configuration.
**When to use:** When the same notification must potentially go to Teams, Slack, or both simultaneously with platform-specific formatting.
**Example:**
```go
// Forwarder dispatches notifications to external platforms
type Forwarder struct {
    repo       IntegrationRepository
    teams      *teams.Client   // nil if not configured
    slack      *slack.Client   // nil if not configured
    cache      *MappingCache   // TTL cache of channel mappings
}

// HandleNotification is registered as a DeliveryCallback on the dispatcher
func (f *Forwarder) HandleNotification(ctx context.Context, notif *models.Notification, decision *preference.DeliveryDecision) {
    mappings, err := f.cache.GetMappingsForModule(ctx, notif.ModuleID)
    if err != nil || len(mappings) == 0 {
        return
    }

    // Most-specific-wins: filter to best match
    best := selectMostSpecific(mappings, notif)
    if best == nil {
        return
    }

    // Dispatch to configured platform(s)
    for _, m := range best {
        switch m.Platform {
        case PlatformTeams:
            go f.teams.PostNotification(ctx, m, notif)
        case PlatformSlack:
            go f.slack.PostNotification(ctx, m, notif)
        }
    }
}
```

### Pattern 2: Inbound Webhook with Signature Verification + User Resolution
**What:** External platforms POST interactions to KMU Hub gateway. The handler verifies the request signature, resolves the external user to a KMU Hub account via the account_links table, then performs the action.
**When to use:** Every inbound interaction from Teams or Slack.
**Example:**
```go
// HandleSlackInteraction receives Slack interactive message callbacks
func (h *IntegrationRoutes) HandleSlackInteraction(w http.ResponseWriter, r *http.Request) {
    // 1. Verify Slack signing secret
    if !slack.VerifyRequest(r, h.slackSigningSecret) {
        respondError(w, http.StatusUnauthorized, "invalid signature")
        return
    }
    // 2. Parse action payload
    var payload slack.InteractionCallback
    json.Unmarshal([]byte(r.FormValue("payload")), &payload)
    // 3. Resolve Slack user -> KMU Hub user
    link, err := h.repo.GetAccountLink(ctx, PlatformSlack, payload.User.ID)
    if err != nil {
        // Send ephemeral "please link your account" message
        return
    }
    // 4. Execute action (acknowledge, reply, approve/reject)
    h.executeAction(ctx, link.KMUHubUserID, payload)
    // 5. Update original card in-place
    h.slackClient.UpdateMessage(ctx, payload.Channel.ID, payload.MessageTs, updatedBlocks)
}
```

### Pattern 3: Account Linking via Slash Command
**What:** Users type `/kmuhub link` in Teams/Slack, which triggers a one-time linking flow generating a short-lived token that the user enters in KMU Hub (or vice versa).
**When to use:** One-time per-user setup for bidirectional interactions.
**Example flow:**
1. User types `/kmuhub link` in Slack/Teams
2. Bot responds with ephemeral message containing a unique link token (valid 5 min)
3. User pastes token in KMU Hub Settings > Integrations > "Link Account" dialog
4. KMU Hub verifies token, creates account_link record (platform, external_user_id, kmuhub_user_id)
5. Subsequent interactions from that external user are automatically resolved

### Pattern 4: Reusable Integration Settings Page (Phase 18-19 ready)
**What:** Admin Settings > Integrations tab uses a card-based layout where each platform is an `IntegrationCard` component showing status, connection info, and "Configure" button. The setup wizard is a 4-step dialog matching Phase 16's automation wizard pattern.
**When to use:** This phase and future Phase 18 (Bexio), Phase 19 (Abacus/RmA).

### Anti-Patterns to Avoid
- **Using incoming webhooks for Teams:** Office 365 Connectors are being retired (deadline ~April 2026). Use Bot Framework instead. This is a hard requirement.
- **Using incoming webhooks for Slack outbound with interactive buttons:** Incoming webhooks cannot handle button responses. Must use `chat.postMessage` with bot token.
- **Storing tokens in plaintext:** All OAuth tokens, bot secrets, and signing secrets MUST go through the existing vault service (encrypted at rest).
- **Synchronous forwarding in notification pipeline:** External API calls must be async (goroutine) to not block notification processing. Use fire-and-forget with error logging.
- **Building a separate integration microservice:** The forwarder is tightly coupled to the notification event flow. Co-host in the notification service binary (same pattern as InboxService in Phase 14).

## Don't Hand-Roll

| Problem | Don't Build | Use Instead | Why |
|---------|-------------|-------------|-----|
| Adaptive Card JSON schema | Custom JSON builder | `go-teams-notify/v2/adaptivecard` package | Typed Go structs with validation, handles schema version, container nesting |
| Slack Block Kit JSON | Raw JSON construction | `slack-go/slack` BlockElement types | Type-safe block construction with all element types |
| Teams bot auth token refresh | Manual OAuth2 token cache | `msbotbuilder-go` adapter | Handles Azure AD token acquisition, caching, refresh automatically |
| Slack OAuth2 flow | Custom OAuth2 implementation | `slack-go/slack` OAuthV2Response + existing gateway OAuth helpers | Standard flow with PKCE, scope negotiation, token exchange |
| Slack request verification | HMAC computation | `slack.VerifyRequest()` from slack-go | Handles timestamp check + HMAC-SHA256 correctly |
| Teams bot request verification | JWT validation | `msbotbuilder-go` adapter auth | Validates Bearer token against Azure AD JWKS |
| Retry with exponential backoff | Custom retry loop | Simple goroutine with `time.Sleep` doubling | Keep it simple: 3 retries, 1s/2s/4s, then log failure and alert admin |

**Key insight:** Both Teams and Slack have complex authentication and message formatting requirements with many edge cases. The Go libraries handle these correctly; hand-rolling would introduce subtle security and compatibility bugs.

## Common Pitfalls

### Pitfall 1: Teams Office 365 Connectors Retirement
**What goes wrong:** Building on incoming webhooks / O365 Connectors that are deprecated and will stop working by ~April 2026.
**Why it happens:** Many tutorials and examples still reference the simpler webhook approach.
**How to avoid:** Use Bot Framework with Azure AD app registration from the start. The `msbotbuilder-go` SDK supports proactive messaging and activity updates.
**Warning signs:** Any code using `https://outlook.office.com/webhook/` URLs or MessageCard format as primary delivery.

### Pitfall 2: Slack Incoming Webhooks Cannot Handle Interactions
**What goes wrong:** Posting via incoming webhook, then discovering buttons don't work because response_url is not available for webhook-posted messages.
**Why it happens:** Incoming webhooks are one-way only. Interactive messages require bot token + `chat.postMessage`.
**How to avoid:** Use Slack App with OAuth install flow and bot token (`xoxb-*`) from the start. Store bot_access_token via vault.
**Warning signs:** Using webhook URLs instead of chat.postMessage for outbound notifications that need interactive buttons.

### Pitfall 3: Rate Limiting on High-Volume Notifications
**What goes wrong:** Slack rate limits (1 msg/sec/channel) or Teams throttling causes notification loss during burst events (e.g., mass CRM import triggering many notifications).
**Why it happens:** No outbound rate limiting, notifications forwarded synchronously.
**How to avoid:** Implement per-platform rate limiter (token bucket: 1/sec for Slack per channel, similar for Teams). Queue overflow notifications and batch or drop with admin alert.
**Warning signs:** HTTP 429 responses from Slack/Teams APIs, lost notifications.

### Pitfall 4: Account Linking Race Conditions
**What goes wrong:** Token-based linking flow has race condition if user generates multiple tokens or if token expires mid-flow.
**Why it happens:** Distributed state between external platform and KMU Hub.
**How to avoid:** Single active token per user per platform. New token invalidates old one. 5-minute expiry with clear user feedback. Idempotent link creation (upsert on external_user_id + platform).
**Warning signs:** Multiple account_link rows for same external user, orphaned tokens.

### Pitfall 5: DSGVO and User Identity in Forwarded Messages
**What goes wrong:** Forwarding user names/avatars to external platforms may violate DSGVO if employees haven't consented to their identity being shared outside KMU Hub.
**Why it happens:** Convenience of showing "Max Mueller assigned you a task" in Teams/Slack.
**How to avoid:** Use bot-only identity by default. Show only the notification title/body without actor name. If actor identity is desired, require explicit opt-in per user in their KMU Hub privacy settings.
**Warning signs:** Personal data flowing to external platforms without consent mechanism.

### Pitfall 6: Teams 5-Second Response Timeout
**What goes wrong:** Teams requires bot to respond to Action.Execute invoke within 5 seconds. If the KMU Hub action (e.g., approve invoice) takes longer, Teams shows an error.
**Why it happens:** Complex backend operations exceed Teams timeout.
**How to avoid:** Respond immediately with "Processing..." card update, then perform the actual action asynchronously and update the card again via proactive message when complete.
**Warning signs:** "Something went wrong" errors in Teams after button clicks.

## Code Examples

### Teams Adaptive Card for a Notification
```go
// Build an Adaptive Card for a KMU Hub notification
func BuildTeamsCard(notif *models.Notification, moduleColor string, moduleIcon string) adaptivecard.Card {
    card := adaptivecard.NewCard()
    card.Schema = "http://adaptivecards.io/schemas/adaptive-card.json"
    card.Version = "1.4"

    // Header with colored accent
    container := adaptivecard.Container{}
    container.Style = adaptivecard.ContainerStyleAccent

    titleBlock := adaptivecard.NewTextBlock(notif.Title, true)
    titleBlock.Weight = adaptivecard.WeightBolder
    titleBlock.Size = adaptivecard.SizeMedium

    container.Items = append(container.Items, titleBlock)

    if notif.Body != nil {
        bodyBlock := adaptivecard.NewTextBlock(*notif.Body, true)
        bodyBlock.Size = adaptivecard.SizeDefault
        container.Items = append(container.Items, bodyBlock)
    }

    card.Body = append(card.Body, container)

    // Context-dependent action buttons
    actions := buildActions(notif)
    card.Actions = actions

    return card
}
```

### Slack Block Kit for a Notification
```go
// Build Slack Block Kit message for a KMU Hub notification
func BuildSlackBlocks(notif *models.Notification, moduleColor string) []slack.Block {
    blocks := []slack.Block{
        // Header section with module context
        slack.NewSectionBlock(
            slack.NewTextBlockObject("mrkdwn", fmt.Sprintf("*%s*", notif.Title), false, false),
            nil,
            nil,
        ),
    }

    if notif.Body != nil {
        blocks = append(blocks, slack.NewSectionBlock(
            slack.NewTextBlockObject("mrkdwn", *notif.Body, false, false),
            nil,
            nil,
        ))
    }

    // Action buttons (context-dependent)
    actions := buildSlackActions(notif)
    if len(actions) > 0 {
        blocks = append(blocks, slack.NewActionBlock(
            fmt.Sprintf("notif_%s", notif.ID),
            actions...,
        ))
    }

    return blocks
}
```

### Database Schema for Integration Config
```sql
-- Platform integration configurations (one per platform per org)
CREATE TABLE integration_configs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('teams', 'slack')),
    is_active BOOLEAN NOT NULL DEFAULT false,
    -- Teams: app_id, app_password (encrypted via vault key reference)
    -- Slack: bot_token, signing_secret (encrypted via vault key reference)
    credentials_vault_key VARCHAR(200) NOT NULL,
    -- Platform-specific metadata (e.g., team_id for Slack, tenant_id for Teams)
    metadata JSONB NOT NULL DEFAULT '{}',
    created_by UUID NOT NULL REFERENCES users(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(platform)
);

-- Channel mappings: which modules forward to which channels
CREATE TABLE integration_channel_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    config_id UUID NOT NULL REFERENCES integration_configs(id) ON DELETE CASCADE,
    channel_id VARCHAR(200) NOT NULL,      -- Teams channel ID or Slack channel ID
    channel_name VARCHAR(200) NOT NULL,    -- Display name for admin UI
    -- Module toggles (which modules forward to this channel)
    modules JSONB NOT NULL DEFAULT '[]',   -- ["crm", "hr", "biz", "work", ...]
    is_active BOOLEAN NOT NULL DEFAULT true,
    -- Teams: conversation reference for proactive messaging
    -- Slack: n/a (channel_id sufficient)
    platform_data JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_integration_mappings_config ON integration_channel_mappings(config_id);
CREATE INDEX idx_integration_mappings_active ON integration_channel_mappings(config_id)
    WHERE is_active = true;

-- Account links: external user <-> KMU Hub user
CREATE TABLE integration_account_links (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(20) NOT NULL CHECK (platform IN ('teams', 'slack')),
    external_user_id VARCHAR(200) NOT NULL,
    kmuhub_user_id UUID NOT NULL REFERENCES users(id),
    external_display_name VARCHAR(200),
    linked_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(platform, external_user_id),
    UNIQUE(platform, kmuhub_user_id)
);

CREATE INDEX idx_account_links_lookup ON integration_account_links(platform, external_user_id);

-- Delivery log: track forwarded notifications for debugging
CREATE TABLE integration_delivery_log (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    notification_id UUID NOT NULL,
    mapping_id UUID NOT NULL REFERENCES integration_channel_mappings(id),
    platform VARCHAR(20) NOT NULL,
    status VARCHAR(20) NOT NULL CHECK (status IN ('sent', 'failed', 'rate_limited')),
    -- Platform response (message_ts for Slack, activity_id for Teams)
    platform_message_id VARCHAR(200),
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_delivery_log_notification ON integration_delivery_log(notification_id);
CREATE INDEX idx_delivery_log_cleanup ON integration_delivery_log(created_at)
    WHERE created_at < NOW() - INTERVAL '30 days';

-- Link tokens for account linking flow
CREATE TABLE integration_link_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    platform VARCHAR(20) NOT NULL,
    external_user_id VARCHAR(200) NOT NULL,
    token_hash VARCHAR(64) NOT NULL,  -- SHA-256 of the token
    expires_at TIMESTAMPTZ NOT NULL,
    used BOOLEAN NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_link_tokens_hash ON integration_link_tokens(token_hash) WHERE NOT used;
```

## State of the Art

| Old Approach | Current Approach | When Changed | Impact |
|--------------|------------------|--------------|--------|
| Teams O365 Connectors / Incoming Webhooks | Bot Framework with Azure AD | Aug 2024 (retirement announced), deadline ~Apr 2026 | MUST use Bot Framework. Connectors will stop working. |
| Teams Action.Submit for card interactions | Action.Execute (Universal Actions) | 2021+ | Action.Execute allows inline card updates; Action.Submit cannot return new card. Use Action.Execute with Action.Submit fallback. |
| Slack legacy attachments | Block Kit | 2019+ | Block Kit is the modern standard. Legacy attachments still work but no new features. |
| Slack classic apps (legacy tokens) | Slack Apps with OAuth V2 | 2020+ | New apps must use OAuth V2 with granular bot scopes. |

**Deprecated/outdated:**
- Teams Office 365 Connectors: Retiring ~April 2026. Do not use.
- Teams MessageCard format: Deprecated in favor of Adaptive Cards.
- Slack incoming webhooks for interactive messages: One-way only, cannot handle button responses.
- Slack classic OAuth (V1): Replaced by OAuth V2 with granular scopes.

## Discretion Decisions (Researcher Recommendations)

### 1. User Identity in Forwarded Messages
**Recommendation:** Bot-only identity by default. Notifications show "KMU Hub" as sender with module-specific icon. Do NOT include actor names/avatars in forwarded messages unless the user has explicitly opted in via their privacy settings.
**Rationale:** DSGVO Article 6 requires lawful basis for processing personal data. Forwarding employee names to external platforms (which may have different data retention policies) without explicit consent is risky. Bot-only identity is the safe default.

### 2. Card Color Scheme per Module
**Recommendation:** Map existing module color tokens from the desk theme system:
- CRM: Blue (#3b82f6)
- Work/Projects: Purple (#8b5cf6)
- HR: Green (#22c55e)
- Finance: Amber (#f59e0b)
- Email: Sky (#0ea5e9)
- Chat: Emerald (#10b981)
- Calendar: Orange (#f97316)
- Documents: Slate (#64748b)
- System: Red (#ef4444)

Use as Adaptive Card accent color (Teams) and Slack attachment color sidebar.

### 3. Error Handling for Failed Deliveries
**Recommendation:** 3 retries with exponential backoff (1s, 2s, 4s). After 3 failures:
- Log to `integration_delivery_log` with status='failed'
- Increment a failure counter per channel mapping
- If >10 consecutive failures: auto-disable the channel mapping, emit a system notification to admin ("Teams channel #sales disconnected: delivery failures")
- Admin sees error badge on the Integration card in settings

### 4. Rate Limiting Strategy
**Recommendation:**
- Slack: Token bucket at 1 msg/sec/channel (matches Slack's documented limit). Queue excess with 30s buffer before dropping.
- Teams: Token bucket at 4 msg/sec (Bot Framework general guidance). Queue excess similarly.
- Both: If queue exceeds 100 pending messages, batch remaining into a digest message ("5 new notifications in CRM") rather than individual cards.

## Open Questions

1. **Teams Bot registration: self-hosted vs SaaS**
   - What we know: Bot Framework requires Azure AD app registration. Self-hosted customers would need their own Azure tenant.
   - What's unclear: Whether we can provide a single multi-tenant Azure AD app for SaaS, or if each customer needs their own registration.
   - Recommendation: For v1, document that Teams integration requires customer's own Azure AD app registration (they paste app_id + app_password into setup wizard). Multi-tenant app registration is a Phase 20+ optimization.

2. **Slack App distribution: workspace install vs org-wide**
   - What we know: Slack App needs to be installed per workspace. OAuth flow handles this.
   - What's unclear: Whether we register a single Slack App (submitted to Slack App Directory) or each KMU Hub instance creates its own Slack App.
   - Recommendation: For v1, each KMU Hub instance uses its own Slack App (customer creates it at api.slack.com, enters client_id + client_secret + signing_secret). Directory listing is a future optimization.

3. **Teams conversation reference persistence**
   - What we know: To send proactive messages, Teams bots need a `conversationReference` obtained when the bot is first added to a channel.
   - What's unclear: Exactly how `msbotbuilder-go` handles conversation reference storage/retrieval.
   - Recommendation: Store conversation reference as JSON in `platform_data` column of `integration_channel_mappings`. Capture on bot installation event.

## Sources

### Primary (HIGH confidence)
- [Slack Block Kit docs](https://api.slack.com/block-kit) -- official Block Kit building documentation
- [Slack interactive messages](https://api.slack.com/interactivity/handling) -- interactive component handling
- [Slack rate limits](https://api.slack.com/docs/rate-limits) -- rate limit tiers and guidelines
- [Slack chat.update](https://api.slack.com/methods/chat.update) -- message update for in-place card replacement
- [Slack OAuth install](https://api.slack.com/authentication/oauth-v2) -- OAuth V2 install flow
- [Teams Adaptive Cards](https://learn.microsoft.com/en-us/adaptive-cards/getting-started/bots) -- bot Adaptive Card usage
- [Teams Universal Actions](https://learn.microsoft.com/en-us/microsoftteams/platform/task-modules-and-cards/cards/universal-actions-for-adaptive-cards/overview) -- Action.Execute for card updates
- [Teams proactive messages](https://learn.microsoft.com/en-us/microsoftteams/platform/bots/how-to/conversations/send-proactive-messages) -- proactive messaging pattern
- [Teams O365 Connector retirement](https://devblogs.microsoft.com/microsoft365dev/retirement-of-office-365-connectors-within-microsoft-teams/) -- official retirement announcement

### Secondary (MEDIUM confidence)
- [slack-go/slack](https://github.com/slack-go/slack) -- Go Slack library (GitHub, widely used, pre-v1)
- [infracloudio/msbotbuilder-go](https://github.com/infracloudio/msbotbuilder-go) -- Go Bot Framework SDK (community maintained, last activity needs verification)
- [atc0005/go-teams-notify](https://github.com/atc0005/go-teams-notify) -- Adaptive Card builder (actively maintained, v2.9+)
- [Slack incoming webhooks limitations](https://api.slack.com/incoming-webhooks) -- one-way limitation documented

### Tertiary (LOW confidence)
- `msbotbuilder-go` activity update capability -- found via pkg.go.dev sample, not verified against latest version
- Teams 5-second response timeout -- documented in MS Learn but exact behavior with Go SDK unverified

## Metadata

**Confidence breakdown:**
- Standard stack: MEDIUM -- slack-go/slack is well-established; msbotbuilder-go is community-maintained with less certainty on maintenance status
- Architecture: HIGH -- follows established project patterns (DeliveryCallback, RouteRegistrar, vault for secrets, co-hosted services)
- Pitfalls: HIGH -- Teams connector retirement is well-documented; Slack limitations are official docs
- Code examples: MEDIUM -- based on library documentation, not running code verification

**Research date:** 2026-02-20
**Valid until:** 2026-03-20 (30 days -- stable domain, but verify msbotbuilder-go activity before planning)
