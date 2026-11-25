package httpapi

import (
	"context"
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"wesmk/internal/authz"
	"wesmk/internal/domain/attendance"
	"wesmk/internal/model"
	"wesmk/internal/ui"
)

var wibZone = time.FixedZone("WIB", 7*60*60)

var shortMonthID = [...]string{"Jan", "Feb", "Mar", "Apr", "Mei", "Jun", "Jul", "Agu", "Sep", "Okt", "Nov", "Des"}

func domainAttendanceViewer(user *model.User) attendance.Viewer {
	if user == nil {
		return attendance.Viewer{}
	}
	return attendance.Viewer{UserID: user.ID, Role: user.Role, StudentID: user.StudentProfileID}
}

func attendanceViewFromDomain(record attendance.Record) ui.AttendanceView {
	return ui.AttendanceView{
		Status:      record.Status,
		StatusLabel: ui.StatusLabel(record.Status),
		Label:       attendanceLabel(record.AttendanceDate, time.Now()),
		Time:        formatAttendanceMoment(record),
		Source:      record.VerifiedSource,
	}
}

func attendanceLabel(attendanceDate string, now time.Time) string {
	if strings.TrimSpace(attendanceDate) == "" {
		return "Presensi Hari Ini"
	}
	if attendanceDate != now.In(wibZone).Format("2006-01-02") {
		return "Presensi Terakhir"
	}
	return "Presensi Hari Ini"
}

func formatAttendanceMoment(record attendance.Record) string {
	dateLabel := formatAttendanceDate(record.AttendanceDate)
	if dateLabel == "" {
		return formatAttendanceStamp(record.UpdatedAt)
	}
	stamp, err := time.Parse(time.RFC3339, strings.TrimSpace(record.UpdatedAt))
	if err == nil {
		stamp = stamp.In(wibZone)
		if stamp.Format("2006-01-02") == record.AttendanceDate {
			return fmt.Sprintf("%s · %02d:%02d WIB", dateLabel, stamp.Hour(), stamp.Minute())
		}
	}
	return dateLabel
}

func formatAttendanceDate(raw string) string {
	date, err := time.Parse("2006-01-02", strings.TrimSpace(raw))
	if err != nil {
		return ""
	}
	return fmt.Sprintf("%02d %s %d", date.Day(), shortMonthID[int(date.Month())-1], date.Year())
}

func formatAttendanceStamp(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	stamp, err := time.Parse(time.RFC3339, raw)
	if err != nil {
		return raw
	}
	stamp = stamp.In(wibZone)
	return fmt.Sprintf("%02d %s %d · %02d:%02d WIB",
		stamp.Day(), shortMonthID[int(stamp.Month())-1], stamp.Year(), stamp.Hour(), stamp.Minute())
}

func attendanceRowFromDomain(record attendance.Record) ui.AttendanceRowView {
	return ui.AttendanceRowView{
		ID:             record.AppID,
		StudentID:      record.StudentAppID,
		StudentName:    record.StudentName,
		NISN:           record.NISN,
		Status:         record.Status,
		StatusLabel:    ui.StatusLabel(record.Status),
		GateTapTime:    formatAttendanceStamp(record.UpdatedAt),
		VerifiedSource: record.VerifiedSource,
	}
}

func attendanceRowsFromDomain(records []attendance.Record) []ui.AttendanceRowView {
	views := make([]ui.AttendanceRowView, 0, len(records))
	for _, record := range records {
		views = append(views, attendanceRowFromDomain(record))
	}
	return views
}

func (s *Server) domainStudentAttendance(ctx context.Context, user *model.User, studentID string) ([]attendance.Record, bool) {
	if !s.cfg.DomainAttendance || s.attendanceSvc == nil || user == nil || studentID == "" {
		return nil, false
	}
	records, err := s.attendanceSvc.ForViewer(ctx, domainAttendanceViewer(user), studentID)
	if err != nil {
		logAttendanceFallback(err)
		return nil, false
	}
	if len(records) == 0 {
		return nil, false
	}
	return records, true
}

func (s *Server) domainAllAttendance(ctx context.Context, user *model.User) ([]attendance.Record, bool) {
	if !s.cfg.DomainAttendance || s.attendanceSvc == nil || user == nil {
		return nil, false
	}
	records, err := s.attendanceSvc.AllForViewer(ctx, domainAttendanceViewer(user))
	if err != nil {
		logAttendanceFallback(err)
		return nil, false
	}
	if len(records) == 0 {
		return nil, false
	}
	return records, true
}

func logAttendanceFallback(err error) {
	// Policy denials and empty rows are expected fallbacks, not incidents.
	if err == nil || errors.Is(err, authz.ErrDenied) || errors.Is(err, attendance.ErrNotFound) {
		return
	}
	log.Printf("domain attendance read failed: %v", err)
}
