/**
 * Company detail page showing company info, linked contacts, and activities.
 *
 * Accessed via /crm/companies/:id. Shows company details with
 * clickable linked contacts list and related activities.
 */
import { useState } from 'react'
import { useParams, useNavigate, Link } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import {
  ArrowLeft,
  Pencil,
  Trash2,
  Globe,
  MapPin,
  Factory,
} from 'lucide-react'
import { toast } from 'sonner'
import { useCompany, useCompanyContacts, useUpdateCompany, useDeleteCompany } from '@/api/hooks/useCompanies'
import { useActivities } from '@/api/hooks/useActivities'
import { Button } from '@/components/ui/button'
import { Badge } from '@/components/ui/badge'
import { Card, CardContent, CardHeader, CardTitle } from '@/components/ui/card'
import { Skeleton } from '@/components/ui/skeleton'
import { Separator } from '@/components/ui/separator'
import { ConfirmDialog } from '@/components/shared'
import { activityTypeLabel, activityTypeIcon } from '../activities/activityUtils'
import { CompanyFormDialog, type CompanyFormData } from './CompanyFormDialog'

export default function CompanyDetailPage() {
  const { t } = useTranslation()
  const { id } = useParams<{ id: string }>()
  const navigate = useNavigate()
  const { data, isLoading, error, refetch } = useCompany(id ?? '')
  const { data: contactsData } = useCompanyContacts(id ?? '')
  const { data: activitiesData } = useActivities({
    company_id: id,
    page_size: 10,
  })

  const updateCompany = useUpdateCompany()
  const deleteCompany = useDeleteCompany()
  const [showEditDialog, setShowEditDialog] = useState(false)
  const [showDeleteConfirm, setShowDeleteConfirm] = useState(false)

  const company = data?.company
  const contacts = contactsData?.contacts ?? []
  const activities = activitiesData?.activities ?? []

  function companyToFormData(): Partial<CompanyFormData> {
    if (!company) return {}
    const cf = (company.customFields ?? {}) as Record<string, unknown>
    const addressParts = (company.address ?? '').split(',').map((s: string) => s.trim())
    return {
      name: company.name ?? '',
      industry: company.industry ?? '',
      website: company.website ?? '',
      phone: (cf._phone as string) ?? '',
      email: (cf._email as string) ?? '',
      street: addressParts[0] ?? '',
      zip: addressParts[1]?.split(' ')[0] ?? '',
      city: addressParts[1]?.split(' ').slice(1).join(' ') ?? '',
      country: addressParts[2] ?? 'Deutschland',
      size: (cf._size as string) ?? '',
      notes: company.notes ?? '',
      tags: (cf._tags as string[]) ?? [],
    }
  }

  async function handleUpdate(form: CompanyFormData) {
    if (!id) return
    const addressParts = [form.street, [form.zip, form.city].filter(Boolean).join(' '), form.country].filter(Boolean)
    try {
      await updateCompany.mutateAsync({
        id,
        name: form.name,
        website: form.website || undefined,
        industry: form.industry || undefined,
        address: addressParts.join(', ') || undefined,
        notes: form.notes || undefined,
        custom_fields: {
          ...(form.phone ? { _phone: form.phone } : {}),
          ...(form.email ? { _email: form.email } : {}),
          ...(form.size ? { _size: form.size } : {}),
          ...(form.tags.length > 0 ? { _tags: form.tags } : {}),
        },
      })
      toast.success(t('crm.companies.updated'))
    } catch {
      toast.error(t('crm.companies.updateError'))
    }
  }

  async function handleDelete() {
    if (!id) return
    try {
      await deleteCompany.mutateAsync(id)
      toast.success(t('crm.companies.deleted'))
      navigate('/kontakte/firmen')
    } catch {
      toast.error(t('crm.companies.deleteError'))
    }
  }

  if (error) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('crm.companies.loadErrorSingle')}
          </p>
          <p className="mt-2 text-sm text-muted-foreground">
            {error instanceof Error ? error.message : t('crm.error.unexpected')}
          </p>
          <Button variant="outline" className="mt-4" onClick={() => refetch()}>
            {t('common.retry')}
          </Button>
        </div>
      </div>
    )
  }

  if (isLoading) {
    return (
      <div className="p-6 space-y-4">
        <Skeleton className="h-8 w-48" />
        <Skeleton className="h-64 w-full" />
        <Skeleton className="h-48 w-full" />
      </div>
    )
  }

  if (!company) {
    return (
      <div className="flex h-full items-center justify-center p-6">
        <div className="text-center">
          <p className="text-lg font-semibold text-foreground">
            {t('crm.companies.notFound')}
          </p>
          <Button
            variant="outline"
            className="mt-4"
            onClick={() => navigate('/kontakte/firmen')}
          >
            {t('crm.backToList')}
          </Button>
        </div>
      </div>
    )
  }

  return (
    <div className="p-6 space-y-6">
      {/* Header */}
      <div className="flex items-center justify-between">
        <div className="flex items-center gap-4">
          <Button
            variant="ghost"
            size="sm"
            onClick={() => navigate('/kontakte/firmen')}
          >
            <ArrowLeft className="h-4 w-4 mr-1" />
            {t('common.back')}
          </Button>
          <h1 className="text-2xl font-semibold text-foreground">
            {company.name}
          </h1>
        </div>
        <div className="flex items-center gap-2">
          <Button variant="outline" size="sm" onClick={() => setShowEditDialog(true)}>
            <Pencil className="h-4 w-4 mr-1" />
            {t('common.edit')}
          </Button>
          <Button
            variant="outline"
            size="sm"
            className="text-destructive"
            onClick={() => setShowDeleteConfirm(true)}
          >
            <Trash2 className="h-4 w-4 mr-1" />
            {t('common.delete')}
          </Button>
        </div>
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
        {/* Company Info */}
        <Card className="lg:col-span-2">
          <CardHeader>
            <CardTitle>{t('crm.companies.info')}</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {company.website && (
              <div className="flex items-center gap-3">
                <Globe className="h-4 w-4 text-muted-foreground" />
                <a
                  href={
                    company.website.startsWith('http')
                      ? company.website
                      : `https://${company.website}`
                  }
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-sm text-primary hover:underline"
                >
                  {company.website}
                </a>
              </div>
            )}
            {company.industry && (
              <div className="flex items-center gap-3">
                <Factory className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">{company.industry}</span>
              </div>
            )}
            {company.address && (
              <div className="flex items-center gap-3">
                <MapPin className="h-4 w-4 text-muted-foreground" />
                <span className="text-sm">{company.address}</span>
              </div>
            )}

            {company.notes && (
              <>
                <Separator />
                <div>
                  <p className="text-sm font-medium text-muted-foreground mb-1">
                    {t('crm.field.notes')}
                  </p>
                  <p className="text-sm whitespace-pre-wrap">{company.notes}</p>
                </div>
              </>
            )}

            {company.customFields &&
              Object.keys(company.customFields).length > 0 && (
                <>
                  <Separator />
                  <div>
                    <p className="text-sm font-medium text-muted-foreground mb-2">
                      {t('crm.field.customFields')}
                    </p>
                    <div className="grid grid-cols-2 gap-2">
                      {Object.entries(company.customFields).map(
                        ([key, value]) => (
                          <div key={key}>
                            <p className="text-xs text-muted-foreground">
                              {key}
                            </p>
                            <p className="text-sm">{String(value)}</p>
                          </div>
                        )
                      )}
                    </div>
                  </div>
                </>
              )}
          </CardContent>
        </Card>

        {/* Tags + Contacts sidebar */}
        <div className="space-y-6">
          <Card>
            <CardHeader>
              <CardTitle>{t('crm.field.tags')}</CardTitle>
            </CardHeader>
            <CardContent>
              {company.tags && company.tags.length > 0 ? (
                <div className="flex flex-wrap gap-2">
                  {company.tags.map((tag) => (
                    <Badge
                      key={tag.id}
                      variant="secondary"
                      style={
                        tag.color
                          ? {
                              backgroundColor: `${tag.color}20`,
                              color: tag.color,
                              borderColor: tag.color,
                            }
                          : undefined
                      }
                    >
                      {tag.name}
                    </Badge>
                  ))}
                </div>
              ) : (
                <p className="text-sm text-muted-foreground">
                  {t('crm.tags.none')}
                </p>
              )}
            </CardContent>
          </Card>

          <Card>
            <CardHeader>
              <CardTitle>{t('crm.contacts.title')} ({contacts.length})</CardTitle>
            </CardHeader>
            <CardContent>
              {contacts.length === 0 ? (
                <p className="text-sm text-muted-foreground">
                  {t('crm.companies.noContacts')}
                </p>
              ) : (
                <div className="space-y-2">
                  {contacts.map((contact) => (
                    <Link
                      key={contact.id}
                      to="/kontakte"
                      className="flex items-center gap-2 rounded-md p-2 text-sm hover:bg-accent transition-colors"
                    >
                      <div className="h-8 w-8 rounded-full bg-secondary flex items-center justify-center text-xs font-medium">
                        {(contact.firstName?.charAt(0) ?? '') +
                          (contact.lastName?.charAt(0) ?? '')}
                      </div>
                      <div className="min-w-0">
                        <p className="truncate font-medium">
                          {contact.firstName} {contact.lastName}
                        </p>
                        {contact.email && (
                          <p className="truncate text-xs text-muted-foreground">
                            {contact.email}
                          </p>
                        )}
                      </div>
                    </Link>
                  ))}
                </div>
              )}
            </CardContent>
          </Card>
        </div>
      </div>

      {/* Activities section */}
      <Card>
        <CardHeader>
          <CardTitle>{t('crm.activities.title')}</CardTitle>
        </CardHeader>
        <CardContent>
          {activities.length === 0 ? (
            <p className="text-sm text-muted-foreground">
              {t('crm.companies.noActivities')}
            </p>
          ) : (
            <div className="space-y-3">
              {activities.map((activity) => {
                const Icon = activityTypeIcon(activity.activity_type)
                return (
                  <div
                    key={activity.id}
                    className="flex items-start gap-3 rounded-md border border-border p-3"
                  >
                    <Icon className="mt-0.5 h-4 w-4 text-muted-foreground shrink-0" />
                    <div className="flex-1 min-w-0">
                      <div className="flex items-center gap-2">
                        <p className="text-sm font-medium truncate">
                          {activity.subject}
                        </p>
                        <Badge variant="outline" className="text-xs shrink-0">
                          {t(activityTypeLabel(activity.activity_type))}
                        </Badge>
                      </div>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {activity.created_at
                          ? new Date(activity.created_at).toLocaleDateString(
                              'de-DE',
                              {
                                day: '2-digit',
                                month: '2-digit',
                                year: 'numeric',
                              }
                            )
                          : ''}
                      </p>
                    </div>
                  </div>
                )
              })}
            </div>
          )}
        </CardContent>
      </Card>

      <CompanyFormDialog
        open={showEditDialog}
        onOpenChange={setShowEditDialog}
        initialData={companyToFormData()}
        onSubmit={handleUpdate}
        isEdit
      />

      <ConfirmDialog
        open={showDeleteConfirm}
        onOpenChange={setShowDeleteConfirm}
        title={t('crm.companies.deleteTitle')}
        description={t('crm.companies.deleteConfirm', { name: company.name })}
        confirmLabel={t('common.delete')}
        variant="destructive"
        onConfirm={handleDelete}
      />
    </div>
  )
}
