import { useState, useEffect, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
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
import { useUpdateEmployee, useEmployees } from '@/api/hooks/hr-hooks'
import type { EmployeeProfile, ContractType } from '@/api/hr-types'

const contractTypeKeys: Record<ContractType, string> = {
  full_time: 'team.contractType.fullTime',
  part_time: 'team.contractType.partTime',
  praktikum: 'team.contractType.internship',
  freelance: 'team.contractType.freelance',
}

interface EditMemberDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  member: EmployeeProfile | null
}

export function EditMemberDialog({ open, onOpenChange, member }: EditMemberDialogProps) {
  const { t } = useTranslation()
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
  const [addressCountry, setAddressCountry] = useState('DE')
  const [emergencyContactName, setEmergencyContactName] = useState('')
  const [emergencyContactPhone, setEmergencyContactPhone] = useState('')

   
  useEffect(() => {
    if (!member || !open) return
    // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
    setDepartment(member.department ?? '')
    setContractType(member.contractType)
    setPositionTitle(member.positionTitle ?? '')
    setManagerUserId(member.managerUserId ?? '')
    setAddressStreet(member.addressStreet ?? '')
    setAddressCity(member.addressCity ?? '')
    setAddressPostalCode(member.addressPostalCode ?? '')
    setAddressCountry(member.addressCountry ?? 'DE')
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

  const memberName = member.userName ?? t('team.member.unknown')

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="max-w-lg max-h-[85vh] overflow-hidden flex flex-col">
        <DialogHeader>
          <DialogTitle>{t('team.member.editTitle', { name: memberName })}</DialogTitle>
        </DialogHeader>

        <div className="flex-1 overflow-y-auto space-y-4 py-2">
          {/* Role & Department */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('team.member.position')}</Label>
              <Input value={positionTitle} onChange={(e) => setPositionTitle(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('team.member.department')}</Label>
              <Select value={department} onValueChange={setDepartment}>
                <SelectTrigger>
                  <SelectValue placeholder={t('team.member.selectPlaceholder')} />
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
              <Label>{t('team.member.contractType')}</Label>
              <Select value={contractType} onValueChange={(v) => setContractType(v as ContractType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {Object.entries(contractTypeKeys).map(([k, v]) => (
                    <SelectItem key={k} value={k}>{t(v)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('team.member.managerUserId')}</Label>
              <Input placeholder={t('common.optional')} value={managerUserId} onChange={(e) => setManagerUserId(e.target.value)} />
            </div>
          </div>

          {/* Address */}
          <div className="space-y-1.5">
            <Label>{t('team.member.street')}</Label>
            <Input placeholder={t('team.member.streetPlaceholder')} value={addressStreet} onChange={(e) => setAddressStreet(e.target.value)} />
          </div>
          <div className="grid grid-cols-3 gap-3">
            <div className="space-y-1.5">
              <Label>{t('team.member.postalCode')}</Label>
              <Input placeholder="10115" value={addressPostalCode} onChange={(e) => setAddressPostalCode(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('team.member.city')}</Label>
              <Input placeholder="Berlin" value={addressCity} onChange={(e) => setAddressCity(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('team.member.country')}</Label>
              <Select value={addressCountry} onValueChange={setAddressCountry}>
                <SelectTrigger><SelectValue /></SelectTrigger>
                <SelectContent>
                  <SelectItem value="DE">{t('team.country.de')}</SelectItem>
                  <SelectItem value="AT">{t('team.country.at')}</SelectItem>
                  <SelectItem value="CH">{t('team.country.ch')}</SelectItem>
                  <SelectItem value="LI">{t('team.country.li')}</SelectItem>
                </SelectContent>
              </Select>
            </div>
          </div>

          {/* Emergency Contact */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('team.member.emergencyContactName')}</Label>
              <Input placeholder={t('team.member.emergencyContactNamePlaceholder')} value={emergencyContactName} onChange={(e) => setEmergencyContactName(e.target.value)} />
            </div>
            <div className="space-y-1.5">
              <Label>{t('team.member.emergencyContactPhone')}</Label>
              <Input placeholder={t('team.member.phonePlaceholder')} value={emergencyContactPhone} onChange={(e) => setEmergencyContactPhone(e.target.value)} />
            </div>
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleSave} disabled={updateEmployee.isPending}>
            {updateEmployee.isPending ? t('team.member.saving') : t('common.save')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
