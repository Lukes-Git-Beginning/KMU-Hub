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

export default function WorkLayout() {
  return (
    <div className="flex h-full flex-col">
      <div className="flex-1 overflow-auto">
        <Routes>
          <Route index element={<Navigate to="/work/projects" replace />} />
          <Route path="projects" element={<ProjectsListPage />} />
          <Route path="projects/:id/*" element={<ProjectDetailPage />} />
          <Route path="my-tasks" element={<MyTasksPage />} />
          <Route path="search" element={<TaskSearchView />} />
        </Routes>
      </div>
    </div>
  )
}
