/**
 * TanStack Query hooks for the Integration module (Teams, Slack, future).
 *
 * Query keys follow the pattern ['integration', domain, ...params] for
 * consistent cache invalidation. Mutations use optimistic updates for
 * enable/disable toggles and invalidation for CRUD operations.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import * as integrationClient from '../integration-client'
import type {
  Platform,
  IntegrationConfigListResponse,
  CreateIntegrationConfigRequest,
  UpdateIntegrationConfigRequest,
  CreateChannelMappingRequest,
  UpdateChannelMappingRequest,
  ChannelMappingListResponse,
} from '../integration-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const integrationKeys = {
  all: ['integration'] as const,
  configs: () => ['integration', 'configs'] as const,
  config: (platform: Platform) =>
    ['integration', 'configs', platform] as const,
  mappings: (platform: Platform) =>
    ['integration', 'mappings', platform] as const,
  link: (platform: Platform) =>
    ['integration', 'link', platform] as const,
}

// ---------------------------------------------------------------------------
// Integration config queries
// ---------------------------------------------------------------------------

/** List all integration configs (admin). */
export function useIntegrationConfigs() {
  return useQuery({
    queryKey: integrationKeys.configs(),
    queryFn: () => integrationClient.listConfigs(),
    select: (data) => data.configs,
    staleTime: 30 * 1000,
  })
}

/** Get a single integration config by platform (admin). */
export function useIntegrationConfig(platform: Platform) {
  return useQuery({
    queryKey: integrationKeys.config(platform),
    queryFn: () => integrationClient.getConfig(platform),
    select: (data) => data.config,
    enabled: !!platform,
  })
}

// ---------------------------------------------------------------------------
// Integration config mutations
// ---------------------------------------------------------------------------

/** Create a new integration config. */
export function useCreateIntegrationConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateIntegrationConfigRequest) =>
      integrationClient.createConfig(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.configs() })
      toast.success('Integration konfiguriert')
    },
    onError: () => {
      toast.error('Fehler beim Konfigurieren der Integration')
    },
  })
}

/** Update an existing integration config with optimistic toggle. */
export function useUpdateIntegrationConfig(platform: Platform) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: UpdateIntegrationConfigRequest) =>
      integrationClient.updateConfig(platform, data),
    onMutate: async (data) => {
      if (data.is_active === undefined) return
      await qc.cancelQueries({ queryKey: integrationKeys.configs() })
      // Optimistic update: toggle is_active in list cache
      qc.setQueriesData<IntegrationConfigListResponse>(
        { queryKey: integrationKeys.configs() },
        (old) => {
          if (!old) return old
          return {
            ...old,
            configs: old.configs.map((c) =>
              c.platform === platform
                ? { ...c, is_active: data.is_active! }
                : c,
            ),
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.configs() })
      toast.success('Integration aktualisiert')
    },
    onError: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.configs() })
      toast.error('Fehler beim Aktualisieren der Integration')
    },
  })
}

/** Delete an integration config. */
export function useDeleteIntegrationConfig(platform: Platform) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => integrationClient.deleteConfig(platform),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.configs() })
      toast.success('Integration entfernt')
    },
    onError: () => {
      toast.error('Fehler beim Entfernen der Integration')
    },
  })
}

/** Send a test notification (fire-and-forget). */
export function useTestIntegrationConfig(platform: Platform) {
  return useMutation({
    mutationFn: () => integrationClient.testConfig(platform),
    onSuccess: (data) => {
      if (data.success) {
        toast.success('Testbenachrichtigung gesendet')
      } else {
        toast.error(data.message || 'Test fehlgeschlagen')
      }
    },
    onError: () => {
      toast.error('Fehler beim Senden der Testbenachrichtigung')
    },
  })
}

// ---------------------------------------------------------------------------
// Channel mapping queries
// ---------------------------------------------------------------------------

/** List channel mappings for a platform. */
export function useChannelMappings(platform: Platform) {
  return useQuery({
    queryKey: integrationKeys.mappings(platform),
    queryFn: () => integrationClient.listMappings(platform),
    select: (data) => data.mappings,
    enabled: !!platform,
    staleTime: 30 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Channel mapping mutations
// ---------------------------------------------------------------------------

/** Create a new channel mapping. */
export function useCreateChannelMapping(platform: Platform) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateChannelMappingRequest) =>
      integrationClient.createMapping(platform, data),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: integrationKeys.mappings(platform),
      })
      toast.success('Kanalzuordnung erstellt')
    },
    onError: () => {
      toast.error('Fehler beim Erstellen der Kanalzuordnung')
    },
  })
}

/** Update a channel mapping with optimistic toggle for is_active. */
export function useUpdateChannelMapping(platform: Platform) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      ...data
    }: { id: string } & UpdateChannelMappingRequest) =>
      integrationClient.updateMapping(id, data),
    onMutate: async ({ id, ...data }) => {
      if (data.is_active === undefined) return
      await qc.cancelQueries({
        queryKey: integrationKeys.mappings(platform),
      })
      qc.setQueriesData<ChannelMappingListResponse>(
        { queryKey: integrationKeys.mappings(platform) },
        (old) => {
          if (!old) return old
          return {
            ...old,
            mappings: old.mappings.map((m) =>
              m.id === id ? { ...m, is_active: data.is_active! } : m,
            ),
          }
        },
      )
    },
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: integrationKeys.mappings(platform),
      })
      toast.success('Kanalzuordnung aktualisiert')
    },
    onError: () => {
      qc.invalidateQueries({
        queryKey: integrationKeys.mappings(platform),
      })
      toast.error('Fehler beim Aktualisieren der Kanalzuordnung')
    },
  })
}

/** Delete a channel mapping. */
export function useDeleteChannelMapping(platform: Platform) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => integrationClient.deleteMapping(id),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: integrationKeys.mappings(platform),
      })
      toast.success('Kanalzuordnung gelöscht')
    },
    onError: () => {
      toast.error('Fehler beim Löschen der Kanalzuordnung')
    },
  })
}

// ---------------------------------------------------------------------------
// Account linking queries
// ---------------------------------------------------------------------------

/** Get account link status for a platform. */
export function useAccountLinkStatus(platform: Platform) {
  return useQuery({
    queryKey: integrationKeys.link(platform),
    queryFn: () => integrationClient.getLinkStatus(platform),
    enabled: !!platform,
  })
}

// ---------------------------------------------------------------------------
// Account linking mutations
// ---------------------------------------------------------------------------

/** Link an external account with a token. */
export function useLinkAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: { token: string }) =>
      integrationClient.linkAccount(data),
    onSuccess: (_data) => {
      qc.invalidateQueries({
        queryKey: ['integration', 'link'],
      })
      toast.success('Konto verknüpft')
    },
    onError: () => {
      toast.error('Token ungültig oder abgelaufen')
    },
  })
}

/** Unlink an external account. */
export function useUnlinkAccount(platform: Platform) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => integrationClient.unlinkAccount(platform),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: integrationKeys.link(platform),
      })
      toast.success('Kontoverknüpfung aufgehoben')
    },
    onError: () => {
      toast.error('Fehler beim Aufheben der Verknüpfung')
    },
  })
}
