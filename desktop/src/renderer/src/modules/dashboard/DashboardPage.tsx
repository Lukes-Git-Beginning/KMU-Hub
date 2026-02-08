import {
  AlertsSection,
  NotificationsFeed,
  ModulesGrid,
  ActivitySection,
  QuickStatsSection,
} from '@/components/dashboard'

function getGreeting(): string {
  const hour = new Date().getHours()
  if (hour < 12) return 'Guten Morgen'
  if (hour < 18) return 'Guten Tag'
  return 'Guten Abend'
}

export default function DashboardPage() {
  return (
    <div className="h-full overflow-auto">
      <div className="p-4 md:p-8">
        {/* Greeting Header */}
        <div className="mb-8">
          <h1 className="text-2xl font-semibold text-foreground">
            {getGreeting()}
          </h1>
          <p className="mt-1 text-sm text-muted-foreground">
            Willkommen im KMU Digital Hub &ndash; Ihre All-in-One Plattform
          </p>
        </div>

        {/* Alerts */}
        <AlertsSection />

        {/* Notifications Feed */}
        <NotificationsFeed />

        {/* Modules Grid */}
        <ModulesGrid />

        {/* Two-Column: Activity + Stats */}
        <div className="grid grid-cols-1 gap-6 lg:grid-cols-3">
          <div className="lg:col-span-2">
            <ActivitySection />
          </div>
          <QuickStatsSection />
        </div>
      </div>
    </div>
  )
}
