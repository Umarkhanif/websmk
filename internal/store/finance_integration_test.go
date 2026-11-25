package store

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"testing"
	"time"

	"wesmk/internal/db"
)

// TestFinanceDomainCutover runs against a disposable Postgres when
// WESMK_TEST_DATABASE_URL is set.
func TestFinanceDomainCutover(t *testing.T) {
	dsn := os.Getenv("WESMK_TEST_DATABASE_URL")
	if dsn == "" {
		t.Skip("WESMK_TEST_DATABASE_URL not set")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pool, err := db.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("open: %v", err)
	}
	defer pool.Close()
	if err := db.Migrate(ctx, pool); err != nil {
		t.Fatalf("migrate: %v", err)
	}

	suffix := time.Now().UnixNano()
	userID := fmt.Sprintf("itest-fin-user-%d", suffix)
	studentAppID := fmt.Sprintf("itest-fin-std-%d", suffix)
	invoiceAppID := fmt.Sprintf("itest-fin-inv-%d", suffix)
	invoiceNumber := fmt.Sprintf("ITEST/%d", suffix)
	nisn := fmt.Sprintf("%010d", suffix%10000000000)
	ledgerID := int64(900000000000 + suffix%100000000000)
	secondLedgerID := ledgerID + 1
	t.Cleanup(func() {
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, userID)
	})

	if _, err := pool.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, password_hash)
		VALUES ($1, $1, 'Siswa Finance', 'student', 'x')
	`, userID); err != nil {
		t.Fatalf("insert user: %v", err)
	}

	doc := map[string]any{
		"students": []any{map[string]any{
			"id": studentAppID, "userId": userID, "nisn": nisn,
			"fullName": "Siswa Finance", "classroomName": "XII TKJ 1",
		}},
		"invoices": []any{map[string]any{
			"id": invoiceAppID, "invoiceNumber": invoiceNumber, "studentId": studentAppID,
			"periodMonthYear": "November 2026", "baseAmount": float64(350000),
			"scholarshipDeduction": float64(0), "finalAmount": float64(350000),
			"paidAmount": float64(0), "status": "UNPAID",
			"vaNumber": "880190058471290", "gatewayProvider": "Test VA",
		}},
		"ledger": []any{map[string]any{
			"id": float64(ledgerID), "invoiceId": invoiceAppID,
			"receiptNumber": "KWT-ITEST-001", "txType": "PAYMENT_CASH",
			"amountDelta": float64(100000), "studentName": "Siswa Finance",
			"actorUserId": userID, "actorName": "Bendahara Uji",
			"treasurerPinVerified": true, "signatureHmac": "sig-001",
			"createdAt": "2026-11-23T10:00:00Z",
		}},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	service := NewPostgres(pool)
	if err := service.PutState(ctx, raw); err != nil {
		t.Fatalf("put state: %v", err)
	}
	if err := service.BackfillDomainData(ctx); err != nil {
		t.Fatalf("backfill: %v", err)
	}

	invoices, err := service.InvoicesForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("invoices: %v", err)
	}
	if len(invoices) != 1 {
		t.Fatalf("invoices = %d, want 1", len(invoices))
	}
	invoice := invoices[0]
	if invoice.AppID != invoiceAppID || invoice.StudentAppID != studentAppID ||
		invoice.Number != invoiceNumber || invoice.FinalAmount != 350000 ||
		invoice.PaidAmount != 0 || invoice.Status != "UNPAID" {
		t.Fatalf("unexpected invoice: %+v", invoice)
	}

	entries, err := service.LedgerForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("ledger: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("ledger = %d, want 1", len(entries))
	}
	entry := entries[0]
	if entry.ID != ledgerID || entry.InvoiceAppID != invoiceAppID ||
		entry.ReceiptNumber != "KWT-ITEST-001" || entry.StudentName != "Siswa Finance" ||
		entry.ActorName != "Bendahara Uji" || entry.AmountDelta != 100000 || !entry.PinVerified {
		t.Fatalf("unexpected ledger entry: %+v", entry)
	}

	// A payment mutation is shadow-synced: paid amount and a second ledger row.
	doc["invoices"].([]any)[0].(map[string]any)["paidAmount"] = float64(100000)
	doc["invoices"].([]any)[0].(map[string]any)["status"] = "PAID"
	doc["ledger"] = append(doc["ledger"].([]any), map[string]any{
		"id": float64(secondLedgerID), "invoiceId": invoiceAppID,
		"receiptNumber": "KWT-ITEST-002", "txType": "PAYMENT_VA",
		"amountDelta": float64(250000), "studentName": "Siswa Finance",
		"actorName": "Webhook VA", "treasurerPinVerified": true,
		"signatureHmac": "sig-002", "createdAt": "2026-11-23T11:00:00Z",
	})
	if err := service.SyncDomainFromState(ctx, doc); err != nil {
		t.Fatalf("sync: %v", err)
	}
	invoices, err = service.InvoicesForStudent(ctx, studentAppID)
	if err != nil || len(invoices) != 1 || invoices[0].PaidAmount != 100000 || invoices[0].Status != "PAID" {
		t.Fatalf("invoice after sync = %+v, err %v", invoices, err)
	}
	entries, err = service.LedgerForStudent(ctx, studentAppID)
	if err != nil || len(entries) != 2 {
		t.Fatalf("ledger after sync = %d, err %v", len(entries), err)
	}

	// The append-only trigger must still reject edits.
	if _, err := pool.Exec(ctx, `UPDATE financial_audit_ledgers SET amount_delta = 0 WHERE id = $1`, ledgerID); err == nil {
		t.Fatal("append-only ledger accepted an UPDATE")
	}

	// Pre-008 invoice rows get app_id on the next backfill.
	if _, err := pool.Exec(ctx, `UPDATE tuition_invoices SET app_id = '' WHERE invoice_number = $1`, invoiceNumber); err != nil {
		t.Fatalf("clear app_id: %v", err)
	}
	if _, err := pool.Exec(ctx, `DELETE FROM financial_ledger_details WHERE ledger_id = $1`, ledgerID); err != nil {
		t.Fatalf("delete details: %v", err)
	}
	if err := service.BackfillDomainData(ctx); err != nil {
		t.Fatalf("re-backfill: %v", err)
	}
	invoices, err = service.InvoicesForStudent(ctx, studentAppID)
	if err != nil || len(invoices) != 1 || invoices[0].AppID != invoiceAppID {
		t.Fatalf("app_id not restored: %+v, err %v", invoices, err)
	}
	entries, err = service.LedgerForStudent(ctx, studentAppID)
	if err != nil {
		t.Fatalf("ledger after re-backfill: %v", err)
	}
	found := false
	for _, item := range entries {
		if item.ID == ledgerID && item.ReceiptNumber == "KWT-ITEST-001" {
			found = true
		}
	}
	if !found {
		t.Fatalf("ledger metadata not restored: %+v", entries)
	}
}
