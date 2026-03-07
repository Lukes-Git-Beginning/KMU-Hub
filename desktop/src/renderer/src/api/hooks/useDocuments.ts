/**
 * TanStack Query hooks for Document Management operations.
 *
 * Query keys follow the pattern ['documents', domain, ...params] for consistent
 * cache invalidation. Mutations invalidate related queries automatically.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  documentFolderApi,
  documentFileApi,
  documentVersionApi,
  documentShareApi,
  documentTagApi,
  documentLinkApi,
  documentSearchApi,
  documentWopiApi,
} from '../clients/document-client'
import type {
  ListFilesParams,
  ListFoldersParams,
  CreateFolderRequest,
  UpdateFolderRequest,
  UpdateFileRequest,
  ShareEntityRequest,
  CreateTagRequest,
  LinkFileRequest,
  SearchFilesParams,
  DocumentFile,
} from '../types/document-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const documentKeys = {
  all: ['documents'] as const,
  folders: (params?: ListFoldersParams) => ['documents', 'folders', params] as const,
  folder: (id: string) => ['documents', 'folder', id] as const,
  folderPath: (id: string) => ['documents', 'folderPath', id] as const,
  files: (params?: ListFilesParams) => ['documents', 'files', params] as const,
  file: (id: string) => ['documents', 'file', id] as const,
  fileDownloadURL: (id: string) => ['documents', 'fileDownloadURL', id] as const,
  versions: (fileId: string) => ['documents', 'versions', fileId] as const,
  shares: (entityType: string, entityId: string) =>
    ['documents', 'shares', entityType, entityId] as const,
  sharedWithMe: (entityType?: string) =>
    ['documents', 'sharedWithMe', entityType] as const,
  tags: () => ['documents', 'tags'] as const,
  fileLinks: (fileId: string) => ['documents', 'fileLinks', fileId] as const,
  search: (query: string, params?: SearchFilesParams) =>
    ['documents', 'search', query, params] as const,
  virtualFiles: (sourceType?: string) =>
    ['documents', 'virtualFiles', sourceType] as const,
}

// ---------------------------------------------------------------------------
// Folder hooks
// ---------------------------------------------------------------------------

export function useDocumentFolders(params?: ListFoldersParams) {
  return useQuery({
    queryKey: documentKeys.folders(params),
    queryFn: () => documentFolderApi.list(params),
  })
}

export function useDocumentFolder(id: string) {
  return useQuery({
    queryKey: documentKeys.folder(id),
    queryFn: () => documentFolderApi.get(id),
    enabled: !!id,
  })
}

export function useFolderPath(id: string) {
  return useQuery({
    queryKey: documentKeys.folderPath(id),
    queryFn: () => documentFolderApi.getPath(id),
    enabled: !!id,
  })
}

export function useCreateFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateFolderRequest) => documentFolderApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
    },
  })
}

export function useUpdateFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: UpdateFolderRequest & { id: string }) =>
      documentFolderApi.update(id, data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
      qc.invalidateQueries({ queryKey: ['documents', 'folder'] })
      qc.invalidateQueries({ queryKey: ['documents', 'folderPath'] })
    },
  })
}

export function useDeleteFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => documentFolderApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
    },
  })
}

export function useInitializeUserSpace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: () => documentFolderApi.initializeUserSpace(),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
    },
  })
}

export function useInitializeTeamSpace() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ teamId, teamName }: { teamId: string; teamName: string }) =>
      documentFolderApi.initializeTeamSpace(teamId, teamName),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
    },
  })
}

// ---------------------------------------------------------------------------
// File hooks
// ---------------------------------------------------------------------------

export function useDocumentFiles(params?: ListFilesParams) {
  return useQuery({
    queryKey: documentKeys.files(params),
    queryFn: () => documentFileApi.list(params),
  })
}

export function useDocumentFile(id: string) {
  return useQuery({
    queryKey: documentKeys.file(id),
    queryFn: () => documentFileApi.get(id),
    enabled: !!id,
  })
}

export function useUpdateFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...data }: UpdateFileRequest & { id: string }) =>
      documentFileApi.update(id, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
      qc.invalidateQueries({ queryKey: documentKeys.file(vars.id) })
    },
    // Optimistic update for favorite toggle
    onMutate: async (vars) => {
      if (vars.is_favorite === undefined) return
      await qc.cancelQueries({ queryKey: ['documents', 'files'] })
      const previousFiles = qc.getQueriesData({ queryKey: ['documents', 'files'] })
      qc.setQueriesData(
        { queryKey: ['documents', 'files'] },
        (old: { files: DocumentFile[]; total: number } | undefined) => {
          if (!old) return old
          return {
            ...old,
            files: old.files.map((f) =>
              f.id === vars.id ? { ...f, is_favorite: vars.is_favorite! } : f,
            ),
          }
        },
      )
      return { previousFiles }
    },
    onError: (_err, _vars, context) => {
      if (context?.previousFiles) {
        for (const [key, data] of context.previousFiles) {
          qc.setQueryData(key, data)
        }
      }
    },
  })
}

export function useDeleteFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => documentFileApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
    },
  })
}

export function useCopyFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      targetFolderId,
    }: {
      id: string
      targetFolderId: string
    }) => documentFileApi.copy(id, targetFolderId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
    },
  })
}

export function useMoveFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      id,
      targetFolderId,
    }: {
      id: string
      targetFolderId: string
    }) => documentFileApi.move(id, targetFolderId),
    onSuccess: () => {
      // Invalidate source and target folder queries
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
      qc.invalidateQueries({ queryKey: ['documents', 'folders'] })
    },
  })
}

export function useFileDownloadURL(id: string) {
  return useQuery({
    queryKey: documentKeys.fileDownloadURL(id),
    queryFn: () => documentFileApi.getDownloadURL(id),
    enabled: !!id,
    staleTime: 60_000, // URLs expire, keep fresh
    gcTime: 120_000,
  })
}

// ---------------------------------------------------------------------------
// Version hooks
// ---------------------------------------------------------------------------

export function useFileVersions(fileId: string) {
  return useQuery({
    queryKey: documentKeys.versions(fileId),
    queryFn: () => documentVersionApi.list(fileId),
    enabled: !!fileId,
  })
}

export function useCreateVersion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      fileId,
      versionLabel,
    }: {
      fileId: string
      versionLabel?: string
    }) => documentVersionApi.create(fileId, { version_label: versionLabel }),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: documentKeys.versions(vars.fileId),
      })
      qc.invalidateQueries({
        queryKey: documentKeys.file(vars.fileId),
      })
    },
  })
}

export function useRevertVersion() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      fileId,
      versionNumber,
    }: {
      fileId: string
      versionNumber: number
    }) => documentVersionApi.revert(fileId, versionNumber),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: documentKeys.versions(vars.fileId),
      })
      qc.invalidateQueries({
        queryKey: documentKeys.file(vars.fileId),
      })
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Share hooks
// ---------------------------------------------------------------------------

export function useShares(entityType: string, entityId: string) {
  return useQuery({
    queryKey: documentKeys.shares(entityType, entityId),
    queryFn: () => documentShareApi.list(entityType, entityId),
    enabled: !!entityType && !!entityId,
  })
}

export function useSharedWithMe(entityType?: string) {
  return useQuery({
    queryKey: documentKeys.sharedWithMe(entityType),
    queryFn: () => documentShareApi.listSharedWithMe(entityType),
  })
}

export function useShareEntity() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: ShareEntityRequest) => documentShareApi.share(data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: documentKeys.shares(vars.entity_type, vars.entity_id),
      })
      qc.invalidateQueries({ queryKey: ['documents', 'sharedWithMe'] })
    },
  })
}

export function useUnshareEntity() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (shareId: string) => documentShareApi.unshare(shareId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'shares'] })
      qc.invalidateQueries({ queryKey: ['documents', 'sharedWithMe'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Tag hooks
// ---------------------------------------------------------------------------

export function useDocumentTags() {
  return useQuery({
    queryKey: documentKeys.tags(),
    queryFn: () => documentTagApi.list(),
  })
}

export function useCreateTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (data: CreateTagRequest) => documentTagApi.create(data),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: documentKeys.tags() })
    },
  })
}

export function useDeleteTag() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => documentTagApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: documentKeys.tags() })
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
    },
  })
}

export function useTagFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fileId, tagId }: { fileId: string; tagId: string }) =>
      documentTagApi.tagFile(fileId, tagId),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
      qc.invalidateQueries({ queryKey: documentKeys.file(vars.fileId) })
      qc.invalidateQueries({ queryKey: documentKeys.tags() })
    },
  })
}

export function useUntagFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ fileId, tagId }: { fileId: string; tagId: string }) =>
      documentTagApi.untagFile(fileId, tagId),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: ['documents', 'files'] })
      qc.invalidateQueries({ queryKey: documentKeys.file(vars.fileId) })
      qc.invalidateQueries({ queryKey: documentKeys.tags() })
    },
  })
}

// ---------------------------------------------------------------------------
// Entity Link hooks
// ---------------------------------------------------------------------------

export function useFileLinks(fileId: string) {
  return useQuery({
    queryKey: documentKeys.fileLinks(fileId),
    queryFn: () => documentLinkApi.listByFile(fileId),
    enabled: !!fileId,
  })
}

export function useLinkFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({
      fileId,
      ...data
    }: LinkFileRequest & { fileId: string }) =>
      documentLinkApi.link(fileId, data),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({
        queryKey: documentKeys.fileLinks(vars.fileId),
      })
    },
  })
}

export function useUnlinkFile() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (linkId: string) => documentLinkApi.unlink(linkId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['documents', 'fileLinks'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Search hooks
// ---------------------------------------------------------------------------

export function useSearchFiles(
  query: string,
  params?: SearchFilesParams,
) {
  return useQuery({
    queryKey: documentKeys.search(query, params),
    queryFn: () => documentSearchApi.searchFiles(query, params),
    enabled: query.length >= 2,
  })
}

export function useVirtualFiles(sourceType?: string) {
  return useQuery({
    queryKey: documentKeys.virtualFiles(sourceType),
    queryFn: () => documentSearchApi.listVirtualFiles(sourceType),
  })
}

// ---------------------------------------------------------------------------
// WOPI hooks
// ---------------------------------------------------------------------------

export function useWOPIToken() {
  return useMutation({
    mutationFn: (fileId: string) => documentWopiApi.generateToken(fileId),
  })
}
