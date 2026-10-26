-- The Report Builder stores the module of its data source. Widen both module
-- CHECK constraints from the six executor modules to every ReportModule the
-- desktop knows (desktop/src/renderer/src/api/berichte-types.ts).
ALTER TABLE report_definitions DROP CONSTRAINT IF EXISTS report_definitions_module_check;
ALTER TABLE report_definitions ADD CONSTRAINT report_definitions_module_check
    CHECK (module IN ('finanzen','crm','helpdesk','inventar','produktion','work','kommunikation',
                      'hr','zeiterfassung','vertraege','einkauf','fuhrpark','rapporte','cross'));

ALTER TABLE report_documents DROP CONSTRAINT IF EXISTS report_documents_module_check;
ALTER TABLE report_documents ADD CONSTRAINT report_documents_module_check
    CHECK (module IN ('finanzen','crm','helpdesk','inventar','produktion','work','kommunikation',
                      'hr','zeiterfassung','vertraege','einkauf','fuhrpark','rapporte','cross'));
