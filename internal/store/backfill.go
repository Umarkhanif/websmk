package store

import (
	"context"
	"encoding/json"
	"log"
)

// BackfillDomainData copies the core entities from the app_state JSON blob into
// the normalized tables created by migration 003. It is idempotent: existing
// rows are never overwritten, so it is safe to run on every boot.
//
// The application still reads the blob; this is the first step of the domain
// migration. Once endpoints read the tables directly, this can be removed.
func (p *Postgres) BackfillDomainData(ctx context.Context) error {
	raw, err := p.GetState(ctx)
	if err != nil || raw == nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}

	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// app student id -> database UUID, keyed by the unique NISN.
	studentIDs := map[string]string{}
	for _, item := range asListAny(doc["students"]) {
		record := asMapAny(item)
		nisn := textAny(record["nisn"])
		if nisn == "" {
			continue
		}
		userID := textAny(record["userId"])
		_, err := tx.Exec(ctx, `
			INSERT INTO students (user_id, nisn, nik, scholarship_type, discount_percent, attitude_score)
			SELECT $1, $2, $3, $4, $5, $6 WHERE EXISTS (SELECT 1 FROM users WHERE id = $1)
			ON CONFLICT (nisn) DO NOTHING
		`, userID, nisn, textAny(record["nik"]), orText(record["scholarshipType"], "NONE"),
			numberAny(record["discountPercent"]), int64(numberAny(record["attitudeScore"])))
		if err != nil {
			return err
		}
		var id string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM students WHERE nisn = $1`, nisn).Scan(&id); err == nil {
			studentIDs[textAny(record["id"])] = id
		}
	}

	// app invoice id -> database UUID.
	invoiceIDs := map[string]string{}
	for _, item := range asListAny(doc["invoices"]) {
		record := asMapAny(item)
		number := textAny(record["invoiceNumber"])
		studentID, ok := studentIDs[textAny(record["studentId"])]
		if number == "" || !ok {
			continue
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO tuition_invoices (
				invoice_number, student_id, period_month_year, base_amount,
				scholarship_deduction, final_amount, paid_amount, status,
				va_number, gateway_provider
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			ON CONFLICT (invoice_number) DO NOTHING
		`, number, studentID, textAny(record["periodMonthYear"]),
			numberAny(record["baseAmount"]), numberAny(record["scholarshipDeduction"]),
			numberAny(record["finalAmount"]), numberAny(record["paidAmount"]),
			orText(record["status"], "UNPAID"), textAny(record["vaNumber"]),
			textAny(record["gatewayProvider"]))
		if err != nil {
			return err
		}
		var id string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM tuition_invoices WHERE invoice_number = $1`, number).Scan(&id); err == nil {
			invoiceIDs[textAny(record["id"])] = id
		}
	}

	// Ledger is append-only; an explicit ID keeps the backfill idempotent.
	for _, item := range asListAny(doc["ledger"]) {
		record := asMapAny(item)
		id := numberAny(record["id"])
		if id <= 0 {
			continue
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO financial_audit_ledgers (
				id, invoice_id, tx_type, amount_delta, actor_user_id,
				treasurer_pin_verified, signature_hmac, created_at
			) VALUES (
				$1, NULLIF($2, '')::uuid, $3, $4,
				(SELECT id FROM users WHERE id = $5), $6, $7,
				COALESCE(NULLIF($8, '')::timestamptz, now())
			) ON CONFLICT (id) DO NOTHING
		`, int64(id), invoiceIDs[textAny(record["invoiceId"])],
			orText(record["txType"], "UNKNOWN"), numberAny(record["amountDelta"]),
			textAny(record["actorUserId"]), record["treasurerPinVerified"] == true,
			textAny(record["signatureHmac"]), textAny(record["createdAt"]))
		if err != nil {
			return err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return err
	}
	log.Printf("[db] backfill: %d siswa, %d tagihan, %d jurnal keuangan dari app_state",
		len(studentIDs), len(invoiceIDs), len(asListAny(doc["ledger"])))
	return nil
}

func asListAny(value any) []any {
	list, _ := value.([]any)
	return list
}

func asMapAny(value any) map[string]any {
	record, _ := value.(map[string]any)
	return record
}

func textAny(value any) string {
	text, _ := value.(string)
	return text
}

func orText(value any, fallback string) string {
	if text := textAny(value); text != "" {
		return text
	}
	return fallback
}

func numberAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return 0
}
