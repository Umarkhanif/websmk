package store

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
)

// BackfillDomainData copies the core entities from the app_state JSON blob into
// the normalized tables created by migration 003. Existing rows are never
// overwritten, so it is safe to run on every boot.
func (p *Postgres) BackfillDomainData(ctx context.Context) error {
	raw, err := p.GetState(ctx)
	if err != nil || raw == nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(raw, &doc); err != nil {
		return err
	}
	if err := p.syncDomain(ctx, doc, false); err != nil {
		return err
	}
	log.Printf("[db] backfill: %d siswa, %d tagihan, %d jurnal keuangan, %d presensi, %d soal, %d sesi ujian, %d PKL, %d BKK dari app_state",
		len(asListAny(doc["students"])), len(asListAny(doc["invoices"])),
		len(asListAny(doc["ledger"])), len(asListAny(doc["attendances"])),
		len(asListAny(doc["examQuestions"])), len(asListAny(doc["proctorStudents"])),
		len(asListAny(doc["pklPlacements"])), len(asListAny(doc["bkkJobs"])))
	return nil
}

// SyncDomainFromState keeps the normalized tables current after a mutation.
// The blob is still the read model, so this shadow copy prevents the domain
// tables from going stale while endpoints are migrated one by one.
func (p *Postgres) SyncDomainFromState(ctx context.Context, doc map[string]any) error {
	return p.syncDomain(ctx, doc, true)
}

func (p *Postgres) syncDomain(ctx context.Context, doc map[string]any, overwrite bool) error {
	tx, err := p.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()

	// The first backfill must not clobber finance rows, but it does fill the
	// profile columns added in migration 007 when the table row predates them.
	// A row without app_id predates migration 007, so this backfill is its
	// upgrade: copy every profile column, including numerics and flags, once.
	// Later boot backfills leave an already-upgraded row alone.
	studentConflict := `DO UPDATE SET
		app_id = COALESCE(NULLIF(students.app_id, ''), EXCLUDED.app_id),
		nik = COALESCE(NULLIF(students.nik, ''), EXCLUDED.nik),
		full_name = COALESCE(NULLIF(students.full_name, ''), EXCLUDED.full_name),
		classroom_name = COALESCE(NULLIF(students.classroom_name, ''), EXCLUDED.classroom_name),
		major_code = COALESCE(NULLIF(students.major_code, ''), EXCLUDED.major_code),
		parent_name = COALESCE(NULLIF(students.parent_name, ''), EXCLUDED.parent_name),
		parent_phone = COALESCE(NULLIF(students.parent_phone, ''), EXCLUDED.parent_phone),
		bmw_choice = COALESCE(NULLIF(students.bmw_choice, ''), EXCLUDED.bmw_choice),
		homeroom_note = COALESCE(NULLIF(students.homeroom_note, ''), EXCLUDED.homeroom_note),
		is_ketua_kelas = CASE WHEN COALESCE(students.app_id, '') = '' THEN EXCLUDED.is_ketua_kelas ELSE students.is_ketua_kelas END,
		is_sekretaris = CASE WHEN COALESCE(students.app_id, '') = '' THEN EXCLUDED.is_sekretaris ELSE students.is_sekretaris END,
		is_alumni = CASE WHEN COALESCE(students.app_id, '') = '' THEN EXCLUDED.is_alumni ELSE students.is_alumni END,
		scholarship_type = CASE WHEN COALESCE(students.app_id, '') = '' THEN EXCLUDED.scholarship_type ELSE students.scholarship_type END,
		discount_percent = CASE WHEN COALESCE(students.app_id, '') = '' THEN EXCLUDED.discount_percent ELSE students.discount_percent END,
		attitude_score = CASE WHEN COALESCE(students.app_id, '') = '' THEN EXCLUDED.attitude_score ELSE students.attitude_score END,
		updated_at = now()`
	// Same upgrade marker as students: fill app_id once on pre-008 rows.
	invoiceConflict := `DO UPDATE SET
		app_id = COALESCE(NULLIF(tuition_invoices.app_id, ''), EXCLUDED.app_id)`
	if overwrite {
		studentConflict = `DO UPDATE SET
			app_id = EXCLUDED.app_id,
			user_id = EXCLUDED.user_id,
			nik = EXCLUDED.nik,
			full_name = EXCLUDED.full_name,
			classroom_name = EXCLUDED.classroom_name,
			major_code = EXCLUDED.major_code,
			parent_name = EXCLUDED.parent_name,
			parent_phone = EXCLUDED.parent_phone,
			bmw_choice = EXCLUDED.bmw_choice,
			homeroom_note = EXCLUDED.homeroom_note,
			is_ketua_kelas = EXCLUDED.is_ketua_kelas,
			is_sekretaris = EXCLUDED.is_sekretaris,
			is_alumni = EXCLUDED.is_alumni,
			scholarship_type = EXCLUDED.scholarship_type,
			discount_percent = EXCLUDED.discount_percent,
			attitude_score = EXCLUDED.attitude_score,
			updated_at = now()`
		invoiceConflict = `DO UPDATE SET
			app_id = EXCLUDED.app_id,
			paid_amount = EXCLUDED.paid_amount,
			status = EXCLUDED.status,
			final_amount = EXCLUDED.final_amount`
	}

	// app student id -> database UUID, keyed by the unique NISN.
	studentIDs := map[string]string{}
	for _, item := range asListAny(doc["students"]) {
		record := asMapAny(item)
		nisn := textAny(record["nisn"])
		if nisn == "" {
			continue
		}
		userID := textAny(record["userId"])
		query := fmt.Sprintf(`
			INSERT INTO students (
				app_id, user_id, nisn, nik, full_name, classroom_name, major_code,
				parent_name, parent_phone, bmw_choice, homeroom_note,
				is_ketua_kelas, is_sekretaris, is_alumni,
				scholarship_type, discount_percent, attitude_score, updated_at
			)
			SELECT $1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17, now()
			WHERE EXISTS (SELECT 1 FROM users WHERE id = $2)
			ON CONFLICT (nisn) %s
		`, studentConflict)
		if _, err := tx.Exec(ctx, query,
			textAny(record["id"]), userID, nisn, textAny(record["nik"]),
			textAny(record["fullName"]), textAny(record["classroomName"]), textAny(record["majorCode"]),
			textAny(record["parentName"]), textAny(record["parentPhone"]),
			textAny(record["bmwChoice"]), textAny(record["homeroomNote"]),
			record["isKetuaKelas"] == true, record["isSekretaris"] == true, record["isAlumni"] == true,
			orText(record["scholarshipType"], "NONE"), numberAny(record["discountPercent"]),
			int64(numberAny(record["attitudeScore"]))); err != nil {
			return err
		}
		var id string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM students WHERE nisn = $1`, nisn).Scan(&id); err == nil {
			studentIDs[textAny(record["id"])] = id
		}
	}

	// Exam bank: one exams row per app exam id, then questions keyed by app id.
	examIDs := map[string]string{}
	for _, item := range asListAny(doc["examQuestions"]) {
		record := asMapAny(item)
		examAppID := textAny(record["examId"])
		if examAppID == "" {
			examAppID = "exam-active"
		}
		if _, seen := examIDs[examAppID]; !seen {
			var examUUID string
			err := tx.QueryRow(ctx, `
				INSERT INTO exams (app_id, exam_code, subject_id, subject_name, is_active)
				VALUES ($1, $1, $2, $2, TRUE)
				ON CONFLICT (app_id) DO UPDATE SET
					subject_id = EXCLUDED.subject_id,
					subject_name = EXCLUDED.subject_name
				RETURNING id::text
			`, examAppID, orText(record["subjectName"], "Ujian Sekolah")).Scan(&examUUID)
			if err != nil {
				return err
			}
			examIDs[examAppID] = examUUID
		}

		options, err := json.Marshal(record["options"])
		if err != nil || len(options) == 0 || string(options) == "null" {
			options = []byte("[]")
		}
		vars, err := json.Marshal(record["dynamicVars"])
		if err != nil || len(vars) == 0 || string(vars) == "null" {
			vars = []byte("{}")
		}
		questionType := orText(record["questionType"], "MCQ")
		if questionType == "MULTIPLE_CHOICE" {
			questionType = "MCQ"
		}
		_, err = tx.Exec(ctx, `
			INSERT INTO exam_questions (
				app_id, exam_id, question_number, question_type, subject_name,
				kd_code, competency_tag, content_html_or_canvas, template_content,
				options_jsonb, dynamic_vars_schema, answer_key, essay_rubric, render_shadow_canvas
			) VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10::jsonb, $11::jsonb, $12, $13, $14)
			ON CONFLICT (app_id) DO UPDATE SET
				exam_id = EXCLUDED.exam_id,
				question_number = EXCLUDED.question_number,
				question_type = EXCLUDED.question_type,
				subject_name = EXCLUDED.subject_name,
				kd_code = EXCLUDED.kd_code,
				competency_tag = EXCLUDED.competency_tag,
				content_html_or_canvas = EXCLUDED.content_html_or_canvas,
				template_content = EXCLUDED.template_content,
				options_jsonb = EXCLUDED.options_jsonb,
				dynamic_vars_schema = EXCLUDED.dynamic_vars_schema,
				answer_key = EXCLUDED.answer_key,
				essay_rubric = EXCLUDED.essay_rubric,
				render_shadow_canvas = EXCLUDED.render_shadow_canvas
		`, textAny(record["id"]), examIDs[examAppID], int64(numberAny(record["questionNumber"])),
			questionType, textAny(record["subjectName"]), textAny(record["kdCode"]),
			textAny(record["competencyTag"]), textAny(record["questionText"]),
			textAny(record["templateContent"]), string(options), string(vars),
			textAny(record["correctKey"]), textAny(record["essayRubric"]),
			record["renderShadowCanvas"] == true)
		if err != nil {
			return err
		}
	}

	// Proctor rows are the active exam sessions, keyed by their app id.
	for _, item := range asListAny(doc["proctorStudents"]) {
		record := asMapAny(item)
		appID := textAny(record["id"])
		studentUUID, ok := studentIDs[textAny(record["studentId"])]
		if appID == "" || !ok {
			continue
		}
		var examArg any
		if defaultExam := examIDs["exam-cbt-tkj-01"]; defaultExam != "" {
			examArg = defaultExam
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO exam_sessions (
				app_id, exam_id, student_id, status, progress,
				yellow_warn_count, lock_count, last_violation, unlock_token,
				emergency_2h, updated_at
			) VALUES ($1, $2::uuid, $3, $4, $5, $6, $7, $8, $9, $10, now())
			ON CONFLICT (app_id) DO UPDATE SET
				exam_id = EXCLUDED.exam_id,
				student_id = EXCLUDED.student_id,
				status = EXCLUDED.status,
				progress = EXCLUDED.progress,
				yellow_warn_count = EXCLUDED.yellow_warn_count,
				lock_count = EXCLUDED.lock_count,
				last_violation = EXCLUDED.last_violation,
				unlock_token = EXCLUDED.unlock_token,
				emergency_2h = EXCLUDED.emergency_2h,
				updated_at = now()
		`, appID, examArg, studentUUID, orText(record["status"], "ACTIVE"),
			textAny(record["progress"]), int64(numberAny(record["warnCount"])),
			int64(numberAny(record["lockCount"])), textAny(record["lastViolation"]),
			textAny(record["unlockToken"]), record["hasEmergency2hTicket"] == true)
		if err != nil {
			return err
		}
	}

	// Attendance is keyed by its app-level ID and belongs to a mapped student.
	for _, item := range asListAny(doc["attendances"]) {
		record := asMapAny(item)
		appID := textAny(record["id"])
		studentUUID, ok := studentIDs[textAny(record["studentId"])]
		if appID == "" || !ok {
			continue
		}
		_, err := tx.Exec(ctx, `			INSERT INTO attendances (
				app_id, student_id, schedule_id, student_name, status, verified_source,
				attendance_date, classroom_id, updated_at
			) VALUES (
				$1, $2, $3, $4, $5, $6,
				COALESCE(NULLIF($7, '')::timestamptz, now())::date,
				(SELECT classroom_id FROM students WHERE id = $2),
				COALESCE(NULLIF($7, '')::timestamptz, now())
			)
			ON CONFLICT (app_id) DO UPDATE SET
				student_id = EXCLUDED.student_id,
				schedule_id = EXCLUDED.schedule_id,
				student_name = EXCLUDED.student_name,
				status = EXCLUDED.status,
				verified_source = EXCLUDED.verified_source,
				attendance_date = EXCLUDED.attendance_date,
				classroom_id = EXCLUDED.classroom_id,
				updated_at = EXCLUDED.updated_at
			`, appID, studentUUID, textAny(record["scheduleId"]), textAny(record["studentName"]),
			textAny(record["status"]), textAny(record["verifiedSource"]), textAny(record["updatedAt"]))
		if err != nil {
			return err
		}
	}

	// app invoice id -> database UUID.
	invoiceIDs := map[string]string{}
	for _, item := range asListAny(doc["invoices"]) {
		record := asMapAny(item)
		number := textAny(record["invoiceNumber"])
		studentID, ok := studentIDs[textAny(record["studentId"])]
		if number == "" || !ok {
			continue
		}
		query := fmt.Sprintf(`
			INSERT INTO tuition_invoices (
				invoice_number, student_id, period_month_year, base_amount,
				scholarship_deduction, final_amount, paid_amount, status,
				va_number, gateway_provider, app_id
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
			ON CONFLICT (invoice_number) %s
		`, invoiceConflict)
		if _, err := tx.Exec(ctx, query, number, studentID, textAny(record["periodMonthYear"]),
			numberAny(record["baseAmount"]), numberAny(record["scholarshipDeduction"]),
			numberAny(record["finalAmount"]), numberAny(record["paidAmount"]),
			orText(record["status"], "UNPAID"), textAny(record["vaNumber"]),
			textAny(record["gatewayProvider"]), textAny(record["id"])); err != nil {
			return err
		}
		var id string
		if err := tx.QueryRow(ctx, `SELECT id::text FROM tuition_invoices WHERE invoice_number = $1`, number).Scan(&id); err == nil {
			invoiceIDs[textAny(record["id"])] = id
		}
	}

	// Ledger is append-only: an explicit ID makes the sync idempotent and the
	// trigger in migration 003 blocks any later edit or deletion.
	for _, item := range asListAny(doc["ledger"]) {
		record := asMapAny(item)
		id := numberAny(record["id"])
		if id <= 0 {
			continue
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO financial_audit_ledgers (
				id, invoice_id, tx_type, amount_delta, actor_user_id,
				treasurer_pin_verified, signature_hmac, created_at
			) VALUES (
				$1, NULLIF($2, '')::uuid, $3, $4,
				(SELECT id FROM users WHERE id = $5), $6, $7,
				COALESCE(NULLIF($8, '')::timestamptz, now())
			) ON CONFLICT (id) DO NOTHING
		`, int64(id), invoiceIDs[textAny(record["invoiceId"])],
			orText(record["txType"], "UNKNOWN"), numberAny(record["amountDelta"]),
			textAny(record["actorUserId"]), record["treasurerPinVerified"] == true,
			textAny(record["signatureHmac"]), textAny(record["createdAt"]))
		if err != nil {
			return err
		}
		// The ledger row is immutable, so receipt/student/actor metadata is
		// upserted here and read through a join.
		if _, err := tx.Exec(ctx, `
			INSERT INTO financial_ledger_details (
				ledger_id, app_invoice_id, receipt_number, student_name, actor_name, reversal_approval
			) VALUES ($1, $2, $3, $4, $5, $6)
			ON CONFLICT (ledger_id) DO UPDATE SET
				app_invoice_id = EXCLUDED.app_invoice_id,
				receipt_number = EXCLUDED.receipt_number,
				student_name = EXCLUDED.student_name,
				actor_name = EXCLUDED.actor_name,
				reversal_approval = EXCLUDED.reversal_approval,
				updated_at = now()
		`, int64(id), textAny(record["invoiceId"]), textAny(record["receiptNumber"]),
			textAny(record["studentName"]), textAny(record["actorName"]),
			textAny(record["reversalApprovalId"])); err != nil {
			return err
		}
	}

	// PKL placements own attendance logs and journals through their database ID.
	placementIDs := map[string]string{}
	for _, item := range asListAny(doc["pklPlacements"]) {
		record := asMapAny(item)
		appID := textAny(record["id"])
		studentUUID, ok := studentIDs[textAny(record["studentId"])]
		if appID == "" || !ok {
			continue
		}
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO pkl_placements (
				app_id, student_id, company_name, industry_sector, address,
				gps_lat, gps_lng, radius_meters, supervisor_name, supervisor_phone,
				supervisor_pin_hash, updated_at
			) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, now())
			ON CONFLICT (app_id) DO UPDATE SET
				student_id = EXCLUDED.student_id,
				company_name = EXCLUDED.company_name,
				industry_sector = EXCLUDED.industry_sector,
				address = EXCLUDED.address,
				gps_lat = EXCLUDED.gps_lat,
				gps_lng = EXCLUDED.gps_lng,
				radius_meters = EXCLUDED.radius_meters,
				supervisor_name = EXCLUDED.supervisor_name,
				supervisor_phone = EXCLUDED.supervisor_phone,
				supervisor_pin_hash = EXCLUDED.supervisor_pin_hash,
				updated_at = now()
			RETURNING id::text
		`, appID, studentUUID, textAny(record["companyName"]), textAny(record["industrySector"]),
			textAny(record["address"]), numberAny(record["gpsLat"]), numberAny(record["gpsLng"]),
			int64(numberAny(record["radiusMeters"])), textAny(record["supervisorName"]),
			textAny(record["supervisorPhone"]), textAny(record["supervisorPin"])).Scan(&id)
		if err != nil {
			return err
		}
		placementIDs[appID] = id
	}

	for _, item := range asListAny(doc["pklAttendanceLogs"]) {
		record := asMapAny(item)
		appID := textAny(record["id"])
		placementUUID, ok := placementIDs[textAny(record["placementId"])]
		if appID == "" || !ok {
			continue
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO pkl_attendance_logs (
				app_id, placement_id, gps_lat, gps_lng, distance_meters,
				is_within_radius, selfie_url, checkin_time, created_at
			) VALUES ($1, $2::uuid, $3, $4, $5, $6, $7,
				COALESCE(NULLIF($8, '')::timestamptz, now()), COALESCE(NULLIF($8, '')::timestamptz, now()))
			ON CONFLICT (app_id) DO UPDATE SET
				placement_id = EXCLUDED.placement_id,
				gps_lat = EXCLUDED.gps_lat,
				gps_lng = EXCLUDED.gps_lng,
				distance_meters = EXCLUDED.distance_meters,
				is_within_radius = EXCLUDED.is_within_radius,
				selfie_url = EXCLUDED.selfie_url,
				checkin_time = EXCLUDED.checkin_time
		`, appID, placementUUID, numberAny(record["gpsLat"]), numberAny(record["gpsLng"]),
			int64(numberAny(record["distanceMeters"])), record["isWithinRadius"] == true,
			textAny(record["selfieUrl"]), textAny(record["checkinTime"]))
		if err != nil {
			return err
		}
	}

	for _, item := range asListAny(doc["pklJournals"]) {
		record := asMapAny(item)
		appID := textAny(record["id"])
		placementUUID, ok := placementIDs[textAny(record["placementId"])]
		if appID == "" || !ok {
			continue
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO pkl_daily_journals (
				app_id, placement_id, journal_date, content, activity_title,
				competency_learned, work_photo_url, supervisor_approved, teacher_approved, approved
			) VALUES ($1, $2::uuid, COALESCE(NULLIF($3, '')::date, CURRENT_DATE), $4, $4, $5, $6, $7, $8, $7)
			ON CONFLICT (app_id) DO UPDATE SET
				placement_id = EXCLUDED.placement_id,
				journal_date = EXCLUDED.journal_date,
				content = EXCLUDED.content,
				activity_title = EXCLUDED.activity_title,
				competency_learned = EXCLUDED.competency_learned,
				work_photo_url = EXCLUDED.work_photo_url,
				supervisor_approved = EXCLUDED.supervisor_approved,
				teacher_approved = EXCLUDED.teacher_approved,
				approved = EXCLUDED.approved
		`, appID, placementUUID, textAny(record["date"]), textAny(record["activityTitle"]),
			textAny(record["competencyLearned"]), textAny(record["workPhotoUrl"]),
			record["supervisorApproved"] == true, record["teacherApproved"] == true)
		if err != nil {
			return err
		}
	}

	// BKK postings are mapped first so applications can point at them.
	jobIDs := map[string]string{}
	for _, item := range asListAny(doc["bkkJobs"]) {
		record := asMapAny(item)
		appID := textAny(record["id"])
		if appID == "" {
			continue
		}
		var id string
		err := tx.QueryRow(ctx, `
			INSERT INTO bkk_job_postings (
				app_id, company_name, job_title, salary_range, majors,
				deadline_date, requirements, applicant_count
			) VALUES ($1, $2, $3, $4, $5, $6::date, $7, $8)
			ON CONFLICT (app_id) DO UPDATE SET
				company_name = EXCLUDED.company_name,
				job_title = EXCLUDED.job_title,
				salary_range = EXCLUDED.salary_range,
				majors = EXCLUDED.majors,
				deadline_date = EXCLUDED.deadline_date,
				requirements = EXCLUDED.requirements,
				applicant_count = EXCLUDED.applicant_count
			RETURNING id::text
		`, appID, textAny(record["companyName"]), textAny(record["jobTitle"]),
			textAny(record["salaryRange"]), stringListAny(record["targetedMajors"]),
			textAny(record["deadlineDate"]), stringListAny(record["requirements"]),
			int64(numberAny(record["applicantCount"]))).Scan(&id)
		if err != nil {
			return err
		}
		jobIDs[appID] = id
	}

	for _, item := range asListAny(doc["bkkApplications"]) {
		record := asMapAny(item)
		appID := textAny(record["id"])
		jobUUID, jobOK := jobIDs[textAny(record["jobId"])]
		studentUUID, studentOK := studentIDs[textAny(record["studentId"])]
		if appID == "" || !jobOK || !studentOK {
			continue
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO bkk_job_applications (
				app_id, posting_id, student_id, status, applied_date, created_at
			) VALUES ($1, $2::uuid, $3, $4, COALESCE(NULLIF($5, '')::date, CURRENT_DATE),
				COALESCE(NULLIF($5, '')::date, CURRENT_DATE)::timestamptz)
			ON CONFLICT (app_id) DO UPDATE SET
				posting_id = EXCLUDED.posting_id,
				student_id = EXCLUDED.student_id,
				status = EXCLUDED.status,
				applied_date = EXCLUDED.applied_date
		`, appID, jobUUID, studentUUID, orText(record["status"], "APPLIED"),
			textAny(record["appliedDate"]))
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func asListAny(value any) []any {
	list, _ := value.([]any)
	return list
}

func stringListAny(value any) []string {
	items := asListAny(value)
	out := make([]string, 0, len(items))
	for _, item := range items {
		if text, ok := item.(string); ok && text != "" {
			out = append(out, text)
		}
	}
	return out
}

func asMapAny(value any) map[string]any {
	record, _ := value.(map[string]any)
	return record
}

func textAny(value any) string {
	text, _ := value.(string)
	return text
}

func orText(value any, fallback string) string {
	if text := textAny(value); text != "" {
		return text
	}
	return fallback
}

func numberAny(value any) float64 {
	switch typed := value.(type) {
	case float64:
		return typed
	case int:
		return float64(typed)
	case int64:
		return float64(typed)
	}
	return 0
}
