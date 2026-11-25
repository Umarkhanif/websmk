-- Finance cutover: the app-level invoice ID is needed to map normalized rows
-- back to the UI/blob IDs. Ledger display metadata lives in its own table
-- because financial_audit_ledgers is append-only and cannot be backfilled with
-- UPDATE.
ALTER TABLE tuition_invoices ADD COLUMN IF NOT EXISTS app_id TEXT;

CREATE UNIQUE INDEX IF NOT EXISTS tuition_invoices_app_id_key
    ON tuition_invoices (app_id) WHERE app_id IS NOT NULL AND app_id <> '';

CREATE TABLE IF NOT EXISTS financial_ledger_details (
    ledger_id BIGINT PRIMARY KEY REFERENCES financial_audit_ledgers(id) ON DELETE CASCADE,
    app_invoice_id TEXT NOT NULL DEFAULT '',
    receipt_number TEXT NOT NULL DEFAULT '',
    student_name TEXT NOT NULL DEFAULT '',
    actor_name TEXT NOT NULL DEFAULT '',
    reversal_approval TEXT NOT NULL DEFAULT '',
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX IF NOT EXISTS financial_ledger_details_invoice_idx
    ON financial_ledger_details (app_invoice_id);
