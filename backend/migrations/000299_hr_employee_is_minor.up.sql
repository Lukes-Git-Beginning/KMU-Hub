-- Youth-protection flag on the personnel file (JArbSchG).
--
-- The shift compliance check (`/api/v1/schichten/compliance`) only knew ArbZG
-- §5 rest periods; nothing in the schema said whether an employee is a minor,
-- so the stricter JArbSchG limits (no night work, max 8 hours a day, no
-- weekends) could not be evaluated at all.
--
-- The flag belongs on the person, not on the shift: it is a property of the
-- employee that every shift they are assigned to inherits.
--
-- Defaults to false so every existing profile keeps its current meaning; a
-- profile is only treated as a minor once HR sets the flag.

ALTER TABLE hr_employee_profiles
    ADD COLUMN is_minor BOOLEAN NOT NULL DEFAULT FALSE;

-- The schichten compliance check looks the flag up per (tenant, employee) on
-- every assignment; without this the lookup is a sequential scan.
CREATE INDEX idx_hr_employee_profiles_tenant_minor
    ON hr_employee_profiles (tenant_id, is_minor)
    WHERE is_minor;
