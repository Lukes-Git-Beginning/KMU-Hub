/**
 * TanStack Query hooks for per-contact GDPR consent.
 *
 * The backend stores one record per consent_type and returns them as a map
 * (consent_type -> latest record), not as a list -- these hooks unwrap that
 * shape once here so no component has to know it. Query keys are nested under
 * the contact so invalidating a contact does not needlessly refetch consents.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { apiClient } from '../client'

/**
 * The consent_type values the backend accepts. Mirrors ValidConsentTypes in
 * backend/internal/crm/consent/postgres_repository.go -- anything else is
 * rejected with 400. marketing_email and marketing_phone are the two the
 * consent asserter enforces on SendEmail and DialerCall.
 */
export const CONSENT_TYPES = [
  'marketing_email',
  'marketing_phone',
  'newsletter',
  'profiling',
  'data_processing',
  'data_sharing',
] as const

export type ConsentType = (typeof CONSENT_TYPES)[number]

/** The legal bases the backend accepts (ValidLegalBases). */
export type LegalBasis = 'consent' | 'legitimate_interest' | 'contract' | 'legal_obligation'

export interface ConsentRecord {
  id?: string
  contact_id?: string
  consent_type?: string
  granted?: boolean
  legal_basis?: string
  source?: string
  ip_address?: string | null
  notes?: string
  granted_at?: string | null
  revoked_at?: string | null
  created_by?: string | null
  created_at?: string
}

const consentsKey = (contactId: string) => ['contacts', contactId, 'consents'] as const

/** Latest consent record per type, keyed by consent_type. */
export function useContactConsents(contactId: string) {
  return useQuery({
    queryKey: consentsKey(contactId),
    queryFn: async () => {
      const { data, error } = await apiClient.GET('/api/v1/contacts/{id}/consents', {
        params: { path: { id: contactId } },
      })
      if (error) throw error
      return (data?.consents ?? {}) as Record<string, ConsentRecord>
    },
    enabled: !!contactId,
  })
}

/** Grant/revoke audit trail for one consent type. Only fetched when expanded. */
export function useConsentHistory(contactId: string, consentType: string, enabled: boolean) {
  return useQuery({
    queryKey: [...consentsKey(contactId), consentType, 'history'],
    queryFn: async () => {
      const { data, error } = await apiClient.GET(
        '/api/v1/contacts/{id}/consents/{type}/history',
        { params: { path: { id: contactId, type: consentType } } },
      )
      if (error) throw error
      return (data?.history ?? []) as ConsentRecord[]
    },
    enabled: enabled && !!contactId && !!consentType,
  })
}

export interface GrantConsentInput {
  consentType: ConsentType | string
  source: string
  legalBasis?: LegalBasis
}

export function useGrantConsent(contactId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ consentType, source, legalBasis }: GrantConsentInput) => {
      const { data, error } = await apiClient.POST('/api/v1/contacts/{id}/consents', {
        params: { path: { id: contactId } },
        body: {
          consent_type: consentType,
          source,
          // lean: a consent recorded by hand in the CRM is always basis
          // "consent"; expose a picker when a module needs to record one of
          // the other three bases (contract, legal_obligation, ...).
          legal_basis: legalBasis ?? 'consent',
        },
      })
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: consentsKey(contactId) })
    },
  })
}

export function useRevokeConsent(contactId: string) {
  const queryClient = useQueryClient()
  return useMutation({
    mutationFn: async ({ consentType, notes }: { consentType: string; notes?: string }) => {
      const { data, error } = await apiClient.DELETE(
        '/api/v1/contacts/{id}/consents/{type}',
        {
          params: { path: { id: contactId, type: consentType } },
          // The handler decodes a body even on DELETE, so send one.
          body: { notes: notes ?? '' },
        },
      )
      if (error) throw error
      return data
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: consentsKey(contactId) })
    },
  })
}
