/**
 * Work module layout with sub-navigation and nested routing.
 *
 * Provides horizontal tab navigation between Work sections:
 * Projects and My Tasks (Meine Aufgaben).
 * Uses React Router's Routes/Route for nested module routing.
 */
import { NavLink, Routes, Route, Navigate } from 'react-router-dom'
import { FolderKanban, CheckSquare } from 'lucide-react'
import { cn } from '@/lib/cn'

import ProjectsListPage from './projects/ProjectsListPage'
import ProjectDetailPage from './projects/ProjectDetailPage'
import MyTasksPage from './tasks/MyTasksPage'

const workNavItems = [
  { to: '/work/projects', icon: FolderKanban, label: 'Projekte' },
  { to: '/work/my-tasks', icon: CheckSquare, label: 'Meine Aufgaben' },
] as const

export default function WorkLayout() {
  return (
    <div className="flex h-full flex-col">
      {/* Sub-navigation bar */}
      <nav className="flex items-center gap-1 border-b border-border bg-card px-6 py-2">
        {workNavItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2 rounded-md px-3 py-1.5 text-sm font-medium transition-colors',
                isActive
                  ? 'bg-secondary text-secondary-foreground'
                  : 'text-muted-foreground hover:bg-accent hover:text-accent-foreground'
              )
            }
          >
            <item.icon className="h-4 w-4" />
            <span>{item.label}</span>
          </NavLink>
        ))}
      </nav>

      {/* Content area */}
      <div className="flex-1 overflow-auto">
        <Routes>
          <Route index element={<Navigate to="/work/projects" replace />} />
          <Route path="projects" element={<ProjectsListPage />} />
          <Route path="projects/:id/*" element={<ProjectDetailPage />} />
          <Route path="my-tasks" element={<MyTasksPage />} />
        </Routes>
      </div>
    </div>
  )
}
