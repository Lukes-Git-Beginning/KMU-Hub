---
phase: 05-desktop-app-shell
plan: 04
subsystem: desktop-chat-notifications
tags: [chat, notifications, websocket, real-time, electron, react-query]
depends_on:
  requires: ["05-01", "05-02"]
  provides: ["Chat module with channels/DMs/messages/threads", "Notification system with bell/center/native push"]
  affects: ["05-05", "05-06", "05-07"]
tech-stack:
  added: ["@radix-ui/react-dialog", "@radix-ui/react-popover", "@radix-ui/react-scroll-area", "@radix-ui/react-tabs"]
  patterns: ["Infinite scroll via useInfiniteQuery", "WebSocket cache sync pattern", "Typing indicator with debounce and auto-expiry", "Native desktop push via Electron IPC"]
key-files:
  created:
    - desktop/src/renderer/src/api/hooks/useChannels.ts
    - desktop/src/renderer/src/api/hooks/useMessages.ts
    - desktop/src/renderer/src/api/hooks/useNotifications.ts
    - desktop/src/renderer/src/modules/chat/channels/ChannelList.tsx
    - desktop/src/renderer/src/modules/chat/channels/ChannelHeader.tsx
    - desktop/src/renderer/src/modules/chat/channels/CreateChannelDialog.tsx
    - desktop/src/renderer/src/modules/chat/messages/MessageList.tsx
    - desktop/src/renderer/src/modules/chat/messages/MessageBubble.tsx
    - desktop/src/renderer/src/modules/chat/messages/MessageInput.tsx
    - desktop/src/renderer/src/modules/chat/threads/ThreadPanel.tsx
    - desktop/src/renderer/src/modules/notifications/NotificationBell.tsx
    - desktop/src/renderer/src/components/ui/dialog.tsx
    - desktop/src/renderer/src/components/ui/popover.tsx
    - desktop/src/renderer/src/components/ui/scroll-area.tsx
    - desktop/src/renderer/src/components/ui/textarea.tsx
    - desktop/src/renderer/src/components/ui/tabs.tsx
  modified:
    - desktop/src/renderer/src/modules/chat/ChatLayout.tsx
    - desktop/src/renderer/src/modules/notifications/NotificationCenter.tsx
    - desktop/src/renderer/src/components/layout/Header.tsx
    - desktop/package.json
    - desktop/package-lock.json
key-decisions:
  - "Typing indicator: 3s debounce on send, 4s auto-expiry on receive"
  - "Message WebSocket: cache mutations via queryClient.setQueryData, invalidation as fallback"
  - "Thread panel: full implementation in Task 1 since tightly coupled with message hooks"
  - "Notification bell: Popover with max 10 items, link to full center"
  - "Native push: triggered only when document.hasFocus() === false"
metrics:
  duration: "~10 minutes"
  completed: "2026-02-07"
---

# Phase 5 Plan 4: Chat Module and Notifications Summary

**Chat module with three-panel layout (channels/messages/threads), real-time WebSocket messaging, typing indicators, notification bell with unread badge, full notification center, and native Electron push notifications.**

## Performance

- Start: 2026-02-07T21:50:22Z
- End: 2026-02-07T22:00:33Z
- Duration: ~10 minutes
- Tasks: 2/2 completed

## Accomplishments

### Task 1: Chat Module Foundation
- **Channel API hooks** (`useChannels.ts`): 8 hooks covering channel CRUD, join/leave, DMs, unread counts, mark-as-read
- **Message API hooks** (`useMessages.ts`): Infinite scroll via `useInfiniteQuery` with cursor-based pagination, send/edit/delete mutations, WebSocket cache sync for message.new/updated/deleted, typing indicator with debounce/auto-expiry, thread reply hooks
- **ChatLayout**: Three-panel design (280px channel list, flex-1 message area, 350px conditional thread panel)
- **ChannelList**: Channels and DMs sections with search filter, unread badges, create channel dialog
- **ChannelHeader**: Channel name, member count, typing indicator display
- **CreateChannelDialog**: Name, description, private toggle with radix dialog
- **MessageList**: Date-grouped messages with infinite scroll, auto-scroll to bottom on new messages
- **MessageBubble**: Avatar, sender name, timestamp, mention highlighting, thread indicator, hover edit/delete actions
- **MessageInput**: Auto-growing textarea, Enter to send, Shift+Enter for newline, typing emission
- **ThreadPanel**: Parent message preview, thread replies list, reply input with real-time updates

### Task 2: Notification System
- **Notification API hooks** (`useNotifications.ts`): List with pagination/filters, unread count with 30s polling fallback, mark read/all-read, preferences CRUD, event types, WebSocket subscription with native push
- **NotificationBell**: Popover with unread badge, 10-item dropdown, mark-all-read link, navigate on click
- **NotificationCenter**: Full page with All/Unread tabs, pagination, notification cards with priority colors, preferences panel with per-event-type in-app/desktop toggles
- **Header update**: Replaced placeholder bell with real NotificationBell, initialized useNotificationWebSocket for always-active listening
- **Native push**: `window.electronAPI.notifications.show()` triggered when app not focused on notification.new

## Task Commits

| Task | Commit | Description |
|------|--------|-------------|
| 1 | cf1bace | feat(05-04): add chat module with channels, messages, and real-time WebSocket |
| 2 | b7495f7 | feat(05-04): add notification system with bell, center, and desktop push |

## Files Created

- `desktop/src/renderer/src/api/hooks/useChannels.ts` - Channel query/mutation hooks
- `desktop/src/renderer/src/api/hooks/useMessages.ts` - Message hooks with WebSocket cache sync
- `desktop/src/renderer/src/api/hooks/useNotifications.ts` - Notification hooks with native push
- `desktop/src/renderer/src/modules/chat/channels/ChannelList.tsx` - Channel/DM sidebar
- `desktop/src/renderer/src/modules/chat/channels/ChannelHeader.tsx` - Channel header with typing
- `desktop/src/renderer/src/modules/chat/channels/CreateChannelDialog.tsx` - Create channel dialog
- `desktop/src/renderer/src/modules/chat/messages/MessageList.tsx` - Scrollable message history
- `desktop/src/renderer/src/modules/chat/messages/MessageBubble.tsx` - Individual message display
- `desktop/src/renderer/src/modules/chat/messages/MessageInput.tsx` - Message composition
- `desktop/src/renderer/src/modules/chat/threads/ThreadPanel.tsx` - Thread reply panel
- `desktop/src/renderer/src/modules/notifications/NotificationBell.tsx` - Header bell popover
- `desktop/src/renderer/src/components/ui/dialog.tsx` - shadcn dialog component
- `desktop/src/renderer/src/components/ui/popover.tsx` - shadcn popover component
- `desktop/src/renderer/src/components/ui/scroll-area.tsx` - shadcn scroll-area component
- `desktop/src/renderer/src/components/ui/textarea.tsx` - shadcn textarea component
- `desktop/src/renderer/src/components/ui/tabs.tsx` - shadcn tabs component

## Files Modified

- `desktop/src/renderer/src/modules/chat/ChatLayout.tsx` - Replaced placeholder with three-panel layout
- `desktop/src/renderer/src/modules/notifications/NotificationCenter.tsx` - Replaced placeholder with full center
- `desktop/src/renderer/src/components/layout/Header.tsx` - Wired NotificationBell and WebSocket listener
- `desktop/package.json` - Added radix dependencies for dialog, popover, scroll-area, tabs

## Decisions Made

1. **Typing indicator timing**: 3-second debounce on sending typing.start, 4-second auto-expiry on receiving (if no typing.stop received). This prevents excessive WebSocket traffic while keeping indicators responsive.

2. **WebSocket cache sync strategy**: Direct cache mutations via `queryClient.setQueryData` for real-time messages, with `invalidateQueries` as fallback on send mutation success. This gives instant display while ensuring eventual consistency.

3. **Thread panel in Task 1**: Implemented the complete thread panel in Task 1 rather than Task 2 because it's tightly coupled with the message WebSocket hooks (thread.reply.new events) and MessageBubble thread indicators.

4. **Native push gating**: Desktop notifications only trigger when `document.hasFocus() === false`, preventing duplicate notifications when the user is actively using the app.

5. **Notification preferences UI**: Embedded in NotificationCenter as a toggleable panel rather than a separate route, keeping settings contextually close to the notification list.

## Deviations from Plan

None -- plan executed exactly as written.

## Issues / Risks

None encountered.

## Next Phase Readiness

- Chat module is fully functional (requires running backend + WebSocket for real-time features)
- Notification system exercises the complete pipeline from backend through gateway WebSocket to Electron native push
- File upload UI is ready for wiring (attachment button placeholder in MessageInput)
- Remaining plans 05-05 through 05-07 can proceed independently

## Self-Check: PASSED
