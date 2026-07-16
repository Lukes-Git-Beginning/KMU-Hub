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
  cancelPO,
  receiveGoods,
  partialReceive,
  listPOLines,
  addPOLine,
  updatePOLine,
  deletePOLine,
  listCatalogItems,
  getCatalogItem,
  createCatalogItem,
  updateCatalogItem,
  deleteCatalogItem,
  listSupplierRatings,
  createSupplierRating,
  deleteSupplierRating,
  listFrameworkContracts,
  getFrameworkContract,
  createFrameworkContract,
  updateFrameworkContract,
  deleteFrameworkContract,
  createContractItem,
  updateContractItem,
  deleteContractItem,
  createContractCall,
  listContractCalls,
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
  ListCatalogItemsParams,
  CreateCatalogItemInput,
  UpdateCatalogItemInput,
  CreateSupplierRatingInput,
  ListFrameworkContractsParams,
  CreateFrameworkContractInput,
  UpdateFrameworkContractInput,
  CreateContractItemInput,
  UpdateContractItemInput,
  CreateContractCallInput,
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
  catalogItems: (params?: ListCatalogItemsParams) => ['einkauf', 'catalog', params] as const,
  catalogItem: (id: string) => ['einkauf', 'catalog', id] as const,
  supplierRatings: (supplierId: string) => ['einkauf', 'suppliers', supplierId, 'ratings'] as const,
  contracts: (params?: ListFrameworkContractsParams) => ['einkauf', 'contracts', params] as const,
  contract: (id: string) => ['einkauf', 'contracts', id] as const,
  contractCalls: (contractId: string) => ['einkauf', 'contracts', contractId, 'calls'] as const,
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

export function useCancelPO() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => cancelPO(id),
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

// Line mutations also invalidate the PO list/detail — the header total_amount
// is derived from the lines (freshly created orders showed 0,00 € otherwise).
export function useAddPOLine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ poId, ...body }: AddPOLineInput & { poId: string }) =>
      addPOLine(poId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.poLines(variables.poId) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] })
    },
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
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.poLines(variables.poId) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] })
    },
  })
}

export function useDeletePOLine() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ poId, lineId }: { poId: string; lineId: string }) =>
      deletePOLine(poId, lineId),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.poLines(variables.poId) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'pos'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Queries + Mutations — Catalog Items
// ---------------------------------------------------------------------------

export function useCatalogItems(params?: ListCatalogItemsParams) {
  return useQuery({
    queryKey: einkaufKeys.catalogItems(params),
    queryFn: () => listCatalogItems(params),
  })
}

export function useCatalogItem(id: string) {
  return useQuery({
    queryKey: einkaufKeys.catalogItem(id),
    queryFn: () => getCatalogItem(id),
    enabled: !!id,
  })
}

export function useCreateCatalogItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateCatalogItemInput) => createCatalogItem(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['einkauf', 'catalog'] }),
  })
}

export function useUpdateCatalogItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateCatalogItemInput & { id: string }) =>
      updateCatalogItem(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.catalogItem(variables.id) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'catalog'] })
    },
  })
}

export function useDeleteCatalogItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteCatalogItem(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['einkauf', 'catalog'] }),
  })
}

// ---------------------------------------------------------------------------
// Queries + Mutations — Supplier Ratings
// ---------------------------------------------------------------------------

export function useSupplierRatings(supplierId: string) {
  return useQuery({
    queryKey: einkaufKeys.supplierRatings(supplierId),
    queryFn: () => listSupplierRatings(supplierId),
    enabled: !!supplierId,
  })
}

export function useCreateSupplierRating() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ supplierId, ...body }: Omit<CreateSupplierRatingInput, 'supplier_id'> & { supplierId: string }) =>
      createSupplierRating(supplierId, body),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: einkaufKeys.supplierRatings(variables.supplierId) }),
  })
}

export function useDeleteSupplierRating() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ supplierId, ratingId }: { supplierId: string; ratingId: string }) =>
      deleteSupplierRating(supplierId, ratingId),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: einkaufKeys.supplierRatings(variables.supplierId) }),
  })
}

// ---------------------------------------------------------------------------
// Queries + Mutations — Framework Contracts
// ---------------------------------------------------------------------------

export function useFrameworkContracts(params?: ListFrameworkContractsParams) {
  return useQuery({
    queryKey: einkaufKeys.contracts(params),
    queryFn: () => listFrameworkContracts(params),
  })
}

export function useFrameworkContract(id: string) {
  return useQuery({
    queryKey: einkaufKeys.contract(id),
    queryFn: () => getFrameworkContract(id),
    enabled: !!id,
  })
}

export function useCreateFrameworkContract() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateFrameworkContractInput) => createFrameworkContract(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['einkauf', 'contracts'] }),
  })
}

export function useUpdateFrameworkContract() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateFrameworkContractInput & { id: string }) =>
      updateFrameworkContract(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.contract(variables.id) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'contracts'] })
    },
  })
}

export function useDeleteFrameworkContract() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteFrameworkContract(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['einkauf', 'contracts'] }),
  })
}

export function useCreateContractItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contractId, ...body }: Omit<CreateContractItemInput, 'contract_id'> & { contractId: string }) =>
      createContractItem(contractId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.contract(variables.contractId) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'contracts'] })
    },
  })
}

export function useUpdateContractItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contractId, itemId, ...body }: UpdateContractItemInput & { contractId: string; itemId: string }) =>
      updateContractItem(contractId, itemId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.contract(variables.contractId) })
    },
  })
}

export function useDeleteContractItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contractId, itemId }: { contractId: string; itemId: string }) =>
      deleteContractItem(contractId, itemId),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.contract(variables.contractId) })
    },
  })
}

export function useContractCalls(contractId: string) {
  return useQuery({
    queryKey: einkaufKeys.contractCalls(contractId),
    queryFn: () => listContractCalls(contractId),
    enabled: !!contractId,
  })
}

export function useCreateContractCall() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ contractId, ...body }: Omit<CreateContractCallInput, 'contract_id'> & { contractId: string }) =>
      createContractCall(contractId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: einkaufKeys.contractCalls(variables.contractId) })
      qc.invalidateQueries({ queryKey: einkaufKeys.contract(variables.contractId) })
      qc.invalidateQueries({ queryKey: ['einkauf', 'contracts'] })
    },
  })
}
