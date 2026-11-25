package store

import (
	"context"

	"wesmk/internal/domain/finance"
)

const invoiceSelect = `
	SELECT
		COALESCE(i.app_id, ''), COALESCE(s.app_id, ''), i.invoice_number,
		i.period_month_year, i.status, i.va_number, i.gateway_provider,
		i.base_amount::bigint, i.scholarship_deduction::bigint,
		i.final_amount::bigint, i.paid_amount::bigint
	FROM tuition_invoices i
	JOIN students s ON s.id = i.student_id`

func (p *Postgres) InvoicesForStudent(ctx context.Context, studentAppID string) ([]finance.Invoice, error) {
	rows, err := p.pool.Query(ctx, invoiceSelect+`
		WHERE s.app_id = $1
		ORDER BY i.invoice_number DESC`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInvoices(rows)
}

func (p *Postgres) AllInvoices(ctx context.Context) ([]finance.Invoice, error) {
	rows, err := p.pool.Query(ctx, invoiceSelect+`
		ORDER BY i.invoice_number DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanInvoices(rows)
}

type invoiceRows interface {
	Next() bool
	Scan(dest ...any) error
	Err() error
}

func scanInvoices(rows invoiceRows) ([]finance.Invoice, error) {
	invoices := make([]finance.Invoice, 0)
	for rows.Next() {
		var invoice finance.Invoice
		if err := rows.Scan(
			&invoice.AppID,
			&invoice.StudentAppID,
			&invoice.Number,
			&invoice.Period,
			&invoice.Status,
			&invoice.VANumber,
			&invoice.Provider,
			&invoice.BaseAmount,
			&invoice.ScholarshipDeduction,
			&invoice.FinalAmount,
			&invoice.PaidAmount,
		); err != nil {
			return nil, err
		}
		invoices = append(invoices, invoice)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return invoices, nil
}

const ledgerSelect = `
	SELECT
		l.id,
		COALESCE(d.app_invoice_id, ''),
		COALESCE(d.receipt_number, ''),
		l.tx_type,
		l.amount_delta::bigint,
		COALESCE(d.student_name, ''),
		COALESCE(d.actor_name, ''),
		l.treasurer_pin_verified,
		COALESCE(d.reversal_approval, ''),
		l.signature_hmac,
		COALESCE(to_char(l.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD"T"HH24:MI:SS"Z"'), '')
	FROM financial_audit_ledgers l
	LEFT JOIN financial_ledger_details d ON d.ledger_id = l.id
	LEFT JOIN tuition_invoices i ON i.id = l.invoice_id
	LEFT JOIN students s ON s.id = i.student_id`

func (p *Postgres) LedgerForStudent(ctx context.Context, studentAppID string) ([]finance.LedgerEntry, error) {
	rows, err := p.pool.Query(ctx, ledgerSelect+`
		WHERE s.app_id = $1
		ORDER BY l.created_at DESC, l.id DESC`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLedger(rows)
}

func (p *Postgres) AllLedger(ctx context.Context) ([]finance.LedgerEntry, error) {
	rows, err := p.pool.Query(ctx, ledgerSelect+`
		ORDER BY l.created_at DESC, l.id DESC`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanLedger(rows)
}

func scanLedger(rows invoiceRows) ([]finance.LedgerEntry, error) {
	entries := make([]finance.LedgerEntry, 0)
	for rows.Next() {
		var entry finance.LedgerEntry
		if err := rows.Scan(
			&entry.ID,
			&entry.InvoiceAppID,
			&entry.ReceiptNumber,
			&entry.TxType,
			&entry.AmountDelta,
			&entry.StudentName,
			&entry.ActorName,
			&entry.PinVerified,
			&entry.ReversalApproval,
			&entry.SignatureHMAC,
			&entry.CreatedAt,
		); err != nil {
			return nil, err
		}
		entries = append(entries, entry)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return entries, nil
}
