package httpapi

import (
	"context"
	"errors"
	"log"

	"wesmk/internal/authz"
	"wesmk/internal/domain/finance"
	"wesmk/internal/model"
	"wesmk/internal/ui"
)

func domainFinanceViewer(user *model.User) finance.Viewer {
	if user == nil {
		return finance.Viewer{}
	}
	return finance.Viewer{UserID: user.ID, Role: user.Role, StudentID: user.StudentProfileID}
}

func invoiceViewFromDomain(invoice finance.Invoice) ui.InvoiceView {
	remaining := invoice.FinalAmount - invoice.PaidAmount
	if remaining < 0 {
		remaining = 0
	}
	return ui.InvoiceView{
		ID:          invoice.AppID,
		Number:      invoice.Number,
		Period:      invoice.Period,
		Status:      invoice.Status,
		StatusLabel: ui.StatusLabel(invoice.Status),
		BaseAmount:  int(invoice.BaseAmount),
		FinalAmount: int(invoice.FinalAmount),
		PaidAmount:  int(invoice.PaidAmount),
		Remaining:   int(remaining),
		VANumber:    invoice.VANumber,
		Provider:    invoice.Provider,
		IsPaid:      invoice.Status == "PAID" || remaining == 0,
	}
}

func ledgerViewFromDomain(entry finance.LedgerEntry) ui.LedgerEntryView {
	return ui.LedgerEntryView{
		ID:               entry.ID,
		InvoiceID:        entry.InvoiceAppID,
		ReceiptNumber:    entry.ReceiptNumber,
		TxType:           entry.TxType,
		AmountDelta:      int(entry.AmountDelta),
		StudentName:      entry.StudentName,
		ActorName:        entry.ActorName,
		PinVerified:      entry.PinVerified,
		ReversalApproval: entry.ReversalApproval,
		SignatureHmac:    entry.SignatureHMAC,
		CreatedAt:        entry.CreatedAt,
	}
}

func invoiceViewsFromDomain(invoices []finance.Invoice) []ui.InvoiceView {
	views := make([]ui.InvoiceView, 0, len(invoices))
	for _, invoice := range invoices {
		views = append(views, invoiceViewFromDomain(invoice))
	}
	return views
}

func ledgerViewsFromDomain(entries []finance.LedgerEntry) []ui.LedgerEntryView {
	views := make([]ui.LedgerEntryView, 0, len(entries))
	for _, entry := range entries {
		views = append(views, ledgerViewFromDomain(entry))
	}
	return views
}

// domainStudentInvoices returns false when the caller should keep the blob
// fallback (feature off, no rows, policy denial, or a read error).
func (s *Server) domainStudentInvoices(ctx context.Context, user *model.User, studentID string) ([]ui.InvoiceView, bool) {
	if !s.cfg.DomainFinance || s.financeSvc == nil || user == nil || studentID == "" {
		return nil, false
	}
	invoices, err := s.financeSvc.InvoicesForViewer(ctx, domainFinanceViewer(user), studentID)
	if err != nil {
		logFinanceFallback(err)
		return nil, false
	}
	if len(invoices) == 0 {
		return nil, false
	}
	return invoiceViewsFromDomain(invoices), true
}

func (s *Server) domainStudentLedger(ctx context.Context, user *model.User, studentID string) ([]ui.LedgerEntryView, bool) {
	if !s.cfg.DomainFinance || s.financeSvc == nil || user == nil || studentID == "" {
		return nil, false
	}
	entries, err := s.financeSvc.LedgerForViewer(ctx, domainFinanceViewer(user), studentID)
	if err != nil {
		logFinanceFallback(err)
		return nil, false
	}
	if len(entries) == 0 {
		return nil, false
	}
	return ledgerViewsFromDomain(entries), true
}

func (s *Server) domainAllInvoices(ctx context.Context, user *model.User) ([]ui.InvoiceView, bool) {
	if !s.cfg.DomainFinance || s.financeSvc == nil || user == nil {
		return nil, false
	}
	invoices, err := s.financeSvc.AllInvoices(ctx, domainFinanceViewer(user))
	if err != nil {
		logFinanceFallback(err)
		return nil, false
	}
	if len(invoices) == 0 {
		return nil, false
	}
	return invoiceViewsFromDomain(invoices), true
}

func (s *Server) domainAllLedger(ctx context.Context, user *model.User) ([]ui.LedgerEntryView, bool) {
	if !s.cfg.DomainFinance || s.financeSvc == nil || user == nil {
		return nil, false
	}
	entries, err := s.financeSvc.AllLedger(ctx, domainFinanceViewer(user))
	if err != nil {
		logFinanceFallback(err)
		return nil, false
	}
	if len(entries) == 0 {
		return nil, false
	}
	return ledgerViewsFromDomain(entries), true
}

func logFinanceFallback(err error) {
	if errors.Is(err, authz.ErrDenied) || errors.Is(err, finance.ErrNotFound) {
		return
	}
	log.Printf("domain finance read failed: %v", err)
}
