// Package finance owns invoice and ledger reads from the normalized tables.
// The ledger itself stays append-only; display metadata lives in a separate
// table so a backfill never edits a posted ledger row.
package finance

import (
	"context"
	"errors"

	"wesmk/internal/authz"
)

var ErrNotFound = errors.New("data keuangan tidak ditemukan")

type Invoice struct {
	AppID                string
	StudentAppID         string
	Number               string
	Period               string
	Status               string
	VANumber             string
	Provider             string
	BaseAmount           int64
	ScholarshipDeduction int64
	FinalAmount          int64
	PaidAmount           int64
}

type LedgerEntry struct {
	ID               int64
	InvoiceAppID     string
	ReceiptNumber    string
	TxType           string
	AmountDelta      int64
	StudentName      string
	ActorName        string
	PinVerified      bool
	ReversalApproval string
	SignatureHMAC    string
	CreatedAt        string
}

type Repository interface {
	InvoicesForStudent(ctx context.Context, studentAppID string) ([]Invoice, error)
	AllInvoices(ctx context.Context) ([]Invoice, error)
	LedgerForStudent(ctx context.Context, studentAppID string) ([]LedgerEntry, error)
	AllLedger(ctx context.Context) ([]LedgerEntry, error)
}

type Viewer struct {
	UserID    string
	Role      string
	StudentID string
}

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) InvoicesForViewer(ctx context.Context, viewer Viewer, studentAppID string) ([]Invoice, error) {
	if err := s.authorize(viewer, studentAppID); err != nil {
		return nil, err
	}
	return s.repo.InvoicesForStudent(ctx, studentAppID)
}

func (s *Service) LedgerForViewer(ctx context.Context, viewer Viewer, studentAppID string) ([]LedgerEntry, error) {
	if err := s.authorize(viewer, studentAppID); err != nil {
		return nil, err
	}
	return s.repo.LedgerForStudent(ctx, studentAppID)
}

func (s *Service) AllInvoices(ctx context.Context, viewer Viewer) ([]Invoice, error) {
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceFinance, authz.Ownership{}) {
		return nil, authz.ErrDenied
	}
	return s.repo.AllInvoices(ctx)
}

func (s *Service) AllLedger(ctx context.Context, viewer Viewer) ([]LedgerEntry, error) {
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceFinance, authz.Ownership{}) {
		return nil, authz.ErrDenied
	}
	return s.repo.AllLedger(ctx)
}

func (s *Service) authorize(viewer Viewer, studentAppID string) error {
	if s == nil || s.repo == nil || studentAppID == "" {
		return ErrNotFound
	}
	own := viewer.StudentID != "" && viewer.StudentID == studentAppID
	if !authz.Can(viewer.Role, authz.ActionRead, authz.ResourceFinance, authz.Ownership{Own: own}) {
		return authz.ErrDenied
	}
	return nil
}
