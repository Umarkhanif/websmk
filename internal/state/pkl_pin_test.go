package state

import (
	"errors"
	"strings"
	"testing"
	"time"
)

func pklFixture(pin string) map[string]any {
	return map[string]any{
		"students": []any{map[string]any{"id": "std-1", "fullName": "Budi"}},
		"pklPlacements": []any{map[string]any{
			"id":            "pkl-1",
			"companyName":   "PT Contoh",
			"supervisorPin": pin,
		}},
		"pklJournals": []any{map[string]any{
			"id":                 "jrn-1",
			"studentId":          "std-1",
			"placementId":        "pkl-1",
			"supervisorApproved": false,
		}},
	}
}

func firstJournal(doc map[string]any) map[string]any {
	return asMap(asList(doc["pklJournals"])[0])
}

func firstPlacement(doc map[string]any) map[string]any {
	return asMap(asList(doc["pklPlacements"])[0])
}

func TestApprovePklJournalByDudiRejectsWrongPinWithoutLeakingIt(t *testing.T) {
	const pin = "123456"
	doc := pklFixture(hashSupervisorPin(pin))

	err := ApprovePklJournalByDudi(doc, "jrn-1", "000000", time.Now())
	if !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid", err)
	}
	if strings.Contains(err.Error(), pin) {
		t.Fatalf("error message leaks the PIN: %q", err.Error())
	}
	if approved, _ := firstJournal(doc)["supervisorApproved"].(bool); approved {
		t.Fatal("journal must not be approved with a wrong PIN")
	}
}

func TestApprovePklJournalByDudiAcceptsHashedPin(t *testing.T) {
	const pin = "123456"
	doc := pklFixture(hashSupervisorPin(pin))

	if err := ApprovePklJournalByDudi(doc, "jrn-1", pin, time.Now()); err != nil {
		t.Fatalf("approve: %v", err)
	}
	journal := firstJournal(doc)
	if approved, _ := journal["supervisorApproved"].(bool); !approved {
		t.Fatal("journal must be approved with the correct PIN")
	}
	if text(journal["approvedAt"]) == "" {
		t.Fatal("approvedAt must be recorded")
	}
}

func TestApprovePklJournalByDudiUpgradesLegacyPlaintextPin(t *testing.T) {
	const pin = "654321"
	doc := pklFixture(pin)

	if err := ApprovePklJournalByDudi(doc, "jrn-1", pin, time.Now()); err != nil {
		t.Fatalf("approve legacy pin: %v", err)
	}
	stored := text(firstPlacement(doc)["supervisorPin"])
	if stored == pin {
		t.Fatal("legacy plaintext PIN must be replaced after a successful check")
	}
	if !strings.HasPrefix(stored, supervisorPinHashPrefix) {
		t.Fatalf("stored PIN = %q, want hash prefix", stored)
	}
}

func TestApprovePklJournalWithoutConfiguredPinFails(t *testing.T) {
	doc := pklFixture("")

	if err := ApprovePklJournalByDudi(doc, "jrn-1", "445566", time.Now()); !errors.Is(err, ErrInvalid) {
		t.Fatalf("error = %v, want ErrInvalid for a missing PIN", err)
	}
	if approved, _ := firstJournal(doc)["supervisorApproved"].(bool); approved {
		t.Fatal("journal must not be approved without a configured PIN")
	}
}

func TestAddPklPlacementRequiresSixDigitPinAndStoresHash(t *testing.T) {
	now := time.Now()
	for _, invalid := range []string{"", "12345", "1234567", "12ab56"} {
		doc := map[string]any{}
		if err := AddPklPlacement(doc, "PT Contoh", "IT", "Alamat", 0, 0, 100, invalid, now); !errors.Is(err, ErrInvalid) {
			t.Fatalf("PIN %q: error = %v, want ErrInvalid", invalid, err)
		}
	}

	doc := map[string]any{}
	if err := AddPklPlacement(doc, "PT Contoh", "IT", "Alamat", 0, 0, 100, "123456", now); err != nil {
		t.Fatalf("add placement: %v", err)
	}
	stored := text(firstPlacement(doc)["supervisorPin"])
	if stored == "123456" || !strings.HasPrefix(stored, supervisorPinHashPrefix) {
		t.Fatalf("stored PIN = %q, want a sha256 hash", stored)
	}
}

func TestPklPlacementPinRedactedFromStudentAndStaffViews(t *testing.T) {
	const pin = "123456"
	doc := pklFixture(hashSupervisorPin(pin))

	studentView := ScopedView(doc, Access{Role: "student", StudentID: "std-1"})
	studentPlacements := asList(studentView["pklPlacements"])
	if len(studentPlacements) == 0 {
		t.Fatal("student view must keep pklPlacements")
	}
	studentPlacement := asMap(studentPlacements[0])
	if _, exists := studentPlacement["supervisorPin"]; exists {
		t.Fatal("student view must not expose supervisorPin")
	}
	if text(studentPlacement["companyName"]) != "PT Contoh" {
		t.Fatal("other placement fields must remain")
	}

	staffView := StaffView(doc, "admin_tu")
	staffPlacements := asList(staffView["pklPlacements"])
	if len(staffPlacements) == 0 {
		t.Fatal("admin TU view must keep pklPlacements")
	}
	if _, exists := asMap(staffPlacements[0])["supervisorPin"]; exists {
		t.Fatal("staff view must not expose the PIN hash")
	}

	if text(firstPlacement(doc)["supervisorPin"]) == "" {
		t.Fatal("redaction must not mutate the stored document")
	}
}
