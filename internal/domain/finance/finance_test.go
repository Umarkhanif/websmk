package finance

import (
	"context"
	"errors"
	"testing"

	"wesmk/internal/authz"
)

type fakeRepo struct {
	invoices []Invoice
	ledger   []LedgerEntry
}

func (f fakeRepo) InvoicesForStudent(context.Context, string) ([]Invoice, error) {
	return f.invoices, nil
}

func (f fakeRepo) AllInvoices(context.Context) ([]Invoice, error) { return f.invoices, nil }

func (f fakeRepo) LedgerForStudent(context.Context, string) ([]LedgerEntry, error) {
	return f.ledger, nil
}

func (f fakeRepo) AllLedger(context.Context) ([]LedgerEntry, error) { return f.ledger, nil }

func TestFinanceViewerPolicy(t *testing.T) {
	service := NewService(fakeRepo{invoices: []Invoice{{AppID: "inv-1"}}})

	if _, err := service.InvoicesForViewer(context.Background(),
		Viewer{Role: "student", StudentID: "std-1"}, "std-1"); err != nil {
		t.Fatalf("student own invoice: %v", err)
	}
	if _, err := service.InvoicesForViewer(context.Background(),
		Viewer{Role: "parent", StudentID: "std-1"}, "std-1"); err != nil {
		t.Fatalf("parent own invoice: %v", err)
	}
	if _, err := service.InvoicesForViewer(context.Background(),
		Viewer{Role: "student", StudentID: "std-2"}, "std-1"); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("student foreign invoice err = %v, want denied", err)
	}
	if _, err := service.InvoicesForViewer(context.Background(),
		Viewer{Role: "teacher"}, "std-1"); !errors.Is(err, authz.ErrDenied) {
		t.Fatalf("teacher invoice err = %v, want denied", err)
	}
	if _, err := service.LedgerForViewer(context.Background(),
		Viewer{Role: "parent", StudentID: "std-1"}, "std-1"); err != nil {
		t.Fatalf("parent ledger: %v", err)
	}
	if _, err := service.InvoicesForViewer(context.Background(),
		Viewer{Role: "parent", StudentID: "std-1"}, ""); !errors.Is(err, ErrNotFound) {
		t.Fatalf("empty student err = %v, want not found", err)
	}
}

func TestFinanceStaffPolicy(t *testing.T) {
	service := NewService(fakeRepo{})
	for _, role := range []string{"treasurer", "admin_tu", "headmaster"} {
		if _, err := service.AllInvoices(context.Background(), Viewer{Role: role}); err != nil {
			t.Fatalf("%s all invoices: %v", role, err)
		}
		if _, err := service.AllLedger(context.Background(), Viewer{Role: role}); err != nil {
			t.Fatalf("%s all ledger: %v", role, err)
		}
	}
	for _, role := range []string{"student", "parent", "teacher", "developer", ""} {
		if _, err := service.AllInvoices(context.Background(), Viewer{Role: role}); !errors.Is(err, authz.ErrDenied) {
			t.Fatalf("%s all invoices err = %v, want denied", role, err)
		}
	}
}
