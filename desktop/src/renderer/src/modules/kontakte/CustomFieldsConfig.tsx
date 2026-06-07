/**
 * CustomFieldsConfig — dialog wrapper around CustomFieldsManager.
 *
 * Accessible from KontaktePage (e.g. settings/gear button). The embeddable
 * body lives in CustomFieldsManager (also used by the CRM settings panel).
 */
import { useTranslation } from 'react-i18next'
import { Settings2 } from 'lucide-react'
import { CustomFieldsManager } from './CustomFieldsManager'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

interface CustomFieldsConfigProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function CustomFieldsConfig({ open, onOpenChange }: CustomFieldsConfigProps) {
  const { t } = useTranslation()

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-2xl max-h-[85vh] overflow-y-auto">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Settings2 className="h-5 w-5 text-muted-foreground" />
            {t('kontakte.customField.title')}
          </DialogTitle>
          <DialogDescription>{t('kontakte.customField.description')}</DialogDescription>
        </DialogHeader>

        <div className="pt-2">
          <CustomFieldsManager />
        </div>
      </DialogContent>
    </Dialog>
  )
}
