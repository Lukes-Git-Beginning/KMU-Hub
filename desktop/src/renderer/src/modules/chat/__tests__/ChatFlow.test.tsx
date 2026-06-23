import { describe, it, expect, vi, beforeEach, beforeAll } from 'vitest'
import { render, screen, waitFor } from '@testing-library/react'
import userEvent from '@testing-library/user-event'
import { QueryClient, QueryClientProvider } from '@tanstack/react-query'
import { MemoryRouter } from 'react-router-dom'
import { TooltipProvider } from '@/components/ui/tooltip'
import ChatLayout from '../ChatLayout'
import { ChannelList } from '../channels/ChannelList'
import { CreateChannelDialog } from '../channels/CreateChannelDialog'
import { MessageInput } from '../messages/MessageInput'

// Polyfill ResizeObserver for jsdom
beforeAll(() => {
  if (typeof globalThis.ResizeObserver === 'undefined') {
    globalThis.ResizeObserver = class {
      observe() {}
      unobserve() {}
      disconnect() {}
    } as unknown as typeof ResizeObserver
  }
})

// Mock sonner toast
vi.mock('sonner', () => ({
  toast: { success: vi.fn(), error: vi.fn(), info: vi.fn() },
}))

// Mock presence store
vi.mock('@/stores/presence', () => ({
  usePresenceStore: (selector: (s: Record<string, unknown>) => unknown) =>
    selector({ presenceMap: { 'usr-002': 'online' } }),
}))

// Mock channel hooks at hook level
const mockChannelsData = {
  channels: [
    { id: 'ch-001', name: 'allgemein', is_dm: false, is_private: false },
    { id: 'ch-002', name: 'development', is_dm: false, is_private: false },
  ],
}

const mockDMsData = {
  channels: [
    { id: 'dm-001', name: 'Anna Beispiel', is_dm: true, other_user_id: 'usr-002' },
  ],
}

const mockUnreadData = {
  unread_counts: { 'ch-001': 3, 'dm-001': 1 },
}

const mockCreateChannelMutateAsync = vi.fn().mockResolvedValue({
  channel: { id: 'ch-new', name: 'neuer-channel' },
})

vi.mock('@/api/hooks/useChannels', () => ({
  useChannels: () => ({ data: mockChannelsData, isLoading: false, error: null }),
  useDMs: () => ({ data: mockDMsData, isLoading: false, error: null }),
  useUnreadCounts: () => ({ data: mockUnreadData, isLoading: false, error: null }),
  useCreateChannel: () => ({
    mutateAsync: mockCreateChannelMutateAsync,
    isPending: false,
  }),
  useStartConversation: () => ({
    mutateAsync: vi.fn().mockResolvedValue({ channel: { id: 'dm-new' } }),
    isPending: false,
  }),
}))

// Mock message hooks
const mockSendMessageMutateAsync = vi.fn().mockResolvedValue({ id: 'msg-new' })

vi.mock('@/api/hooks/useMessages', () => ({
  useSendMessage: () => ({
    mutateAsync: mockSendMessageMutateAsync,
    isPending: false,
  }),
  useSendTypingIndicator: () => vi.fn(),
}))

// Mock MentionAutocomplete (complex sub-component)
vi.mock('../messages/MentionAutocomplete', () => ({
  MentionAutocomplete: () => null,
}))

// Mock FileDropZone to just render children
vi.mock('../messages/FileDropZone', () => ({
  FileDropZone: ({ children }: { children: React.ReactNode }) => <div>{children}</div>,
}))

// Mock MessageList (depends on infinite query)
vi.mock('../messages/MessageList', () => ({
  MessageList: ({ channelId }: { channelId: string }) => (
    <div data-testid="message-list">Messages for {channelId}</div>
  ),
}))

// Mock ChannelHeader
vi.mock('../channels/ChannelHeader', () => ({
  ChannelHeader: ({ channelId }: { channelId: string }) => (
    <div data-testid="channel-header">Header: {channelId}</div>
  ),
}))

// Mock ThreadPanel
vi.mock('../threads/ThreadPanel', () => ({
  ThreadPanel: () => <div data-testid="thread-panel">Thread</div>,
}))

function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  })
}

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = createQueryClient()
  return render(
    <QueryClientProvider client={queryClient}>
      <TooltipProvider>
        <MemoryRouter>{ui}</MemoryRouter>
      </TooltipProvider>
    </QueryClientProvider>,
  )
}

// ---------------------------------------------------------------------------
// ChatLayout
// ---------------------------------------------------------------------------

describe('ChatLayout', () => {
  it('renders empty state when no channel selected', () => {
    renderWithProviders(<ChatLayout />)

    expect(screen.getByRole('heading', { level: 2 })).toHaveTextContent('chat.empty.title')
  })

  it('shows channel list sidebar', () => {
    renderWithProviders(<ChatLayout />)

    expect(screen.getByText('chat.channels.title')).toBeInTheDocument()
  })

  it('shows message area when channel is selected', async () => {
    const user = userEvent.setup()
    renderWithProviders(<ChatLayout />)

    await user.click(screen.getByText('allgemein'))

    await waitFor(() => {
      expect(screen.getByTestId('message-list')).toBeInTheDocument()
    })
  })
})

// ---------------------------------------------------------------------------
// ChannelList
// ---------------------------------------------------------------------------

describe('ChannelList', () => {
  let onSelectChannel: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onSelectChannel = vi.fn()
  })

  it('renders channels from hook data', () => {
    renderWithProviders(
      <ChannelList selectedChannelId={null} onSelectChannel={onSelectChannel} />,
    )

    expect(screen.getByText('allgemein')).toBeInTheDocument()
    expect(screen.getByText('development')).toBeInTheDocument()
  })

  it('renders DMs', () => {
    renderWithProviders(
      <ChannelList selectedChannelId={null} onSelectChannel={onSelectChannel} />,
    )

    expect(screen.getByText('Anna Beispiel')).toBeInTheDocument()
  })

  it('shows unread badges', () => {
    renderWithProviders(
      <ChannelList selectedChannelId={null} onSelectChannel={onSelectChannel} />,
    )

    expect(screen.getByText('3')).toBeInTheDocument()
    expect(screen.getByText('1')).toBeInTheDocument()
  })

  it('filters channels by search', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <ChannelList selectedChannelId={null} onSelectChannel={onSelectChannel} />,
    )

    await user.type(screen.getByPlaceholderText('chat.channels.searchPlaceholder'), 'dev')

    await waitFor(() => {
      expect(screen.getByText('development')).toBeInTheDocument()
      expect(screen.queryByText('allgemein')).not.toBeInTheDocument()
    })
  })
})

// ---------------------------------------------------------------------------
// CreateChannelDialog
// ---------------------------------------------------------------------------

describe('CreateChannelDialog', () => {
  let onOpenChange: ReturnType<typeof vi.fn>
  let onCreated: ReturnType<typeof vi.fn>

  beforeEach(() => {
    onOpenChange = vi.fn()
    onCreated = vi.fn()
    mockCreateChannelMutateAsync.mockClear()
  })

  it('renders create channel form', () => {
    renderWithProviders(
      <CreateChannelDialog open={true} onOpenChange={onOpenChange} onCreated={onCreated} />,
    )

    expect(screen.getByText('chat.createChannel.title')).toBeInTheDocument()
    expect(screen.getByPlaceholderText('chat.createChannel.namePlaceholder')).toBeInTheDocument()
  })

  it('submits channel creation', async () => {
    const user = userEvent.setup()
    renderWithProviders(
      <CreateChannelDialog open={true} onOpenChange={onOpenChange} onCreated={onCreated} />,
    )

    await user.type(
      screen.getByPlaceholderText('chat.createChannel.namePlaceholder'),
      'neuer-channel',
    )
    await user.click(screen.getByRole('button', { name: 'common.create' }))

    await waitFor(() => {
      expect(onCreated).toHaveBeenCalledWith('ch-new')
    })
  })

  it('disables submit without channel name', () => {
    renderWithProviders(
      <CreateChannelDialog open={true} onOpenChange={onOpenChange} onCreated={onCreated} />,
    )

    const submitBtn = screen.getByRole('button', { name: 'common.create' })
    expect(submitBtn).toBeDisabled()
  })
})

// ---------------------------------------------------------------------------
// MessageInput
// ---------------------------------------------------------------------------

describe('MessageInput', () => {
  it('renders with placeholder text', () => {
    renderWithProviders(<MessageInput channelId="ch-001" />)

    expect(screen.getByPlaceholderText('chat.input.placeholder')).toBeInTheDocument()
  })

  it('sends message on Enter', async () => {
    const user = userEvent.setup()
    renderWithProviders(<MessageInput channelId="ch-001" />)

    const textarea = screen.getByPlaceholderText('chat.input.placeholder')
    await user.type(textarea, 'Hallo Welt!')
    await user.keyboard('{Enter}')

    await waitFor(() => {
      expect(mockSendMessageMutateAsync).toHaveBeenCalledWith({
        channelId: 'ch-001',
        content: 'Hallo Welt!',
        parentMessageId: undefined,
      })
    })
  })
})
