-- A sixth exit type for the team page's "deactivate" button.
--
-- Deactivating an employee is an offboard (the personnel record has to say the
-- person left, exit date and type are mandatory there), but the button has no
-- dialog to ask for a reason. 'other' is what it records; the five types from
-- 000282 stay for the offboard dialog.

ALTER TABLE hr_employee_profiles DROP CONSTRAINT chk_hr_employee_exit_type;

ALTER TABLE hr_employee_profiles
    ADD CONSTRAINT chk_hr_employee_exit_type
        CHECK (exit_type IS NULL OR exit_type IN (
            'resignation', 'termination', 'fixed_term_expired',
            'mutual_termination', 'retirement', 'other'
        ));
