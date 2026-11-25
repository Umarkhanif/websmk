package store

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"

	"wesmk/internal/domain/identity"
)

// StudentProfile reads the normalized profile row for an app-level student ID.
// The users join supplies a name for rows created before full_name was added.
func (p *Postgres) StudentProfile(ctx context.Context, studentAppID string) (*identity.StudentProfile, error) {
	const query = `
		SELECT
			COALESCE(s.app_id, ''), s.user_id, COALESCE(s.nisn, ''), s.nik,
			COALESCE(NULLIF(s.full_name, ''), u.full_name, ''),
			COALESCE(s.classroom_id::text, ''),
			s.classroom_name, s.major_code, s.parent_name, s.parent_phone,
			s.is_ketua_kelas, s.is_sekretaris, s.is_alumni,
			s.scholarship_type, s.discount_percent::float8, s.attitude_score,
			s.bmw_choice, s.homeroom_note
		FROM students s
		JOIN users u ON u.id = s.user_id
		WHERE s.app_id = $1 OR s.user_id = $1
		LIMIT 1`

	var profile identity.StudentProfile
	err := p.pool.QueryRow(ctx, query, studentAppID).Scan(
		&profile.AppID,
		&profile.UserID,
		&profile.NISN,
		&profile.NIK,
		&profile.FullName,
		&profile.ClassroomID,
		&profile.ClassroomName,
		&profile.MajorCode,
		&profile.ParentName,
		&profile.ParentPhone,
		&profile.IsKetuaKelas,
		&profile.IsSekretaris,
		&profile.IsAlumni,
		&profile.ScholarshipType,
		&profile.DiscountPercent,
		&profile.AttitudeScore,
		&profile.BMWChoice,
		&profile.HomeroomNote,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, identity.ErrProfileNotFound
	}
	if err != nil {
		return nil, err
	}
	return &profile, nil
}
