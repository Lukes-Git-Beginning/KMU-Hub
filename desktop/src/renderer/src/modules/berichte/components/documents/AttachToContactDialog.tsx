import { useTranslation } from 'react-i18next'
import { User } from 'lucide-react'
import { toast } from 'sonner'
import type { ReportDocument } from '@/api/berichte-types'
import { useAttachReportToContact } from '@/api/hooks/useBerichte'
import { useContacts } from '@/api/hooks/useContacts'
import { AttachReportDialog, type AttachItem } from './AttachReportDialog'

interface AttachToContactDialogProps {
  doc: ReportDocument
  open: boolean
  onClose: () => void
}

/** R-5a — attach the report as a reference to a CRM contact. */
export function AttachToContactDialog({ doc, open, onClose }: AttachToContactDialogProps) {
  const { t } = useTranslation()
  const { data, isLoading } = useContacts({ page_size: 200 })
  const attach = useAttachReportToContact()

  const items: AttachItem[] = (data?.contacts ?? []).map((c) => {
    const contact = c as { id: string; firstName?: string; lastName?: string; companyName?: string }
    return {
      id: contact.id,
      label: `${contact.firstName ?? ''} ${contact.lastName ?? ''}`.trim() || contact.id,
      hint: contact.companyName,
    }
  })

  return (
    <AttachReportDialog
      open={open}
      onClose={onClose}
      subtitle={doc.title}
      icon={User}
      title={t('berichte.docs.attach.contactTitle')}
      searchPlaceholder={t('berichte.docs.attach.searchContact')}
      emptyLabel={t('berichte.docs.attach.noContacts')}
      items={items}
      isLoading={isLoading}
      pending={attach.isPending}
      onPick={(item) =>
        attach.mutate(
          { contactId: item.id, doc: { id: doc.id, title: doc.title } },
          {
            onSuccess: () => {
              toast.success(t('berichte.docs.attach.contactDone', { contact: item.label }))
              onClose()
            },
            onError: (err) => toast.error((err as Error).message),
          },
        )
      }
    />
  )
}
