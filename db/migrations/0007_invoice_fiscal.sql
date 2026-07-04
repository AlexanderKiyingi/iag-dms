-- Wave 4: invoice fiscalisation — URA EFRIS receipt + generated document URL.
ALTER TABLE dms_invoices ADD COLUMN IF NOT EXISTS efris_status TEXT NOT NULL DEFAULT '';
ALTER TABLE dms_invoices ADD COLUMN IF NOT EXISTS ura_receipt  TEXT NOT NULL DEFAULT '';
ALTER TABLE dms_invoices ADD COLUMN IF NOT EXISTS document_url TEXT NOT NULL DEFAULT '';
