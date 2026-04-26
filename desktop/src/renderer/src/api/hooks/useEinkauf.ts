/**
 * TanStack Query hooks for the Einkauf (purchasing) module.
 *
 * Queries for suppliers, purchase orders, and PO lines.
 * Mutations for full CRUD + workflow actions (submit, receive, partial-receive).
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listSuppliers,
  getSupplier,
  createSupplier,
  updateSupplier,
  deleteSupplier,
  listPOs,
  getPO,
  createPO,
  updatePO,
  deletePO,
  submitPO,
  receiveGoods,
  partialReceive,
  listPOLines,
  addPOLine,
  updatePOLine,
  deletePOLine,
} from '../einkauf-client'
import type {
  CreateSupplierInput,
  UpdateSupplierInput,
  ListSuppliersParams,
  CreatePOInput,
  UpdatePOInput,
  ListPOsParams,
  AddPOLineInput,
  UpdatePOLineInput,
  PartialReceiveInput,
} from '../einkauf-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const einkaufKeys = {
  all: ['einkauf'] as const,
  suppliers: (params?: ListSuppliersParams) => ['einkauf', 'suppliers', params] as const,
  supplier: (id: string) => ['einkauf', 'suppliers', id] as const,
  pos: (params?: ListPOsParams) => ['einkauf', 'pos', params] as const,
  po: (id: string) => ['einkauf', 'pos', id] as const,
  poLines: (poId: string) => ['einkauf', 'pos', poId, 'lines'] as const,
}

// ---------------------------------------------------------------------------
// Queries — Suppliers
// ---------------------------------------------------------------------------

export function useSuppliers(params?: ListSuppliersParams) {
  return useQuery({
    queryKey: einkaufKeys.suppliers(params),
    queryFn: () => listSuppliers(params),
  })
}

export function useSupplier(id: string) {
  return useQuery({
    queryKey: einkaufKeys.supplier(id),
    queryFn: () => getSupplier(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Purchase Orders
// ---------------------------------------------------------------------------

export function usePOs(params?: ListPOsParams) {
  return useQuery({
    queryKey: einkaufKeys.pos(params),
    queryFn: () => listPOs(params),
  })
}

export function usePO(id: string) {
  return useQuery({
    queryKey: einkaufKeys.po(id),
    queryFn: () => getPO(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — PO Lines
// ---------------------------------------------------------------------------

export function usePOLines(poId: string) {
  return useQuery({
    queryKey: einkaufKeys.poLines(poId),
    queryFn: () => listPOLines(poId),
    enabled: !!poId,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Suppliers
// ---------------------------------------------------------------------------

export function useCreateSupplier() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateSupplierInput) => createSupplier(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['einkauf', 'suppliers'] }),
  })
}

export function useUpdateSupplier() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateSupplierInput & { id: string }) =>
      updateSupplier(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.supplier(variables.id) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'suppliers'] })
    },
  })
}

export function useDeleteSupplier() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteSupplier(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['einkauf', 'suppliers'] }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Purchase Orders
// ---------------------------------------------------------------------------

export function useCreatePO() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreatePOInput) => createPO(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] }),
  })
}

export function useUpdatePO() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdatePOInput & { id: string }) => updatePO(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.po(variables.id) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] })
    },
  })
}

export function useDeletePO() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deletePO(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] }),
  })
}

export function useSubmitPO() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => submitPO(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.po(id) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] })
    },
  })
}

export function useReceiveGoods() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => receiveGoods(id),
    onSuccess: (_data, id) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.po(id) })
      qc.invalidateQueries({ queryKey: einkaufKeys.poLines(id) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] })
    },
  })
}

export function usePartialReceive() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ poId, ...body }: PartialReceiveInput & { poId: string }) =>
      partialReceive(poId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.po(variables.poId) })
      qc.invalidateQueries({ queryKey: einkaufKeys.poLines(variables.poId) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — PO Lines
// ---------------------------------------------------------------------------

export function useAddPOLine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ poId, ...body }: AddPOLineInput & { poId: string }) =>
      addPOLine(poId, body),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: einkaufKeys.poLines(variables.poId) }),
  })
}

export function useUpdatePOLine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      poId,
      lineId,
      ...body
    }: UpdatePOLineInput & { poId: string; lineId: string }) =>
      updatePOLine(poId, lineId, body),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: einkaufKeys.poLines(variables.poId) }),
  })
}

export function useDeletePOLine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ poId, lineId }: { poId: string; lineId: string }) =>
      deletePOLine(poId, lineId),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: einkaufKeys.poLines(variables.poId) }),
  })
}
