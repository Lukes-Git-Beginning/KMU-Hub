/**
 * Expenses & transactions (Ausgaben/Transaktionen) — TanStack Query layer.
 *
 * Mock-first: the data still lives in the Zustand finance store (which also backs
 * the deprecated buchhaltung folder, so it must stay), but the finanzen tabs now
 * consume it through TanStack hooks — consistent with the rest of the module and
 * swap-ready. Replace the queryFn/mutationFn bodies with apiClient calls once
 * `/api/v1/finance/expenses` + `/transactions` land (see backend-gaps.md).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import { useFinanceStore, type Expense, type Transaction } from '@/stores/finance'

const EXPENSES_KEY = ['finance', 'expenses'] as const
const TRANSACTIONS_KEY = ['finance', 'transactions'] as const

function delay<T>(value: T): Promise<T> {
  return new Promise((resolve) => setTimeout(() => resolve(value), 100))
}

// --- Reads -----------------------------------------------------------------

export function useExpenses() {
  return useQuery({
    queryKey: EXPENSES_KEY,
    queryFn: () => delay([...useFinanceStore.getState().expenses]),
  })
}

export function useTransactions() {
  return useQuery({
    queryKey: TRANSACTIONS_KEY,
    queryFn: () => delay([...useFinanceStore.getState().transactions]),
  })
}

// --- Expense mutations -----------------------------------------------------

export function useAddExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (expense: Omit<Expense, 'id'>) => {
      useFinanceStore.getState().addExpense(expense)
      return delay(true)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

export function useApproveExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => {
      useFinanceStore.getState().approveExpense(id)
      return delay(true)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

export function useRejectExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => {
      useFinanceStore.getState().rejectExpense(id)
      return delay(true)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

export function useDeleteExpense() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => {
      useFinanceStore.getState().deleteExpense(id)
      return delay(true)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: EXPENSES_KEY }),
  })
}

// --- Transaction mutations -------------------------------------------------

export function useDeleteTransaction() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => {
      useFinanceStore.getState().deleteTransaction(id)
      return delay(true)
    },
    onSuccess: () => qc.invalidateQueries({ queryKey: TRANSACTIONS_KEY }),
  })
}

export type { Expense, Transaction }
