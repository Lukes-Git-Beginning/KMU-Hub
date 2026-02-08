# D5: Module Styling — Plan

## Goal

Lukes funktionale Module visuell aufwerten. Konsistentes Design-System ueber alle
Seiten hinweg. Fokus: die Arbeitsoberflaeche soll professionell und angenehm aussehen.

## Tasks

### 1. Dashboard Widgets
- 6 Widgets (RecentContacts, DealPipeline, UnreadMessages, ActivityFeed, QuickActions, NotificationSummary)
- Konsistente Karten-Styles mit Desk-Theme-Integration
- Hover-Effekte, bessere Typografie-Hierarchie
- Edit-Mode visuell deutlicher machen

### 2. CRM Module
- Listen-Seiten (Kontakte, Firmen, Deals): Tabellen aufwerten, Zeilen-Hover, bessere Spacing
- Detail-Seiten: Layout-Verfeinerung, Tabs/Sections, Avatar-Darstellung
- Pipeline-View: Kanban-Spalten visuell aufwerten
- Such-Seite: bessere Ergebnis-Darstellung

### 3. Chat Module
- 3-Panel-Layout: Proportionen, Trennlinien, Scroll-Verhalten
- Nachrichten-Blasen: Design, Timestamp, Read-Receipts
- Channel-Liste: aktiver Channel, Unread-Badge, Hover
- Message-Input: Toolbar-Styling

### 4. Notifications
- NotificationBell: Badge-Animation
- Notification-Center: Item-Cards, Zeitstempel, Icons pro Typ
- Filter-Tabs aufwerten

### 5. Leere Zustaende (Empty States)
- Illustrierte Empty-States fuer jedes Modul
- Passend zum Desk-Theme (z.B. leerer Schreibtisch-Illustration)

## Files (voraussichtlich)

Betrifft hauptsaechlich Module unter:
- modules/dashboard/
- modules/crm/
- modules/chat/
- modules/notifications/
- components/widgets/

Und ggf. neue shared Components:
- components/ui/ (erweiterte shadcn Komponenten)
