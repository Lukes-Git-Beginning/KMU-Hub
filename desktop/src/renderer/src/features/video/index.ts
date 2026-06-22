/**
 * Video call feature components.
 *
 * Exports all video call UI components for clean barrel imports:
 *   import { VideoCallView, FloatingCallBar } from '@/features/video'
 */

export { VideoCallView } from './VideoCallView'
export type { VideoCallViewProps } from './VideoCallView'

export { PreJoinScreen } from './PreJoinScreen'
export type { PreJoinScreenProps } from './PreJoinScreen'

export { CallControls } from './CallControls'
export type { CallControlsProps } from './CallControls'

export { ScreenShareView } from './ScreenShareView'
export type { ScreenShareViewProps } from './ScreenShareView'

export { FloatingCallBar } from './FloatingCallBar'
export type { FloatingCallBarProps } from './FloatingCallBar'

export { IncomingCallOverlay } from './IncomingCallOverlay'
export type { IncomingCallOverlayProps } from './IncomingCallOverlay'

export { RecordingConsentDialog } from './RecordingConsentDialog'
export type { RecordingConsentDialogProps } from './RecordingConsentDialog'

export { InCallChat } from './InCallChat'
export type { InCallChatProps } from './InCallChat'

export { ReactionLayer, ReactionPicker, useReactionChannel } from './ReactionLayer'
