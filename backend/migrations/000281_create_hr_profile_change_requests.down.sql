-- Reverse of 000281. Dropping the table loses the change-request history; the
-- employee profiles themselves keep every value an approval already wrote.

DROP TABLE IF EXISTS hr_profile_change_requests;
