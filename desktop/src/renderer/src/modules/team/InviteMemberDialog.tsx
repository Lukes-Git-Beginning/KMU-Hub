import { useState, useMemo } from 'react'
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
import { useCreateEmployee, useEmployees } from '@/api/hooks/hr-hooks'
import type { ContractType } from '@/api/hr-types'

const contractTypeMap: Record<string, ContractType> = {
  Vollzeit: 'full_time',
  Teilzeit: 'part_time',
  Praktikum: 'praktikum',
  Freelance: 'freelance',
}

interface InviteMemberDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function InviteMemberDialog({ open, onOpenChange }: InviteMemberDialogProps) {
  const createEmployee = useCreateEmployee()
  const { data: employeesData } = useEmployees()

  const departments = useMemo(() => {
    const employees = employeesData?.employees ?? []
    return [...new Set(employees.map((e) => e.department).filter(Boolean))] as string[]
  }, [employeesData])

  const [firstName, setFirstName] = useState('')
  const [lastName, setLastName] = useState('')
  const [email, setEmail] = useState('')
  const [phone, setPhone] = useState('')
  const [role, setRole] = useState('')
  const [department, setDepartment] = useState('')
  const [contractType, setContractType] = useState<'Vollzeit' | 'Teilzeit' | 'Praktikum' | 'Freelance'>('Vollzeit')
  const [workload, setWorkload] = useState(100)
  // TODO: location is not in EmployeeProfile — kept for UI but not sent to API
  const [location, setLocation] = useState('Zürich')
  const [welcomeMessage, setWelcomeMessage] = useState('')

  const reset = () => {
    setFirstName('')
    setLastName('')
    setEmail('')
    setPhone('')
    setRole('')
    setDepartment('')
    setContractType('Vollzeit')
    setWorkload(100)
    setLocation('Zürich')
    setWelcomeMessage('')
  }

  const handleInvite = () => {
    if (!firstName.trim() || !lastName.trim() || !email.trim()) return

    createEmployee.mutate(
      {
        firstName: firstName.trim(),
        lastName: lastName.trim(),
        email: email.trim(),
        phone: phone.trim() || undefined,
        temporaryPassword: crypto.randomUUID().slice(0, 12),
        roles: ['member'],
        department: department || undefined,
        positionTitle: role.trim() || undefined,
        contractType: contractTypeMap[contractType],
        workDaysPerWeek: 5,
        annualLeaveDays: 25,
        workloadPercent: workload,
        startDate: new Date().toISOString().split('T')[0],
        addressCountry: 'CH',
        sendInviteEmail: true,
      },
      {
        onSuccess: () => {
          toast.success(`Einladung an ${firstName} ${lastName} gesendet`)
          reset()
          onOpenChange(false)
        },
      },
    )
  }

  return (
    <Dialog open={open} onOpenChange={(v) => { if (!v) reset(); onOpenChange(v) }}>
      <DialogContent className="max-w-md">
        <DialogHeader>
          <DialogTitle>Mitglied einladen</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Name */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Vorname *</Label>
              <Input
                autoFocus
                placeholder="Vorname"
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Nachname *</Label>
              <Input
                placeholder="Nachname"
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>

          {/* Email */}
          <div className="space-y-1.5">
            <Label>E-Mail *</Label>
            <Input
              type="email"
              placeholder="email@firma.ch"
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          {/* Phone */}
          <div className="space-y-1.5">
            <Label>Telefon</Label>
            <Input
              placeholder="+41 79 ..."
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
            />
          </div>

          {/* Role & Department */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Rolle</Label>
              <Input
                placeholder="z.B. Developer"
                value={role}
                onChange={(e) => setRole(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>Abteilung</Label>
              <Select value={department} onValueChange={setDepartment}>
                <SelectTrigger>
                  <SelectValue placeholder="Wählen..." />
                </SelectTrigger>
                <SelectContent>
                  {departments.map((d) => (
                    <SelectItem key={d} value={d}>{d}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Contract & Workload */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Vertragsart</Label>
              <Select value={contractType} onValueChange={(v) => setContractType(v as typeof contractType)}>
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
              <Input
                type="number"
                min={10}
                max={100}
                step={10}
                value={workload}
                onChange={(e) => setWorkload(Number(e.target.value))}
              />
            </div>
          </div>

          {/* Location */}
          <div className="space-y-1.5">
            <Label>Standort</Label>
            <Input
              placeholder="Zürich"
              value={location}
              onChange={(e) => setLocation(e.target.value)}
            />
          </div>

          {/* Welcome message */}
          <div className="space-y-1.5">
            <Label>Willkommensnachricht (optional)</Label>
            <Textarea
              placeholder="Schreibe eine persönliche Nachricht..."
              value={welcomeMessage}
              onChange={(e) => setWelcomeMessage(e.target.value)}
              rows={3}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>
            Abbrechen
          </Button>
          <Button onClick={handleInvite} disabled={!firstName.trim() || !lastName.trim() || !email.trim() || createEmployee.isPending}>
            {createEmployee.isPending ? 'Sendet...' : 'Einladung senden'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
