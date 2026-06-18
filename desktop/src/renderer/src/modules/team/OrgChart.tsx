import { useState, useMemo } from 'react'
import { useTranslation } from 'react-i18next'
import {
  ChevronDown,
  ChevronRight,
  Mail,
  Phone,
  Users,
  Building2,
  Crown,
  Search,
  ZoomIn,
  ZoomOut,
  Maximize2,
} from 'lucide-react'
import { useEmployees } from '@/api/hooks/hr-hooks'
import { useMeetingsStore } from '@/stores/meetings'
import { useNavigationStore } from '@/stores/navigation'
import type { EmployeeProfile } from '@/api/hr-types'
import { EmptyState, InlineStat, SkeletonList } from '@/components/shared'
import { DetailModal } from '@/components/shared/DetailModal'

// ============================================================
// Types
// ============================================================

interface OrgNode {
  id: string
  name: string
  initials: string
  role: string
  department: string
  email: string
  children: OrgNode[]
  isManager?: boolean
}

// ============================================================
// Build tree from EmployeeProfile[]
// ============================================================

function buildOrgTree(employees: EmployeeProfile[]): OrgNode[] {
  // Build a map of userId -> OrgNode
  const nodeMap = new Map<string, OrgNode>()
  for (const emp of employees) {
    const name = emp.userName ?? 'N/A'
    const parts = name.split(' ')
    const initials = parts.map((p) => p[0]).join('').slice(0, 2).toUpperCase()
    nodeMap.set(emp.userId, {
      id: emp.id,
      name,
      initials,
      role: emp.positionTitle ?? '',
      department: emp.department ?? 'Sonstige',
      email: emp.userEmail ?? '',
      children: [],
      isManager: false,
    })
  }

  // Build parent-child relationships
  const rootNodes: OrgNode[] = []
  for (const emp of employees) {
    const node = nodeMap.get(emp.userId)
    if (!node) continue

    if (emp.managerUserId && nodeMap.has(emp.managerUserId)) {
      const parent = nodeMap.get(emp.managerUserId)!
      parent.children.push(node)
      parent.isManager = true
    } else {
      rootNodes.push(node)
    }
  }

  return rootNodes
}

// ============================================================
// Helpers
// ============================================================

const DEPT_COLORS: Record<string, { bg: string; text: string; border: string }> = {
  Management: { bg: 'bg-primary-light', text: 'text-primary', border: 'border-primary/30' },
  Entwicklung: { bg: 'bg-info-light', text: 'text-info', border: 'border-info/30' },
  Design: { bg: 'bg-warning-light', text: 'text-warning', border: 'border-warning/30' },
  Marketing: { bg: 'bg-success-light', text: 'text-success', border: 'border-success/30' },
  Vertrieb: { bg: 'bg-error-light', text: 'text-error', border: 'border-error/30' },
  'Human Resources': { bg: 'bg-secondary', text: 'text-foreground', border: 'border-border' },
}

const DEFAULT_DEPT_COLOR = { bg: 'bg-secondary', text: 'text-foreground', border: 'border-border' }

function countNodes(nodes: OrgNode[]): number {
  return nodes.reduce((s, n) => s + 1 + countNodes(n.children), 0)
}

function flattenNodes(nodes: OrgNode[]): OrgNode[] {
  return nodes.flatMap((n) => [n, ...flattenNodes(n.children)])
}

function getMaxDepth(nodes: OrgNode[], depth = 1): number {
  if (nodes.length === 0) return depth
  return Math.max(...nodes.map((n) => getMaxDepth(n.children, depth + 1)))
}

// ============================================================
// OrgNode Card Component
// ============================================================

function OrgNodeCard({
  node,
  level,
  collapsedIds,
  onToggle,
  onSelect,
  searchHighlight,
}: {
  node: OrgNode
  level: number
  collapsedIds: Set<string>
  onToggle: (id: string) => void
  onSelect: (node: OrgNode) => void
  searchHighlight: string
}) {
  const isCollapsed = collapsedIds.has(node.id)
  const hasChildren = node.children.length > 0
  const colors = DEPT_COLORS[node.department] ?? DEFAULT_DEPT_COLOR
  const isHighlighted = searchHighlight && node.name.toLowerCase().includes(searchHighlight.toLowerCase())

  return (
    <div className={`${level > 0 ? 'ml-8' : ''}`}>
      {/* Connector line */}
      {level > 0 && (
        <div className="relative">
          <div className="absolute left-[-20px] top-0 bottom-1/2 w-px bg-border" />
          <div className="absolute left-[-20px] top-1/2 w-5 h-px bg-border" />
        </div>
      )}

      {/* Card */}
      <div
        className={`inline-flex items-center gap-3 rounded-lg border bg-card p-3 mb-2 cursor-pointer transition-all hover:shadow-[var(--shadow-card-hover)] ${
          isHighlighted ? 'ring-2 ring-primary border-primary' : colors.border
        }`}
        onClick={() => onSelect(node)}
      >
        {hasChildren && (
          <button
            onClick={(e) => { e.stopPropagation(); onToggle(node.id) }}
            className="rounded-md p-0.5 text-muted-foreground hover:bg-secondary transition-colors"
          >
            {isCollapsed ? <ChevronRight className="h-3.5 w-3.5" /> : <ChevronDown className="h-3.5 w-3.5" />}
          </button>
        )}

        <div className={`flex h-10 w-10 items-center justify-center rounded-full ${colors.bg} text-sm font-medium ${colors.text}`}>
          {node.isManager ? <Crown className="h-4 w-4" /> : node.initials}
        </div>

        <div>
          <div className="flex items-center gap-2">
            <span className="text-sm font-medium text-foreground">{node.name}</span>
            {node.isManager && node.children.length > 0 && (
              <span className="rounded-full bg-secondary px-1.5 py-0.5 text-[10px] text-muted-foreground">
                {node.children.length} MA
              </span>
            )}
          </div>
          <p className="text-xs text-muted-foreground">{node.role}</p>
          <span className={`inline-block rounded-full px-1.5 py-0.5 text-[9px] font-medium mt-0.5 ${colors.bg} ${colors.text}`}>
            {node.department}
          </span>
        </div>
      </div>

      {/* Children */}
      {hasChildren && !isCollapsed && (
        <div className="relative pl-5 border-l border-border ml-5">
          {node.children.map((child) => (
            <OrgNodeCard
              key={child.id}
              node={child}
              level={level + 1}
              collapsedIds={collapsedIds}
              onToggle={onToggle}
              onSelect={onSelect}
              searchHighlight={searchHighlight}
            />
          ))}
        </div>
      )}
    </div>
  )
}

// ============================================================
// Main Component
// ============================================================

export function OrgChart() {
  const { t } = useTranslation()
  const { startCall } = useMeetingsStore()
  const { setIntent } = useNavigationStore()
  const { data: employeesData, isLoading } = useEmployees()
  const employees = useMemo(() => employeesData?.employees ?? [], [employeesData?.employees])

  const rootNodes = useMemo(() => buildOrgTree(employees), [employees])
  const allNodes = useMemo(() => flattenNodes(rootNodes), [rootNodes])
  const totalEmployees = useMemo(() => countNodes(rootNodes), [rootNodes])
  const departments = useMemo(() => [...new Set(allNodes.map((n) => n.department))], [allNodes])
  const managers = useMemo(() => allNodes.filter((n) => n.isManager), [allNodes])
  const hierarchyLevels = useMemo(() => rootNodes.length > 0 ? getMaxDepth(rootNodes) : 0, [rootNodes])

  const [collapsedIds, setCollapsedIds] = useState<Set<string>>(new Set())
  const [selectedNode, setSelectedNode] = useState<OrgNode | null>(null)
  const [search, setSearch] = useState('')
  const [zoom, setZoom] = useState(100)

  const toggleNode = (id: string) => {
    setCollapsedIds((prev) => {
      const next = new Set(prev)
      if (next.has(id)) next.delete(id)
      else next.add(id)
      return next
    })
  }

  const expandAll = () => setCollapsedIds(new Set())
  const collapseAll = () => {
    const ids = allNodes.filter((n) => n.children.length > 0).map((n) => n.id)
    setCollapsedIds(new Set(ids))
  }

  if (isLoading) {
    return <SkeletonList rows={5} className="max-w-md" />
  }

  if (employees.length === 0) {
    return (
      <EmptyState
        icon={Users}
        title={t('team.orgChart.noEmployees')}
        description={t('team.orgChart.noEmployeesDescription')}
      />
    )
  }

  return (
    <div className="space-y-4">
      {/* KPI Row */}
      <dl className="mb-4 flex flex-wrap items-end gap-x-10 gap-y-3 border-b border-border pb-4">
        <InlineStat label={t('team.orgChart.totalEmployees')} value={totalEmployees} />
        <InlineStat label={t('team.orgChart.departments')} value={departments.length} />
        <InlineStat label={t('team.orgChart.managers')} value={managers.length} />
        <InlineStat label={t('team.orgChart.hierarchyLevels')} value={hierarchyLevels} />
      </dl>

      {/* Toolbar */}
      <div className="flex items-center gap-3">
        <div className="relative flex-1 max-w-sm">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 h-4 w-4 text-muted-foreground" />
          <input
            type="text"
            placeholder={t('team.orgChart.searchPlaceholder')}
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            className="w-full rounded-lg border border-border bg-card pl-9 pr-3 py-2 text-sm text-foreground placeholder:text-input-placeholder focus:outline-none focus:ring-2 focus:ring-focus-ring"
          />
        </div>
        <div className="flex items-center gap-1">
          <button
            onClick={() => setZoom((z) => Math.max(60, z - 10))}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
            title={t('team.orgChart.zoomOut')}
          >
            <ZoomOut className="h-4 w-4" />
          </button>
          <span className="text-xs text-muted-foreground w-10 text-center tabular-nums">{zoom}%</span>
          <button
            onClick={() => setZoom((z) => Math.min(150, z + 10))}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
            title={t('team.orgChart.zoomIn')}
          >
            <ZoomIn className="h-4 w-4" />
          </button>
          <button
            onClick={() => setZoom(100)}
            className="rounded-md p-1.5 text-muted-foreground hover:bg-secondary transition-colors"
            title={t('team.orgChart.resetZoom')}
          >
            <Maximize2 className="h-4 w-4" />
          </button>
        </div>
        <button
          onClick={expandAll}
          className="rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          {t('team.orgChart.expandAll')}
        </button>
        <button
          onClick={collapseAll}
          className="rounded-md border border-border px-3 py-1.5 text-xs text-foreground hover:bg-secondary transition-colors"
        >
          {t('team.orgChart.collapseAll')}
        </button>
      </div>

      {/* Org Chart */}
      <div
        className="rounded-lg border border-border bg-card p-6 overflow-auto min-h-[400px]"
        style={{ transform: `scale(${zoom / 100})`, transformOrigin: 'top left' }}
      >
        {rootNodes.map((node) => (
          <OrgNodeCard
            key={node.id}
            node={node}
            level={0}
            collapsedIds={collapsedIds}
            onToggle={toggleNode}
            onSelect={setSelectedNode}
            searchHighlight={search}
          />
        ))}
      </div>

      {/* Detail Modal (zentriertes Cosmi-Fenster, sticky Header/Close) */}
      <DetailModal
        open={!!selectedNode}
        onClose={() => setSelectedNode(null)}
        title={selectedNode?.name}
        subtitle={selectedNode?.role}
        badge={selectedNode && (
          <span className={`inline-block rounded-full px-2 py-0.5 text-[10px] font-medium ${
            (DEPT_COLORS[selectedNode.department] ?? DEFAULT_DEPT_COLOR).bg
          } ${(DEPT_COLORS[selectedNode.department] ?? DEFAULT_DEPT_COLOR).text}`}>
            {selectedNode.department}
          </span>
        )}
        maxWidth="max-w-lg"
        footer={selectedNode && (
          <div className="flex gap-2">
            <button
              onClick={() => setIntent({ type: 'compose-email', data: { to: selectedNode.email, name: selectedNode.name } })}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <Mail className="h-3.5 w-3.5" />
              {t('team.detail.email', { defaultValue: 'E-Mail' })}
            </button>
            <button
              onClick={() => startCall(selectedNode.name, selectedNode.initials)}
              className="flex-1 flex items-center justify-center gap-1.5 rounded-lg border border-border py-2 text-xs text-foreground hover:bg-secondary transition-colors"
            >
              <Phone className="h-3.5 w-3.5" />
              {t('team.detail.call')}
            </button>
          </div>
        )}
      >
        {selectedNode && (
          <div className="space-y-4">
            <div className="flex items-center gap-3">
              <div className={`flex h-14 w-14 items-center justify-center rounded-full ${
                (DEPT_COLORS[selectedNode.department] ?? DEFAULT_DEPT_COLOR).bg
              } text-base font-bold ${(DEPT_COLORS[selectedNode.department] ?? DEFAULT_DEPT_COLOR).text}`}>
                {selectedNode.isManager ? <Crown className="h-5 w-5" /> : selectedNode.initials}
              </div>
              <div className="min-w-0">
                <h3 className="text-base font-semibold text-foreground">{selectedNode.name}</h3>
                <p className="text-sm text-muted-foreground">{selectedNode.role}</p>
              </div>
            </div>

            <div className="space-y-2.5 text-sm">
              <div className="flex items-center gap-2 text-muted-foreground">
                <Building2 className="h-4 w-4 flex-shrink-0" />
                <span>{selectedNode.department}</span>
              </div>
              <div className="flex items-center gap-2 text-muted-foreground">
                <Mail className="h-4 w-4 flex-shrink-0" />
                <span className="truncate">{selectedNode.email}</span>
              </div>
              {selectedNode.isManager && selectedNode.children.length > 0 && (
                <div className="flex items-center gap-2 text-muted-foreground">
                  <Users className="h-4 w-4 flex-shrink-0" />
                  <span>{selectedNode.children.length} {t('team.orgChart.directReports')}</span>
                </div>
              )}
            </div>

            {selectedNode.children.length > 0 && (
              <div className="pt-3 border-t border-border">
                <p className="text-xs font-medium text-muted-foreground mb-2">{t('team.orgChart.directReportsLabel')}:</p>
                <div className="space-y-1.5">
                  {selectedNode.children.map((child) => (
                    <button
                      key={child.id}
                      onClick={() => setSelectedNode(child)}
                      className="flex items-center gap-2 w-full rounded-md px-2 py-1.5 text-left hover:bg-secondary/50 transition-colors"
                    >
                      <div className="flex h-7 w-7 items-center justify-center rounded-full bg-secondary text-[10px] font-medium text-foreground">
                        {child.initials}
                      </div>
                      <div>
                        <p className="text-xs font-medium text-foreground">{child.name}</p>
                        <p className="text-[10px] text-muted-foreground">{child.role}</p>
                      </div>
                    </button>
                  ))}
                </div>
              </div>
            )}
          </div>
        )}
      </DetailModal>

      {/* Department Legend */}
      <div className="flex flex-wrap items-center gap-3">
        <span className="text-xs font-medium text-muted-foreground">{t('team.orgChart.departments')}:</span>
        {departments.map((dept) => {
          const colors = DEPT_COLORS[dept] ?? DEFAULT_DEPT_COLOR
          return (
            <div key={dept} className="flex items-center gap-1.5">
              <div className={`h-3 w-3 rounded-full ${colors.bg} border ${colors.border}`} />
              <span className="text-xs text-muted-foreground">{dept}</span>
            </div>
          )
        })}
      </div>
    </div>
  )
}
