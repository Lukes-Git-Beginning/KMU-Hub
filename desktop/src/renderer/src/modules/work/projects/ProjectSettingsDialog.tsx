/**
 * Project settings dialog for editing name, description, statuses, members.
 *
 * Provides tabs for project info, status management, member management,
 * and project archival/template creation.
 */
import { useState, useEffect } from 'react'
import { useTranslation } from 'react-i18next'
import { Plus, X, Trash2, Save, Archive, Copy } from 'lucide-react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { Textarea } from '@/components/ui/textarea'
import { Badge } from '@/components/ui/badge'
import { Separator } from '@/components/ui/separator'
import {
  useProject,
  useUpdateProject,
  useArchiveProject,
  useProjectStatuses,
  useCreateStatus,
  useUpdateStatus,
  useDeleteStatus,
  useProjectMembers,
  useRemoveMember,
  useSaveAsTemplate,
} from '@/api/hooks/useProjects'

interface ProjectSettingsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  projectId: string
}

export default function ProjectSettingsDialog({
  open,
  onOpenChange,
  projectId,
}: ProjectSettingsDialogProps) {
  const { t } = useTranslation()
  const { data: projectData } = useProject(projectId)
  const { data: statusesData } = useProjectStatuses(projectId)
  const { data: membersData } = useProjectMembers(projectId)

  const updateProject = useUpdateProject()
  const archiveProject = useArchiveProject()
  const createStatus = useCreateStatus()
  const _updateStatus = useUpdateStatus()
  const deleteStatus = useDeleteStatus()
  const removeMember = useRemoveMember()
  const saveAsTemplate = useSaveAsTemplate()

  const project = projectData?.project
  const statuses = statusesData?.statuses ?? []
  const members = membersData?.members ?? []

  const [name, setName] = useState('')
  const [description, setDescription] = useState('')
  const [newStatusName, setNewStatusName] = useState('')
  const [newStatusColor, setNewStatusColor] = useState('#6b7280')
  const [templateName, setTemplateName] = useState('')
  const [tab, setTab] = useState<'info' | 'statuses' | 'members'>('info')

   
  useEffect(() => {
    if (project) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setName(project.name ?? '')
      setDescription(project.description ?? '')
    }
  }, [project])

  async function handleSaveInfo() {
    await updateProject.mutateAsync({
      id: projectId,
      name: name.trim(),
      description: description.trim(),
    })
  }

  async function handleAddStatus() {
    if (!newStatusName.trim()) return
    await createStatus.mutateAsync({
      projectId,
      name: newStatusName.trim(),
      color: newStatusColor,
    })
    setNewStatusName('')
    setNewStatusColor('#6b7280')
  }

  async function handleArchive() {
    if (!confirm(t('work.settings.archiveConfirm'))) return
    await archiveProject.mutateAsync(projectId)
    onOpenChange(false)
  }

  async function handleSaveAsTemplate() {
    if (!templateName.trim()) return
    await saveAsTemplate.mutateAsync({
      projectId,
      template_name: templateName.trim(),
    })
    setTemplateName('')
  }

  const tabs = [
    { key: 'info' as const, labelKey: 'work.settings.general' },
    { key: 'statuses' as const, labelKey: 'common.status' },
    { key: 'members' as const, labelKey: 'work.settings.members' },
  ]

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle>{t('work.settings.title')}</DialogTitle>
          <DialogDescription>
            {t('work.settings.description', { name: project?.name ?? t('work.settings.project') })}
          </DialogDescription>
        </DialogHeader>

        {/* Tab navigation */}
        <div className="flex gap-1 border-b border-border">
          {tabs.map((tb) => (
            <button
              key={tb.key}
              className={`px-3 py-2 text-sm font-medium border-b-2 transition-colors ${
                tab === tb.key
                  ? 'border-primary text-foreground'
                  : 'border-transparent text-muted-foreground hover:text-foreground'
              }`}
              onClick={() => setTab(tb.key)}
            >
              {t(tb.labelKey)}
            </button>
          ))}
        </div>

        {/* Info tab */}
        {tab === 'info' && (
          <div className="space-y-4">
            <div className="space-y-2">
              <Label htmlFor="settings-name">{t('work.projects.projectName')}</Label>
              <Input
                id="settings-name"
                value={name}
                onChange={(e) => setName(e.target.value)}
              />
            </div>
            <div className="space-y-2">
              <Label htmlFor="settings-description">{t('work.projects.description')}</Label>
              <Textarea
                id="settings-description"
                value={description}
                onChange={(e) => setDescription(e.target.value)}
                rows={3}
              />
            </div>
            <Button
              onClick={handleSaveInfo}
              disabled={updateProject.isPending}
              className="gap-1"
            >
              <Save className="h-4 w-4" />
              {updateProject.isPending ? t('work.settings.saving') : t('common.save')}
            </Button>

            <Separator />

            {/* Template */}
            <div className="space-y-2">
              <Label>{t('work.settings.saveAsTemplate')}</Label>
              <div className="flex gap-2">
                <Input
                  placeholder={t('work.settings.templateNamePlaceholder')}
                  value={templateName}
                  onChange={(e) => setTemplateName(e.target.value)}
                />
                <Button
                  variant="outline"
                  onClick={handleSaveAsTemplate}
                  disabled={saveAsTemplate.isPending || !templateName.trim()}
                  className="gap-1 shrink-0"
                >
                  <Copy className="h-4 w-4" />
                  {t('common.save')}
                </Button>
              </div>
            </div>

            <Separator />

            {/* Archive */}
            <Button
              variant="outline"
              className="text-destructive gap-1"
              onClick={handleArchive}
              disabled={archiveProject.isPending}
            >
              <Archive className="h-4 w-4" />
              {t('work.settings.archiveProject')}
            </Button>
          </div>
        )}

        {/* Statuses tab */}
        {tab === 'statuses' && (
          <div className="space-y-4">
            <div className="space-y-2">
              {statuses.map((status) => (
                <div key={status.id} className="flex items-center gap-2">
                  <div
                    className="h-4 w-4 rounded-full shrink-0 border border-border"
                    style={{ backgroundColor: status.color || '#6b7280' }}
                  />
                  <span className="flex-1 text-sm">{status.name}</span>
                  {status.is_default && (
                    <Badge variant="secondary" className="text-xs">{t('work.projects.default')}</Badge>
                  )}
                  {status.is_closed && (
                    <Badge variant="outline" className="text-xs">{t('work.projects.done')}</Badge>
                  )}
                  <Button
                    variant="ghost"
                    size="icon"
                    className="h-7 w-7 text-destructive"
                    onClick={() => {
                      if (confirm(t('work.settings.deleteStatusConfirm', { name: status.name }))) {
                        deleteStatus.mutate(status.id ?? '')
                      }
                    }}
                  >
                    <Trash2 className="h-3 w-3" />
                  </Button>
                </div>
              ))}
            </div>

            <Separator />

            {/* Add new status */}
            <div className="flex items-center gap-2">
              <input
                type="color"
                value={newStatusColor}
                onChange={(e) => setNewStatusColor(e.target.value)}
                className="h-8 w-8 cursor-pointer rounded border border-border"
              />
              <Input
                placeholder={t('work.settings.newStatusPlaceholder')}
                value={newStatusName}
                onChange={(e) => setNewStatusName(e.target.value)}
                className="flex-1"
                onKeyDown={(e) => {
                  if (e.key === 'Enter') {
                    e.preventDefault()
                    handleAddStatus()
                  }
                }}
              />
              <Button
                variant="outline"
                size="sm"
                onClick={handleAddStatus}
                disabled={createStatus.isPending || !newStatusName.trim()}
                className="gap-1"
              >
                <Plus className="h-3 w-3" />
                {t('work.settings.add')}
              </Button>
            </div>
          </div>
        )}

        {/* Members tab */}
        {tab === 'members' && (
          <div className="space-y-4">
            {members.length === 0 ? (
              <p className="text-sm text-muted-foreground">
                {t('work.settings.noMembers')}
              </p>
            ) : (
              <div className="space-y-2">
                {members.map((member) => (
                  <div key={member.user_id} className="flex items-center gap-3">
                    <div className="flex-1 min-w-0">
                      <p className="text-sm font-medium truncate">
                        {member.first_name} {member.last_name}
                      </p>
                    </div>
                    <Badge variant="outline" className="text-xs shrink-0">
                      {member.role === 'owner'
                        ? t('work.settings.roleOwner')
                        : member.role === 'member'
                        ? t('work.settings.roleMember')
                        : t('work.settings.roleViewer')}
                    </Badge>
                    {member.role !== 'owner' && (
                      <Button
                        variant="ghost"
                        size="icon"
                        className="h-7 w-7 text-destructive"
                        onClick={() => {
                          if (confirm(t('work.settings.removeMemberConfirm', { name: `${member.first_name} ${member.last_name}` }))) {
                            removeMember.mutate({
                              projectId,
                              userId: member.user_id ?? '',
                            })
                          }
                        }}
                      >
                        <X className="h-3 w-3" />
                      </Button>
                    )}
                  </div>
                ))}
              </div>
            )}
          </div>
        )}
      </DialogContent>
    </Dialog>
  )
}
