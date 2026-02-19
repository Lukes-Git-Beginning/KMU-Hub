/**
 * TanStack Query hooks for Email operations.
 *
 * Query keys follow the pattern ['email', domain, ...params] for consistent
 * cache invalidation. Mutations invalidate related queries automatically.
 */
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query'
import {
  emailAccountApi,
  emailFolderApi,
  emailMessageApi,
  emailSendApi,
  emailSignatureApi,
  emailLinkApi,
  emailSyncApi,
  emailAttachmentApi,
} from '../email-client'
import type {
  CreateEmailAccountRequest,
  UpdateEmailAccountRequest,
  TestEmailConnectionRequest,
  ListMessagesParams,
  SendEmailRequest,
  SaveDraftRequest,
  ReplyEmailRequest,
  ForwardEmailRequest,
  CreateSignatureRequest,
  UpdateSignatureRequest,
} from '../email-types'

// ---------------------------------------------------------------------------
// Query key factory
// ---------------------------------------------------------------------------

export const emailKeys = {
  all: ['email'] as const,
  account: (userId: string) => ['email', 'account', userId] as const,
  folders: (accountId: string) => ['email', 'folders', accountId] as const,
  folder: (id: string) => ['email', 'folder', id] as const,
  messages: (params: ListMessagesParams) => ['email', 'messages', params] as const,
  message: (id: string) => ['email', 'message', id] as const,
  thread: (threadId: string) => ['email', 'thread', threadId] as const,
  signatures: (userId: string) => ['email', 'signatures', userId] as const,
  signature: (id: string) => ['email', 'signature', id] as const,
  links: (messageId: string) => ['email', 'links', messageId] as const,
  contactEmails: (contactId: string) => ['email', 'contactEmails', contactId] as const,
  syncStatus: (accountId: string) => ['email', 'syncStatus', accountId] as const,
}

// ---------------------------------------------------------------------------
// Account hooks
// ---------------------------------------------------------------------------

export function useEmailAccount(userId: string) {
  return useQuery({
    queryKey: emailKeys.account(userId),
    queryFn: () => emailAccountApi.get(userId),
    enabled: !!userId,
  })
}

export function useCreateEmailAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateEmailAccountRequest) => emailAccountApi.create(req),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: emailKeys.account(vars.user_id) })
    },
  })
}

export function useUpdateEmailAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...rest }: UpdateEmailAccountRequest) => emailAccountApi.update(id, rest),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'account'] })
    },
  })
}

export function useDeleteEmailAccount() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => emailAccountApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email'] })
    },
  })
}

export function useTestEmailConnection() {
  return useMutation({
    mutationFn: (req: TestEmailConnectionRequest) => emailAccountApi.testConnection(req),
  })
}

// ---------------------------------------------------------------------------
// Folder hooks
// ---------------------------------------------------------------------------

export function useEmailFolders(accountId: string) {
  return useQuery({
    queryKey: emailKeys.folders(accountId),
    queryFn: () => emailFolderApi.list(accountId),
    enabled: !!accountId,
  })
}

export function useEmailFolder(id: string) {
  return useQuery({
    queryKey: emailKeys.folder(id),
    queryFn: () => emailFolderApi.get(id),
    enabled: !!id,
  })
}

export function useSyncFolders() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (accountId: string) => emailFolderApi.sync(accountId),
    onSuccess: (_data, accountId) => {
      qc.invalidateQueries({ queryKey: emailKeys.folders(accountId) })
    },
  })
}

// ---------------------------------------------------------------------------
// Message hooks
// ---------------------------------------------------------------------------

export function useEmailMessages(params: ListMessagesParams) {
  return useQuery({
    queryKey: emailKeys.messages(params),
    queryFn: () => emailMessageApi.list(params),
    enabled: !!params.folder_id,
  })
}

export function useEmailMessage(id: string) {
  return useQuery({
    queryKey: emailKeys.message(id),
    queryFn: () => emailMessageApi.get(id),
    enabled: !!id,
  })
}

export function useEmailThread(threadId: string) {
  return useQuery({
    queryKey: emailKeys.thread(threadId),
    queryFn: () => emailMessageApi.getThread(threadId),
    enabled: !!threadId,
  })
}

export function useMarkEmailRead() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => emailMessageApi.markRead(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
      qc.invalidateQueries({ queryKey: ['email', 'folders'] })
    },
  })
}

export function useMarkEmailUnread() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => emailMessageApi.markUnread(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
      qc.invalidateQueries({ queryKey: ['email', 'folders'] })
    },
  })
}

export function useToggleEmailStar() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => emailMessageApi.toggleStar(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
    },
  })
}

export function useMoveEmailToFolder() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ messageId, targetFolderId }: { messageId: string; targetFolderId: string }) =>
      emailMessageApi.moveToFolder(messageId, targetFolderId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
      qc.invalidateQueries({ queryKey: ['email', 'folders'] })
    },
  })
}

export function useDeleteEmailMessage() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => emailMessageApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
      qc.invalidateQueries({ queryKey: ['email', 'folders'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Send / Compose hooks
// ---------------------------------------------------------------------------

export function useSendEmail() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: SendEmailRequest) => emailSendApi.send(req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
      qc.invalidateQueries({ queryKey: ['email', 'folders'] })
    },
  })
}

export function useSaveDraft() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: SaveDraftRequest) => emailSendApi.saveDraft(req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
      qc.invalidateQueries({ queryKey: ['email', 'folders'] })
    },
  })
}

export function useReplyEmail() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: ReplyEmailRequest) => emailSendApi.reply(req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
      qc.invalidateQueries({ queryKey: ['email', 'folders'] })
    },
  })
}

export function useForwardEmail() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: ForwardEmailRequest) => emailSendApi.forward(req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'messages'] })
      qc.invalidateQueries({ queryKey: ['email', 'folders'] })
    },
  })
}

// ---------------------------------------------------------------------------
// Signature hooks
// ---------------------------------------------------------------------------

export function useEmailSignatures(userId: string) {
  return useQuery({
    queryKey: emailKeys.signatures(userId),
    queryFn: () => emailSignatureApi.list(userId),
    enabled: !!userId,
  })
}

export function useEmailSignature(id: string) {
  return useQuery({
    queryKey: emailKeys.signature(id),
    queryFn: () => emailSignatureApi.get(id),
    enabled: !!id,
  })
}

export function useCreateSignature() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (req: CreateSignatureRequest) => emailSignatureApi.create(req),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'signatures'] })
    },
  })
}

export function useUpdateSignature() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, ...rest }: UpdateSignatureRequest) => emailSignatureApi.update(id, rest),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'signatures'] })
    },
  })
}

export function useDeleteSignature() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => emailSignatureApi.delete(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'signatures'] })
    },
  })
}

export function useSetDefaultSignature() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, userId }: { id: string; userId: string }) =>
      emailSignatureApi.setDefault(id, userId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ['email', 'signatures'] })
    },
  })
}

// ---------------------------------------------------------------------------
// CRM Link hooks
// ---------------------------------------------------------------------------

export function useEmailContactLinks(messageId: string) {
  return useQuery({
    queryKey: emailKeys.links(messageId),
    queryFn: () => emailLinkApi.getByMessage(messageId),
    enabled: !!messageId,
  })
}

export function useLinkEmailToContact() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ messageId, contactId, linkType }: { messageId: string; contactId: string; linkType?: 'auto' | 'manual' }) =>
      emailLinkApi.link(messageId, contactId, linkType),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: emailKeys.links(vars.messageId) })
      qc.invalidateQueries({ queryKey: ['email', 'contactEmails'] })
    },
  })
}

export function useUnlinkEmailFromContact() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ messageId, contactId }: { messageId: string; contactId: string }) =>
      emailLinkApi.unlink(messageId, contactId),
    onSuccess: (_data, vars) => {
      qc.invalidateQueries({ queryKey: emailKeys.links(vars.messageId) })
      qc.invalidateQueries({ queryKey: ['email', 'contactEmails'] })
    },
  })
}

export function useContactEmails(contactId: string, page?: number, perPage?: number) {
  return useQuery({
    queryKey: emailKeys.contactEmails(contactId),
    queryFn: () => emailLinkApi.getContactEmails(contactId, page, perPage),
    enabled: !!contactId,
  })
}

// ---------------------------------------------------------------------------
// Sync hooks
// ---------------------------------------------------------------------------

export function useSyncStatus(accountId: string) {
  return useQuery({
    queryKey: emailKeys.syncStatus(accountId),
    queryFn: () => emailSyncApi.getStatus(accountId),
    enabled: !!accountId,
    refetchInterval: 30_000,
  })
}

export function useTriggerSync() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (accountId: string) => emailSyncApi.trigger(accountId),
    onSuccess: (_data, accountId) => {
      qc.invalidateQueries({ queryKey: emailKeys.syncStatus(accountId) })
    },
  })
}

// ---------------------------------------------------------------------------
// Attachment hooks
// ---------------------------------------------------------------------------

export function useUploadAttachment() {
  return useMutation({
    mutationFn: ({ accountId, file }: { accountId: string; file: File }) =>
      emailAttachmentApi.upload(accountId, file),
  })
}

export function useAttachmentDownloadURL(id: string) {
  return useQuery({
    queryKey: ['email', 'attachment', id],
    queryFn: () => emailAttachmentApi.getDownloadURL(id),
    enabled: !!id,
  })
}
