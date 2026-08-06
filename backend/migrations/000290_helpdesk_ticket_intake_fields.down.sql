ALTER TABLE tickets
    DROP COLUMN IF EXISTS custom_fields,
    DROP COLUMN IF EXISTS requester_is_external,
    DROP COLUMN IF EXISTS requester_name,
    DROP COLUMN IF EXISTS requester_email,
    DROP COLUMN IF EXISTS channel;
