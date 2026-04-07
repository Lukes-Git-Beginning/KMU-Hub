/**
 * TanStack Query hooks for CalDAV/CardDAV settings and admin management.
 *
 * Query keys: ['caldav', domain, ...params]
 * Mutations invalidate relevant queries on success.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { toast } from 'sonner'
import i18next from 'i18next'
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
      toast.error(i18next.t('api.caldav.error.passwordCreate'))
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
      toast.success(i18next.t('api.caldav.passwordRevoked'))
    },
    onError: () => {
      toast.error(i18next.t('api.caldav.error.passwordRevoke'))
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
      toast.success(i18next.t('api.caldav.enabled'))
    },
    onError: () => {
      toast.error(i18next.t('api.caldav.error.enable'))
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
      toast.success(i18next.t('api.caldav.disabled'))
    },
    onError: () => {
      toast.error(i18next.t('api.caldav.error.disable'))
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
      toast.success(i18next.t('api.caldav.settingsUpdated'))
    },
    onError: () => {
      toast.error(i18next.t('api.caldav.error.settingsUpdate'))
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
      toast.success(i18next.t('api.caldav.allPasswordsRevoked'))
    },
    onError: () => {
      toast.error(i18next.t('api.caldav.error.allPasswordsRevoke'))
    },
  })
}
