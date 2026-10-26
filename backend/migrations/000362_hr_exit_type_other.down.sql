-- Fails while a profile still carries exit_type 'other'; those rows need a
-- decision (which of the five types they really were) before going back.
ALTER TABLE hr_employee_profiles DROP CONSTRAINT chk_hr_employee_exit_type;

ALTER TABLE hr_employee_profiles
    ADD CONSTRAINT chk_hr_employee_exit_type
        CHECK (exit_type IS NULL OR exit_type IN (
            'resignation', 'termination', 'fixed_term_expired',
            'mutual_termination', 'retirement'
        ));
