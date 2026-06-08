/**
 * In-app member profile window (large centered dialog).
 *
 * Opened by clicking a member card/row in TeamPage. Hosts the shared
 * MemberProfileContent on a roomy surface so the full profile — contact,
 * employment, leave, modules, payroll, documents — fits comfortably without
 * the cramped side panel.
 */
import { useTranslation } from 'react-i18next'
import { Dialog, DialogContent, DialogTitle } from '@/components/ui/dialog'
import { MemberProfileContent } from './MemberProfileContent'

interface MemberProfileDialogProps {
  memberId: string | null
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function MemberProfileDialog({ memberId, open, onOpenChange }: MemberProfileDialogProps) {
  const { t } = useTranslation()
  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent
        showCloseButton={false}
        className="flex h-[82vh] max-w-3xl flex-col gap-0 overflow-hidden rounded-2xl p-0"
      >
        <DialogTitle className="sr-only">{t('team.detail.title')}</DialogTitle>
        {memberId && (
          <MemberProfileContent
            memberId={memberId}
            onNavigateAway={() => onOpenChange(false)}
            onClose={() => onOpenChange(false)}
          />
        )}
      </DialogContent>
    </Dialog>
  )
}
