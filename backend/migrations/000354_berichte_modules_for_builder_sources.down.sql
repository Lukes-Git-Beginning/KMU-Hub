-- Rows saved under one of the widened modules would violate the narrow
-- constraint; fold them into 'cross' (the module-neutral bucket) first.
UPDATE report_definitions SET module = 'cross'
 WHERE module NOT IN ('finanzen','crm','helpdesk','inventar','produktion','cross');
ALTER TABLE report_definitions DROP CONSTRAINT IF EXISTS report_definitions_module_check;
ALTER TABLE report_definitions ADD CONSTRAINT report_definitions_module_check
    CHECK (module IN ('finanzen','crm','helpdesk','inventar','produktion','cross'));

UPDATE report_documents SET module = 'cross'
 WHERE module NOT IN ('finanzen','crm','helpdesk','inventar','produktion','cross');
ALTER TABLE report_documents DROP CONSTRAINT IF EXISTS report_documents_module_check;
ALTER TABLE report_documents ADD CONSTRAINT report_documents_module_check
    CHECK (module IN ('finanzen','crm','helpdesk','inventar','produktion','cross'));
