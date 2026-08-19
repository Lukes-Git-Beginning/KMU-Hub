/**
 * TanStack Query hook for GDPR right-to-erasure requests on a contact.
 *
 * Mirrors useDeleteContact (useContacts.ts) rather than the useConsents.ts
 * binding style: this fires from the contact list/detail flow where the
 * target contact varies per call, not from a panel mounted for one fixed
 * contact.
 *
 * Filing the request only creates a record with status "pending" -- it does
 * NOT anonymize anything. The actual erasure runs separately and requires the
 * admin role (POST /api/v1/gdpr/deletion-requests/{id}/process, route_crm_ext.go
 * HandleProcessDeletion). This hook intentionally stops at the request step:
 * unlike the GDPR export flow (useGDPRExports in useSecurity.ts), there is no
 * list endpoint for pending deletion requests to build an admin review queue
 * around, so wiring up execution here would either skip review entirely or
 * require backend work that is out of scope for this change.
 */
import { useMutation } from '@tanstack/react-query'
import { apiClient } from '../client'

export interface ContactErasureRequest {
  id?: string
  contact_id?: string
  status?: 'pending' | 'processing' | 'completed'
  reason?: string
  created_at?: string
}

export function useRequestContactErasure() {
  return useMutation({
    mutationFn: async (contactId: string) => {
      const { data, error } = await apiClient.POST(
        '/api/v1/contacts/{id}/gdpr/deletion-request',
        {
          params: { path: { id: contactId } },
          // The handler decodes a body unconditionally (see useConsents.ts
          // useRevokeConsent), so send one even without a reason.
          body: { reason: '' },
        },
      )
      if (error) throw new Error(error.error || 'request failed')
      return data as ContactErasureRequest
    },
  })
}
