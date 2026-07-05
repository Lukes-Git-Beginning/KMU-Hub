import { useState, useMemo } from 'react'
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
import { Textarea } from '@/components/ui/textarea'
import { toast } from 'sonner'
import { useCreateEmployee, useEmployees } from '@/api/hooks/hr-hooks'
import type { ContractType } from '@/api/hr-types'

const CONTRACT_TYPE_LABELS: Record<ContractType, string> = {
  full_time: 'team.contractType.fullTime',
  part_time: 'team.contractType.partTime',
  mini_job:  'team.contractType.miniJob',
  intern:    'team.contractType.internship',
  temporary: 'team.contractType.temporary',
}

interface InviteMemberDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
}

export function InviteMemberDialog({ open, onOpenChange }: InviteMemberDialogProps) {
  const { t } = useTranslation()
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
  const [contractType, setContractType] = useState<ContractType>('full_time')
  const [workload, setWorkload] = useState(100)
  const [welcomeMessage, setWelcomeMessage] = useState('')

  const reset = () => {
    setFirstName('')
    setLastName('')
    setEmail('')
    setPhone('')
    setRole('')
    setDepartment('')
    setContractType('full_time')
    setWorkload(100)
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
        contractType: contractType,
        workDaysPerWeek: 5,
        annualLeaveDays: 25,
        workloadPercent: workload,
        startDate: new Date().toISOString().split('T')[0],
        addressCountry: 'DE',
        sendInviteEmail: true,
      },
      {
        onSuccess: () => {
          toast.success(t('team.invite.sent', { name: `${firstName} ${lastName}` }))
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
          <DialogTitle>{t('team.invite.title')}</DialogTitle>
        </DialogHeader>

        <div className="space-y-4 py-2">
          {/* Name */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('team.invite.firstName')} *</Label>
              <Input
                autoFocus
                placeholder={t('team.invite.firstName')}
                value={firstName}
                onChange={(e) => setFirstName(e.target.value)}
              />
            </div>
            <div className="space-y-1.5">
              <Label>{t('team.invite.lastName')} *</Label>
              <Input
                placeholder={t('team.invite.lastName')}
                value={lastName}
                onChange={(e) => setLastName(e.target.value)}
              />
            </div>
          </div>

          {/* Email */}
          <div className="space-y-1.5">
            <Label>{t('team.invite.email')} *</Label>
            <Input
              type="email"
              placeholder={t('team.invite.emailPlaceholder')}
              value={email}
              onChange={(e) => setEmail(e.target.value)}
            />
          </div>

          {/* Phone */}
          <div className="space-y-1.5">
            <Label>{t('team.invite.phone')}</Label>
            <Input
              placeholder={t('team.member.phonePlaceholder')}
              value={phone}
              onChange={(e) => setPhone(e.target.value)}
            />
          </div>

          {/* Role & Department */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('team.invite.role')}</Label>
              <Input
                placeholder={t('team.invite.rolePlaceholder')}
                value={role}
                onChange={(e) => setRole(e.target.value)}
              />
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

          {/* Contract & Workload */}
          <div className="grid grid-cols-2 gap-3">
            <div className="space-y-1.5">
              <Label>{t('team.member.contractType')}</Label>
              <Select value={contractType} onValueChange={(v) => setContractType(v as ContractType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {(Object.entries(CONTRACT_TYPE_LABELS) as [ContractType, string][]).map(([k, labelKey]) => (
                    <SelectItem key={k} value={k}>{t(labelKey)}</SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>
            <div className="space-y-1.5">
              <Label>{t('team.member.workload')}</Label>
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

          {/* Welcome message */}
          <div className="space-y-1.5">
            <Label>{t('team.invite.welcomeMessage')}</Label>
            <Textarea
              placeholder={t('team.invite.welcomeMessagePlaceholder')}
              value={welcomeMessage}
              onChange={(e) => setWelcomeMessage(e.target.value)}
              rows={3}
            />
          </div>
        </div>

        <DialogFooter>
          <Button variant="outline" onClick={() => { reset(); onOpenChange(false) }}>
            {t('common.cancel')}
          </Button>
          <Button onClick={handleInvite} disabled={!firstName.trim() || !lastName.trim() || !email.trim() || createEmployee.isPending}>
            {createEmployee.isPending ? t('team.invite.sending') : t('team.invite.sendInvitation')}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
