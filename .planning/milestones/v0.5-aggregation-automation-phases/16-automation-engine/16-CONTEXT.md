# Phase 16: Automation Engine - Context

**Gathered:** 2026-02-20
**Status:** Ready for planning

<domain>
## Phase Boundary

Users can automate repetitive workflows across all Hub modules using trigger-condition-action rules. This is the "killer feature" of an all-in-one platform — cross-module automation that standalone tools can't match. Builds on Phase 14's event infrastructure (LISTEN/NOTIFY). Scope: workflow builder, trigger/action registries, condition engine, execution logs, pre-built templates. NOT in scope: external integrations (Zapier/Make-style), AI-driven automations, or plugin-triggered workflows (Phase 20).

</domain>

<decisions>
## Implementation Decisions

### Workflow Builder UX
- **Hybrid approach**: Simple wizard for basic automations, full visual node editor (react-flow) for complex multi-branch flows
- **Wizard model**: 4-step linear flow — Trigger → Condition → Action(s) → Review. Allows multiple actions per trigger. Review step shows full summary before saving
- **Advanced mode**: Full react-flow node editor with draggable nodes, connectors, zoom/pan. User can switch between wizard and canvas views
- **Access model**: Role-based with scope — admins create org-wide automations, managers create for their team, members create personal automations (only affecting their own items)

### Trigger & Action Catalog
- **Full cross-module coverage from v1**: All four module groups (CRM, Email/Inbox, Finance, HR/Calendar) get trigger and action support
- **Priority CRM triggers**: Deal stage change, new contact/company created
- **Cross-module actions**: Always allowed — a CRM trigger can fire an Email action, a Finance action, a Calendar action, etc. No restrictions on combinations
- **Event mechanism**: Hybrid — most triggers subscribe to Phase 14 LISTEN/NOTIFY events directly; time-based triggers (e.g., "invoice overdue for 7 days") use scheduled polling checks

### Pre-built Templates
- **10-15 templates** shipping in v1, fully editable — broad coverage across all modules
- **Discovery**: Both a browsable template gallery AND contextual suggestions (on first use, on empty state, and contextually when relevant)
- **Must-have templates**: "Invoice overdue → Dunning workflow" and "Leave approved → Calendar event + team notification" are essential
- **Organization**: Default grouping by module/use case (Vertrieb, Personal, Finanzen, Kommunikation) with toggle to switch to complexity-based grouping (Einfach, Mittel, Fortgeschritten)

### Conditional Logic Depth
- **Expression language support** (e.g., expr-lang) — full power including string operations, date math, regex
- **UI-first with expression fallback**: Simple conditions use dropdown UI (field, operator, value). Power users can toggle to a raw expression text field with autocomplete and syntax highlighting
- **Chained actions**: Action output feeds into next action (e.g., created invoice ID becomes input to next step). Enables powerful multi-step workflows
- **Soft step limit**: Default maximum of 10 actions per automation chain, admins can raise the limit. Prevents runaway workflows while allowing power users flexibility

### Claude's Discretion
- Exact react-flow node types and edge styling
- Expression language choice (expr-lang vs alternatives)
- Execution log retention and cleanup policy
- Error handling and retry behavior for failed actions
- Template content for the 10-15 pre-built automations beyond the two must-haves
- Polling interval for time-based triggers

</decisions>

<specifics>
## Specific Ideas

- Wizard should feel like a 4-step guided flow, not a complex form — Trigger → Condition → Action(s) → Review
- The visual node editor should use react-flow for the full canvas experience (like n8n)
- Template gallery should support both module-based and complexity-based browsing with a toggle
- Conditions should be accessible via simple dropdowns for most users, with a toggle to raw expression mode for power users
- Cross-module is the core value — "Deal Won → Create Invoice" type flows are what differentiate an all-in-one from standalone tools

</specifics>

<deferred>
## Deferred Ideas

None — discussion stayed within phase scope

</deferred>

---

*Phase: 16-automation-engine*
*Context gathered: 2026-02-20*
