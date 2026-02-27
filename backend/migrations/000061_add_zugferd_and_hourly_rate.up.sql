-- ZUGFeRD support: NULL = plain PDF, 'MINIMUM', 'BASIC_WL', 'EN16931'
ALTER TABLE finance_invoices ADD COLUMN IF NOT EXISTS zugferd_profile VARCHAR(20);

-- Audit trail for time-tracking-to-invoice conversions
ALTER TABLE finance_invoices ADD COLUMN IF NOT EXISTS time_tracking_source JSONB;
-- Format: { "employee_id": "...", "date_from": "...", "date_to": "...", "total_minutes": 123, "entry_ids": ["..."] }

-- Hourly rate for billing (per employee profile)
ALTER TABLE hr_employee_profiles ADD COLUMN IF NOT EXISTS hourly_rate DECIMAL(10,2);
