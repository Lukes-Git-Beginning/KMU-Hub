import { useState, useEffect } from 'react'
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
import { Textarea } from '@/components/ui/textarea'
import { toast } from 'sonner'
import { useTeamStore, type TeamMember } from '@/stores/team'

interface EditMemberDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  member: TeamMember | null
}

export function EditMemberDialog({ open, onOpenChange, member }: EditMemberDialogProps) {
  const { updateMember, departments } = useTeamStore()

  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [mobile, setMobile] = useState('')
  const [role, setRole] = useState('')
  const [department, setDepartment] = useState('')
  const [contractType, setContractType] = useState<TeamMember['contractType']>('Vollzeit')
  const [workload, setWorkload] = useState(100)
  const [location, setLocation] = useState('')
  const [manager, setManager] = useState('')
  const [notes, setNotes] = useState('')
  const [skills, setSkills] = useState('')

  useEffect(() => {
    if (!member || !open) return
    setFirstName(member.firstName)
    setLastName(member.lastName)
    setEmail(member.email)
    setPhone(member.phone)
    setMobile(member.mobile ?? '')
    setRole(member.role)
    setDepartment(member.department)
    setContractType(member.contractType)
    setWorkload(member.workload)
    setLocation(member.location)
    setManager(member.manager ?? '')
    setNotes(member.notes ?? '')
    setSkills(member.skills.join(', '))
  }, [member, open])

  const handleSave = () => {
    if (!member || !firstName.trim() || !lastName.trim()) return

    updateMember(member.id, {
      firstName: firstName.trim(),
      lastName: lastName.trim(),
      email: email.trim(),
      phone: phone.trim(),
      mobile: mobile.trim() || undefined,
      role: role.trim(),
      department,
      contractType,
      workload,
      location: location.trim(),
      manager: manager.trim() || undefined,
      notes: notes.trim() || undefined,
      skills: skills.split(',').map((s) => s.trim()).filter(Boolean),
    })

    toast.success(`${firstName} ${lastName} aktualisiert`)
    onOpenChange(false)
  }

  if (!member) return null

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>Mitglied bearbeiten</DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-2">
          {/* Name */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Vorname *</Label>
              <Input value={firstName} onChange={(e) => setFirstName(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Nachname *</Label>
              <Input value={lastName} onChange={(e) => setLastName(e.target.value)} />
            </div>
          </div>

          {/* Contact */}
          <div className="space-y-1.5">
            <Label>E-Mail</Label>
            <Input type="email" value={email} onChange={(e) => setEmail(e.target.value)} />
          </div>
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Telefon</Label>
              <Input value={phone} onChange={(e) => setPhone(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Mobil</Label>
              <Input value={mobile} onChange={(e) => setMobile(e.target.value)} />
            </div>
          </div>

          {/* Role & Department */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Rolle</Label>
              <Input value={role} onChange={(e) => setRole(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Abteilung</Label>
              <Select value={department} onValueChange={setDepartment}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {departments.map((d) => (
                    <SelectItem key={d.id} value={d.name}>{d.name}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Contract */}
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label>Vertragsart</Label>
              <Select value={contractType} onValueChange={(v) => setContractType(v as TeamMember['contractType'])}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="Vollzeit">Vollzeit</SelectItem>
                  <SelectItem value="Teilzeit">Teilzeit</SelectItem>
                  <SelectItem value="Praktikum">Praktikum</SelectItem>
                  <SelectItem value="Freelance">Freelance</SelectItem>
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Pensum (%)</Label>
              <Input type="number" min={10} max={100} step={10} value={workload} onChange={(e) => setWorkload(Number(e.target.value))} />
            </div>
            <div className="space-y-1.5">
              <Label>Standort</Label>
              <Input value={location} onChange={(e) => setLocation(e.target.value)} />
            </div>
          </div>

          {/* Manager */}
          <div className="space-y-1.5">
            <Label>Vorgesetzter</Label>
            <Input placeholder="Name des Vorgesetzten" value={manager} onChange={(e) => setManager(e.target.value)} />
          </div>

          {/* Skills */}
          <div className="space-y-1.5">
            <Label>Skills (kommagetrennt)</Label>
            <Input
              placeholder="React, TypeScript, Go..."
              value={skills}
              onChange={(e) => setSkills(e.target.value)}
            />
          </div>

          {/* Notes */}
          <div className="space-y-1.5">
            <Label>Notizen</Label>
            <Textarea
              placeholder="Interne Notizen..."
              value={notes}
              onChange={(e) => setNotes(e.target.value)}
              rows={3}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSave} disabled={!firstName.trim() || !lastName.trim()}>
            Speichern
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
