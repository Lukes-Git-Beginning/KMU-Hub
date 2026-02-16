import { forwardRef } from 'react'
import { Input } from '@/components/ui/input'
import { Label } from '@/components/ui/label'
import { cn } from '@/lib'

interface FormFieldProps extends React.InputHTMLAttributes<HTMLInputElement> {
  label: string
  error?: string
  description?: string
}

export const FormField = forwardRef<HTMLInputElement, FormFieldProps>(
  ({ label, error, description, className, id, required, ...props }, ref) => {
    const fieldId = id || label.toLowerCase().replace(/\s+/g, '-')

    return (
      <div className={cn('space-y-1.5', className)}>
        <Label htmlFor={fieldId} className="text-sm font-medium text-[var(--body)]">
          {label}
          {required && <span className="ml-0.5 text-destructive">*</span>}
        </Label>
        <Input
          ref={ref}
          id={fieldId}
          className={cn(error && 'border-destructive')}
          {...props}
        />
        {description && !error && (
          <p className="text-xs text-[var(--muted)]">{description}</p>
        )}
        {error && <p className="text-xs text-destructive">{error}</p>}
      </div>
    )
  }
)

FormField.displayName = 'FormField'
