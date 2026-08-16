ALTER TABLE contacts
    DROP CONSTRAINT IF EXISTS chk_contacts_salutation,
    DROP CONSTRAINT IF EXISTS chk_contacts_category,
    DROP CONSTRAINT IF EXISTS chk_contacts_status;

ALTER TABLE contacts
    DROP COLUMN IF EXISTS salutation,
    DROP COLUMN IF EXISTS title,
    DROP COLUMN IF EXISTS mobile,
    DROP COLUMN IF EXISTS department,
    DROP COLUMN IF EXISTS address_street,
    DROP COLUMN IF EXISTS address_zip,
    DROP COLUMN IF EXISTS address_city,
    DROP COLUMN IF EXISTS address_country,
    DROP COLUMN IF EXISTS website,
    DROP COLUMN IF EXISTS linkedin,
    DROP COLUMN IF EXISTS xing,
    DROP COLUMN IF EXISTS category,
    DROP COLUMN IF EXISTS status;
