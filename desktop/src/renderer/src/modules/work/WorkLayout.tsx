/**
 * Work module layout with nested routing.
 *
 * Routes between Work sections: Projects, My Tasks, and Search.
 * Navigation is handled by the main sidebar/topnav — no sub-nav needed.
 */
import { Routes, Route, Navigate } from 'react-router-dom'

import ProjectsListPage from './projects/ProjectsListPage'
import ProjectDetailPage from './projects/ProjectDetailPage'
import MyTasksPage from './tasks/MyTasksPage'
import TaskSearchView from './components/TaskSearchView'
import { useWorkPrefsStore } from '@/stores/workPrefs'

/** Index redirect — honours the user's default-project preference. */
function WorkIndexRedirect() {
  const defaultProjectId = useWorkPrefsStore((s) => s.defaultProjectId)
  const to = defaultProjectId ? `/work/projects/${defaultProjectId}` : '/work/projects'
  return <Navigate to={to} replace />
}

export default function WorkLayout() {
  return (
    <div className="flex h-full flex-col">
      <div className="flex-1 overflow-auto">
        <Routes>
          <Route index element={<WorkIndexRedirect />} />
          <Route path="projects" element={<ProjectsListPage />} />
          <Route path="projects/:id/*" element={<ProjectDetailPage />} />
          <Route path="my-tasks" element={<MyTasksPage />} />
          <Route path="search" element={<TaskSearchView />} />
        </Routes>
      </div>
    </div>
  )
}
