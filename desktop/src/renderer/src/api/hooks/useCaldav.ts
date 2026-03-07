/**
 * TanStack Query hooks for CalDAV/CardDAV settings and admin management.
 *
 * Query keys: ['caldav', domain, ...params]
 * Mutations invalidate relevant queries on success.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import * as caldavClient from '../caldav-client'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const caldavKeys = {
  all: ['caldav'] as const,
  passwords: () => ['caldav', 'passwords'] as const,
  status: () => ['caldav', 'status'] as const,
  admin: {
    settings: () => ['caldav', 'admin', 'settings'] as const,
    users: () => ['caldav', 'admin', 'users'] as const,
  },
}

// ---------------------------------------------------------------------------
// User hooks
// ---------------------------------------------------------------------------

/** List app-specific passwords for the current user. */
export function useAppPasswords() {
  return useQuery({
    queryKey: caldavKeys.passwords(),
    queryFn: () => caldavClient.getAppPasswords(),
    select: (data) => data.passwords,
  })
}

/** Get CalDAV/CardDAV status for the current user. */
export function useCalDAVStatus() {
  return useQuery({
    queryKey: caldavKeys.status(),
    queryFn: () => caldavClient.getCalDAVStatus(),
  })
}

/** Create a new app-specific password. */
export function useCreateAppPassword() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (label: string) => caldavClient.createAppPassword(label),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: caldavKeys.passwords() })
      qc.invalidateQueries({ queryKey: caldavKeys.status() })
    },
    onError: () => {
      toast.error('Fehler beim Erstellen des Passworts')
    },
  })
}

/** Revoke an app-specific password. */
export function useRevokeAppPassword() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => caldavClient.revokeAppPassword(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: caldavKeys.passwords() })
      qc.invalidateQueries({ queryKey: caldavKeys.status() })
      toast.success('Passwort widerrufen')
    },
    onError: () => {
      toast.error('Fehler beim Widerrufen des Passworts')
    },
  })
}

/** Enable CalDAV for the current user. */
export function useEnableCalDAV() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => caldavClient.enableCalDAV(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: caldavKeys.status() })
      toast.success('CalDAV/CardDAV aktiviert')
    },
    onError: () => {
      toast.error('Fehler beim Aktivieren')
    },
  })
}

/** Test CalDAV/CardDAV connection. */
export function useTestCalDAVConnection() {
  return useMutation({
    mutationFn: () => caldavClient.testCalDAVConnection(),
  })
}

/** Disable CalDAV for the current user. */
export function useDisableCalDAV() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => caldavClient.disableCalDAV(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: caldavKeys.status() })
      toast.success('CalDAV/CardDAV deaktiviert')
    },
    onError: () => {
      toast.error('Fehler beim Deaktivieren')
    },
  })
}

// ---------------------------------------------------------------------------
// Admin hooks
// ---------------------------------------------------------------------------

/** Get org-wide CalDAV/CardDAV settings (admin only). */
export function useAdminCalDAVSettings() {
  return useQuery({
    queryKey: caldavKeys.admin.settings(),
    queryFn: () => caldavClient.getAdminCalDAVSettings(),
  })
}

/** Set org-wide CalDAV/CardDAV enabled/disabled (admin only). */
export function useSetAdminCalDAVSettings() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (enabled: boolean) =>
      caldavClient.setAdminCalDAVSettings(enabled),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: caldavKeys.admin.settings() })
      qc.invalidateQueries({ queryKey: caldavKeys.status() })
      toast.success('CalDAV-Einstellungen aktualisiert')
    },
    onError: () => {
      toast.error('Fehler beim Aktualisieren der Einstellungen')
    },
  })
}

/** List users with CalDAV enabled (admin only). */
export function useAdminCalDAVUsers() {
  return useQuery({
    queryKey: caldavKeys.admin.users(),
    queryFn: () => caldavClient.getAdminCalDAVUsers(),
    select: (data) => data.users,
  })
}

/** Admin revoke all passwords for a user. */
export function useAdminRevokeUserPasswords() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (userId: string) =>
      caldavClient.adminRevokeUserPasswords(userId),
    onSuccess: (_data, _userId) => {
      qc.invalidateQueries({ queryKey: caldavKeys.admin.users() })
      toast.success('Alle Passwoerter des Benutzers widerrufen')
    },
    onError: () => {
      toast.error('Fehler beim Widerrufen der Passwoerter')
    },
  })
}
