import { useState } from 'react'
import { useTranslation } from 'react-i18next'
import { CheckSquare, FolderDown, Link2, Share2, User } from 'lucide-react'
import { Popover, PopoverContent, PopoverTrigger } from '@/components/ui/popover'
import type { ReportDocument } from '@/api/berichte-types'
import { AttachToTaskDialog } from './AttachToTaskDialog'
import { AttachToContactDialog } from './AttachToContactDialog'
import { SaveToDocumentsDialog } from './SaveToDocumentsDialog'
import { ShareLinkDialog } from './ShareLinkDialog'

interface ShareActionsMenuProps {
  doc: ReportDocument
}

/**
 * Consolidated "Teilen / Verteilen" menu in the report reader header (R-5).
 * Bundles the integration actions so the header stays uncluttered: attach to a
 * task (B6-1), to a contact/deal (B6-2), save as PDF to documents (B6-3) and
 * external share links (B6-4/B6-5) land here as they are built.
 */
export function ShareActionsMenu({ doc }: ShareActionsMenuProps) {
  const { t } = useTranslation()
  const [menuOpen, setMenuOpen] = useState(false)
  const [attachTaskOpen, setAttachTaskOpen] = useState(false)
  const [attachContactOpen, setAttachContactOpen] = useState(false)
  const [saveDocOpen, setSaveDocOpen] = useState(false)
  const [shareLinkOpen, setShareLinkOpen] = useState(false)

  const itemCls =
    'flex w-full items-center gap-2.5 rounded-lg px-2 py-1.5 text-xs text-foreground transition-colors hover:bg-secondary'

  return (
    <>
      <Popover open={menuOpen} onOpenChange={setMenuOpen}>
        <PopoverTrigger asChild>
          <button
            type="button"
            className="flex items-center gap-1.5 rounded-lg border border-border px-2.5 py-1.5 text-xs text-muted-foreground transition-colors hover:bg-secondary hover:text-foreground"
          >
            <Share2 className="h-3.5 w-3.5" />
            {t('berichte.docs.share.menu')}
          </button>
        </PopoverTrigger>
        <PopoverContent className="w-56 p-1" align="end" sideOffset={4}>
          <button
            type="button"
            onClick={() => {
              setMenuOpen(false)
              setAttachTaskOpen(true)
            }}
            className={itemCls}
          >
            <CheckSquare className="h-3.5 w-3.5 text-muted-foreground" />
            {t('berichte.docs.attach.taskAction')}
          </button>
          <button
            type="button"
            onClick={() => {
              setMenuOpen(false)
              setAttachContactOpen(true)
            }}
            className={itemCls}
          >
            <User className="h-3.5 w-3.5 text-muted-foreground" />
            {t('berichte.docs.attach.contactAction')}
          </button>
          <div className="my-1 h-px bg-border-muted" />
          <button
            type="button"
            onClick={() => {
              setMenuOpen(false)
              setSaveDocOpen(true)
            }}
            className={itemCls}
          >
            <FolderDown className="h-3.5 w-3.5 text-muted-foreground" />
            {t('berichte.docs.saveDoc.action')}
          </button>
          <button
            type="button"
            onClick={() => {
              setMenuOpen(false)
              setShareLinkOpen(true)
            }}
            className={itemCls}
          >
            <Link2 className="h-3.5 w-3.5 text-muted-foreground" />
            {t('berichte.docs.share.linkAction')}
          </button>
        </PopoverContent>
      </Popover>

      <AttachToTaskDialog doc={doc} open={attachTaskOpen} onClose={() => setAttachTaskOpen(false)} />
      <AttachToContactDialog
        doc={doc}
        open={attachContactOpen}
        onClose={() => setAttachContactOpen(false)}
      />
      <SaveToDocumentsDialog doc={doc} open={saveDocOpen} onClose={() => setSaveDocOpen(false)} />
      <ShareLinkDialog doc={doc} open={shareLinkOpen} onClose={() => setShareLinkOpen(false)} />
    </>
  )
}
