---
phase: 08-video-voice-meetings
plan: 07
subsystem: ui
tags: [livekit, components-react, react, typescript, video, call-ui, screen-share, recording-consent, dsgvo]

# Dependency graph
requires:
  - phase: 08-06
    provides: "LiveKit SDK, types, API client, TanStack Query hooks, Zustand video store"
provides:
  - "VideoCallView with gallery (GridLayout) and speaker (FocusLayout) modes"
  - "PreJoinScreen for camera/mic preview before joining"
  - "CallControls for mic/camera/screen-share/record/leave"
  - "ScreenShareView replacing main area with shared screen + sidebar thumbnails"
  - "FloatingCallBar for persistent call controls when navigating away (bottom-right pill)"
  - "IncomingCallOverlay fullscreen overlay with accept/decline"
  - "RecordingConsentDialog DSGVO-compliant consent with blur/mute consequence explanation"
affects: [08-08, 08-09, 10-email-crm-import-export, 11-documents-files]

# Tech tracking
tech-stack:
  added: []
  patterns: ["LiveKitRoom wrapper with connection management", "Gallery vs Speaker view state toggle", "Active speaker detection auto-switches to speaker view", "Screen share detection triggers layout change", "Fixed floating bar with z-50", "Fullscreen overlay with z-[100]", "Radix AlertDialog for DSGVO consent (no dismiss without response)", "Web Audio API for ringtone generation (400Hz sine wave)"]

key-files:
  created:
    - desktop/src/renderer/src/features/video/VideoCallView.tsx
    - desktop/src/renderer/src/features/video/PreJoinScreen.tsx
    - desktop/src/renderer/src/features/video/CallControls.tsx
    - desktop/src/renderer/src/features/video/ScreenShareView.tsx
    - desktop/src/renderer/src/features/video/FloatingCallBar.tsx
    - desktop/src/renderer/src/features/video/IncomingCallOverlay.tsx
    - desktop/src/renderer/src/features/video/RecordingConsentDialog.tsx
    - desktop/src/renderer/src/features/video/index.ts
  modified: []

key-decisions:
  - "Gallery view uses GridLayout with responsive columns: 1-9 participants = CSS Grid calc, 10-25 = dynamic wrap"
  - "Speaker view uses FocusLayout + CarouselLayout sidebar for thumbnails"
  - "Active speaker events auto-switch to speaker view (useSortedParticipants hook)"
  - "Clicking participant in gallery manually switches to speaker view with focus"
  - "Screen share detection via useTracks filtering for Track.Source.ScreenShare"
  - "FloatingCallBar positioned fixed bottom-right with z-50, pill shape with backdrop-blur"
  - "Duration timer increments via setInterval when floating bar visible"
  - "IncomingCallOverlay uses Web Audio API to generate ringtone (no audio file needed)"
  - "RecordingConsentDialog explains DSGVO Art. 6 compliance and blur/mute consequence for declined consent"
  - "Dialog uses Radix AlertDialog with no escape key, no click-outside dismiss"

patterns-established:
  - "LiveKitRoom as top-level wrapper with token/wsUrl props"
  - "Conditional rendering: GridLayout for gallery, FocusLayout for speaker, ScreenShareView for screen share"
  - "CallControls use @livekit/components-react hooks (useLocalParticipant, setMicrophoneEnabled, etc.)"
  - "Fixed floating UI with high z-index for persistent controls"
  - "Fullscreen overlays with z-[100] for modal interruptions (incoming calls)"
  - "German DSGVO compliance text in all recording-related UI"

# Metrics
duration: 6min
completed: 2026-02-11
---

# Phase 8 Plan 7: Video Call UI and Screen Share Summary

**Complete video call experience with LiveKit components-react: gallery/speaker layouts, screen sharing, floating call bar, incoming call overlay, and DSGVO recording consent**

## Performance

- **Duration:** 6 min
- **Started:** 2026-02-11T17:38:48+01:00
- **Completed:** 2026-02-11T17:40:50+01:00
- **Tasks:** 2 auto, 1 checkpoint (human-verify approved)
- **Files modified:** 8 (7 components + barrel index)

## Accomplishments
- VideoCallView with LiveKitRoom wrapping gallery (GridLayout) and speaker (FocusLayout) modes, switchable via view toggle or active speaker detection
- ScreenShareView automatically replaces main area when screen share track detected, thumbnails move to sidebar
- CallControls with mic/camera/screen-share/record/leave buttons using LiveKit hooks
- FloatingCallBar persistent bottom-right pill with mute/camera/hangup + MM:SS duration timer
- IncomingCallOverlay fullscreen with pulsing avatar, caller name, accept/decline, and Web Audio API ringtone
- RecordingConsentDialog DSGVO-compliant with German text explaining blur/mute consequence for declined consent

## Task Commits

Each task was committed atomically:

1. **Task 1: VideoCallView + PreJoin + CallControls + ScreenShare** - `1e275b9` (feat)
   - VideoCallView with LiveKitRoom, gallery/speaker toggle, active speaker auto-switch
   - Gallery uses GridLayout (responsive 1-25 participants), speaker uses FocusLayout + CarouselLayout
   - Click participant to focus in speaker mode
   - ScreenShareView replaces main area with shared screen + sidebar thumbnails
   - CallControls with mic/camera/screen-share/record/leave buttons
   - PreJoinScreen with camera/mic preview via PreJoin prefab
   - 826 lines added across 5 files

2. **Task 2: FloatingCallBar + IncomingCallOverlay + RecordingConsentDialog** - `77331be` (feat)
   - FloatingCallBar: fixed bottom-right pill with z-50, mute/camera/hangup + MM:SS timer
   - Duration timer via setInterval, click bar to navigate back to call
   - IncomingCallOverlay: fullscreen z-[100] with pulsing avatar, Web Audio API ringtone (400Hz sine wave)
   - Accept joins via useJoinCall, decline clears incoming state
   - RecordingConsentDialog: Radix AlertDialog with DSGVO Art. 6 reference in German
   - Explains blur+mute consequence for declined consent, no dismiss without response
   - 599 lines added across 3 files

**Plan metadata:** (Next commit after approval)

## Files Created/Modified

Created:
- `desktop/src/renderer/src/features/video/VideoCallView.tsx` - Main call view with LiveKitRoom, gallery/speaker toggle, active speaker detection
- `desktop/src/renderer/src/features/video/PreJoinScreen.tsx` - Camera/mic preview before joining via PreJoin prefab
- `desktop/src/renderer/src/features/video/CallControls.tsx` - Bottom control bar with mic/camera/screen-share/record/leave buttons
- `desktop/src/renderer/src/features/video/ScreenShareView.tsx` - Layout for screen sharing (shared screen as main + sidebar thumbnails)
- `desktop/src/renderer/src/features/video/FloatingCallBar.tsx` - Persistent bottom-right pill with call controls + duration timer
- `desktop/src/renderer/src/features/video/IncomingCallOverlay.tsx` - Fullscreen incoming call overlay with accept/decline + ringtone
- `desktop/src/renderer/src/features/video/RecordingConsentDialog.tsx` - DSGVO-compliant consent dialog with blur/mute explanation
- `desktop/src/renderer/src/features/video/index.ts` - Barrel export for clean imports

## Decisions Made

1. **Gallery vs Speaker view toggle:** User can switch manually via button or let active speaker detection auto-switch to speaker view. Clicking a participant in gallery mode focuses that participant in speaker mode.

2. **Screen share layout:** When screen share track detected (via useTracks filtering for Track.Source.ScreenShare), view auto-switches to ScreenShareView which replaces the main area with shared screen and moves participant thumbnails to a vertical sidebar.

3. **FloatingCallBar positioning:** Fixed bottom-right with z-50, pill shape with backdrop-blur. Duration timer increments via setInterval when bar is visible. Clicking bar navigates back to call view.

4. **IncomingCallOverlay ringtone:** Uses Web Audio API to generate a 400Hz sine wave pattern instead of requiring an audio file. This keeps the implementation self-contained.

5. **RecordingConsentDialog compliance:** Implements DSGVO Article 6 compliance by explaining that declining consent results in blurred video and muted audio in the recording (selective consent). Dialog cannot be dismissed without responding (no escape key, no click-outside).

6. **German DSGVO text:** All recording-related UI uses German text to match the DACH KMU target audience and comply with EU data sovereignty requirements.

## Deviations from Plan

None - plan executed exactly as written.

## Issues Encountered

None.

## User Setup Required

None - no external service configuration required.

## Next Phase Readiness

Video call UI foundation complete. Ready for:
- Plan 08-08: Meeting lifecycle (lobby, notes, action items, summary)
- Plan 08-09: Reactions + presence integration + call-from-chat
- Future phases: Email integration (send calendar invites), document sharing (attach files to calls)

All 7 components in features/video/ covering the full call lifecycle are now available for integration.

---
*Phase: 08-video-voice-meetings*
*Completed: 2026-02-11*

## Self-Check: PASSED
