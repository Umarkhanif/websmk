package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"wesmk/internal/ui"
)

func TestPublicVerifyPageDoesNotLeakSupervisorPIN(t *testing.T) {
	var buffer bytes.Buffer
	if err := PublicVerify(ui.PublicVerifyPage{ActiveTab: "dudi"}).Render(context.Background(), &buffer); err != nil {
		t.Fatalf("render: %v", err)
	}

	html := buffer.String()
	for _, leaked := range []string{
		"445566",
		"112233",
		"778899",
		"PIN resmi dari nota kerja sama sekolah (demo",
	} {
		if strings.Contains(html, leaked) {
			t.Fatalf("public verify page leaks %q", leaked)
		}
	}
	if !strings.Contains(html, "Kode Jurnal PKL") {
		t.Fatal("the DUDI search form should remain available")
	}
	if strings.Contains(html, `name="journal_id"`) {
		t.Fatal("an empty search page must not render the approval form")
	}
}

func TestPublicVerifyDudiRendersOnlyRequestedJournal(t *testing.T) {
	journal := ui.PklJournalView{
		ID:                "pkl-jrn-0001",
		StudentName:       "Budi Santoso",
		CompanyName:       "PT Contoh",
		Date:              "2026-11-20",
		ActivityTitle:     "Konfigurasi VLAN",
		CompetencyLearned: "VLAN & trunking",
	}

	var buffer bytes.Buffer
	page := ui.PublicVerifyPage{
		ActiveTab:       "dudi",
		Query:           journal.ID,
		SelectedJournal: &journal,
		CSRFToken:       "token-uji",
	}
	if err := PublicVerify(page).Render(context.Background(), &buffer); err != nil {
		t.Fatalf("render: %v", err)
	}

	html := buffer.String()
	for _, want := range []string{journal.ID, journal.StudentName, journal.ActivityTitle, "Sahkan Jurnal Magang"} {
		if !strings.Contains(html, want) {
			t.Fatalf("rendered page is missing %q", want)
		}
	}
	if strings.Contains(html, "<select") || strings.Contains(html, "<table") {
		t.Fatal("the DUDI page must not list every journal")
	}
}
