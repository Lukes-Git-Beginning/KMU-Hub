/**
 * CalDAV/CardDAV settings API client.
 *
 * Fetch wrapper for app-specific password management and admin CalDAV settings.
 * Follows the calendar-client.ts pattern: typed fetch with auth header injection
 * and 401 retry.
 *
 * Gateway routes: /api/v1/caldav/* and /api/v1/admin/caldav/*
 */
import { authenticatedRequest } from './utils/authenticatedFetch'

// ---------------------------------------------------------------------------
// Types
// ---------------------------------------------------------------------------

export interface AppPassword {
  id: string
  label: string
  password_prefix: string
  last_used_at: string | null
  created_at: string
  revoked_at: string | null
}

export interface CreatePasswordResponse {
  password: string
  id: string
  label: string
  prefix: string
}

export interface CalDAVStatus {
  org_enabled: boolean
  user_enabled: boolean
  password_count: number
  user_id: string
  caldav_url: string
  carddav_url: string
}

export interface AdminCalDAVSettings {
  enabled: boolean
}

export interface AdminCalDAVUser {
  id: string
  email: string
  first_name: string
  last_name: string
  caldav_enabled: boolean
  password_count: number
  last_used: string | null
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

function request<T>(opts: { method: string; path: string; body?: unknown }): Promise<T> {
  return authenticatedRequest<T>(opts)
}

// ---------------------------------------------------------------------------
// User API
// ---------------------------------------------------------------------------

export function getAppPasswords() {
  return request<{ passwords: AppPassword[] }>({
    method: 'GET',
    path: '/api/v1/caldav/passwords',
  })
}

export function createAppPassword(label: string) {
  return request<CreatePasswordResponse>({
    method: 'POST',
    path: '/api/v1/caldav/passwords',
    body: { label },
  })
}

export function revokeAppPassword(id: string) {
  return request<{ status: string }>({
    method: 'DELETE',
    path: `/api/v1/caldav/passwords/${id}`,
  })
}

export function getCalDAVStatus() {
  return request<CalDAVStatus>({
    method: 'GET',
    path: '/api/v1/caldav/status',
  })
}

export function enableCalDAV() {
  return request<{ status: string }>({
    method: 'PUT',
    path: '/api/v1/caldav/enable',
  })
}

export function disableCalDAV() {
  return request<{ status: string }>({
    method: 'PUT',
    path: '/api/v1/caldav/disable',
  })
}

export interface CalDAVTestResult {
  success: boolean
  message?: string
  caldav_reachable: boolean
  carddav_reachable: boolean
}

export function testCalDAVConnection() {
  return request<CalDAVTestResult>({
    method: 'POST',
    path: '/api/v1/caldav/test',
  })
}

// ---------------------------------------------------------------------------
// Admin API
// ---------------------------------------------------------------------------

export function getAdminCalDAVSettings() {
  return request<AdminCalDAVSettings>({
    method: 'GET',
    path: '/api/v1/admin/caldav/settings',
  })
}

export function setAdminCalDAVSettings(enabled: boolean) {
  return request<AdminCalDAVSettings>({
    method: 'PUT',
    path: '/api/v1/admin/caldav/settings',
    body: { enabled },
  })
}

export function getAdminCalDAVUsers() {
  return request<{ users: AdminCalDAVUser[] }>({
    method: 'GET',
    path: '/api/v1/admin/caldav/users',
  })
}

export function adminRevokeUserPasswords(userId: string) {
  return request<{ revoked_count: number }>({
    method: 'DELETE',
    path: `/api/v1/admin/caldav/users/${userId}/passwords`,
  })
}
