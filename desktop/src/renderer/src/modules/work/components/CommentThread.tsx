/**
 * Task comment thread with flat list, quote-reply, and @mentions.
 *
 * Displays comments chronologically with author, timestamp, content,
 * and optional quoted preview. Input supports @mention dropdown for
 * project members and quote-reply with preview.
 */
import { useState, useRef, useEffect, useMemo } from 'react'
import { MessageSquare, Reply, Pencil, Trash2, X, Send } from 'lucide-react'
import { cn } from '@/lib'
import { Button } from '@/components/ui/button'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import {
  useTaskComments,
  useCreateComment,
  useUpdateComment,
  useDeleteComment,
  type TaskComment,
} from '@/api/hooks/useTaskComments'
import { useProjectMembers } from '@/api/hooks/useProjects'
import { useAuthStore } from '@/stores/auth'

interface CommentThreadProps {
  taskId: string
  projectId: string
}

function getInitials(name?: string): string {
  if (!name) return '?'
  return name
    .split(' ')
    .map((p) => p[0])
    .join('')
    .toUpperCase()
    .slice(0, 2)
}

function formatRelativeTime(dateStr?: string): string {
  if (!dateStr) return ''
  const date = new Date(dateStr)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffMin = Math.floor(diffMs / 60000)
  const diffHours = Math.floor(diffMs / 3600000)
  const diffDays = Math.floor(diffMs / 86400000)

  if (diffMin < 1) return 'gerade eben'
  if (diffMin < 60) return `vor ${diffMin} Min.`
  if (diffHours < 24) return `vor ${diffHours} Std.`
  if (diffDays < 7) return `vor ${diffDays} Tag${diffDays === 1 ? '' : 'en'}`
  return date.toLocaleDateString('de-DE', {
    day: '2-digit',
    month: '2-digit',
    year: '2-digit',
  })
}

export default function CommentThread({
  taskId,
  projectId,
}: CommentThreadProps) {
  const currentUserId = useAuthStore((s) => s.user?.id)
  const { data: commentsData, isLoading } = useTaskComments(taskId)
  const { data: membersData } = useProjectMembers(projectId)
  const createComment = useCreateComment()
  const updateComment = useUpdateComment()
  const deleteComment = useDeleteComment()

  const comments = commentsData?.comments ?? []
  const members = membersData?.members ?? []

  const [inputValue, setInputValue] = useState('')
  const [quotedComment, setQuotedComment] = useState<TaskComment | null>(null)
  const [editingId, setEditingId] = useState<string | null>(null)
  const [editValue, setEditValue] = useState('')
  const [mentionOpen, setMentionOpen] = useState(false)
  const [mentionQuery, setMentionQuery] = useState('')
  const [mentionIndex, setMentionIndex] = useState(0)
  const inputRef = useRef<HTMLTextAreaElement>(null)
  const endRef = useRef<HTMLDivElement>(null)

  // Filter members for mention dropdown
  const filteredMembers = useMemo(() => {
    if (!mentionQuery) return members
    const q = mentionQuery.toLowerCase()
    return members.filter(
      (m) =>
        (m.display_name ?? '').toLowerCase().includes(q) ||
        (m.email ?? '').toLowerCase().includes(q)
    )
  }, [members, mentionQuery])

  // Auto-scroll when new comments arrive
  useEffect(() => {
    endRef.current?.scrollIntoView({ behavior: 'smooth' })
  }, [comments.length])

  function handleInputChange(e: React.ChangeEvent<HTMLTextAreaElement>) {
    const value = e.target.value
    setInputValue(value)

    // Detect @mention trigger
    const cursorPos = e.target.selectionStart ?? value.length
    const textBeforeCursor = value.slice(0, cursorPos)
    const lastAt = textBeforeCursor.lastIndexOf('@')

    if (lastAt >= 0) {
      const afterAt = textBeforeCursor.slice(lastAt + 1)
      // Only open mention if no space after @ (still typing name)
      if (!afterAt.includes(' ') && afterAt.length <= 30) {
        setMentionOpen(true)
        setMentionQuery(afterAt)
        setMentionIndex(0)
        return
      }
    }
    setMentionOpen(false)
  }

  function insertMention(displayName: string) {
    const cursorPos = inputRef.current?.selectionStart ?? inputValue.length
    const textBeforeCursor = inputValue.slice(0, cursorPos)
    const lastAt = textBeforeCursor.lastIndexOf('@')
    const before = inputValue.slice(0, lastAt)
    const after = inputValue.slice(cursorPos)
    const newValue = `${before}@${displayName} ${after}`
    setInputValue(newValue)
    setMentionOpen(false)
    inputRef.current?.focus()
  }

  function handleKeyDown(e: React.KeyboardEvent<HTMLTextAreaElement>) {
    if (mentionOpen) {
      if (e.key === 'ArrowDown') {
        e.preventDefault()
        setMentionIndex((i) => Math.min(i + 1, filteredMembers.length - 1))
        return
      }
      if (e.key === 'ArrowUp') {
        e.preventDefault()
        setMentionIndex((i) => Math.max(i - 1, 0))
        return
      }
      if (e.key === 'Enter' || e.key === 'Tab') {
        e.preventDefault()
        const member = filteredMembers[mentionIndex]
        if (member) {
          insertMention(member.display_name ?? member.email ?? '')
        }
        return
      }
      if (e.key === 'Escape') {
        setMentionOpen(false)
        return
      }
    }

    // Send on Enter (without Shift)
    if (e.key === 'Enter' && !e.shiftKey) {
      e.preventDefault()
      handleSend()
    }
  }

  async function handleSend() {
    const content = inputValue.trim()
    if (!content) return

    try {
      await createComment.mutateAsync({
        taskId,
        content,
        quoted_comment_id: quotedComment?.id,
      })
      setInputValue('')
      setQuotedComment(null)
    } catch {
      // Error handled by mutation state
    }
  }

  async function handleEditSave(commentId: string) {
    const content = editValue.trim()
    if (!content) return

    try {
      await updateComment.mutateAsync({ commentId, taskId, content })
      setEditingId(null)
      setEditValue('')
    } catch {
      // Error handled by mutation state
    }
  }

  async function handleDelete(commentId: string) {
    try {
      await deleteComment.mutateAsync({ commentId, taskId })
    } catch {
      // Error handled by mutation state
    }
  }

  function startEdit(comment: TaskComment) {
    setEditingId(comment.id ?? null)
    setEditValue(comment.content ?? '')
  }

  function startQuote(comment: TaskComment) {
    setQuotedComment(comment)
    inputRef.current?.focus()
  }

  // Render @mentions as highlighted spans in comment content
  function renderContent(content?: string) {
    if (!content) return null
    // Match @mentions (word characters after @)
    const parts = content.split(/(@\S+)/g)
    return parts.map((part, i) => {
      if (part.startsWith('@')) {
        return (
          <span
            key={i}
            className="rounded bg-primary/10 px-1 py-0.5 text-primary font-medium"
          >
            {part}
          </span>
        )
      }
      return <span key={i}>{part}</span>
    })
  }

  if (isLoading) {
    return (
      <div className="py-4 text-center text-sm text-muted-foreground">
        Kommentare werden geladen...
      </div>
    )
  }

  return (
    <div className="flex flex-col">
      {/* Comment list */}
      {comments.length === 0 ? (
        <div className="flex flex-col items-center py-6 text-center">
          <MessageSquare className="h-8 w-8 text-muted-foreground/30" />
          <p className="mt-2 text-sm text-muted-foreground">
            Noch keine Kommentare
          </p>
        </div>
      ) : (
        <div className="space-y-3">
          {comments.map((comment) => {
            const isOwn = comment.author_id === currentUserId
            const isEditing = editingId === comment.id

            return (
              <div
                key={comment.id}
                className="group rounded-lg border border-border bg-card p-3"
              >
                {/* Header */}
                <div className="flex items-center gap-2">
                  <Avatar className="h-6 w-6">
                    <AvatarFallback className="text-[10px]">
                      {getInitials(comment.author_name)}
                    </AvatarFallback>
                  </Avatar>
                  <span className="text-sm font-medium">
                    {comment.author_name}
                  </span>
                  <span className="text-xs text-muted-foreground">
                    {formatRelativeTime(comment.created_at)}
                  </span>
                  {comment.updated_at &&
                    comment.updated_at !== comment.created_at && (
                      <span className="text-xs text-muted-foreground italic">
                        (bearbeitet)
                      </span>
                    )}

                  {/* Actions (on hover) */}
                  <div className="ml-auto flex items-center gap-1 opacity-0 group-hover:opacity-100 transition-opacity">
                    <button
                      type="button"
                      className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                      onClick={() => startQuote(comment)}
                      title="Antworten"
                    >
                      <Reply className="h-3.5 w-3.5" />
                    </button>
                    {isOwn && (
                      <>
                        <button
                          type="button"
                          className="rounded p-1 text-muted-foreground hover:text-foreground hover:bg-accent transition-colors"
                          onClick={() => startEdit(comment)}
                          title="Bearbeiten"
                        >
                          <Pencil className="h-3.5 w-3.5" />
                        </button>
                        <button
                          type="button"
                          className="rounded p-1 text-muted-foreground hover:text-destructive hover:bg-destructive/10 transition-colors"
                          onClick={() => comment.id && handleDelete(comment.id)}
                          title="Löschen"
                        >
                          <Trash2 className="h-3.5 w-3.5" />
                        </button>
                      </>
                    )}
                  </div>
                </div>

                {/* Quoted preview */}
                {comment.quoted_content_preview && (
                  <div className="mt-2 rounded border-l-2 border-muted-foreground/30 bg-muted/50 px-3 py-1.5 text-xs text-muted-foreground line-clamp-2">
                    {comment.quoted_content_preview}
                  </div>
                )}

                {/* Content */}
                {isEditing ? (
                  <div className="mt-2 space-y-2">
                    <textarea
                      className="w-full rounded-md border border-input bg-background px-3 py-2 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring"
                      rows={3}
                      value={editValue}
                      onChange={(e) => setEditValue(e.target.value)}
                      onKeyDown={(e) => {
                        if (e.key === 'Enter' && !e.shiftKey) {
                          e.preventDefault()
                          if (comment.id) handleEditSave(comment.id)
                        }
                        if (e.key === 'Escape') {
                          setEditingId(null)
                          setEditValue('')
                        }
                      }}
                      autoFocus
                    />
                    <div className="flex gap-2">
                      <Button
                        size="sm"
                        variant="outline"
                        onClick={() => {
                          setEditingId(null)
                          setEditValue('')
                        }}
                      >
                        Abbrechen
                      </Button>
                      <Button
                        size="sm"
                        onClick={() => {
                          if (comment.id) handleEditSave(comment.id)
                        }}
                        disabled={updateComment.isPending}
                      >
                        Speichern
                      </Button>
                    </div>
                  </div>
                ) : (
                  <div className="mt-1.5 text-sm whitespace-pre-wrap">
                    {renderContent(comment.content)}
                  </div>
                )}
              </div>
            )
          })}
          <div ref={endRef} />
        </div>
      )}

      {/* Comment input */}
      <div className="mt-4 relative">
        {/* Quoted preview above input */}
        {quotedComment && (
          <div className="flex items-start gap-2 rounded-t-md border border-b-0 border-input bg-muted/50 px-3 py-2">
            <Reply className="mt-0.5 h-3.5 w-3.5 shrink-0 text-muted-foreground" />
            <div className="flex-1 min-w-0">
              <p className="text-xs font-medium text-muted-foreground">
                {quotedComment.author_name}
              </p>
              <p className="text-xs text-muted-foreground line-clamp-2">
                {quotedComment.content}
              </p>
            </div>
            <button
              type="button"
              className="rounded p-0.5 text-muted-foreground hover:text-foreground"
              onClick={() => setQuotedComment(null)}
            >
              <X className="h-3.5 w-3.5" />
            </button>
          </div>
        )}

        <div
          className={cn(
            'relative',
            quotedComment ? 'rounded-b-md' : 'rounded-md'
          )}
        >
          <textarea
            ref={inputRef}
            className={cn(
              'w-full border border-input bg-background px-3 py-2 pr-10 text-sm resize-none focus:outline-none focus:ring-2 focus:ring-ring',
              quotedComment
                ? 'rounded-b-md rounded-t-none'
                : 'rounded-md'
            )}
            rows={2}
            placeholder="Kommentar schreiben... (@erwaehnen, Shift+Enter für neue Zeile)"
            value={inputValue}
            onChange={handleInputChange}
            onKeyDown={handleKeyDown}
          />
          <button
            type="button"
            className={cn(
              'absolute right-2 bottom-2 rounded p-1 transition-colors',
              inputValue.trim()
                ? 'text-primary hover:bg-primary/10'
                : 'text-muted-foreground/40 cursor-not-allowed'
            )}
            onClick={handleSend}
            disabled={!inputValue.trim() || createComment.isPending}
          >
            <Send className="h-4 w-4" />
          </button>
        </div>

        {/* @mention dropdown */}
        {mentionOpen && filteredMembers.length > 0 && (
          <div className="absolute bottom-full left-0 mb-1 w-64 rounded-md border border-border bg-popover p-1 shadow-md z-50">
            {filteredMembers.slice(0, 8).map((member, i) => (
              <button
                key={member.user_id}
                type="button"
                className={cn(
                  'flex w-full items-center gap-2 rounded px-2 py-1.5 text-sm transition-colors',
                  i === mentionIndex
                    ? 'bg-accent text-accent-foreground'
                    : 'hover:bg-accent/50'
                )}
                onMouseDown={(e) => {
                  e.preventDefault()
                  insertMention(member.display_name ?? member.email ?? '')
                }}
                onMouseEnter={() => setMentionIndex(i)}
              >
                <Avatar className="h-5 w-5">
                  <AvatarFallback className="text-[9px]">
                    {getInitials(member.display_name)}
                  </AvatarFallback>
                </Avatar>
                <span className="truncate">
                  {member.display_name || member.email}
                </span>
              </button>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}
