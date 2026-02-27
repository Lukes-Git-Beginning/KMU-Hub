/**
 * CRM module layout with sub-navigation and nested routing.
 *
 * Provides horizontal tab navigation between CRM sections:
 * Contacts, Companies, Deals, Activities, and Search.
 * Uses React Router's Routes/Route for nested module routing.
 */
import { NavLink, Routes, Route, Navigate } from 'react-router-dom'
import { Users, Building2, TrendingUp, Activity, Search } from 'lucide-react'
import { cn } from '@/lib/cn'

import ContactsListPage from './contacts/ContactsListPage'
import ContactDetailPage from './contacts/ContactDetailPage'
import CompaniesListPage from './companies/CompaniesListPage'
import CompanyDetailPage from './companies/CompanyDetailPage'
import DealsListPage from './deals/DealsListPage'
import DealDetailPage from './deals/DealDetailPage'
import ActivitiesListPage from './activities/ActivitiesListPage'
import CRMSearchPage from './search/CRMSearchPage'

const crmNavItems = [
  { to: '/crm/contacts', icon: Users, label: 'Kontakte' },
  { to: '/crm/companies', icon: Building2, label: 'Unternehmen' },
  { to: '/crm/deals', icon: TrendingUp, label: 'Deals' },
  { to: '/crm/activities', icon: Activity, label: 'Aktivitäten' },
  { to: '/crm/search', icon: Search, label: 'Suche' },
] as const

export default function CRMLayout() {
  return (
    <div className="flex h-full flex-col animate-fade-in">
      {/* Sub-navigation bar */}
      <nav className="flex items-center gap-1 border-b border-border bg-card px-6 py-2">
        {crmNavItems.map((item) => (
          <NavLink
            key={item.to}
            to={item.to}
            className={({ isActive }) =>
              cn(
                'flex items-center gap-2 rounded-lg px-3 py-1.5 text-sm font-medium transition-all',
                isActive
                  ? 'bg-primary/10 text-primary tab-accent-active'
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
          <Route index element={<Navigate to="/crm/contacts" replace />} />
          <Route path="contacts" element={<ContactsListPage />} />
          <Route path="contacts/:id" element={<ContactDetailPage />} />
          <Route path="companies" element={<CompaniesListPage />} />
          <Route path="companies/:id" element={<CompanyDetailPage />} />
          <Route path="deals" element={<DealsListPage />} />
          <Route path="deals/:id" element={<DealDetailPage />} />
          <Route path="activities" element={<ActivitiesListPage />} />
          <Route path="search" element={<CRMSearchPage />} />
        </Routes>
      </div>
    </div>
  )
}
