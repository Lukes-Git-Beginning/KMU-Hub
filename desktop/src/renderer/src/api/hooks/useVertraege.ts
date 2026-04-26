/**
 * TanStack Query hooks for the Vertraege (contracts) module.
 *
 * Queries for contracts, parties, and reminders.
 * Mutations for the full contract lifecycle, party management,
 * and reminder scheduling.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listContracts,
  getContract,
  createContract,
  updateContract,
  deleteContract,
  uploadDocument,
  listParties,
  addParty,
  removeParty,
  listReminders,
  createReminder,
  updateReminder,
  deleteReminder,
} from '../vertraege-client'
import type {
  CreateContractInput,
  UpdateContractInput,
  ListContractsParams,
  AddPartyInput,
  CreateReminderInput,
  UpdateReminderInput,
} from '../vertraege-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const vertraegeKeys = {
  all: ['vertraege'] as const,
  contracts: (params?: ListContractsParams) => ['vertraege', 'contracts', params] as const,
  contract: (id: string) => ['vertraege', 'contracts', id] as const,
  parties: (contractId: string) => ['vertraege', 'contracts', contractId, 'parties'] as const,
  reminders: (contractId: string, onlyPending?: boolean) =>
    ['vertraege', 'contracts', contractId, 'reminders', { onlyPending }] as const,
}

// ---------------------------------------------------------------------------
// Queries — Contracts
// ---------------------------------------------------------------------------

export function useContracts(params?: ListContractsParams) {
  return useQuery({
    queryKey: vertraegeKeys.contracts(params),
    queryFn: () => listContracts(params),
  })
}

export function useContract(id: string) {
  return useQuery({
    queryKey: vertraegeKeys.contract(id),
    queryFn: () => getContract(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Parties
// ---------------------------------------------------------------------------

export function useParties(contractId: string) {
  return useQuery({
    queryKey: vertraegeKeys.parties(contractId),
    queryFn: () => listParties(contractId),
    enabled: !!contractId,
  })
}

// ---------------------------------------------------------------------------
// Queries — Reminders
// ---------------------------------------------------------------------------

export function useReminders(contractId: string, onlyPending?: boolean) {
  return useQuery({
    queryKey: vertraegeKeys.reminders(contractId, onlyPending),
    queryFn: () => listReminders(contractId, onlyPending),
    enabled: !!contractId,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Contracts
// ---------------------------------------------------------------------------

export function useCreateContract() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateContractInput) => createContract(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vertraege', 'contracts'] }),
  })
}

export function useUpdateContract() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateContractInput & { id: string }) =>
      updateContract(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: vertraegeKeys.contract(variables.id) })
      qc.invalidateQueries({ queryKey: ['vertraege', 'contracts'] })
    },
  })
}

export function useDeleteContract() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteContract(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['vertraege', 'contracts'] }),
  })
}

export function useUploadDocument() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (contractId: string) => uploadDocument(contractId),
    onSuccess: (_data, contractId) =>
      qc.invalidateQueries({ queryKey: vertraegeKeys.contract(contractId) }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Parties
// ---------------------------------------------------------------------------

export function useAddParty() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contractId, ...input }: AddPartyInput & { contractId: string }) =>
      addParty(contractId, input),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: vertraegeKeys.parties(variables.contractId) }),
  })
}

export function useRemoveParty() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contractId, partyId }: { contractId: string; partyId: string }) =>
      removeParty(contractId, partyId),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: vertraegeKeys.parties(variables.contractId) }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Reminders
// ---------------------------------------------------------------------------

export function useCreateReminder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      contractId,
      ...input
    }: CreateReminderInput & { contractId: string }) => createReminder(contractId, input),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: vertraegeKeys.reminders(variables.contractId) }),
  })
}

export function useUpdateReminder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      contractId,
      reminderId,
      ...input
    }: UpdateReminderInput & { contractId: string; reminderId: string }) =>
      updateReminder(contractId, reminderId, input),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: vertraegeKeys.reminders(variables.contractId) }),
  })
}

export function useDeleteReminder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contractId, reminderId }: { contractId: string; reminderId: string }) =>
      deleteReminder(contractId, reminderId),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: vertraegeKeys.reminders(variables.contractId) }),
  })
}
