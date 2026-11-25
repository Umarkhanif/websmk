package httpapi

import (
	"testing"

	"wesmk/internal/domain/identity"
	"wesmk/internal/ui"
)

func TestApplyDomainStudentProfileOverridesDisplayFields(t *testing.T) {
	view := ui.StudentView{
		FullName:        "Nama dari blob",
		NISN:            "lama",
		AttitudeScore:   80,
		IsKetuaKelas:    false,
		DiscountPercent: 0,
	}
	profile := &identity.StudentProfile{
		FullName:        "Nama dari tabel",
		NISN:            "0058471290",
		NIK:             "3201284509050001",
		ClassroomName:   "XII TKJ 1",
		ParentName:      "Bapak Ahmad",
		ScholarshipType: "KIP",
		DiscountPercent: 25,
		AttitudeScore:   92,
		BMWChoice:       "BEKERJA",
		HomeroomNote:    "Catatan wali",
		IsKetuaKelas:    true,
	}

	applyDomainStudentProfile(&view, profile)

	if view.FullName != profile.FullName || view.NISN != profile.NISN || view.NIK != profile.NIK {
		t.Fatalf("identity not overridden: %+v", view)
	}
	if view.ClassroomName != "XII TKJ 1" || view.ParentName != "Bapak Ahmad" {
		t.Fatalf("class/parent not overridden: %+v", view)
	}
	if view.AttitudeScore != 92 || view.DiscountPercent != 25 || !view.IsKetuaKelas {
		t.Fatalf("numeric/flag fields not overridden: %+v", view)
	}
	if view.BMWChoice != "BEKERJA" || view.HomeroomNote != "Catatan wali" {
		t.Fatalf("BMW/homeroom not overridden: %+v", view)
	}
}

func TestApplyDomainStudentProfileKeepsBlobFallback(t *testing.T) {
	view := ui.StudentView{FullName: "Nama blob", NISN: "nisn-blob", NIK: "nik-blob"}
	applyDomainStudentProfile(&view, &identity.StudentProfile{})

	if view.FullName != "Nama blob" || view.NISN != "nisn-blob" || view.NIK != "nik-blob" {
		t.Fatalf("empty table values must not wipe the blob fallback: %+v", view)
	}
}
