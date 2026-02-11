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
import { Save, Info, TestTube } from 'lucide-react'
import { FormattedMessage } from 'react-intl'
import { toast } from 'sonner'
import { Button } from '@/components/ui/button'
import { Input } from '@/components/ui/input'
import { Switch } from '@/components/ui/switch'
import { Label } from '@/components/ui/label'
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from '@/components/ui/card'
import { useAuthStore } from '@/stores/auth'
import { usePasswordPolicy, useUpdatePasswordPolicy, useValidatePassword } from '@/api/hooks/useSecurity'

/** Strength meter for password testing. */
function StrengthBar({ score }: { score: number }) {
  const level =
    score >= 80
      ? { labelId: 'password.policy.strength.strong', color: 'bg-green-500', pct: 100 }
      : score >= 60
        ? { labelId: 'password.policy.strength.good', color: 'bg-blue-500', pct: 75 }
        : score >= 40
          ? { labelId: 'password.policy.strength.fair', color: 'bg-yellow-500', pct: 50 }
          : { labelId: 'password.policy.strength.weak', color: 'bg-red-500', pct: 25 }

  return (
    <div className="space-y-1">
      <div className="h-2 w-full rounded-full bg-muted">
        <div
          className={`h-2 rounded-full transition-all duration-300 ${level.color}`}
          style={{ width: `${level.pct}%` }}
        />
      </div>
      <p className="text-xs text-muted-foreground">
        <FormattedMessage id={level.labelId} /> ({Math.round(score)} bits)
      </p>
    </div>
  )
}

export default function PasswordPolicyPage() {
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
        onSuccess: () => toast.success(<FormattedMessage id="password.policy.saved" />),
        onError: () => toast.error(<FormattedMessage id="common.error" />),
      }
    )
  }, [
    minLength,
    minEntropy,
    maxAgeDays,
    reuseCount,
    requireUppercase,
    requireLowercase,
    requireDigit,
    requireSpecial,
    updatePolicy,
  ])

  if (!isAdmin) {
    return <Navigate to="/" replace />
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center h-full">
        <p className="text-sm text-muted-foreground">
          <FormattedMessage id="common.loading" />
        </p>
      </div>
    )
  }

  return (
    <div className="h-full overflow-auto">
      <div className="mx-auto max-w-3xl px-6 py-6 space-y-6">
        <div>
          <h1 className="text-2xl font-semibold text-foreground">
            <FormattedMessage id="password.policy.title" />
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            <FormattedMessage id="password.policy.description" />
          </p>
        </div>

        {/* Core Settings */}
        <Card>
          <CardHeader>
            <CardTitle className="text-lg">
              <FormattedMessage id="password.policy.title" />
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-6">
            {/* Min Length */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label>
                  <FormattedMessage id="password.policy.minLength" />
                </Label>
                <Input
                  type="number"
                  min={8}
                  max={128}
                  value={minLength}
                  onChange={(e) => setMinLength(Number(e.target.value))}
                />
                <p className="text-xs text-muted-foreground">
                  <FormattedMessage
                    id="password.policy.minLengthChars"
                    values={{ count: minLength }}
                  />
                </p>
              </div>

              {/* Min Entropy */}
              <div className="space-y-1.5">
                <Label>
                  <FormattedMessage id="password.policy.minEntropy" />
                </Label>
                <Input
                  type="number"
                  min={20}
                  max={200}
                  value={minEntropy}
                  onChange={(e) => setMinEntropy(Number(e.target.value))}
                />
                <p className="text-xs text-muted-foreground">{minEntropy} bits</p>
              </div>
            </div>

            {/* Max Age & Reuse */}
            <div className="grid grid-cols-2 gap-4">
              <div className="space-y-1.5">
                <Label>
                  <FormattedMessage id="password.policy.maxAge" />
                </Label>
                <Input
                  type="number"
                  min={0}
                  max={365}
                  value={maxAgeDays}
                  onChange={(e) => setMaxAgeDays(Number(e.target.value))}
                  placeholder="0 = no expiry"
                />
                {maxAgeDays > 0 && (
                  <p className="text-xs text-muted-foreground">
                    <FormattedMessage
                      id="password.policy.maxAgeDays"
                      values={{ days: maxAgeDays }}
                    />
                  </p>
                )}
              </div>

              <div className="space-y-1.5">
                <Label>
                  <FormattedMessage id="password.policy.reuseCount" />
                </Label>
                <Input
                  type="number"
                  min={0}
                  max={24}
                  value={reuseCount}
                  onChange={(e) => setReuseCount(Number(e.target.value))}
                />
                <p className="text-xs text-muted-foreground">
                  <FormattedMessage
                    id="password.policy.reuseCountPasswords"
                    values={{ count: reuseCount }}
                  />
                </p>
              </div>
            </div>

            {/* Complexity Toggles */}
            <div className="space-y-3">
              <div className="flex items-center justify-between">
                <Label>
                  <FormattedMessage id="password.policy.requireUppercase" />
                </Label>
                <Switch checked={requireUppercase} onCheckedChange={setRequireUppercase} />
              </div>
              <div className="flex items-center justify-between">
                <Label>
                  <FormattedMessage id="password.policy.requireLowercase" />
                </Label>
                <Switch checked={requireLowercase} onCheckedChange={setRequireLowercase} />
              </div>
              <div className="flex items-center justify-between">
                <Label>
                  <FormattedMessage id="password.policy.requireDigit" />
                </Label>
                <Switch checked={requireDigit} onCheckedChange={setRequireDigit} />
              </div>
              <div className="flex items-center justify-between">
                <Label>
                  <FormattedMessage id="password.policy.requireSpecial" />
                </Label>
                <Switch checked={requireSpecial} onCheckedChange={setRequireSpecial} />
              </div>
            </div>

            {/* NIST Info Box */}
            <div className="flex gap-3 rounded-lg bg-blue-50 p-4 dark:bg-blue-950/30">
              <Info className="h-5 w-5 shrink-0 text-blue-600 dark:text-blue-400 mt-0.5" />
              <p className="text-sm text-blue-800 dark:text-blue-300">
                <FormattedMessage id="password.policy.entropyInfo" />
              </p>
            </div>

            {/* Save */}
            <Button onClick={handleSave} disabled={updatePolicy.isPending}>
              <Save className="mr-2 h-4 w-4" />
              <FormattedMessage id="common.save" />
            </Button>
          </CardContent>
        </Card>

        {/* Live Test */}
        <Card>
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-lg">
              <TestTube className="h-5 w-5" />
              <FormattedMessage id="password.policy.testPassword" />
            </CardTitle>
          </CardHeader>
          <CardContent className="space-y-3">
            <div className="flex gap-2">
              <Input
                type="text"
                value={testPassword}
                onChange={(e) => setTestPassword(e.target.value)}
                placeholder="Enter a password to test..."
                className="flex-1"
              />
              <Button
                variant="outline"
                onClick={() => { if (testPassword) validatePw.mutate(testPassword) }}
                disabled={!testPassword || validatePw.isPending}
              >
                <FormattedMessage id="password.policy.testPassword" />
              </Button>
            </div>
            {validatePw.data && (
              <div className="space-y-2">
                <StrengthBar score={validatePw.data.valid ? 80 : Math.max(10, 80 - validatePw.data.failures.length * 20)} />
                {validatePw.data.failures && validatePw.data.failures.length > 0 && (
                  <ul className="text-xs text-destructive space-y-1">
                    {validatePw.data.failures.map((err: string, i: number) => (
                      <li key={i}>- {err}</li>
                    ))}
                  </ul>
                )}
                {validatePw.data.valid && (
                  <p className="text-xs text-green-600">
                    <FormattedMessage id="common.success" />
                  </p>
                )}
              </div>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
