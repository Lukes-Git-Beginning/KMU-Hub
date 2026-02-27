/**
 * TanStack Query hooks for the DATEV upload integration module.
 *
 * Query keys follow the pattern ['datevUpload', domain, ...params] for
 * consistent cache invalidation. Mutations invalidate related queries
 * and show German toast notifications.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import * as datevUploadClient from '../datev-upload-client'
import type { DatevUploadConfig } from '../datev-upload-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const datevUploadKeys = {
  all: ['datevUpload'] as const,
  connection: () => ['datevUpload', 'connection'] as const,
  config: () => ['datevUpload', 'config'] as const,
  logs: (limit?: number) => ['datevUpload', 'logs', limit] as const,
}

// ---------------------------------------------------------------------------
// Connection queries
// ---------------------------------------------------------------------------

export function useDatevConnectionStatus() {
  return useQuery({
    queryKey: datevUploadKeys.connection(),
    queryFn: () => datevUploadClient.getConnectionStatus(),
    refetchInterval: 30_000,
  })
}

// ---------------------------------------------------------------------------
// Connection mutations
// ---------------------------------------------------------------------------

export function useDatevGetAuthURL() {
  return useMutation({
    mutationFn: (redirectUrl: string) => datevUploadClient.getAuthorizationURL(redirectUrl),
    onError: () => {
      toast.error('Fehler beim Abrufen der DATEV-Anmelde-URL')
    },
  })
}

export function useDatevDisconnect() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => datevUploadClient.disconnect(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: datevUploadKeys.all })
      toast.success('DATEV-Verbindung getrennt')
    },
    onError: () => {
      toast.error('Fehler beim Trennen der DATEV-Verbindung')
    },
  })
}

// ---------------------------------------------------------------------------
// Config queries
// ---------------------------------------------------------------------------

export function useDatevUploadConfig() {
  return useQuery({
    queryKey: datevUploadKeys.config(),
    queryFn: () => datevUploadClient.getUploadConfig(),
    staleTime: 60 * 1000,
  })
}

// ---------------------------------------------------------------------------
// Config mutations
// ---------------------------------------------------------------------------

export function useDatevUpdateConfig() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (config: Partial<DatevUploadConfig>) =>
      datevUploadClient.updateUploadConfig(config),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: datevUploadKeys.config() })
      toast.success('DATEV-Konfiguration gespeichert')
    },
    onError: () => {
      toast.error('Fehler beim Speichern der DATEV-Konfiguration')
    },
  })
}

// ---------------------------------------------------------------------------
// Upload mutations
// ---------------------------------------------------------------------------

export function useDatevUploadBuchungsstapel() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ startDate, endDate }: { startDate: string; endDate: string }) =>
      datevUploadClient.uploadBuchungsstapel(startDate, endDate),
    onSuccess: (data) => {
      if (data.success) {
        toast.success('Buchungsstapel an DATEV hochgeladen')
      } else {
        toast.error(data.error_message || 'Upload fehlgeschlagen')
      }
      qc.invalidateQueries({ queryKey: datevUploadKeys.logs() })
    },
    onError: () => {
      toast.error('Fehler beim Hochladen des Buchungsstapels')
    },
  })
}

export function useDatevUploadBeleg() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (invoiceId: string) => datevUploadClient.uploadBeleg(invoiceId),
    onSuccess: (data) => {
      if (data.success) {
        toast.success('Beleg an DATEV hochgeladen')
      } else {
        toast.error(data.error_message || 'Beleg-Upload fehlgeschlagen')
      }
      qc.invalidateQueries({ queryKey: datevUploadKeys.logs() })
    },
    onError: () => {
      toast.error('Fehler beim Hochladen des Belegs')
    },
  })
}

// ---------------------------------------------------------------------------
// Log queries
// ---------------------------------------------------------------------------

export function useDatevUploadLogs(limit = 20) {
  return useQuery({
    queryKey: datevUploadKeys.logs(limit),
    queryFn: () => datevUploadClient.listUploadLogs(limit),
    refetchInterval: 15_000,
  })
}
