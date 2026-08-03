-- Offboarding state on the personnel file.
--
-- hr_employee_profiles had no notion of an employee having left: the desktop
-- offboard dialog and its MSW handler both write status/exitDate/exitType, and
-- none of the three columns existed. Without them an offboard could deactivate
-- the account but the personnel record would still read as a current employee.
--
-- status defaults to 'active' so every existing row keeps its meaning; the
-- exit_* columns stay NULL until someone actually leaves.

ALTER TABLE hr_employee_profiles
    ADD COLUMN status        VARCHAR(20) NOT NULL DEFAULT 'active',
    ADD COLUMN last_work_day DATE,
    ADD COLUMN exit_date     DATE,
    ADD COLUMN exit_type     VARCHAR(30),
    ADD COLUMN exit_reason   TEXT;

ALTER TABLE hr_employee_profiles
    ADD CONSTRAINT chk_hr_employee_status
        CHECK (status IN ('active', 'inactive'));

-- The five exit types the offboard dialog offers
-- (desktop/src/renderer/src/modules/team/OffboardEmployeeDialog.tsx). Kept as a
-- CHECK rather than an enum for the same reason the contract type is: the
-- gateway marshals proto enums as numbers and the frontend types this as a
-- string union.
ALTER TABLE hr_employee_profiles
    ADD CONSTRAINT chk_hr_employee_exit_type
        CHECK (exit_type IS NULL OR exit_type IN (
            'resignation', 'termination', 'fixed_term_expired',
            'mutual_termination', 'retirement'
        ));

-- An inactive profile without an exit date is a half-written offboard; the
-- constraint makes that state unreachable rather than merely unlikely.
ALTER TABLE hr_employee_profiles
    ADD CONSTRAINT chk_hr_employee_exit_complete
        CHECK (status = 'active' OR (exit_date IS NOT NULL AND exit_type IS NOT NULL));

-- The employee list filters current staff; without this the predicate is a
-- sequential scan over every profile of the tenant.
CREATE INDEX idx_hr_employee_profiles_status
    ON hr_employee_profiles (tenant_id, status);
