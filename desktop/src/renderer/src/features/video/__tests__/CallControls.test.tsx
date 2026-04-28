/**
 * Tests for CallControls — verifies that the RecordingInitiatorDialog (pre-consent)
 * is shown when the record button is clicked, and that confirmInitiatorConsent is
 * called before startRecording completes.
 */
import { describe, it, expect, vi, beforeEach } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'

// ---------------------------------------------------------------------------
// Mock LiveKit hooks — they require a live Room context in production
// ---------------------------------------------------------------------------
vi.mock('@livekit/components-react', () => ({
  useLocalParticipant: () => ({
    localParticipant: {
      setMicrophoneEnabled: vi.fn(),
      setCameraEnabled: vi.fn(),
      setScreenShareEnabled: vi.fn(),
    },
    isMicrophoneEnabled: true,
    isCameraEnabled: true,
    isScreenShareEnabled: false,
  }),
  useRoomContext: () => ({ disconnect: vi.fn() }),
  useIsRecording: () => false,
}))

// ---------------------------------------------------------------------------
// Mock API hooks
// ---------------------------------------------------------------------------

const mockStartRecordingMutateAsync = vi.fn()
const mockConfirmInitiatorConsentMutateAsync = vi.fn()
const mockStopRecordingMutateAsync = vi.fn()
const mockEndCallMutateAsync = vi.fn()

vi.mock('@/api/hooks/useVideo', () => ({
  useStartRecording: () => ({
    mutateAsync: mockStartRecordingMutateAsync,
    isPending: false,
  }),
  useStopRecording: () => ({
    mutateAsync: mockStopRecordingMutateAsync,
    isPending: false,
  }),
  useEndCall: () => ({
    mutateAsync: mockEndCallMutateAsync,
    isPending: false,
  }),
  useConfirmInitiatorConsent: () => ({
    mutateAsync: mockConfirmInitiatorConsentMutateAsync,
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

// ---------------------------------------------------------------------------
// Mock Radix AlertDialog — render children directly in tests
// ---------------------------------------------------------------------------

vi.mock('@/components/ui/alert-dialog', () => ({
  AlertDialog: ({ children, open }: { children: React.ReactNode; open: boolean }) =>
    open ? <div data-testid="alert-dialog">{children}</div> : null,
  AlertDialogContent: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogHeader: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogTitle: ({ children }: { children: React.ReactNode }) => <h2>{children}</h2>,
  AlertDialogDescription: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
  AlertDialogFooter: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

vi.mock('@/components/ui/button', () => ({
  Button: ({ children, onClick, disabled }: { children: React.ReactNode; onClick?: () => void; disabled?: boolean }) => (
    <button onClick={onClick} disabled={disabled}>{children}</button>
  ),
}))

vi.mock('@/lib', () => ({
  cn: (...args: unknown[]) => args.filter(Boolean).join(' '),
}))

// ---------------------------------------------------------------------------
// Test subject
// ---------------------------------------------------------------------------

import { CallControls } from '../CallControls'

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

function renderControls(callId = 'call-123') {
  return render(<CallControls callId={callId} />)
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe('CallControls — pre-recording consent dialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    mockStartRecordingMutateAsync.mockResolvedValue({ id: 'rec-abc' })
    mockConfirmInitiatorConsentMutateAsync.mockResolvedValue({ stamped: true })
  })

  it('shows the initiator pre-dialog when record button is clicked', async () => {
    renderControls()

    const recordButton = screen.getByTitle('Aufnahme starten')
    await userEvent.click(recordButton)

    await waitFor(() => {
      expect(screen.getByTestId('alert-dialog')).toBeInTheDocument()
    })
  })

  it('calls startRecording before showing the pre-dialog', async () => {
    renderControls()

    const recordButton = screen.getByTitle('Aufnahme starten')
    await userEvent.click(recordButton)

    expect(mockStartRecordingMutateAsync).toHaveBeenCalledWith('call-123')
  })

  it('calls confirmInitiatorConsent with the recording ID when confirmed', async () => {
    renderControls()

    const recordButton = screen.getByTitle('Aufnahme starten')
    await userEvent.click(recordButton)

    await waitFor(() => screen.getByTestId('alert-dialog'))

    // Click confirm button (key = features.video.recordingInitiator.confirm)
    const confirmButton = screen.getByText('features.video.recordingInitiator.confirm')
    await userEvent.click(confirmButton)

    await waitFor(() => {
      expect(mockConfirmInitiatorConsentMutateAsync).toHaveBeenCalledWith('rec-abc')
    })
  })

  it('does not call confirmInitiatorConsent when cancel is clicked', async () => {
    renderControls()

    const recordButton = screen.getByTitle('Aufnahme starten')
    await userEvent.click(recordButton)

    await waitFor(() => screen.getByTestId('alert-dialog'))

    const cancelButton = screen.getByText('features.video.recordingInitiator.cancel')
    await userEvent.click(cancelButton)

    expect(mockConfirmInitiatorConsentMutateAsync).not.toHaveBeenCalled()
  })
})
