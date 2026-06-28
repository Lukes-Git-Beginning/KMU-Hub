/**
 * TanStack Query hooks for the Inventar module.
 *
 * Queries for items, movements, warnings, and stock reports.
 * Mutations for item lifecycle, stock adjustments, transfers,
 * movement recording, and warning management.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  listItems,
  getItem,
  createItem,
  updateItem,
  deleteItem,
  adjustStock,
  transferStock,
  recordMovement,
  listMovements,
  getStockHistory,
  listWarnings,
  createWarning,
  updateWarning,
  acknowledgeWarning,
  getStockReport,
  listLocations,
  getLocation,
  createLocation,
  updateLocation,
  deleteLocation,
  listInventurSessions,
  getInventurSession,
  createInventurSession,
  updateInventurSessionStatus,
  deleteInventurSession,
  upsertInventurCount,
  bookInventurDifferences,
  listItemAttachments,
  createItemAttachment,
  deleteItemAttachment,
} from '../inventar-client'
import { presignUpload } from '../utils/presignUpload'
import type {
  CreateItemInput,
  UpdateItemInput,
  ListItemsParams,
  AdjustStockInput,
  TransferStockInput,
  RecordMovementInput,
  ListMovementsParams,
  CreateWarningInput,
  UpdateWarningInput,
  AcknowledgeWarningInput,
  ListWarningsParams,
  CreateLocationInput,
  UpdateLocationInput,
  CreateInventurSessionInput,
  InventurStatus,
} from '../inventar-types'

// ---------------------------------------------------------------------------
// Query keys
// ---------------------------------------------------------------------------

export const inventarKeys = {
  all: ['inventar'] as const,
  items: (params?: ListItemsParams) => ['inventar', 'items', params] as const,
  item: (id: string) => ['inventar', 'items', id] as const,
  movements: (itemId: string, params?: ListMovementsParams) =>
    ['inventar', 'items', itemId, 'movements', params] as const,
  history: (itemId: string, params?: ListMovementsParams) =>
    ['inventar', 'items', itemId, 'history', params] as const,
  warnings: (params?: ListWarningsParams) => ['inventar', 'warnings', params] as const,
  report: () => ['inventar', 'report'] as const,
  attachments: (itemId: string) => ['inventar', 'items', itemId, 'attachments'] as const,
}

// ---------------------------------------------------------------------------
// Queries — Items
// ---------------------------------------------------------------------------

export function useInventarItems(params?: ListItemsParams) {
  return useQuery({
    queryKey: inventarKeys.items(params),
    queryFn: () => listItems(params),
  })
}

export function useInventarItem(id: string) {
  return useQuery({
    queryKey: inventarKeys.item(id),
    queryFn: () => getItem(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Movements
// ---------------------------------------------------------------------------

export function useInventarMovements(itemId: string, params?: ListMovementsParams) {
  return useQuery({
    queryKey: inventarKeys.movements(itemId, params),
    queryFn: () => listMovements(itemId, params),
    enabled: !!itemId,
  })
}

export function useStockHistory(itemId: string, params?: ListMovementsParams) {
  return useQuery({
    queryKey: inventarKeys.history(itemId, params),
    queryFn: () => getStockHistory(itemId, params),
    enabled: !!itemId,
  })
}

// ---------------------------------------------------------------------------
// Queries — Warnings
// ---------------------------------------------------------------------------

export function useStockWarnings(params?: ListWarningsParams) {
  return useQuery({
    queryKey: inventarKeys.warnings(params),
    queryFn: () => listWarnings(params),
  })
}

// ---------------------------------------------------------------------------
// Queries — Report
// ---------------------------------------------------------------------------

export function useStockReport() {
  return useQuery({
    queryKey: inventarKeys.report(),
    queryFn: getStockReport,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Items
// ---------------------------------------------------------------------------

export function useCreateInventarItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateItemInput) => createItem(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'items'] }),
  })
}

export function useUpdateInventarItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...input }: UpdateItemInput & { id: string }) =>
      updateItem(id, input),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: inventarKeys.item(variables.id) })
      qc.invalidateQueries({ queryKey: ['inventar', 'items'] })
    },
  })
}

export function useDeleteInventarItem() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteItem(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inventar', 'items'] })
      qc.invalidateQueries({ queryKey: inventarKeys.report() })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Stock operations
// ---------------------------------------------------------------------------

export function useAdjustStock() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ itemId, ...body }: AdjustStockInput & { itemId: string }) =>
      adjustStock(itemId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: inventarKeys.item(variables.itemId) })
      qc.invalidateQueries({ queryKey: ['inventar', 'items'] })
      qc.invalidateQueries({ queryKey: inventarKeys.movements(variables.itemId) })
      qc.invalidateQueries({ queryKey: inventarKeys.warnings() })
      qc.invalidateQueries({ queryKey: inventarKeys.report() })
    },
  })
}

export function useTransferStock() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (body: TransferStockInput) => transferStock(body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: inventarKeys.item(variables.from_item_id) })
      qc.invalidateQueries({ queryKey: inventarKeys.item(variables.to_item_id) })
      qc.invalidateQueries({ queryKey: ['inventar', 'items'] })
      qc.invalidateQueries({ queryKey: inventarKeys.warnings() })
      qc.invalidateQueries({ queryKey: inventarKeys.report() })
    },
  })
}

export function useRecordMovement() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ itemId, ...body }: RecordMovementInput & { itemId: string }) =>
      recordMovement(itemId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: inventarKeys.movements(variables.itemId) })
      qc.invalidateQueries({ queryKey: inventarKeys.history(variables.itemId) })
    },
  })
}

// ---------------------------------------------------------------------------
// Mutations — Warnings
// ---------------------------------------------------------------------------

export function useCreateStockWarning() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateWarningInput) => createWarning(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'warnings'] }),
  })
}

export function useUpdateStockWarning() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateWarningInput & { id: string }) =>
      updateWarning(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'warnings'] }),
  })
}

export function useAcknowledgeStockWarning() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: AcknowledgeWarningInput & { id: string }) =>
      acknowledgeWarning(id, body),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inventar', 'warnings'] })
      qc.invalidateQueries({ queryKey: inventarKeys.report() })
    },
  })
}

// ---------------------------------------------------------------------------
// Queries — Locations
// ---------------------------------------------------------------------------

export function useInventarLocations(params?: { page?: number; page_size?: number }) {
  return useQuery({
    queryKey: ['inventar', 'locations', params] as const,
    queryFn: () => listLocations(params),
  })
}

export function useInventarLocation(id: string) {
  return useQuery({
    queryKey: ['inventar', 'locations', id] as const,
    queryFn: () => getLocation(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Queries — Inventur Sessions
// ---------------------------------------------------------------------------

export function useInventurSessions(params?: { page?: number; page_size?: number }) {
  return useQuery({
    queryKey: ['inventar', 'inventur', params] as const,
    queryFn: () => listInventurSessions(params),
  })
}

export function useInventurSession(id: string) {
  return useQuery({
    queryKey: ['inventar', 'inventur', id] as const,
    queryFn: () => getInventurSession(id),
    enabled: !!id,
  })
}

// ---------------------------------------------------------------------------
// Mutations — Locations
// ---------------------------------------------------------------------------

export function useCreateInventarLocation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateLocationInput) => createLocation(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'locations'] }),
  })
}

export function useUpdateInventarLocation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...body }: UpdateLocationInput & { id: string }) => updateLocation(id, body),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'locations'] }),
  })
}

export function useDeleteInventarLocation() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteLocation(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'locations'] }),
  })
}

// ---------------------------------------------------------------------------
// Mutations — Inventur Sessions
// ---------------------------------------------------------------------------

export function useCreateInventurSession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateInventurSessionInput) => createInventurSession(input),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'inventur'] }),
  })
}

export function useUpdateInventurSessionStatus() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, status }: { id: string; status: InventurStatus }) =>
      updateInventurSessionStatus(id, { status }),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'inventur'] }),
  })
}

export function useDeleteInventurSession() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => deleteInventurSession(id),
    onSuccess: () => qc.invalidateQueries({ queryKey: ['inventar', 'inventur'] }),
  })
}

export function useUpsertInventurCount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ sessionId, ...body }: { sessionId: string; item_id: string; counted: number }) =>
      upsertInventurCount(sessionId, body),
    onSuccess: (_data, variables) => {
      qc.invalidateQueries({ queryKey: ['inventar', 'inventur', variables.sessionId] })
    },
  })
}

export function useBookInventurDifferences() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ sessionId, booked_by }: { sessionId: string; booked_by?: string }) =>
      bookInventurDifferences(sessionId, booked_by ? { booked_by } : undefined),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['inventar', 'inventur'] })
      qc.invalidateQueries({ queryKey: ['inventar', 'items'] })
      qc.invalidateQueries({ queryKey: inventarKeys.report() })
    },
  })
}

// ---------------------------------------------------------------------------
// Item Attachments
// ---------------------------------------------------------------------------

export function useItemAttachments(itemId: string) {
  return useQuery({
    queryKey: inventarKeys.attachments(itemId),
    queryFn: () => listItemAttachments(itemId),
    enabled: !!itemId,
  })
}

/** Uploads a file browser-direct (scope=inventar) then registers it on the item. */
export function useUploadItemAttachment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: async ({ itemId, file }: { itemId: string; file: File }) => {
      const objectKey = await presignUpload('inventar', file)
      return createItemAttachment(itemId, {
        name: file.name,
        object_key: objectKey,
        file_type: file.type || 'application/octet-stream',
      })
    },
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: inventarKeys.attachments(variables.itemId) }),
  })
}

export function useDeleteItemAttachment() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id }: { id: string; itemId: string }) => deleteItemAttachment(id),
    onSuccess: (_data, variables) =>
      qc.invalidateQueries({ queryKey: inventarKeys.attachments(variables.itemId) }),
  })
}
