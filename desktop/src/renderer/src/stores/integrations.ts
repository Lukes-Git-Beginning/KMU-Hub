import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type IntegrationConnectionStatus =
  | 'connected'
  | 'disconnected'
  | 'syncing'
  | 'error'

export interface IntegrationState {
  status: IntegrationConnectionStatus
  connectedAt?: string
  lastSync?: string
  fieldValues: Record<string, string | boolean>
}

interface IntegrationStoreState {
  integrations: Record<string, IntegrationState>

  getStatus: (id: string) => IntegrationConnectionStatus
  getFieldValues: (id: string) => Record<string, string | boolean>
  connect: (id: string) => void
  disconnect: (id: string) => void
  setFieldValues: (id: string, values: Record<string, string | boolean>) => void
  setStatus: (id: string, status: IntegrationConnectionStatus) => void
  triggerSync: (id: string) => void
}

export const useIntegrationStore = create<IntegrationStoreState>()(
  persist(
    (set, get) => ({
      integrations: {},

      getStatus: (id: string) => {
        return get().integrations[id]?.status ?? 'disconnected'
      },

      getFieldValues: (id: string) => {
        return get().integrations[id]?.fieldValues ?? {}
      },

      connect: (id: string) => {
        set((state) => ({
          integrations: {
            ...state.integrations,
            [id]: {
              ...state.integrations[id],
              status: 'connected',
              connectedAt: new Date().toISOString(),
              fieldValues: state.integrations[id]?.fieldValues ?? {},
            },
          },
        }))
      },

      disconnect: (id: string) => {
        set((state) => ({
          integrations: {
            ...state.integrations,
            [id]: {
              ...state.integrations[id],
              status: 'disconnected',
              connectedAt: undefined,
              lastSync: undefined,
              fieldValues: state.integrations[id]?.fieldValues ?? {},
            },
          },
        }))
      },

      setFieldValues: (id: string, values: Record<string, string | boolean>) => {
        set((state) => ({
          integrations: {
            ...state.integrations,
            [id]: {
              ...state.integrations[id],
              status: state.integrations[id]?.status ?? 'disconnected',
              fieldValues: {
                ...(state.integrations[id]?.fieldValues ?? {}),
                ...values,
              },
            },
          },
        }))
      },

      setStatus: (id: string, status: IntegrationConnectionStatus) => {
        set((state) => ({
          integrations: {
            ...state.integrations,
            [id]: {
              ...state.integrations[id],
              status,
              fieldValues: state.integrations[id]?.fieldValues ?? {},
            },
          },
        }))
      },

      triggerSync: (id: string) => {
        const { setStatus } = get()
        setStatus(id, 'syncing')
        setTimeout(() => {
          set((state) => ({
            integrations: {
              ...state.integrations,
              [id]: {
                ...state.integrations[id],
                status: 'connected',
                lastSync: new Date().toISOString(),
                fieldValues: state.integrations[id]?.fieldValues ?? {},
              },
            },
          }))
        }, 1500)
      },
    }),
    { name: 'kmuhub-integrations' },
  ),
)
