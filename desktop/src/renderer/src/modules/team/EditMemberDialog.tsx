import { useState, useEffect, useMemo } from 'react'
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
import { useUpdateEmployee, useEmployees } from '@/api/hooks/hr-hooks'
import type { EmployeeProfile, ContractType } from '@/api/hr-types'

const contractTypeLabels: Record<ContractType, string> = {
  full_time: 'Vollzeit',
  part_time: 'Teilzeit',
  praktikum: 'Praktikum',
  freelance: 'Freelance',
}

interface EditMemberDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  member: EmployeeProfile | null
}

export function EditMemberDialog({ open, onOpenChange, member }: EditMemberDialogProps) {
  const updateEmployee = useUpdateEmployee()
  const { data: employeesData } = useEmployees()

  const departments = useMemo(() => {
    const employees = employeesData?.employees ?? []
    return [...new Set(employees.map((e) => e.department).filter(Boolean))] as string[]
  }, [employeesData])

  // TODO: phone, mobile, skills, notes, location are not in EmployeeProfile — form fields kept for UI but not sent to API
  const [department, setDepartment] = useState('')
  const [contractType, setContractType] = useState<ContractType>('full_time')
  const [positionTitle, setPositionTitle] = useState('')
  const [managerUserId, setManagerUserId] = useState('')
  const [addressStreet, setAddressStreet] = useState('')
  const [addressCity, setAddressCity] = useState('')
  const [addressPostalCode, setAddressPostalCode] = useState('')
  const [addressCountry, setAddressCountry] = useState('CH')
  const [emergencyContactName, setEmergencyContactName] = useState('')
  const [emergencyContactPhone, setEmergencyContactPhone] = useState('')

  useEffect(() => {
    if (!member || !open) return
    setDepartment(member.department ?? '')
    setContractType(member.contractType)
    setPositionTitle(member.positionTitle ?? '')
    setManagerUserId(member.managerUserId ?? '')
    setAddressStreet(member.addressStreet ?? '')
    setAddressCity(member.addressCity ?? '')
    setAddressPostalCode(member.addressPostalCode ?? '')
    setAddressCountry(member.addressCountry ?? 'CH')
    setEmergencyContactName(member.emergencyContactName ?? '')
    setEmergencyContactPhone(member.emergencyContactPhone ?? '')
  }, [member, open])

  const handleSave = () => {
    if (!member) return

    updateEmployee.mutate(
      {
        id: member.id,
        data: {
          department: department || undefined,
          positionTitle: positionTitle || undefined,
          contractType: contractType,
          managerUserId: managerUserId || undefined,
          addressStreet: addressStreet || undefined,
          addressCity: addressCity || undefined,
          addressPostalCode: addressPostalCode || undefined,
          addressCountry: addressCountry || undefined,
          emergencyContactName: emergencyContactName || undefined,
          emergencyContactPhone: emergencyContactPhone || undefined,
        },
      },
      {
        onSuccess: () => {
          onOpenChange(false)
        },
      },
    )
  }

  if (!member) return null

  const memberName = member.userName ?? 'Unbekannt'

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>Mitglied bearbeiten — {memberName}</DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-2">
          {/* Role & Department */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Position</Label>
              <Input value={positionTitle} onChange={(e) => setPositionTitle(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Abteilung</Label>
              <Select value={department} onValueChange={setDepartment}>
                <SelectTrigger>
                  <SelectValue placeholder="Waehlen..." />
                </SelectTrigger>
                <SelectContent>
                  {departments.map((d) => (
                    <SelectItem key={d} value={d}>{d}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Contract */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Vertragsart</Label>
              <Select value={contractType} onValueChange={(v) => setContractType(v as ContractType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(contractTypeLabels).map(([k, v]) => (
                    <SelectItem key={k} value={k}>{v}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>Vorgesetzter (User-ID)</Label>
              <Input placeholder="Optional" value={managerUserId} onChange={(e) => setManagerUserId(e.target.value)} />
            </div>
          </div>

          {/* Address */}
          <div className="space-y-1.5">
            <Label>Strasse</Label>
            <Input placeholder="Musterstrasse 1" value={addressStreet} onChange={(e) => setAddressStreet(e.target.value)} />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label>PLZ</Label>
              <Input placeholder="8000" value={addressPostalCode} onChange={(e) => setAddressPostalCode(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Ort</Label>
              <Input placeholder="Zuerich" value={addressCity} onChange={(e) => setAddressCity(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Land</Label>
              <Select value={addressCountry} onValueChange={setAddressCountry}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="CH">Schweiz</SelectItem>
                  <SelectItem value="DE">Deutschland</SelectItem>
                  <SelectItem value="AT">Oesterreich</SelectItem>
                  <SelectItem value="LI">Liechtenstein</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Emergency Contact */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>Notfallkontakt Name</Label>
              <Input placeholder="Anna Muster" value={emergencyContactName} onChange={(e) => setEmergencyContactName(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>Notfallkontakt Telefon</Label>
              <Input placeholder="+41 79 ..." value={emergencyContactPhone} onChange={(e) => setEmergencyContactPhone(e.target.value)} />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Abbrechen
          </Button>
          <Button onClick={handleSave} disabled={updateEmployee.isPending}>
            {updateEmployee.isPending ? 'Speichert...' : 'Speichern'}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
