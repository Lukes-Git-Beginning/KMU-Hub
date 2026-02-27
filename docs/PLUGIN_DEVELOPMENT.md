# Plugin Development Guide

## Architecture Overview

KMU Hub's plugin system provides two levels of extensibility:

1. **Config Plugins** (Zero-Code) — Validation rules, workflow rules, and custom fields defined via JSON configuration. ~80% of industry customization needs.
2. **WASM Plugins** (Code) — Complex logic compiled to WebAssembly, running in a sandboxed wazero runtime with limited host API access.

### Service Architecture

```
┌─────────────────────────────────────────────────┐
│                  Gateway (:8080)                │
│         /api/v1/plugins/* endpoints             │
└────────────────────┬────────────────────────────┘
                     │ gRPC
┌────────────────────▼────────────────────────────┐
│              Plugin Service (:50060)            │
│  ┌──────────┐ ┌──────────────┐ ┌────────────┐  │
│  │ Manifests│ │ Installations│ │ KV Store   │  │
│  └──────────┘ └──────────────┘ └────────────┘  │
│  ┌──────────┐ ┌──────────────┐ ┌────────────┐  │
│  │Validation│ │  Workflow     │ │ Execution  │  │
│  │ Engine   │ │  Engine       │ │ Logger     │  │
│  └──────────┘ └──────────────┘ └────────────┘  │
│  ┌──────────────────────────────────────────┐   │
│  │         WASM Runtime (wazero)            │   │
│  │  ┌──────────┐  ┌──────────┐  ┌────────┐ │   │
│  │  │ Sandbox  │  │ Host API │  │ Rate   │ │   │
│  │  │ (64MB,5s)│  │ (16 fns) │  │Limiter │ │   │
│  │  └──────────┘  └──────────┘  └────────┘ │   │
│  └──────────────────────────────────────────┘   │
└─────────────────────────────────────────────────┘
         ▲                    │
         │ ExecuteHooks RPC   │ Graceful Degradation
         │                    ▼
┌────────┴──────┐  ┌────────────────┐  ┌──────────┐
│   CRM Service │  │  Work Service  │  │Biz Service│
│  before/after │  │  before/after  │  │before/after│
│  create/update│  │  create/update │  │create/update│
└───────────────┘  └────────────────┘  └──────────┘
```

**Graceful Degradation**: All services work normally when the Plugin Service is down. Hook calls simply return no modifications.

---

## 1. Config Plugin (Zero-Code)

### Custom Fields

Define custom fields in your manifest or industry template:

```json
{
  "custom_fields": [
    {"name": "license_plate", "label": "Kennzeichen", "type": "text"},
    {"name": "mileage", "label": "Kilometerstand", "type": "number"},
    {"name": "vehicle_type", "label": "Fahrzeugtyp", "type": "select", "options": ["PKW", "LKW"]}
  ]
}
```

Supported types: `text`, `number`, `date`, `select`, `boolean`, `email`, `url`, `phone`

### Validation Rules

Config-based validation rules are evaluated without WASM:

```json
{
  "validation_rules": [
    {
      "name": "Kennzeichen-Format",
      "entity_type": "contact",
      "field_name": "license_plate",
      "rule_type": "regex",
      "rule_config": {"pattern": "^[A-Z]{1,3}-[A-Z]{1,2}\\s?\\d{1,4}$", "required": false},
      "error_message": "Ungültiges Kennzeichen"
    }
  ]
}
```

**Rule Types:**
| Type | Config | Description |
|------|--------|-------------|
| `regex` | `{pattern, required}` | Regular expression match |
| `range` | `{min, max, required}` | Numeric range check |
| `required_if` | `{depends_on, operator, value}` | Conditional required |
| `format` | `{format, required}` | Predefined format (email, url, date, phone, iban) |
| `enum` | `{values, required}` | Allowed values list |

### Workflow Rules

Automate actions based on entity events:

```json
{
  "workflow_rules": [
    {
      "name": "Wartung fällig",
      "trigger_event": "after_update_contact",
      "conditions": [
        {"field": "mileage", "operator": "gte", "value": 15000}
      ],
      "actions": [
        {
          "type": "create_task",
          "config": {"title": "Wartung fällig: {{license_plate}}", "priority": "high"}
        }
      ]
    }
  ]
}
```

**Condition Operators:** `eq`, `neq`, `gt`, `lt`, `gte`, `lte`, `contains`, `starts_with`, `ends_with`, `exists`, `not_exists`

**Action Types:** `create_task`, `send_notification`, `set_field`, `log`

---

## 2. WASM Plugin Development

### Prerequisites

- Go 1.21+ (for `wasip1` build target)
- Plugin SDK at `backend/internal/plugin/sdk/`

### Project Structure

```
plugins/my-plugin/
├── manifest.json      # Plugin manifest
├── main.go           # Entry point (exports handle_hook)
├── handlers.go       # Hook handler implementations
└── calculations.go   # Business logic
```

### manifest.json

```json
{
  "slug": "my-plugin",
  "name": "My Plugin",
  "description": "Description of what it does",
  "version": "1.0.0",
  "author": "Your Name",
  "plugin_type": "wasm",
  "permissions": ["crm:read", "work:write"],
  "settings_schema": {
    "type": "object",
    "properties": {
      "my_setting": {"type": "string", "default": "value"}
    }
  },
  "hook_registrations": [
    {"hook_type": "after_create", "module": "crm", "entity_type": "contact", "priority": 10}
  ]
}
```

### Entry Point

```go
package main

//export handle_hook
func handle_hook() int32 {
    // Read hook request from shared memory
    // Process the hook
    // Return 0 (no modification) or 1 (entity modified)
    return 0
}

func main() {}
```

### Build

```bash
cd plugins/my-plugin
GOOS=wasip1 GOARCH=wasm go build -o my-plugin.wasm .
```

### Host API Functions

Available through the `kmuhub` host module:

| Function | Permission | Description |
|----------|-----------|-------------|
| `kv_get(keyPtr, keyLen) → ptr\|len` | — | Read from KV store |
| `kv_set(keyPtr, keyLen, valPtr, valLen) → status` | — | Write to KV store |
| `kv_delete(keyPtr, keyLen) → status` | — | Delete from KV store |
| `config_get(keyPtr, keyLen) → ptr\|len` | — | Read plugin config |
| `log_info(msgPtr, msgLen)` | — | Log info message |
| `log_warn(msgPtr, msgLen)` | — | Log warning message |
| `log_error(msgPtr, msgLen)` | — | Log error message |

### Sandbox Limits

- **Memory**: 64MB max
- **Execution Time**: 5 seconds max
- **Rate Limit**: 100 host API calls/minute per plugin
- **No filesystem access**
- **No network access**

### Permission Model

Plugins declare required permissions in their manifest. An admin must approve these permissions before the plugin can be activated.

Available permissions:
- `crm:read` — Read contacts, companies, deals, activities
- `crm:write` — Modify CRM entities
- `work:read` — Read tasks and projects
- `work:write` — Create/modify tasks and projects
- `biz:read` — Read invoices, quotes
- `biz:write` — Modify business entities
- `events:read` — Subscribe to system events

---

## 3. Industry Templates

Industry templates are config-only packages that bundle custom fields, validation rules, and workflow rules for specific industries.

### Creating a Template

Templates are stored in the `industry_templates` table. Use the admin UI or API to create them:

```bash
POST /api/v1/plugins/templates
{
  "slug": "my-industry",
  "name": "My Industry",
  "industry": "Category",
  "icon": "icon-name",
  "custom_fields": [...],
  "validation_rules": [...],
  "workflow_rules": [...]
}
```

### Applying a Template

```bash
POST /api/v1/plugins/templates/{template_id}/apply
{
  "tenant_id": "...",
  "applied_by": "..."
}
```

This creates validation rules and workflow rules for the tenant based on the template configuration.

### Built-in Templates

| Template | Industry | Key Features |
|----------|----------|-------------|
| **Handwerk & Bau** | Construction | Baustelle, Gewerk, Auftrags-Nr, Baubeginn-Erinnerung |
| **Beratung & Dienstleistung** | Consulting | Mandat-Nr, Stundensatz, Budget, Budget-Warnung bei 80% |
| **Handel & Vertrieb** | Trade | Liefertermin, Versandart, Tracking-Nr, Kreditlimit |

---

## 4. Fuhrpark Reference Plugin — Walkthrough

The Fuhrpark (fleet management) plugin is a complete reference implementation showing both config and WASM capabilities.

### What it does

- Tracks vehicles as CRM contacts with custom fields (license plate, mileage, inspection date)
- Validates license plate formats for DE/AT/CH
- Creates maintenance tasks when mileage thresholds are reached
- Sends notifications when TÜV/HU inspections are due
- Calculates fuel costs using the KV store

### Files

```
backend/plugins/fuhrpark/
├── manifest.json      # Full manifest with permissions, hooks, fields, rules
├── main.go           # WASM entry point + hook handlers
└── calculations.go   # Fuel cost calculation + maintenance planning
```

### Hook Flow

1. **Contact Created** (`after_create` on `crm:contact`)
   - Checks if contact has `license_plate` custom field
   - Initializes fuel cost tracking in KV store

2. **Contact Updated** (`after_update` on `crm:contact`)
   - Reads mileage, compares with last maintenance
   - If interval exceeded → creates maintenance task
   - Checks TÜV date → sends warning notification

3. **Task Created** (`before_create` on `work:task`)
   - Can enrich auto-created maintenance tasks with vehicle data

### Settings

| Setting | Default | Description |
|---------|---------|-------------|
| `maintenance_interval_km` | 15,000 | Km between maintenance |
| `tuev_warning_days` | 30 | Days before TÜV warning |
| `insurance_warning_days` | 30 | Days before insurance warning |
| `fuel_currency` | EUR | EUR or CHF |
| `country` | DE | For license plate validation |

---

## 5. API Reference

### Plugin Endpoints

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/v1/plugins/manifests` | List all manifests |
| POST | `/api/v1/plugins/manifests` | Create manifest (admin) |
| GET | `/api/v1/plugins/manifests/{id}` | Get manifest |
| DELETE | `/api/v1/plugins/manifests/{id}` | Delete manifest (admin) |
| GET | `/api/v1/plugins/installations` | List installations |
| POST | `/api/v1/plugins/installations` | Install plugin (admin) |
| POST | `/api/v1/plugins/installations/{id}/enable` | Enable (admin) |
| POST | `/api/v1/plugins/installations/{id}/disable` | Disable (admin) |
| DELETE | `/api/v1/plugins/installations/{id}` | Uninstall (admin) |
| POST | `/api/v1/plugins/installations/{id}/permissions` | Approve perms (admin) |
| GET | `/api/v1/plugins/installations/{id}/settings` | Get settings |
| PUT | `/api/v1/plugins/installations/{id}/settings` | Update settings (admin) |
| GET | `/api/v1/plugins/validation-rules` | List rules |
| POST | `/api/v1/plugins/validation-rules` | Create rule (admin) |
| PUT | `/api/v1/plugins/validation-rules/{id}` | Update rule (admin) |
| DELETE | `/api/v1/plugins/validation-rules/{id}` | Delete rule (admin) |
| GET | `/api/v1/plugins/workflow-rules` | List rules |
| POST | `/api/v1/plugins/workflow-rules` | Create rule (admin) |
| PUT | `/api/v1/plugins/workflow-rules/{id}` | Update rule (admin) |
| DELETE | `/api/v1/plugins/workflow-rules/{id}` | Delete rule (admin) |
| GET | `/api/v1/plugins/templates` | List templates |
| POST | `/api/v1/plugins/templates/{id}/apply` | Apply template (admin) |
| GET | `/api/v1/plugins/execution-logs` | List logs (admin) |
