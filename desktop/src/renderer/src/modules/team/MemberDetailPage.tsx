/**
 * Deep-link route page for a member profile (/team/member/:id).
 *
 * Thin chrome around the shared MemberProfileContent — a sticky top bar with a
 * back control. The primary way to view a profile is the in-app window
 * (MemberProfileDialog) opened from the team list; this route exists for
 * direct links and full-screen viewing.
 */
import { useParams, useNavigate } from 'react-router-dom'
import { useTranslation } from 'react-i18next'
import { ArrowLeft } from 'lucide-react'
import { Button } from '@/components/ui/button'
import { MemberProfileContent } from './MemberProfileContent'

export default function MemberDetailPage() {
  const { t } = useTranslation()
  const navigate = useNavigate()
  const { id = '' } = useParams<{ id: string }>()

  return (
    <div className="flex h-full flex-col overflow-hidden">
      <div className="flex shrink-0 items-center gap-2 border-b border-border bg-background px-4 py-2.5">
        <Button variant="ghost" size="sm" onClick={() => navigate('/team')}>
          <ArrowLeft className="mr-1 h-4 w-4" />
          {t('common.back')}
        </Button>
        <span className="text-sm text-muted-foreground">{t('team.detail.title')}</span>
      </div>
      <div className="min-h-0 w-full flex-1">
        <MemberProfileContent memberId={id} />
      </div>
    </div>
  )
}
