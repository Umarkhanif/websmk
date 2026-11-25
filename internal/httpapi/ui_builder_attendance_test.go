package httpapi

import (
	"testing"
	"time"

	"wesmk/internal/domain/attendance"
)

func TestAttendanceViewsFromDomain(t *testing.T) {
	record := attendance.Record{
		AppID: "att-1", StudentAppID: "std-1", StudentName: "Budi", NISN: "005",
		AttendanceDate: "2026-11-23", Status: "HADIR_GERBANG", VerifiedSource: "RFID Gerbang",
		UpdatedAt: "2026-11-23T06:48:15Z",
	}

	view := attendanceViewFromDomain(record)
	if view.Status != record.Status || view.Source != record.VerifiedSource || view.Time != "23 Nov 2026 · 13:48 WIB" || view.StatusLabel == "" || view.Label == "" {
		t.Fatalf("unexpected dashboard view: %+v", view)
	}

	row := attendanceRowFromDomain(record)
	if row.ID != "att-1" || row.StudentID != "std-1" || row.NISN != "005" ||
		row.StatusLabel == "" || row.GateTapTime != "23 Nov 2026 · 13:48 WIB" {
		t.Fatalf("unexpected row view: %+v", row)
	}
}

func TestAttendanceLabelAndStamp(t *testing.T) {
	now := time.Date(2026, 11, 24, 10, 0, 0, 0, wibZone)
	if got := attendanceLabel("2026-11-24", now); got != "Presensi Hari Ini" {
		t.Fatalf("today label = %q", got)
	}
	if got := attendanceLabel("2026-11-23", now); got != "Presensi Terakhir" {
		t.Fatalf("stale label = %q", got)
	}
	if got := formatAttendanceStamp("2026-10-05T20:55:25Z"); got != "06 Okt 2026 · 03:55 WIB" {
		t.Fatalf("formatted stamp = %q", got)
	}
	if got := formatAttendanceMoment(attendance.Record{AttendanceDate: "2026-11-23", UpdatedAt: "2026-11-23T06:48:15Z"}); got != "23 Nov 2026 · 13:48 WIB" {
		t.Fatalf("same-day moment = %q", got)
	}
	if got := formatAttendanceMoment(attendance.Record{AttendanceDate: "2026-11-23", UpdatedAt: "2026-11-23T20:55:25Z"}); got != "23 Nov 2026" {
		t.Fatalf("cross-midnight moment = %q", got)
	}
	if got := formatAttendanceStamp("06:48 WIB"); got != "06:48 WIB" {
		t.Fatalf("unparsed stamp = %q", got)
	}
}
