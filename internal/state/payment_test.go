package state

import (
	"errors"
	"strings"
	"testing"
	"time"
)

const paymentState = `{
  "students": [
    {"id": "std-1", "fullName": "Budi"},
    {"id": "std-2", "fullName": "Rahma"}
  ],
  "invoices": [
    {"id": "inv-1", "studentId": "std-1", "studentName": "Budi", "finalAmount": 350000, "paidAmount": 100000, "status": "PARTIAL"},
    {"id": "inv-2", "studentId": "std-2", "studentName": "Rahma", "finalAmount": 200000, "paidAmount": 0, "status": "UNPAID"}
  ],
  "ledger": [],
  "paymentIntents": []
}`

const testSecret = "unit-test-receipt-secret"

func TestSettlePaymentIntentIssuesServerReceipt(t *testing.T) {
	doc := mustDoc(t, paymentState)
	now := time.Date(2026, 10, 25, 10, 0, 0, 0, time.UTC)

	intent, err := CreatePaymentIntent(doc, "std-1", "inv-1", now)
	if err != nil {
		t.Fatalf("create intent: %v", err)
	}
	result, err := SettlePaymentIntent(doc, intent.IntentID, "ref-1", testSecret, now)
	if err != nil {
		t.Fatalf("settle intent: %v", err)
	}
	if result.Amount != 250000 || result.Status != "PAID" {
		t.Fatalf("result = %+v, want 250000 PAID", result)
	}
	if !strings.HasPrefix(result.ReceiptNumber, "KWT-VA-") {
		t.Fatalf("receipt = %q, want a KWT-VA number", result.ReceiptNumber)
	}

	invoice := findInvoice(doc, "inv-1", "std-1")
	if paid, _ := number(invoice["paidAmount"]); paid != 350000 {
		t.Fatalf("invoice paid = %v, want 350000", paid)
	}
	if status := text(invoice["status"]); status != "PAID" {
		t.Fatalf("invoice status = %q, want PAID", status)
	}

	ledger := asList(doc["ledger"])
	if len(ledger) != 1 {
		t.Fatalf("ledger entries = %d, want 1", len(ledger))
	}
	entry := asMap(ledger[0])
	if got := text(entry["signatureHmac"]); got == "" || got == "hmac_sha256_" {
		t.Fatalf("ledger signature = %q, want a server HMAC", got)
	}
	expected := ReceiptSignature(testSecret, "inv-1", result.ReceiptNumber, 250000, result.PaidAt)
	if got := text(entry["signatureHmac"]); got != expected {
		t.Fatalf("ledger signature = %q, want %q", got, expected)
	}

	replay, err := SettlePaymentIntent(doc, intent.IntentID, "ref-1", testSecret, now.Add(time.Minute))
	if err != nil {
		t.Fatalf("replay payment: %v", err)
	}
	if !replay.AlreadyPaid || replay.ReceiptNumber != result.ReceiptNumber {
		t.Fatalf("replay = %+v, want the original receipt", replay)
	}
	if got := len(asList(doc["ledger"])); got != 1 {
		t.Fatalf("ledger entries after replay = %d, want 1", got)
	}
}

func TestPaymentIsScopedToTheOwner(t *testing.T) {
	doc := mustDoc(t, paymentState)

	if _, err := CreatePaymentIntent(doc, "std-1", "inv-2", time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign invoice error = %v, want ErrNotFound", err)
	}
	if _, err := CreatePaymentIntent(doc, "std-2", "inv-1", time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("foreign intent error = %v, want ErrNotFound", err)
	}
	if _, err := CreatePaymentIntent(doc, "std-ghost", "inv-1", time.Now()); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown student error = %v, want ErrNotFound", err)
	}
}

func TestPaymentIntentIsReusedAndSettledByCallback(t *testing.T) {
	doc := mustDoc(t, paymentState)
	now := time.Date(2026, 10, 25, 11, 0, 0, 0, time.UTC)

	first, err := CreatePaymentIntent(doc, "std-2", "inv-2", now)
	if err != nil {
		t.Fatalf("create intent: %v", err)
	}
	if first.Status != "PENDING" || first.Amount != 200000 || first.VANumber == "" {
		t.Fatalf("intent = %+v, want a PENDING 200000 VA", first)
	}
	again, err := CreatePaymentIntent(doc, "std-2", "inv-2", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("recreate intent: %v", err)
	}
	if again.IntentID != first.IntentID {
		t.Fatalf("intent id changed from %q to %q, want reuse", first.IntentID, again.IntentID)
	}
	if got := len(asList(doc["paymentIntents"])); got != 1 {
		t.Fatalf("intents = %d, want 1", got)
	}

	if _, err := SettlePaymentIntent(doc, "pi-missing", "ref-1", testSecret, now); !errors.Is(err, ErrNotFound) {
		t.Fatalf("unknown intent error = %v, want ErrNotFound", err)
	}

	paid, err := SettlePaymentIntent(doc, first.IntentID, "ref-1", testSecret, now.Add(2*time.Minute))
	if err != nil {
		t.Fatalf("settle intent: %v", err)
	}
	if paid.Status != "PAID" || paid.AlreadyPaid {
		t.Fatalf("settled = %+v, want a first-time PAID", paid)
	}
	if got := len(asList(doc["ledger"])); got != 1 {
		t.Fatalf("ledger entries = %d, want 1", got)
	}

	replay, err := SettlePaymentIntent(doc, first.IntentID, "ref-1", testSecret, now.Add(3*time.Minute))
	if err != nil {
		t.Fatalf("replay settle: %v", err)
	}
	if !replay.AlreadyPaid {
		t.Fatalf("replay = %+v, want AlreadyPaid", replay)
	}
	if got := len(asList(doc["ledger"])); got != 1 {
		t.Fatalf("ledger entries after replay = %d, want 1", got)
	}
}

func TestExpiredPaymentIntentIsNotSettled(t *testing.T) {
	doc := mustDoc(t, paymentState)
	now := time.Date(2026, 10, 25, 12, 0, 0, 0, time.UTC)

	intent, err := CreatePaymentIntent(doc, "std-2", "inv-2", now)
	if err != nil {
		t.Fatalf("create intent: %v", err)
	}

	late := now.Add(48 * time.Hour)
	if _, err := SettlePaymentIntent(doc, intent.IntentID, "ref-late", testSecret, late); !errors.Is(err, ErrConflict) {
		t.Fatalf("late settlement error = %v, want ErrConflict", err)
	}
	if got := len(asList(doc["ledger"])); got != 0 {
		t.Fatalf("ledger entries = %d, want 0 for an expired intent", got)
	}

	fresh, err := CreatePaymentIntent(doc, "std-2", "inv-2", late)
	if err != nil {
		t.Fatalf("create fresh intent: %v", err)
	}
	if fresh.IntentID == intent.IntentID {
		t.Fatal("expired intent must not be reused")
	}
}
