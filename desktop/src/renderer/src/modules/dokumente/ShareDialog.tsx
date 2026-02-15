import { useState } from 'react'
import {
  Dialog,
  DialogContent,
  DialogHeader,
  DialogTitle,
  DialogFooter,
} from '@/components/ui/dialog'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from '@/components/ui/select'
import { Link, X, Users } from 'lucide-react'
import { toast } from 'sonner'

interface ShareDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  fileName: string
  currentShares: { name: string; permission: 'view' | 'edit' }[]
}

const availableUsers = [
  'Anna Mueller', 'Michael Berg', 'Sarah Klein', 'Thomas Weber',
  'Lisa Schmidt', 'Peter Koch', 'Jonas Diaz', 'Eva Brunner',
]

export function ShareDialog({ open, onOpenChange, fileName, currentShares }: ShareDialogProps) {
  const [search, setSearch] = useState('')
  const [shares, setShares] = useState(currentShares)
  const [permission, setPermission] = useState<'view' | 'edit'>('view')

  const filteredUsers = availableUsers.filter(
    (u) =>
      !shares.some((s) => s.name === u) &&
      u.toLowerCase().includes(search.toLowerCase())
  )

  const addShare = (name: string) => {
    setShares([...shares, { name, permission }])
    setSearch('')
  }

  const removeShare = (name: string) => {
    setShares(shares.filter((s) => s.name !== name))
  }

  const copyLink = () => {
    toast.success('Link kopiert')
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle className="flex items-center gap-2">
            <Users className="h-5 w-5" />
            Datei teilen
          </DialogTitle>
        </DialogHeader>

        <p className="text-sm text-muted-foreground truncate">{fileName}</p>

        <div className="space-y-4 py-2">
          {/* Add person */}
          <div className="space-y-1.5">
            <Label>Person hinzufügen</Label>
            <div className="flex gap-2">
              <Input
                placeholder="Name suchen..."
                value={search}
                onChange={(e) => setSearch(e.target.value)}
                className="flex-1"
              />
              <Select value={permission} onValueChange={(v) => setPermission(v as typeof permission)}>
                <SelectTrigger className="w-32">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="view">Ansehen</SelectItem>
                  <SelectItem value="edit">Bearbeiten</SelectItem>
                </SelectContent>
              </Select>
            </div>
            {search && filteredUsers.length > 0 && (
              <div className="max-h-28 overflow-y-auto rounded-md border bg-card p-1">
                {filteredUsers.map((user) => (
                  <button
                    key={user}
                    onClick={() => addShare(user)}
                    className="flex w-full items-center gap-2 rounded-sm px-2 py-1.5 text-sm hover:bg-secondary"
                  >
                    <span className="flex h-6 w-6 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                      {user.split(' ').map((n) => n[0]).join('')}
                    </span>
                    {user}
                  </button>
                ))}
              </div>
            )}
          </div>

          {/* Current shares */}
          {shares.length > 0 && (
            <div className="space-y-1.5">
              <Label className="text-xs text-muted-foreground">Geteilt mit</Label>
              {shares.map((s) => (
                <div key={s.name} className="flex items-center justify-between rounded-md border border-border px-3 py-2">
                  <div className="flex items-center gap-2">
                    <span className="flex h-7 w-7 items-center justify-center rounded-full bg-primary/10 text-[10px] font-medium text-primary">
                      {s.name.split(' ').map((n) => n[0]).join('')}
                    </span>
                    <span className="text-sm text-foreground">{s.name}</span>
                  </div>
                  <div className="flex items-center gap-1.5">
                    <span className="text-xs text-muted-foreground">
                      {s.permission === 'edit' ? 'Bearbeiten' : 'Ansehen'}
                    </span>
                    <button
                      onClick={() => removeShare(s.name)}
                      className="rounded p-0.5 text-muted-foreground hover:text-red-500"
                    >
                      <X className="h-3.5 w-3.5" />
                    </button>
                  </div>
                </div>
              ))}
            </div>
          )}

          {/* Copy link */}
          <Button variant="outline" className="w-full" onClick={copyLink}>
            <Link className="mr-1.5 h-4 w-4" />
            Link kopieren
          </Button>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>Schliessen</Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
