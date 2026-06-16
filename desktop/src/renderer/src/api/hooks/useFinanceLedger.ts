/**
 * Expenses & transactions (Ausgaben/Transaktionen) — TanStack Query layer.
 *
 * finanzen P2: now backed by MSW handlers (`/api/v1/finance/expenses` +
 * `/transactions`) through `financeExpenseApi` / `financeTransactionApi`. The
 * mock state is stateful, so approve/reject/create/Kontierung/Beleg-Upload
 * survive within a session. Swap to the real backend by pointing the API client
 * at live routes — no hook changes needed (see backend-gaps.md §finanzen).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  financeExpenseApi,
  financeTransactionApi,
  type CreateExpenseRequest,
  type UpdateExpenseRequest,
} from '@/api/finance-client'
import type { Expense, Transaction } from '@/stores/finance'

const EXPENSES_KEY = ['finance', 'expenses'] as const
const TRANSACTIONS_KEY = ['finance', 'transactions'] as const

// --- Reads -----------------------------------------------------------------

export function useExpenses() {
  return useQuery({
    queryKey: EXPENSES_KEY,
    queryFn: async () => (await financeExpenseApi.list()).expenses,
  })
}

export function useTransactions() {
  return useQuery({
    queryKey: TRANSACTIONS_KEY,
    queryFn: async () => (await financeTransactionApi.list()).transactions,
  })
}

// --- Expense mutations -----------------------------------------------------

export function useAddExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (expense: CreateExpenseRequest) => financeExpenseApi.create(expense),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

export function useUpdateExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, data }: { id: string; data: UpdateExpenseRequest }) =>
      financeExpenseApi.update(id, data),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

export function useApproveExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeExpenseApi.approve(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

export function useRejectExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeExpenseApi.reject(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

export function useAttachReceipt() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, receiptName }: { id: string; receiptName: string }) =>
      financeExpenseApi.attachReceipt(id, receiptName),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

export function useDeleteExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeExpenseApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

// --- Transaction mutations -------------------------------------------------

export function useDeleteTransaction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => financeTransactionApi.delete(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: TRANSACTIONS_KEY }),
  })
}

export type { Expense, Transaction }
