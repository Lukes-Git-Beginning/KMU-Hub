/**
 * TanStack Query hooks for the Lexware integration module.
 *
 * Query keys follow the pattern ['lexware', domain, ...params] for
 * consistent cache invalidation. Mutations invalidate related queries
 * and show German toast notifications.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import * as lexwareClient from '../lexware-client'
import type { LexwareFieldMappingEntry } from '../lexware-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const lexwareKeys = {
  all: ['lexware'] as const,
  connection: () => ['lexware', 'connection'] as const,
  syncStatus: () => ['lexware', 'sync-status'] as const,
  syncLogs: (limit?: number) => ['lexware', 'sync-logs', limit] as const,
  fieldMappings: (entityType: string) =>
    ['lexware', 'field-mappings', entityType] as const,
}

// ---------------------------------------------------------------------------
// Connection queries
// ---------------------------------------------------------------------------

export function useLexwareConnectionStatus() {
  return useQuery({
    queryKey: lexwareKeys.connection(),
    queryFn: () => lexwareClient.getConnectionStatus(),
    refetchInterval: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Connection mutations
// ---------------------------------------------------------------------------

export function useLexwareConnect() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (apiKey: string) => lexwareClient.connect(apiKey),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: lexwareKeys.connection() })
      toast.success('Lexware-Verbindung hergestellt')
    },
    onError: () => {
      toast.error('Fehler beim Verbinden mit Lexware')
    },
  })
}

export function useLexwareDisconnect() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => lexwareClient.disconnect(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: lexwareKeys.all })
      toast.success('Lexware-Verbindung getrennt')
    },
    onError: () => {
      toast.error('Fehler beim Trennen der Lexware-Verbindung')
    },
  })
}

export function useLexwareTestConnection() {
  return useMutation({
    mutationFn: () => lexwareClient.testConnection(),
    onSuccess: (data) => {
      if (data.success) {
        toast.success('Lexware-Verbindung erfolgreich getestet')
      } else {
        toast.error(data.error_message || 'Verbindungstest fehlgeschlagen')
      }
    },
    onError: () => {
      toast.error('Fehler beim Testen der Lexware-Verbindung')
    },
  })
}

// ---------------------------------------------------------------------------
// Sync queries
// ---------------------------------------------------------------------------

export function useLexwareSyncStatus(enabled = true) {
  return useQuery({
    queryKey: lexwareKeys.syncStatus(),
    queryFn: () => lexwareClient.getSyncStatus(),
    refetchInterval: 10_000,
    enabled,
  })
}

export function useLexwareSyncLogs(limit = 20) {
  return useQuery({
    queryKey: lexwareKeys.syncLogs(limit),
    queryFn: () => lexwareClient.listSyncLogs(limit),
    refetchInterval: 15_000,
  })
}

// ---------------------------------------------------------------------------
// Sync mutations
// ---------------------------------------------------------------------------

export function useLexwareTriggerSync() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (syncType?: string) => lexwareClient.triggerSync(syncType),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: lexwareKeys.syncStatus() })
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

export function useLexwareFieldMappings(entityType: string) {
  return useQuery({
    queryKey: lexwareKeys.fieldMappings(entityType),
    queryFn: () => lexwareClient.getFieldMappings(entityType),
    staleTime: 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Field mapping mutations
// ---------------------------------------------------------------------------

export function useLexwareUpdateFieldMappings(entityType: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (mappings: LexwareFieldMappingEntry[]) =>
      lexwareClient.updateFieldMappings(entityType, mappings),
    onSuccess: () => {
      qc.invalidateQueries({
        queryKey: lexwareKeys.fieldMappings(entityType),
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

export function useLexwarePushInvoice() {
  return useMutation({
    mutationFn: (invoiceId: string) => lexwareClient.pushInvoice(invoiceId),
    onSuccess: (data) => {
      if (data.success) {
        toast.success(`Rechnung an Lexware gesendet${data.lexware_id ? ` (ID: ${data.lexware_id})` : ''}`)
      } else {
        toast.error(data.error_message || 'Fehler beim Senden der Rechnung')
      }
    },
    onError: () => {
      toast.error('Fehler beim Senden der Rechnung an Lexware')
    },
  })
}

export function useLexwarePushQuote() {
  return useMutation({
    mutationFn: (quoteId: string) => lexwareClient.pushQuote(quoteId),
    onSuccess: (data) => {
      if (data.success) {
        toast.success(`Offerte an Lexware gesendet${data.lexware_id ? ` (ID: ${data.lexware_id})` : ''}`)
      } else {
        toast.error(data.error_message || 'Fehler beim Senden der Offerte')
      }
    },
    onError: () => {
      toast.error('Fehler beim Senden der Offerte an Lexware')
    },
  })
}
