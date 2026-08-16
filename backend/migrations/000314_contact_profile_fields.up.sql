-- The contact form has always shown these fields; the backend never stored them.
--
-- ContactFormDialog collects salutation, academic title, mobile, department, a
-- four-part address, website, LinkedIn, Xing, category and status. Against the
-- real backend the adapter dropped all of them (custom_fields: []) and saving
-- reported success — nine of them silently, which is what G0-6 describes.
-- ContactDetailPanel already renders every one of these, so it starts working
-- the moment they persist.
--
-- Not custom fields: those need an admin to define each field per installation
-- and hand out UUIDs, and models.FieldType has no url or composite-address
-- type. These are ordinary CRM fields, like phone and position already are.
--
-- RLS: contacts carries tenant_id NOT NULL (000070) and a tenant policy
-- (000120). Adding columns inherits both — no new policy needed.

ALTER TABLE contacts
    ADD COLUMN salutation      VARCHAR(10),
    -- Academic title (Dr./Prof.), distinct from `position` (job role). The
    -- OpenAPI spec called `position` "title" and knew no `position` at all,
    -- which is why the adapter needed a cast; that drift is fixed alongside.
    ADD COLUMN title           VARCHAR(50),
    ADD COLUMN mobile          VARCHAR(50),
    ADD COLUMN department      VARCHAR(100),
    ADD COLUMN address_street  VARCHAR(255),
    ADD COLUMN address_zip     VARCHAR(20),
    ADD COLUMN address_city    VARCHAR(100),
    ADD COLUMN address_country VARCHAR(100),
    ADD COLUMN website         VARCHAR(255),
    ADD COLUMN linkedin        VARCHAR(255),
    ADD COLUMN xing            VARCHAR(255),
    -- category and status describe the relationship and its state. They are
    -- deliberately separate from lifecycle_stage / lead_status (000259), which
    -- track the sales pipeline: a contact can be an 'active' 'partner' and have
    -- no lead status at all. Keep the two pairs apart when reading this table.
    ADD COLUMN category        VARCHAR(20),
    ADD COLUMN status          VARCHAR(20);

ALTER TABLE contacts
    ADD CONSTRAINT chk_contacts_salutation
        CHECK (salutation IS NULL OR salutation IN ('Herr', 'Frau')),
    ADD CONSTRAINT chk_contacts_category
        CHECK (category IS NULL OR category IN ('employee', 'customer', 'partner')),
    ADD CONSTRAINT chk_contacts_status
        CHECK (status IS NULL OR status IN ('active', 'prospect', 'inactive'));

COMMENT ON COLUMN contacts.title IS 'Academic title (Dr./Prof.) — job role lives in position';
COMMENT ON COLUMN contacts.category IS 'Relationship type; unrelated to lifecycle_stage (sales pipeline)';
COMMENT ON COLUMN contacts.status IS 'Relationship state; unrelated to lead_status (sales pipeline)';
