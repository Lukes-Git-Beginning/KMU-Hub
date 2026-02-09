import { create } from 'zustand'
import { persist } from 'zustand/middleware'

export type FileType = 'pdf' | 'word' | 'excel' | 'image' | 'video' | 'archive' | 'other'

export interface DocFile {
  id: string
  name: string
  type: FileType
  size: string
  sizeBytes: number
  date: string
  folderId: string
  tags: string[]
  createdBy: string
  isFavorite: boolean
  isShared: boolean
  isVault: boolean
  sharedWith: { name: string; permission: 'view' | 'edit' }[]
  versions: { version: string; date: string; author: string }[]
}

export interface DocFolder {
  id: string
  name: string
  parentId: string | null
  icon: 'folder' | 'share' | 'star' | 'lock'
  isSystem: boolean
}

interface DocumentsState {
  files: DocFile[]
  folders: DocFolder[]
  addFile: (file: Omit<DocFile, 'id' | 'versions'>) => void
  removeFile: (id: string) => void
  renameFile: (id: string, name: string) => void
  moveFile: (id: string, folderId: string) => void
  toggleFavorite: (id: string) => void
  toggleShare: (id: string) => void
  updateFileTags: (id: string, tags: string[]) => void
  addFolder: (name: string, parentId: string | null) => void
  renameFolder: (id: string, name: string) => void
  deleteFolder: (id: string) => void
  totalStorageUsed: () => number
}

const mockFolders: DocFolder[] = [
  { id: 'root', name: 'Alle Dateien', parentId: null, icon: 'folder', isSystem: true },
  { id: 'projects', name: 'Projekte', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'contracts', name: 'Vertraege', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'invoices', name: 'Rechnungen', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'marketing', name: 'Marketing', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'hr', name: 'HR', parentId: 'root', icon: 'folder', isSystem: false },
  { id: 'shared', name: 'Geteilt mit mir', parentId: null, icon: 'share', isSystem: true },
  { id: 'favorites', name: 'Favoriten', parentId: null, icon: 'star', isSystem: true },
  { id: 'vault', name: 'Tresor', parentId: null, icon: 'lock', isSystem: true },
  { id: 'vault-hr', name: 'HR Dokumente', parentId: 'vault', icon: 'folder', isSystem: false },
  { id: 'vault-finance', name: 'Finanzen', parentId: 'vault', icon: 'folder', isSystem: false },
]

const mockFiles: DocFile[] = [
  {
    id: 'd1', name: 'Projektplan_Q1_2026.pdf', type: 'pdf', size: '2.4 MB', sizeBytes: 2516582,
    date: '2026-02-07', folderId: 'projects', tags: ['Q1', 'Planung'], createdBy: 'Anna Mueller',
    isFavorite: true, isShared: true, isVault: false,
    sharedWith: [{ name: 'Michael Berg', permission: 'edit' }, { name: 'Sarah Klein', permission: 'view' }],
    versions: [{ version: '1.2', date: '2026-02-07', author: 'Anna Mueller' }, { version: '1.1', date: '2026-02-03', author: 'Anna Mueller' }, { version: '1.0', date: '2026-01-28', author: 'Michael Berg' }],
  },
  {
    id: 'd2', name: 'Vertrag_Kunde_ABC.pdf', type: 'pdf', size: '1.8 MB', sizeBytes: 1887437,
    date: '2026-02-06', folderId: 'contracts', tags: ['Vertrag', 'ABC GmbH'], createdBy: 'Peter Koch',
    isFavorite: false, isShared: false, isVault: true,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-02-06', author: 'Peter Koch' }],
  },
  {
    id: 'd3', name: 'Budget_2026.xlsx', type: 'excel', size: '540 KB', sizeBytes: 552960,
    date: '2026-02-05', folderId: 'invoices', tags: ['Budget', 'Finanzen'], createdBy: 'Michael Berg',
    isFavorite: true, isShared: true, isVault: false,
    sharedWith: [{ name: 'Anna Mueller', permission: 'edit' }],
    versions: [{ version: '2.0', date: '2026-02-05', author: 'Michael Berg' }, { version: '1.0', date: '2026-01-15', author: 'Michael Berg' }],
  },
  {
    id: 'd4', name: 'Logo_Redesign_v3.png', type: 'image', size: '4.2 MB', sizeBytes: 4404019,
    date: '2026-02-04', folderId: 'marketing', tags: ['Design', 'Branding'], createdBy: 'Sarah Klein',
    isFavorite: false, isShared: true, isVault: false,
    sharedWith: [{ name: 'Lisa Schmidt', permission: 'view' }],
    versions: [{ version: '3.0', date: '2026-02-04', author: 'Sarah Klein' }, { version: '2.0', date: '2026-01-20', author: 'Sarah Klein' }],
  },
  {
    id: 'd5', name: 'Onboarding_Guide.docx', type: 'word', size: '890 KB', sizeBytes: 911360,
    date: '2026-02-03', folderId: 'hr', tags: ['HR', 'Onboarding'], createdBy: 'Lisa Schmidt',
    isFavorite: false, isShared: true, isVault: false,
    sharedWith: [{ name: 'Anna Mueller', permission: 'edit' }, { name: 'Peter Koch', permission: 'view' }],
    versions: [{ version: '1.1', date: '2026-02-03', author: 'Lisa Schmidt' }],
  },
  {
    id: 'd6', name: 'Produktvideo_Final.mp4', type: 'video', size: '156 MB', sizeBytes: 163577856,
    date: '2026-02-02', folderId: 'marketing', tags: ['Video', 'Marketing'], createdBy: 'Jonas Diaz',
    isFavorite: false, isShared: false, isVault: false,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-02-02', author: 'Jonas Diaz' }],
  },
  {
    id: 'd7', name: 'Archiv_2025.zip', type: 'archive', size: '340 MB', sizeBytes: 356515840,
    date: '2026-01-31', folderId: 'root', tags: ['Archiv', '2025'], createdBy: 'Peter Koch',
    isFavorite: false, isShared: false, isVault: false,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-01-31', author: 'Peter Koch' }],
  },
  {
    id: 'd8', name: 'Mitarbeiterakte_vertraulich.pdf', type: 'pdf', size: '3.1 MB', sizeBytes: 3250586,
    date: '2026-02-01', folderId: 'vault-hr', tags: ['HR', 'Vertraulich'], createdBy: 'Anna Mueller',
    isFavorite: false, isShared: false, isVault: true,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-02-01', author: 'Anna Mueller' }],
  },
  {
    id: 'd9', name: 'Sprint_Retro_Notes.docx', type: 'word', size: '120 KB', sizeBytes: 122880,
    date: '2026-02-07', folderId: 'projects', tags: ['Sprint', 'Retro'], createdBy: 'Michael Berg',
    isFavorite: false, isShared: true, isVault: false,
    sharedWith: [{ name: 'Anna Mueller', permission: 'view' }],
    versions: [{ version: '1.0', date: '2026-02-07', author: 'Michael Berg' }],
  },
  {
    id: 'd10', name: 'Kampagne_Assets.zip', type: 'archive', size: '89 MB', sizeBytes: 93323264,
    date: '2026-02-06', folderId: 'marketing', tags: ['Marketing', 'Assets'], createdBy: 'Sarah Klein',
    isFavorite: true, isShared: false, isVault: false,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-02-06', author: 'Sarah Klein' }],
  },
  {
    id: 'd11', name: 'Angebot_Steiner_Bau.pdf', type: 'pdf', size: '1.5 MB', sizeBytes: 1572864,
    date: '2026-02-08', folderId: 'contracts', tags: ['Angebot', 'Steiner Bau'], createdBy: 'Anna Mueller',
    isFavorite: false, isShared: true, isVault: false,
    sharedWith: [{ name: 'Thomas Weber', permission: 'view' }],
    versions: [{ version: '1.0', date: '2026-02-08', author: 'Anna Mueller' }],
  },
  {
    id: 'd12', name: 'Bilanz_Q4_2025.xlsx', type: 'excel', size: '780 KB', sizeBytes: 798720,
    date: '2026-01-15', folderId: 'vault-finance', tags: ['Bilanz', 'Q4'], createdBy: 'Michael Berg',
    isFavorite: false, isShared: false, isVault: true,
    sharedWith: [], versions: [{ version: '1.0', date: '2026-01-15', author: 'Michael Berg' }],
  },
]

let nextFileId = 13
let nextFolderId = 12

export const useDocumentsStore = create<DocumentsState>()(
  persist(
    (set, get) => ({
      files: mockFiles,
      folders: mockFolders,

      addFile: (file) =>
        set((state) => ({
          files: [
            { ...file, id: `d${nextFileId++}`, versions: [{ version: '1.0', date: file.date, author: file.createdBy }] },
            ...state.files,
          ],
        })),

      removeFile: (id) =>
        set((state) => ({ files: state.files.filter((f) => f.id !== id) })),

      renameFile: (id, name) =>
        set((state) => ({
          files: state.files.map((f) => (f.id === id ? { ...f, name } : f)),
        })),

      moveFile: (id, folderId) =>
        set((state) => ({
          files: state.files.map((f) => (f.id === id ? { ...f, folderId } : f)),
        })),

      toggleFavorite: (id) =>
        set((state) => ({
          files: state.files.map((f) =>
            f.id === id ? { ...f, isFavorite: !f.isFavorite } : f
          ),
        })),

      toggleShare: (id) =>
        set((state) => ({
          files: state.files.map((f) =>
            f.id === id ? { ...f, isShared: !f.isShared } : f
          ),
        })),

      updateFileTags: (id, tags) =>
        set((state) => ({
          files: state.files.map((f) => (f.id === id ? { ...f, tags } : f)),
        })),

      addFolder: (name, parentId) =>
        set((state) => ({
          folders: [
            ...state.folders,
            { id: `folder-${nextFolderId++}`, name, parentId: parentId || 'root', icon: 'folder', isSystem: false },
          ],
        })),

      renameFolder: (id, name) =>
        set((state) => ({
          folders: state.folders.map((f) => (f.id === id ? { ...f, name } : f)),
        })),

      deleteFolder: (id) =>
        set((state) => ({
          folders: state.folders.filter((f) => f.id !== id),
          files: state.files.map((f) => (f.folderId === id ? { ...f, folderId: 'root' } : f)),
        })),

      totalStorageUsed: () => get().files.reduce((sum, f) => sum + f.sizeBytes, 0),
    }),
    { name: 'kmuhub-documents' }
  )
)
