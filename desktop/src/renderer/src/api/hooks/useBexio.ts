/**
 * TanStack Query hooks for the Bexio integration module.
 *
 * Query keys follow the pattern ['bexio', domain, ...params] for
 * consistent cache invalidation. Mutations invalidate related queries
 * and show German toast notifications.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import * as bexioClient from '../bexio-client'
import type {
  BexioFieldMappingEntry,
  BexioEntityType,
} from '../bexio-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const bexioKeys = {
  all: ['bexio'] as const,
  connection: () => ['bexio', 'connection'] as const,
  syncStatus: () => ['bexio', 'sync-status'] as const,
  syncLogs: (limit?: number) => ['bexio', 'sync-logs', limit] as const,
  fieldMappings: (entityType: BexioEntityType) =>
    ['bexio', 'field-mappings', entityType] as const,
}

// ---------------------------------------------------------------------------
// Connection queries
// ---------------------------------------------------------------------------

export function useBexioConnectionStatus() {
  return useQuery({
    queryKey: bexioKeys.connection(),
    queryFn: () => bexioClient.getConnectionStatus(),
    staleTime: 30 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Connection mutations
// ---------------------------------------------------------------------------

export function useBexioGetAuthURL() {
  return useMutation({
    mutationFn: () => bexioClient.getAuthorizationURL(),
    onError: () => {
      toast.error('Fehler beim Abrufen der Bexio-Anmelde-URL')
    },
  })
}

export function useBexioDisconnect() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => bexioClient.disconnect(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: bexioKeys.connection() })
      toast.success('Bexio-Verbindung getrennt')
    },
    onError: () => {
      toast.error('Fehler beim Trennen der Bexio-Verbindung')
    },
  })
}

// ---------------------------------------------------------------------------
// Sync queries
// ---------------------------------------------------------------------------

export function useBexioSyncStatus(enabled = true) {
  return useQuery({
    queryKey: bexioKeys.syncStatus(),
    queryFn: () => bexioClient.getSyncStatus(),
    staleTime: 10 * 1000,
    enabled,
  })
}

export function useBexioSyncLogs(limit = 20) {
  return useQuery({
    queryKey: bexioKeys.syncLogs(limit),
    queryFn: () => bexioClient.listSyncLogs(limit),
    select: (data) => data.logs,
    staleTime: 15 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Sync mutations
// ---------------------------------------------------------------------------

export function useBexioTriggerSync() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (syncType?: string) => bexioClient.triggerSync(syncType),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: bexioKeys.syncStatus() })
      toast.success('Synchronisierung gestartet')
    },
    onError: () => {
      toast.error('Fehler beim Starten der Synchronisierung')
    },
  })
}

// ---------------------------------------------------------------------------
// Field mapping queries
// ---------------------------------------------------------------------------

export function useBexioFieldMappings(entityType: BexioEntityType) {
  return useQuery({
    queryKey: bexioKeys.fieldMappings(entityType),
    queryFn: () => bexioClient.getFieldMappings(entityType),
    select: (data) => data.mappings,
    staleTime: 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Field mapping mutations
// ---------------------------------------------------------------------------

export function useBexioUpdateFieldMappings(entityType: BexioEntityType) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (mappings: BexioFieldMappingEntry[]) =>
      bexioClient.updateFieldMappings(entityType, mappings),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: bexioKeys.fieldMappings(entityType),
      })
      toast.success('Feld-Zuordnungen gespeichert')
    },
    onError: () => {
      toast.error('Fehler beim Speichern der Feld-Zuordnungen')
    },
  })
}

// ---------------------------------------------------------------------------
// Manual push mutations
// ---------------------------------------------------------------------------

export function useBexioPushInvoice() {
  return useMutation({
    mutationFn: (invoiceId: string) => bexioClient.pushInvoice(invoiceId),
    onSuccess: (data) => {
      toast.success(`Rechnung an Bexio gesendet (ID: ${data.bexio_id})`)
    },
    onError: () => {
      toast.error('Fehler beim Senden der Rechnung an Bexio')
    },
  })
}

export function useBexioPushQuote() {
  return useMutation({
    mutationFn: (quoteId: string) => bexioClient.pushQuote(quoteId),
    onSuccess: (data) => {
      toast.success(`Offerte an Bexio gesendet (ID: ${data.bexio_id})`)
    },
    onError: () => {
      toast.error('Fehler beim Senden der Offerte an Bexio')
    },
  })
}
