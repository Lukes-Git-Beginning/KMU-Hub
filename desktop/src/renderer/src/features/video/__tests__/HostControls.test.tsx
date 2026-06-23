/**
 * Tests for HostControls -- Wave 3 moderation UI.
 *
 * Verifies:
 * 1. ParticipantActions renders mute/kick buttons for non-local participants.
 * 2. Co-host toggle shows for organizer, hidden for regular host.
 * 3. GlobalHostControls renders mute-all and lock/unlock buttons.
 * 4. MeetingLockIndicator shows only when locked.
 * 5. 403 errors display the correct i18n key toast.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// ---------------------------------------------------------------------------
// Mock API hooks
// ---------------------------------------------------------------------------

const mockMutateAsync = vi.fn()

vi.mock('@/api/hooks/useVideo', () => ({
  useMuteParticipant: (_meetingId: string) => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
  useKickParticipant: (_meetingId: string) => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
  usePromoteCoHost: (_meetingId: string) => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
  useDemoteCoHost: (_meetingId: string) => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
  useMuteAllParticipants: (_meetingId: string) => ({
    mutateAsync: mockMutateAsync,
    isPending: false,
  }),
  useSetMeetingLock: (_meetingId: string) => ({
    mutateAsync: mockMutateAsync,
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
  toast: { success: vi.fn(), error: vi.fn() },
}))

// ---------------------------------------------------------------------------
// Test subjects
// ---------------------------------------------------------------------------

import { ParticipantActions, GlobalHostControls, MeetingLockIndicator } from '../HostControls'
import { toast } from 'sonner'

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('ParticipantActions', () => {
  const defaultProps = {
    participantIdentity: 'user-abc',
    meetingId: 'meeting-1',
    isOrganizer: false,
    isCoHost: false,
  }

  beforeEach(() => {
    vi.clearAllMocks()
    mockMutateAsync.mockResolvedValue(undefined)
  })

  it('renders mute and kick buttons', () => {
    render(<ParticipantActions {...defaultProps} />)
    expect(
      screen.getByTitle('features.video.hostControls.muteParticipant'),
    ).toBeInTheDocument()
    expect(
      screen.getByTitle('features.video.hostControls.kickParticipant'),
    ).toBeInTheDocument()
  })

  it('does NOT render co-host button when isOrganizer=false', () => {
    render(<ParticipantActions {...defaultProps} isOrganizer={false} />)
    expect(
      screen.queryByTitle('features.video.hostControls.makeCoHost'),
    ).not.toBeInTheDocument()
  })

  it('renders make-co-host button when isOrganizer=true and not yet co-host', () => {
    render(<ParticipantActions {...defaultProps} isOrganizer={true} isCoHost={false} />)
    expect(
      screen.getByTitle('features.video.hostControls.makeCoHost'),
    ).toBeInTheDocument()
  })

  it('renders remove-co-host button when isOrganizer=true and already co-host', () => {
    render(<ParticipantActions {...defaultProps} isOrganizer={true} isCoHost={true} />)
    expect(
      screen.getByTitle('features.video.hostControls.removeCoHost'),
    ).toBeInTheDocument()
  })

  it('calls mutateAsync with participantIdentity when mute button is clicked', async () => {
    render(<ParticipantActions {...defaultProps} />)
    await userEvent.click(
      screen.getByTitle('features.video.hostControls.muteParticipant'),
    )
    expect(mockMutateAsync).toHaveBeenCalledWith('user-abc')
  })

  it('calls mutateAsync with participantIdentity when kick button is clicked', async () => {
    render(<ParticipantActions {...defaultProps} />)
    await userEvent.click(
      screen.getByTitle('features.video.hostControls.kickParticipant'),
    )
    expect(mockMutateAsync).toHaveBeenCalledWith('user-abc')
  })

  it('shows error403 toast on 403 response', async () => {
    mockMutateAsync.mockRejectedValueOnce({ status: 403 })
    render(<ParticipantActions {...defaultProps} />)
    await userEvent.click(
      screen.getByTitle('features.video.hostControls.muteParticipant'),
    )
    await waitFor(() => {
      expect(toast.error).toHaveBeenCalledWith(
        'features.video.hostControls.error403',
      )
    })
  })
})

describe('GlobalHostControls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockMutateAsync.mockResolvedValue(undefined)
  })

  it('renders mute-all and lock buttons when not locked', () => {
    render(<GlobalHostControls meetingId="meeting-1" isLocked={false} />)
    expect(
      screen.getByTitle('features.video.hostControls.muteAll'),
    ).toBeInTheDocument()
    expect(
      screen.getByTitle('features.video.hostControls.lock'),
    ).toBeInTheDocument()
  })

  it('renders unlock button when locked', () => {
    render(<GlobalHostControls meetingId="meeting-1" isLocked={true} />)
    expect(
      screen.getByTitle('features.video.hostControls.unlock'),
    ).toBeInTheDocument()
  })

  it('calls mutateAsync when mute-all is clicked', async () => {
    render(<GlobalHostControls meetingId="meeting-1" isLocked={false} />)
    await userEvent.click(
      screen.getByTitle('features.video.hostControls.muteAll'),
    )
    expect(mockMutateAsync).toHaveBeenCalledWith()
  })

  it('calls mutateAsync(true) when lock button is clicked', async () => {
    render(<GlobalHostControls meetingId="meeting-1" isLocked={false} />)
    await userEvent.click(
      screen.getByTitle('features.video.hostControls.lock'),
    )
    expect(mockMutateAsync).toHaveBeenCalledWith(true)
  })

  it('calls mutateAsync(false) when unlock button is clicked', async () => {
    render(<GlobalHostControls meetingId="meeting-1" isLocked={true} />)
    await userEvent.click(
      screen.getByTitle('features.video.hostControls.unlock'),
    )
    expect(mockMutateAsync).toHaveBeenCalledWith(false)
  })
})

describe('MeetingLockIndicator', () => {
  it('renders nothing when not locked', () => {
    const { container } = render(<MeetingLockIndicator isLocked={false} />)
    expect(container.firstChild).toBeNull()
  })

  it('renders lock badge when locked', () => {
    render(<MeetingLockIndicator isLocked={true} />)
    expect(
      screen.getByTitle('features.video.hostControls.locked'),
    ).toBeInTheDocument()
  })
})