import { LogOut } from 'lucide-react'
import { cn } from '@/lib/cn'
import { useAuthStore } from '@/stores/auth'
import { Avatar, AvatarFallback } from '@/components/ui/avatar'
import { Button } from '@/components/ui/button'
import {
  Tooltip,
  TooltipContent,
  TooltipTrigger,
} from '@/components/ui/tooltip'

const MOCK_USER = {
  firstName: 'Darien',
  lastName: 'Müller',
  email: 'darien@zentria.tech',
  roles: ['admin'],
}

interface SidebarUserProps {
  collapsed: boolean
}

export function SidebarUser({ collapsed }: SidebarUserProps) {
  const storeUser = useAuthStore((s) => s.user)
  const logout = useAuthStore((s) => s.logout)

  const user = storeUser ?? MOCK_USER

  const initials = `${user.firstName.charAt(0)}${user.lastName.charAt(0)}`.toUpperCase()
  const displayName = `${user.firstName} ${user.lastName}`

  return (
    <div className={cn('flex items-center gap-3 p-3', collapsed && 'justify-center')}>
      <Tooltip delayDuration={collapsed ? 100 : 999999}>
        <TooltipTrigger asChild>
          <div className="relative shrink-0">
            <Avatar className="h-8 w-8">
              <AvatarFallback className="bg-sidebar-primary text-sidebar-primary-foreground text-xs">
                {initials}
              </AvatarFallback>
            </Avatar>
            {/* Online status dot */}
            <span className="absolute -bottom-0.5 -right-0.5 h-3 w-3 rounded-full border-2 border-sidebar bg-success" />
          </div>
        </TooltipTrigger>
        {collapsed && (
          <TooltipContent side="right">
            <p>{displayName}</p>
            <p className="text-xs text-muted-foreground">Online</p>
          </TooltipContent>
        )}
      </Tooltip>

      {!collapsed && (
        <>
          <div className="min-w-0 flex-1">
            <p className="truncate text-sm font-medium text-sidebar-foreground">{displayName}</p>
            <p className="truncate text-xs text-sidebar-muted">{user.email}</p>
          </div>
          <Tooltip>
            <TooltipTrigger asChild>
              <Button
                variant="ghost"
                size="icon"
                className="h-8 w-8 shrink-0 text-sidebar-muted hover:bg-sidebar-accent hover:text-sidebar-foreground"
                onClick={() => logout()}
              >
                <LogOut className="h-4 w-4" />
              </Button>
            </TooltipTrigger>
            <TooltipContent>Abmelden</TooltipContent>
          </Tooltip>
        </>
      )}
    </div>
  )
}
