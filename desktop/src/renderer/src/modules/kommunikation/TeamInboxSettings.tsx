/**
 * Team inbox settings dialog.
 *
 * Allows editing team inbox name, description, assignment mode
 * (manual/round-robin), visibility (open/private), and member management.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { X, UserPlus, Trash2 } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { ConfirmDialog } from '@/components/shared'
import type {
  TeamInbox,
  AssignmentMode,
  TeamVisibility,
  TeamMemberRole,
} from '@/api/inbox-types'
import {
  useTeamMembers,
  useUpdateTeamInbox,
  useDeleteTeamInbox,
  useAddTeamMember,
  useRemoveTeamMember,
} from '@/api/hooks/useInbox'

interface TeamInboxSettingsProps {
  team: TeamInbox | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function TeamInboxSettings({
  team,
  open,
  onOpenChange,
}: TeamInboxSettingsProps) {
  const { t } = useTranslation()
  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [assignmentMode, setAssignmentMode] =
    useState<AssignmentMode>('manual')
  const [visibility, setVisibility] = useState<TeamVisibility>('open')
  const [deleteConfirm, setDeleteConfirm] = useState(false)

  // New member form
  const [newMemberUserId, setNewMemberUserId] = useState('')
  const [newMemberRole, setNewMemberRole] = useState<TeamMemberRole>('member')

  const { data: members } = useTeamMembers(team?.id ?? '')
  const updateTeam = useUpdateTeamInbox()
  const deleteTeam = useDeleteTeamInbox()
  const addMember = useAddTeamMember()
  const removeMember = useRemoveTeamMember()

   
  useEffect(() => {
    if (team) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setName(team.name)
      setDescription(team.description ?? '')
      setAssignmentMode(team.assignment_mode)
      setVisibility(team.visibility)
    }
  }, [team])

  if (!team) return null

  const handleSave = () => {
    updateTeam.mutate(
      {
        id: team.id,
        name,
        description: description || undefined,
        assignment_mode: assignmentMode,
        visibility,
      },
      { onSuccess: () => onOpenChange(false) },
    )
  }

  const handleDelete = () => {
    deleteTeam.mutate(team.id, {
      onSuccess: () => {
        setDeleteConfirm(false)
        onOpenChange(false)
      },
    })
  }

  const handleAddMember = () => {
    if (!newMemberUserId.trim()) return
    addMember.mutate(
      {
        teamId: team.id,
        userId: newMemberUserId.trim(),
        role: newMemberRole,
      },
      {
        onSuccess: () => {
          setNewMemberUserId('')
          setNewMemberRole('member')
        },
      },
    )
  }

  return (
    <>
      <Dialog open={open} onOpenChange={onOpenChange}>
        <DialogContent className="max-w-lg max-h-[80vh] overflow-y-auto">
          <DialogHeader>
            <DialogTitle>{t('kommunikation.teamInbox.settingsTitle', { name: team.name })}</DialogTitle>
            <DialogDescription>
              {t('kommunikation.teamInbox.settingsDescription')}
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 pt-2">
            {/* Name */}
            <div>
              <label className="text-xs font-medium text-foreground">
                {t('kommunikation.teamInbox.name')}
              </label>
              <input
                type="text"
                value={name}
                onChange={(e) => setName(e.target.value)}
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
            </div>

            {/* Description */}
            <div>
              <label className="text-xs font-medium text-foreground">
                {t('kommunikation.teamInbox.description')}
              </label>
              <textarea
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={2}
                className="mt-1 w-full rounded-md border border-border bg-background px-3 py-2 text-sm text-foreground focus:outline-none focus:ring-2 focus:ring-focus-ring resize-none"
              />
            </div>

            {/* Assignment mode */}
            <div>
              <label className="text-xs font-medium text-foreground">
                {t('kommunikation.teamInbox.assignmentMode')}
              </label>
              <div className="mt-1 flex gap-4">
                <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer">
                  <input
                    type="radio"
                    name="assignmentMode"
                    value="manual"
                    checked={assignmentMode === 'manual'}
                    onChange={() => setAssignmentMode('manual')}
                    className="accent-primary"
                  />
                  {t('kommunikation.teamInbox.manual')}
                </label>
                <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer">
                  <input
                    type="radio"
                    name="assignmentMode"
                    value="round_robin"
                    checked={assignmentMode === 'round_robin'}
                    onChange={() => setAssignmentMode('round_robin')}
                    className="accent-primary"
                  />
                  {t('kommunikation.inbox.roundRobin')}
                </label>
              </div>
            </div>

            {/* Visibility */}
            <div>
              <label className="text-xs font-medium text-foreground">
                {t('kommunikation.teamInbox.visibility')}
              </label>
              <div className="mt-1 flex gap-4">
                <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer">
                  <input
                    type="radio"
                    name="visibility"
                    value="open"
                    checked={visibility === 'open'}
                    onChange={() => setVisibility('open')}
                    className="accent-primary"
                  />
                  {t('kommunikation.inbox.visibilityOpen')}
                </label>
                <label className="flex items-center gap-2 text-sm text-foreground cursor-pointer">
                  <input
                    type="radio"
                    name="visibility"
                    value="private"
                    checked={visibility === 'private'}
                    onChange={() => setVisibility('private')}
                    className="accent-primary"
                  />
                  {t('kommunikation.inbox.visibilityPrivate')}
                </label>
              </div>
            </div>

            {/* Members */}
            <div>
              <label className="text-xs font-medium text-foreground">
                {t('kommunikation.inbox.members')}
              </label>
              <div className="mt-1 space-y-1">
                {members?.map((m) => (
                  <div
                    key={m.user_id}
                    className="flex items-center justify-between rounded-md border border-border px-3 py-2"
                  >
                    <div className="flex items-center gap-2">
                      <span className="text-sm text-foreground">
                        {m.user_id}
                      </span>
                      <span
                        className={`rounded-full px-2 py-0.5 text-[10px] font-medium ${
                          m.role === 'admin'
                            ? 'bg-primary/10 text-primary'
                            : 'bg-secondary text-secondary-foreground'
                        }`}
                      >
                        {m.role === 'admin' ? t('kommunikation.inbox.roleAdmin') : t('kommunikation.inbox.roleMember')}
                      </span>
                    </div>
                    <button
                      onClick={() =>
                        removeMember.mutate({
                          teamId: team.id,
                          userId: m.user_id,
                        })
                      }
                      className="rounded p-1 text-muted-foreground hover:text-red-500 hover:bg-secondary transition-colors"
                      title={t('kommunikation.inbox.removeMember')}
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
                {(!members || members.length === 0) && (
                  <p className="text-xs text-muted-foreground py-2">
                    {t('kommunikation.inbox.noMembers')}
                  </p>
                )}
              </div>

              {/* Add member */}
              <div className="mt-2 flex gap-2">
                <input
                  type="text"
                  value={newMemberUserId}
                  onChange={(e) => setNewMemberUserId(e.target.value)}
                  placeholder={t('kommunikation.inbox.userIdPlaceholder')}
                  className="flex-1 rounded-md border border-border bg-background px-3 py-1.5 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-1 focus:ring-focus-ring"
                />
                <select
                  value={newMemberRole}
                  onChange={(e) =>
                    setNewMemberRole(e.target.value as TeamMemberRole)
                  }
                  className="rounded-md border border-border bg-background px-2 py-1.5 text-sm text-foreground focus:outline-none focus:ring-1 focus:ring-focus-ring"
                >
                  <option value="member">{t('kommunikation.inbox.roleMember')}</option>
                  <option value="admin">{t('kommunikation.inbox.roleAdmin')}</option>
                </select>
                <button
                  onClick={handleAddMember}
                  disabled={!newMemberUserId.trim()}
                  className="flex items-center gap-1 rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
                >
                  <UserPlus className="h-3.5 w-3.5" />
                </button>
              </div>
            </div>

            {/* Actions */}
            <div className="flex items-center justify-between pt-2 border-t border-border">
              <button
                onClick={() => setDeleteConfirm(true)}
                className="flex items-center gap-1.5 rounded-md px-3 py-1.5 text-xs text-red-500 hover:bg-red-50 dark:hover:bg-red-950/30 transition-colors"
              >
                <Trash2 className="h-3.5 w-3.5" />
                {t('common.delete')}
              </button>
              <div className="flex gap-2">
                <button
                  onClick={() => onOpenChange(false)}
                  className="rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
                >
                  {t('common.cancel')}
                </button>
                <button
                  onClick={handleSave}
                  disabled={!name.trim() || updateTeam.isPending}
                  className="rounded-md bg-primary px-3 py-1.5 text-xs text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
                >
                  {t('common.save')}
                </button>
              </div>
            </div>
          </div>
        </DialogContent>
      </Dialog>

      <ConfirmDialog
        open={deleteConfirm}
        onOpenChange={setDeleteConfirm}
        title={t('kommunikation.inbox.deleteTitle')}
        description={t('kommunikation.inbox.deleteDescription', { name: team.name })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </>
  )
}
