package store

import (
	"context"
	"encoding/json"

	"wesmk/internal/domain/exam"
)

const examQuestionSelect = `
	SELECT
		q.app_id, COALESCE(e.app_id, ''), q.question_number, q.question_type,
		q.subject_name, q.kd_code, q.competency_tag,
		q.content_html_or_canvas, q.template_content,
		COALESCE(q.options_jsonb, '[]'::jsonb),
		q.answer_key, q.essay_rubric,
		COALESCE(q.dynamic_vars_schema, '{}'::jsonb),
		q.render_shadow_canvas
	FROM exam_questions q
	LEFT JOIN exams e ON e.id = q.exam_id`

func (p *Postgres) ExamQuestions(ctx context.Context) ([]exam.Question, error) {
	rows, err := p.pool.Query(ctx, examQuestionSelect+`
		ORDER BY q.question_number, q.app_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	questions := make([]exam.Question, 0)
	for rows.Next() {
		var question exam.Question
		var optionsRaw, varsRaw []byte
		if err := rows.Scan(
			&question.AppID,
			&question.ExamAppID,
			&question.QuestionNumber,
			&question.QuestionType,
			&question.SubjectName,
			&question.KDCode,
			&question.CompetencyTag,
			&question.QuestionText,
			&question.TemplateContent,
			&optionsRaw,
			&question.CorrectKey,
			&question.EssayRubric,
			&varsRaw,
			&question.RenderShadowCanvas,
		); err != nil {
			return nil, err
		}
		if len(optionsRaw) > 0 {
			if err := json.Unmarshal(optionsRaw, &question.Options); err != nil {
				return nil, err
			}
		}
		if len(varsRaw) > 0 {
			if err := json.Unmarshal(varsRaw, &question.DynamicVars); err != nil {
				return nil, err
			}
		}
		questions = append(questions, question)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return questions, nil
}

const examSessionSelect = `
	SELECT
		s.app_id, COALESCE(st.app_id, ''), COALESCE(st.full_name, ''), COALESCE(st.nisn, ''),
		s.status, s.progress, s.yellow_warn_count, s.lock_count,
		s.last_violation, s.unlock_token, s.emergency_2h
	FROM exam_sessions s
	JOIN students st ON st.id = s.student_id`

func (p *Postgres) ExamSessions(ctx context.Context) ([]exam.Session, error) {
	rows, err := p.pool.Query(ctx, examSessionSelect+`
		ORDER BY s.updated_at DESC, s.app_id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExamSessions(rows)
}

func (p *Postgres) ExamSessionForStudent(ctx context.Context, studentAppID string) ([]exam.Session, error) {
	rows, err := p.pool.Query(ctx, examSessionSelect+`
		WHERE st.app_id = $1
		ORDER BY s.updated_at DESC, s.app_id`, studentAppID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanExamSessions(rows)
}

func scanExamSessions(rows invoiceRows) ([]exam.Session, error) {
	sessions := make([]exam.Session, 0)
	for rows.Next() {
		var session exam.Session
		if err := rows.Scan(
			&session.AppID,
			&session.StudentAppID,
			&session.StudentName,
			&session.NISN,
			&session.Status,
			&session.Progress,
			&session.WarnCount,
			&session.LockCount,
			&session.LastViolation,
			&session.UnlockToken,
			&session.Emergency2h,
		); err != nil {
			return nil, err
		}
		sessions = append(sessions, session)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return sessions, nil
}
