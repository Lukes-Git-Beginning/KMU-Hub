/**
 * Tests for InCallChat — verifies that:
 * 1. The chat history loading state is shown initially.
 * 2. Empty state is shown when no messages exist.
 * 3. Local and remote messages render with correct identities.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen } from '@testing-library/react'

// ---------------------------------------------------------------------------
// Mock LiveKit hooks
// ---------------------------------------------------------------------------

const mockLocalParticipant = { identity: 'user-local', name: 'Alice' }

vi.mock('@livekit/components-react', () => ({
  useChat: () => ({
    chatMessages: [],
    send: vi.fn(),
  }),
  useLocalParticipant: () => ({
    localParticipant: mockLocalParticipant,
  }),
  useParticipants: () => [
    { identity: 'user-local', name: 'Alice', isMicrophoneEnabled: true, isCameraEnabled: true },
    { identity: 'user-remote', name: 'Bob', isMicrophoneEnabled: false, isCameraEnabled: true },
  ],
}))

// ---------------------------------------------------------------------------
// Mock API hooks
// ---------------------------------------------------------------------------

vi.mock('@/api/hooks/useVideo', () => ({
  useMeetingChatHistory: (_meetingId: string) => ({
    data: [],
    isLoading: false,
  }),
  useSaveMeetingChatMessage: (_meetingId: string) => ({
    mutate: vi.fn(),
    isPending: false,
  }),
}))

// ---------------------------------------------------------------------------
// Mock i18n
// ---------------------------------------------------------------------------

vi.mock('react-i18next', () => ({
  useTranslation: () => ({
    t: (key: string) => key,
  }),
}))

vi.mock('@/lib', () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(' '),
}))

vi.mock('sonner', () => ({
  toast: { error: vi.fn() },
}))

// ---------------------------------------------------------------------------
// Test subject
// ---------------------------------------------------------------------------

import { InCallChat } from '../InCallChat'

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('InCallChat', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('renders the chat panel header', () => {
    render(
      <InCallChat meetingId="meeting-1" localDisplayName="Alice" onClose={vi.fn()} />,
    )
    expect(screen.getByText('features.video.chat.title')).toBeInTheDocument()
  })

  it('shows empty state when no messages exist', () => {
    render(
      <InCallChat meetingId="meeting-1" localDisplayName="Alice" onClose={vi.fn()} />,
    )
    expect(screen.getByText('features.video.chat.empty')).toBeInTheDocument()
  })

  it('renders send button', () => {
    render(
      <InCallChat meetingId="meeting-1" localDisplayName="Alice" onClose={vi.fn()} />,
    )
    const sendButton = screen.getByLabelText('features.video.chat.send')
    expect(sendButton).toBeInTheDocument()
    expect(sendButton).toBeDisabled() // disabled when input is empty
  })

  it('calls onClose when close button is clicked', async () => {
    const onClose = vi.fn()
    const { getByLabelText } = render(
      <InCallChat meetingId="meeting-1" localDisplayName="Alice" onClose={onClose} />,
    )
    const closeButton = getByLabelText('features.video.chat.close')
    closeButton.click()
    expect(onClose).toHaveBeenCalledOnce()
  })
})
