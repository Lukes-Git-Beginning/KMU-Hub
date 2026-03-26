import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog'
import { cn } from '@/lib'
import { AlertTriangle, Trash2, Info } from 'lucide-react'

interface ConfirmDialogProps {
  open: boolean
  onOpenChange: (open: boolean) => void
  title: string
  description: string
  confirmLabel?: string
  cancelLabel?: string
  variant?: 'default' | 'destructive' | 'warning'
  onConfirm: () => void
}

const variantConfig = {
  default: {
    icon: Info,
    iconBg: 'bg-primary/10',
    iconClass: 'text-primary',
    buttonClass: '',
  },
  destructive: {
    icon: Trash2,
    iconBg: 'bg-error-light',
    iconClass: 'text-destructive',
    buttonClass: 'bg-destructive text-destructive-foreground hover:bg-destructive/90 focus:ring-destructive',
  },
  warning: {
    icon: AlertTriangle,
    iconBg: 'bg-warning-light',
    iconClass: 'text-warning-foreground',
    buttonClass: 'bg-warning text-warning-foreground hover:bg-warning/90 focus:ring-warning',
  },
}

export function ConfirmDialog({
  open,
  onOpenChange,
  title,
  description,
  confirmLabel = 'Bestaetigen',
  cancelLabel = 'Abbrechen',
  variant = 'default',
  onConfirm,
}: ConfirmDialogProps) {
  const config = variantConfig[variant]
  const Icon = config.icon

  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <div className="flex items-start gap-3">
            <div className={cn('mt-0.5 shrink-0 rounded-lg p-2', config.iconBg)}>
              <Icon className={cn('h-5 w-5', config.iconClass)} />
            </div>
            <div className="space-y-2">
              <AlertDialogTitle>{title}</AlertDialogTitle>
              <AlertDialogDescription>{description}</AlertDialogDescription>
            </div>
          </div>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>{cancelLabel}</AlertDialogCancel>
          <AlertDialogAction
            className={cn(config.buttonClass)}
            onClick={onConfirm}
          >
            {confirmLabel}
          </AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
