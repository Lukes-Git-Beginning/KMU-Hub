-- Migration 000041 DOWN: Drop email tables in reverse dependency order

DROP TRIGGER IF EXISTS trg_email_messages_search_vector ON email_messages;
DROP FUNCTION IF EXISTS email_messages_search_vector_update();

DROP TABLE IF EXISTS email_signatures;
DROP TABLE IF EXISTS email_attachments;
DROP TABLE IF EXISTS email_contact_links;
DROP TABLE IF EXISTS email_messages;
DROP TABLE IF EXISTS email_folders;
DROP TABLE IF EXISTS email_accounts;
