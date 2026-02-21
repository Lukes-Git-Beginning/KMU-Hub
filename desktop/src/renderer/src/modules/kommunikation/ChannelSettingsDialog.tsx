import { useState } from 'react'
import {
  Mail,
  MessageCircle,
  Globe,
  Headphones,
  Users,
  CheckCircle2,
  XCircle,
  RefreshCw,
} from 'lucide-react'
import { toast } from 'sonner'
import type { CommunicationChannel } from '@/types/communication'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogDescription,
} from '@/components/ui/dialog'

// ---------------------------------------------------------------------------
// Channel config
// ---------------------------------------------------------------------------

interface ChannelConfig {
  id: CommunicationChannel
  label: string
  icon: typeof Mail
  color: string
  connected: boolean
  account?: string
  lastSync?: string
}

const defaultChannels: ChannelConfig[] = [
  { id: 'email', label: 'E-Mail', icon: Mail, color: 'text-blue-500', connected: true, account: 'info@firma.ch', lastSync: '2026-02-20T10:30:00' },
  { id: 'teams', label: 'Microsoft Teams', icon: Users, color: 'text-violet-500', connected: true, account: 'Organisation: Firma AG', lastSync: '2026-02-20T10:28:00' },
  { id: 'whatsapp', label: 'WhatsApp Business', icon: MessageCircle, color: 'text-green-500', connected: false },
  { id: 'widget', label: 'Website-Widget', icon: Globe, color: 'text-orange-500', connected: true, account: 'firma.ch/chat', lastSync: '2026-02-20T10:25:00' },
  { id: 'portal', label: 'Kundenportal', icon: Headphones, color: 'text-teal-500', connected: true, account: 'portal.firma.ch', lastSync: '2026-02-20T09:00:00' },
]

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface ChannelSettingsDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function ChannelSettingsDialog({ open, onOpenChange }: ChannelSettingsDialogProps) {
  const [channels] = useState(defaultChannels)

  const handleConnect = (id: CommunicationChannel) => {
    toast.success(`${id} Verbindung wird eingerichtet... (Mock)`)
  }

  const handleSync = (id: CommunicationChannel) => {
    toast.success(`${id} synchronisiert (Mock)`)
  }

  const formatLastSync = (dateStr?: string): string => {
    if (!dateStr) return ''
    try {
      return new Date(dateStr).toLocaleString('de-CH', {
        day: '2-digit',
        month: '2-digit',
        hour: '2-digit',
        minute: '2-digit',
      })
    } catch {
      return dateStr
    }
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Kanal-Einstellungen</DialogTitle>
          <DialogDescription>
            Verbinde und konfiguriere deine Kommunikationskanaele.
          </DialogDescription>
        </DialogHeader>

        <div className="space-y-2 pt-2">
          {channels.map((ch) => {
            const Icon = ch.icon
            return (
              <div
                key={ch.id}
                className="flex items-center gap-3 rounded-lg border border-border px-3 py-3"
              >
                {/* Icon */}
                <div className={`flex h-8 w-8 shrink-0 items-center justify-center rounded-lg bg-secondary ${ch.color}`}>
                  <Icon className="h-4 w-4" />
                </div>

                {/* Info */}
                <div className="min-w-0 flex-1">
                  <div className="flex items-center gap-1.5">
                    <span className="text-sm font-medium text-foreground">{ch.label}</span>
                    {ch.connected ? (
                      <CheckCircle2 className="h-3.5 w-3.5 text-success shrink-0" />
                    ) : (
                      <XCircle className="h-3.5 w-3.5 text-muted-foreground shrink-0" />
                    )}
                  </div>
                  {ch.connected ? (
                    <div className="text-[11px] text-muted-foreground">
                      {ch.account}
                      {ch.lastSync && (
                        <span> · Sync: {formatLastSync(ch.lastSync)}</span>
                      )}
                    </div>
                  ) : (
                    <span className="text-[11px] text-muted-foreground">Nicht verbunden</span>
                  )}
                </div>

                {/* Action */}
                {ch.connected ? (
                  <button
                    onClick={() => handleSync(ch.id)}
                    className="rounded-md p-1.5 text-muted-foreground hover:bg-accent hover:text-foreground transition-colors"
                    title="Synchronisieren"
                  >
                    <RefreshCw className="h-4 w-4" />
                  </button>
                ) : (
                  <button
                    onClick={() => handleConnect(ch.id)}
                    className="rounded-md bg-primary px-3 py-1.5 text-xs font-medium text-primary-foreground hover:bg-primary/90 transition-colors"
                  >
                    Verbinden
                  </button>
                )}
              </div>
            )
          })}
        </div>
      </DialogContent>
    </Dialog>
  )
}
