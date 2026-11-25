package store

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"

	"wesmk/internal/model"
)

var (
	ErrStudentNISNExists = errors.New("NISN sudah terdaftar sebagai siswa")
	ErrParentPhoneExists = errors.New("nomor WhatsApp wali sudah terdaftar")
)

// StudentOnboarding is the normalized part of admitting one student. The state
// blob still carries the UI snapshot, but auth, profile, and the parent link are
// created here first so the student can actually sign in.
type StudentOnboarding struct {
	StudentID           string
	StudentUserID       string
	ParentUserID        string
	FullName            string
	NISN                string
	ClassroomID         string
	ClassroomName       string
	MajorCode           string
	ParentName          string
	ParentPhone         string
	StudentPasswordHash string
	ParentPasswordHash  string
}

// CreateStudentWithAccounts creates the student account, parent account and
// student profile in one transaction. It refuses duplicate NISN or parent
// phone instead of silently overwriting an existing family account.
func (p *Postgres) CreateStudentWithAccounts(ctx context.Context, input StudentOnboarding) error {
	phone := model.NormalizePhone(input.ParentPhone)
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, student_profile_id, password_hash, is_active)
		VALUES ($1, $2, $3, 'student', $4, $5, TRUE)
	`, input.StudentUserID, input.NISN, input.FullName, input.StudentID, input.StudentPasswordHash); err != nil {
		if isUniqueViolation(err, "users_identifier_key") {
			return ErrStudentNISNExists
		}
		return fmt.Errorf("create student user: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO users (id, identifier, full_name, role, phone_wa, student_profile_id, password_hash, is_active)
		VALUES ($1, $2, $3, 'parent', $2, $4, $5, TRUE)
	`, input.ParentUserID, phone, input.ParentName, input.StudentID, input.ParentPasswordHash); err != nil {
		if isUniqueViolation(err, "users_identifier_key") {
			return ErrParentPhoneExists
		}
		return fmt.Errorf("create parent user: %w", err)
	}

	var parentProfileID string
	if err := tx.QueryRow(ctx, `
		INSERT INTO parent_profiles (user_id, phone_wa)
		VALUES ($1, $2)
		RETURNING id::text
	`, input.ParentUserID, phone).Scan(&parentProfileID); err != nil {
		return fmt.Errorf("create parent profile: %w", err)
	}

	if _, err := tx.Exec(ctx, `
		INSERT INTO students (
			app_id, user_id, nisn, full_name, classroom_id, classroom_name, major_code,
			parent_name, parent_phone, parent_id, updated_at
		)
		VALUES ($1, $2, $3, $4, NULLIF($5, '')::uuid, $6, $7, $8, $9, $10, now())
	`, input.StudentID, input.StudentUserID, input.NISN, input.FullName,
		input.ClassroomID, input.ClassroomName, input.MajorCode, input.ParentName, phone, parentProfileID); err != nil {
		if isUniqueViolation(err, "students_nisn_key") {
			return ErrStudentNISNExists
		}
		return fmt.Errorf("create student profile: %w", err)
	}

	return tx.Commit(ctx)
}

// DeleteStudentOnboarding rolls back the database half when the state snapshot
// cannot be written.
func (p *Postgres) DeleteStudentOnboarding(ctx context.Context, studentUserID, parentUserID string) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, studentUserID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `DELETE FROM users WHERE id = $1`, parentUserID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) || pgErr.Code != "23505" {
		return false
	}
	return constraint == "" || pgErr.ConstraintName == constraint
}
