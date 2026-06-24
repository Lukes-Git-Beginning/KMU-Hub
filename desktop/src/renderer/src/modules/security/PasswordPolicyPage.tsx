/**
 * Admin page for configuring organization-wide password policies.
 *
 * Allows setting minimum length, entropy requirements, max age,
 * reuse prevention, and complexity toggles (uppercase, lowercase, digit, special).
 * Includes a live test section using useValidatePassword for real-time feedback.
 * Follows NIST SP 800-63B recommendations for entropy-based validation.
 */
import { useState, useEffect, useCallback } from 'react'
import { Navigate } from 'react-router-dom'
import { KeyRound, Save, Info, TestTube } from 'lucide-react'
import { useTranslation } from 'react-i18next'
import { toast } from 'sonner'
import { Switch } from '@/components/ui/switch'
import { useAuthStore } from '@/stores/auth'
import { usePasswordPolicy, useUpdatePasswordPolicy, useValidatePassword } from '@/api/hooks/useSecurity'

/** Strength meter for password testing. */
function StrengthBar({ score }: { score: number }) {
  const level =
    score >= 80
      ? { labelId: 'password.policy.strength.strong', color: 'bg-success', pct: 100 }
      : score >= 60
        ? { labelId: 'password.policy.strength.good', color: 'bg-info', pct: 75 }
        : score >= 40
          ? { labelId: 'password.policy.strength.fair', color: 'bg-warning', pct: 50 }
          : { labelId: 'password.policy.strength.weak', color: 'bg-error', pct: 25 }

  const { t } = useTranslation()

  return (
    <div className="space-y-1">
      <div className="h-2 w-full rounded-full bg-secondary">
        <div
          className={`h-2 rounded-full transition-all duration-300 ${level.color}`}
          style={{ width: `${level.pct}%` }}
        />
      </div>
      <p className="text-xs text-muted-foreground">
        {t(level.labelId)} ({Math.round(score)} bits)
      </p>
    </div>
  )
}

export default function PasswordPolicyPage() {
  const { t } = useTranslation()
  const user = useAuthStore((s) => s.user)
  const isAdmin = user?.roles.includes('admin')

  const { data: policy, isLoading } = usePasswordPolicy()
  const updatePolicy = useUpdatePasswordPolicy()

  // Form state
  const [minLength, setMinLength] = useState(12)
  const [minEntropy, setMinEntropy] = useState(50)
  const [maxAgeDays, setMaxAgeDays] = useState(0)
  const [reuseCount, setReuseCount] = useState(5)
  const [requireUppercase, setRequireUppercase] = useState(false)
  const [requireLowercase, setRequireLowercase] = useState(false)
  const [requireDigit, setRequireDigit] = useState(false)
  const [requireSpecial, setRequireSpecial] = useState(false)

  // Test password
  const [testPassword, setTestPassword] = useState('')
  const validatePw = useValidatePassword()

  // Sync form from server data
   
  useEffect(() => {
    if (policy) {
      // eslint-disable-next-line react-hooks/set-state-in-effect -- sync form fields from prop/API data
      setMinLength(policy.min_length ?? 12)
      setMinEntropy(policy.min_entropy ?? 50)
      setMaxAgeDays(policy.max_age_days ?? 0)
      setReuseCount(policy.prevent_reuse_count ?? 5)
      setRequireUppercase(policy.require_uppercase ?? false)
      setRequireLowercase(policy.require_lowercase ?? false)
      setRequireDigit(policy.require_digit ?? false)
      setRequireSpecial(policy.require_special ?? false)
    }
  }, [policy])

  const handleSave = useCallback(() => {
    updatePolicy.mutate(
      {
        min_length: minLength,
        min_entropy: minEntropy,
        max_age_days: maxAgeDays || null,
        prevent_reuse_count: reuseCount,
        require_uppercase: requireUppercase,
        require_lowercase: requireLowercase,
        require_digit: requireDigit,
        require_special: requireSpecial,
      } as Parameters<typeof updatePolicy.mutate>[0],
      {
        onSuccess: () => toast.success(t('password.policy.saved')),
        onError: () => toast.error(t('common.error')),
      },
    )
  }, [
    minLength, minEntropy, maxAgeDays, reuseCount,
    requireUppercase, requireLowercase, requireDigit, requireSpecial,
    updatePolicy, t,
  ])

  // Compute policy strength indicator
  const policyStrength = (() => {
    let score = 0
    if (minLength >= 12) score += 25
    else if (minLength >= 8) score += 10
    if (minEntropy >= 50) score += 25
    else if (minEntropy >= 30) score += 10
    if (requireUppercase) score += 10
    if (requireLowercase) score += 10
    if (requireDigit) score += 10
    if (requireSpecial) score += 10
    if (reuseCount >= 5) score += 10
    return score
  })()

  const strengthLabel =
    policyStrength >= 80 ? { text: t('password.strength.strong'), css: 'bg-success-light text-success' }
      : policyStrength >= 50 ? { text: t('password.strength.moderate'), css: 'bg-warning-light text-warning' }
        : { text: t('password.strength.weak'), css: 'bg-error-light text-error' }

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-sm text-muted-foreground">
          {t('common.loading')}
        </p>
      </div>
    )
  }

  const switchRows: { labelId: string; checked: boolean; onChange: (v: boolean) => void }[] = [
    { labelId: 'password.policy.requireUppercase', checked: requireUppercase, onChange: setRequireUppercase },
    { labelId: 'password.policy.requireLowercase', checked: requireLowercase, onChange: setRequireLowercase },
    { labelId: 'password.policy.requireDigit', checked: requireDigit, onChange: setRequireDigit },
    { labelId: 'password.policy.requireSpecial', checked: requireSpecial, onChange: setRequireSpecial },
  ]

  return (
    <div className="flex-1 overflow-y-auto p-6">
      <div className="mx-auto max-w-3xl">
        {/* Header */}
        <div className="flex flex-col sm:flex-row sm:items-center justify-between mb-6 gap-4">
          <div className="flex items-center gap-4">
            <div className="flex h-12 w-12 items-center justify-center rounded-lg bg-primary-light">
              <KeyRound className="h-6 w-6 text-primary" />
            </div>
            <div>
              <h1 className="text-foreground">
                {t('password.policy.title')}
              </h1>
              <div className="flex items-center gap-2 mt-1">
                <span className={`rounded-full px-2.5 py-0.5 text-xs font-medium ${strengthLabel.css}`}>
                  {strengthLabel.text}
                </span>
              </div>
            </div>
          </div>
          <button
            onClick={handleSave}
            disabled={updatePolicy.isPending}
            className="flex items-center gap-2 rounded-lg bg-primary px-4 py-2 text-sm text-primary-foreground hover:bg-button-primary-hover transition-colors disabled:opacity-50"
          >
            <Save className="h-4 w-4" />
            {t('common.save')}
          </button>
        </div>

        {/* Core Settings */}
        <div className="rounded-lg border border-border bg-card p-6 glass-surface mb-6">
          <h3 className="text-base font-semibold text-foreground mb-5">
            {t('password.policy.title')}
          </h3>

          {/* Min Length & Entropy */}
          <div className="grid grid-cols-2 gap-4 mb-5">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('password.policy.minLength')}
              </label>
              <input
                type="number"
                min={8}
                max={128}
                value={minLength}
                onChange={(e) => setMinLength(Number(e.target.value))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground tabular-nums focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
              <p className="text-xs text-muted-foreground">
                {t('password.policy.minLengthChars', { count: minLength })}
              </p>
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('password.policy.minEntropy')}
              </label>
              <input
                type="number"
                min={20}
                max={200}
                value={minEntropy}
                onChange={(e) => setMinEntropy(Number(e.target.value))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground tabular-nums focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
              <p className="text-xs text-muted-foreground">{minEntropy} bits</p>
            </div>
          </div>

          {/* Max Age & Reuse */}
          <div className="grid grid-cols-2 gap-4 mb-5">
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('password.policy.maxAge')}
              </label>
              <input
                type="number"
                min={0}
                max={365}
                value={maxAgeDays}
                onChange={(e) => setMaxAgeDays(Number(e.target.value))}
                placeholder={t('password.noExpiryPlaceholder')}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground tabular-nums placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
              {maxAgeDays > 0 && (
                <p className="text-xs text-muted-foreground">
                  {t('password.policy.maxAgeDays', { days: maxAgeDays })}
                </p>
              )}
            </div>
            <div className="space-y-1.5">
              <label className="text-sm font-medium text-foreground">
                {t('password.policy.reuseCount')}
              </label>
              <input
                type="number"
                min={0}
                max={24}
                value={reuseCount}
                onChange={(e) => setReuseCount(Number(e.target.value))}
                className="w-full rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground tabular-nums focus:outline-none focus:ring-2 focus:ring-focus-ring"
              />
              <p className="text-xs text-muted-foreground">
                {t('password.policy.reuseCountPasswords', { count: reuseCount })}
              </p>
            </div>
          </div>

          {/* Complexity Toggles */}
          <div className="mb-5">
            {switchRows.map((row, i) => (
              <div
                key={row.labelId}
                className={`flex items-center justify-between py-3 ${
                  i < switchRows.length - 1 ? 'border-b border-border-muted' : ''
                }`}
              >
                <label className="text-sm text-foreground">
                  {t(row.labelId)}
                </label>
                <Switch checked={row.checked} onCheckedChange={row.onChange} />
              </div>
            ))}
          </div>

          {/* NIST Info Box */}
          <div className="flex gap-3 rounded-lg bg-info-light p-4">
            <Info className="h-5 w-5 shrink-0 text-info mt-0.5" />
            <p className="text-sm text-info">
              {t('password.policy.entropyInfo')}
            </p>
          </div>
        </div>

        {/* Live Test */}
        <div className="rounded-lg border border-border bg-card p-6 glass-surface">
          <div className="flex items-center gap-2 mb-4">
            <TestTube className="h-5 w-5 text-primary" />
            <h3 className="text-base font-semibold text-foreground">
              {t('password.policy.testPassword')}
            </h3>
          </div>

          <div className="flex gap-2 mb-3">
            <input
              type="text"
              value={testPassword}
              onChange={(e) => setTestPassword(e.target.value)}
              placeholder={t('password.testPlaceholder')}
              className="flex-1 rounded-lg border border-border bg-card px-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
            />
            <button
              onClick={() => { if (testPassword) validatePw.mutate(testPassword) }}
              disabled={!testPassword || validatePw.isPending}
              className="rounded-lg border border-border px-4 py-2 text-sm text-foreground hover:bg-secondary transition-colors disabled:opacity-40"
            >
              {t('password.policy.testPassword')}
            </button>
          </div>

          {validatePw.data && (
            <div className="space-y-2">
              <StrengthBar score={validatePw.data.valid ? 80 : Math.max(10, 80 - validatePw.data.failures.length * 20)} />
              {validatePw.data.failures && validatePw.data.failures.length > 0 && (
                <ul className="text-xs text-error space-y-1">
                  {validatePw.data.failures.map((err: string, i: number) => (
                    <li key={i}>- {err}</li>
                  ))}
                </ul>
              )}
              {validatePw.data.valid && (
                <p className="text-xs text-success font-medium">
                  {t('common.success')}
                </p>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  )
}
