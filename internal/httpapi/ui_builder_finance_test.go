package httpapi

import (
	"testing"

	"wesmk/internal/domain/finance"
)

func TestInvoiceViewFromDomain(t *testing.T) {
	view := invoiceViewFromDomain(finance.Invoice{
		AppID: "inv-1", Status: "UNPAID",
		FinalAmount: 350000, PaidAmount: 100000,
	})
	if view.IsPaid || view.Remaining != 250000 || view.StatusLabel == "" {
		t.Fatalf("unexpected unpaid view: %+v", view)
	}

	paid := invoiceViewFromDomain(finance.Invoice{
		AppID: "inv-2", Status: "PAID",
		FinalAmount: 350000, PaidAmount: 350000,
	})
	if !paid.IsPaid || paid.Remaining != 0 {
		t.Fatalf("unexpected paid view: %+v", paid)
	}

	overpaid := invoiceViewFromDomain(finance.Invoice{Status: "PAID", FinalAmount: 100, PaidAmount: 150})
	if overpaid.Remaining != 0 {
		t.Fatalf("remaining must not go negative: %+v", overpaid)
	}
}

func TestLedgerViewFromDomain(t *testing.T) {
	view := ledgerViewFromDomain(finance.LedgerEntry{
		ID: 7, InvoiceAppID: "inv-1", ReceiptNumber: "KWT-1",
		TxType: "PAYMENT_CASH", AmountDelta: 250000, StudentName: "Budi",
		ActorName: "Bendahara", PinVerified: true, SignatureHMAC: "sig",
	})
	if view.ID != 7 || view.InvoiceID != "inv-1" || view.ReceiptNumber != "KWT-1" ||
		view.AmountDelta != 250000 || !view.PinVerified || view.SignatureHmac != "sig" {
		t.Fatalf("unexpected ledger view: %+v", view)
	}
}
