/**
 * TypeScript types for the Document Management API.
 *
 * Mirrors proto/backend response shapes from Plan 11-01/11-02.
 * All types exported for use by document-client.ts and TanStack Query hooks.
 */

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export type FolderSpaceType = 'personal' | 'team' | 'project'
export type SharePermission = 'read' | 'write'
export type VirtualFileSource = 'chat' | 'email' | 'task'
export type FileSortField = 'name' | 'size' | 'date' | 'type'
export type SortDirection = 'asc' | 'desc'

// ---------------------------------------------------------------------------
// Core Models
// ---------------------------------------------------------------------------

export interface DocumentFolder {
  id: string
  name: string
  parent_id: string | null
  space_type: FolderSpaceType
  space_id: string
  is_system: boolean
  icon: string
  created_by: string
  created_at: string
  updated_at: string
  file_count: number
}

export interface DocumentFile {
  id: string
  folder_id: string
  filename: string
  mime_type: string
  file_size: number
  storage_key: string
  thumbnail_key: string | null
  current_version: number
  owner_id: string
  is_favorite: boolean
  is_deleted: boolean
  tags: DocumentTag[]
  created_at: string
  updated_at: string
}

export interface DocumentFileVersion {
  id: string
  file_id: string
  version_number: number
  version_label: string | null
  storage_key: string
  file_size: number
  created_by: string
  created_by_name: string
  created_at: string
}

export interface DocumentShare {
  id: string
  entity_type: 'file' | 'folder'
  entity_id: string
  shared_with_user_id: string
  shared_with_user_name: string
  permission: SharePermission
  shared_by: string
  shared_by_name: string
  created_at: string
}

export interface DocumentTag {
  id: string
  name: string
  color: string | null
  file_count: number
  created_by: string
  created_at: string
}

export interface DocumentEntityLink {
  id: string
  file_id: string
  entity_type: string
  entity_id: string
  entity_name: string
  linked_by: string
  created_at: string
}

export interface VirtualFile {
  id: string
  filename: string
  mime_type: string
  file_size: number
  storage_key: string
  source_type: VirtualFileSource
  source_id: string
  source_name: string
  uploaded_by: string
  uploaded_by_name: string
  created_at: string
}

export interface FolderPathSegment {
  id: string
  name: string
}

export type DocumentActivityAction =
  | 'uploaded'
  | 'renamed'
  | 'moved'
  | 'copied'
  | 'downloaded'
  | 'shared'
  | 'version_created'
  | 'reverted'

/** One entry of a file's audit trail (who did what, when). Mock-first —
 *  the backend activity log is tracked in .planning/backend-gaps.md. */
export interface DocumentActivity {
  id: string
  file_id: string
  action: DocumentActivityAction
  actor_id: string
  actor_name: string
  detail: string | null
  created_at: string
}

export interface FileSearchResult {
  file: DocumentFile
  rank: number
  snippet: string
}

// ---------------------------------------------------------------------------
// API Request / Response types
// ---------------------------------------------------------------------------

export interface ListFilesParams {
  folder_id?: string
  owner_id?: string
  is_favorite?: boolean
  tag_ids?: string[]
  sort_field?: FileSortField
  sort_dir?: SortDirection
  limit?: number
  offset?: number
}

export interface ListFoldersParams {
  parent_id?: string
  space_type?: FolderSpaceType
  space_id?: string
}

export interface CreateFolderRequest {
  name: string
  parent_id?: string
  space_type: FolderSpaceType
  space_id: string
}

export interface UpdateFolderRequest {
  name?: string
  parent_id?: string
}

export interface UpdateFileRequest {
  filename?: string
  folder_id?: string
  is_favorite?: boolean
}

export interface ShareEntityRequest {
  entity_type: 'file' | 'folder'
  entity_id: string
  shared_with_user_id: string
  permission: SharePermission
}

export interface CreateTagRequest {
  name: string
  color?: string
}

export interface LinkFileRequest {
  entity_type: string
  entity_id: string
}

export interface SearchFilesParams {
  folder_id?: string
  tag_ids?: string[]
  limit?: number
}

// ---------------------------------------------------------------------------
// Global search types
// ---------------------------------------------------------------------------

export interface GlobalSearchResult {
  id: string
  title: string
  description: string
  module: string
  url: string
  score: number
}

export interface GlobalSearchResponse {
  query: string
  modules: Record<string, { results: GlobalSearchResult[]; total: number }>
}

// ---------------------------------------------------------------------------
// Paginated response wrappers
// ---------------------------------------------------------------------------

export interface ListFilesResponse {
  files: DocumentFile[]
  total: number
}

export interface ListFoldersResponse {
  folders: DocumentFolder[]
  total: number
}

export interface ListVersionsResponse {
  versions: DocumentFileVersion[]
}

export interface ListSharesResponse {
  shares: DocumentShare[]
}

export interface ListTagsResponse {
  tags: DocumentTag[]
}

export interface ListEntityLinksResponse {
  links: DocumentEntityLink[]
}

export interface SearchFilesResponse {
  results: FileSearchResult[]
  total: number
}

export interface ListVirtualFilesResponse {
  files: VirtualFile[]
  total: number
}

export interface DownloadURLResponse {
  url: string
}

export interface ListActivityResponse {
  activities: DocumentActivity[]
}

export interface WOPITokenResponse {
  access_token: string
  access_token_ttl: number
}
