/**
 * TanStack Query hooks for Integration module.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  integrationConfigApi,
  accountLinkApi,
  channelMappingApi,
} from '../integration-client'
import type {
  CreateIntegrationRequest,
  UpdateIntegrationRequest,
  CreateChannelMappingRequest,
} from '../integration-types'

export const integrationKeys = {
  all: ['integrations'] as const,
  configs: () => ['integrations', 'configs'] as const,
  links: () => ['integrations', 'links'] as const,
  mappings: (integrationId: string) =>
    ['integrations', 'mappings', integrationId] as const,
}

// ---------------------------------------------------------------------------
// Config hooks
// ---------------------------------------------------------------------------

export function useIntegrationConfigs() {
  return useQuery({
    queryKey: integrationKeys.configs(),
    queryFn: () => integrationConfigApi.list(),
    select: (data) => data.configs,
  })
}

export function useCreateIntegration() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateIntegrationRequest) =>
      integrationConfigApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.configs() })
    },
  })
}

export function useUpdateIntegration() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateIntegrationRequest }) =>
      integrationConfigApi.update(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.configs() })
    },
  })
}

export function useDeleteIntegration() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => integrationConfigApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.configs() })
    },
  })
}

export function useTestIntegration() {
  return useMutation({
    mutationFn: (id: string) => integrationConfigApi.test(id),
  })
}

// ---------------------------------------------------------------------------
// Account linking hooks
// ---------------------------------------------------------------------------

export function useAccountLinks() {
  return useQuery({
    queryKey: integrationKeys.links(),
    queryFn: () => accountLinkApi.list(),
    select: (data) => data.links,
  })
}

export function useLinkAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ platform, token }: { platform: string; token: string }) =>
      accountLinkApi.link(platform, token),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.links() })
    },
  })
}

export function useUnlinkAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (platform: string) => accountLinkApi.unlink(platform),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: integrationKeys.links() })
    },
  })
}

// ---------------------------------------------------------------------------
// Channel mapping hooks
// ---------------------------------------------------------------------------

export function useChannelMappings(integrationId: string) {
  return useQuery({
    queryKey: integrationKeys.mappings(integrationId),
    queryFn: () => channelMappingApi.list(integrationId),
    enabled: !!integrationId,
    select: (data) => data.mappings,
  })
}

export function useCreateChannelMapping() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      integrationId,
      data,
    }: {
      integrationId: string
      data: CreateChannelMappingRequest
    }) => channelMappingApi.create(integrationId, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: integrationKeys.mappings(vars.integrationId),
      })
    },
  })
}

export function useDeleteChannelMapping() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      integrationId,
      mappingId,
    }: {
      integrationId: string
      mappingId: string
    }) => channelMappingApi.delete(integrationId, mappingId),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: integrationKeys.mappings(vars.integrationId),
      })
    },
  })
}
