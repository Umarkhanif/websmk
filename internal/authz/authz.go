// Package authz is the single place that answers "may this role do this?".
// Handlers and UI builders must not re-implement role checks.
package authz

import "errors"

// ErrDenied is returned when a policy check fails.
var ErrDenied = errors.New("aksi tidak diizinkan untuk peran ini")

type Action string

const (
	ActionRead    Action = "read"
	ActionCreate  Action = "create"
	ActionUpdate  Action = "update"
	ActionDelete  Action = "delete"
	ActionApprove Action = "approve"
	ActionSubmit  Action = "submit"
)

type Resource string

const (
	ResourceStudentProfile Resource = "student_profile"
	ResourceFinance        Resource = "finance"
	ResourceAttendance     Resource = "attendance"
	ResourceExam           Resource = "exam"
	ResourceDiscipline     Resource = "discipline"
	ResourcePkl            Resource = "pkl"
	ResourceBkk            Resource = "bkk"
	ResourceLetter         Resource = "letter"
	ResourceSchoolSettings Resource = "school_settings"
	ResourceBugReport      Resource = "bug_report"
)

// Ownership says whether the resource belongs to the viewer. The caller is
// responsible for resolving it from server-side data, never from a form field.
type Ownership struct {
	Own bool
}

// Can applies the role/resource/action matrix from docs/master-plan-2026-11-23.md.
// Unknown roles and combinations are denied by default.
func Can(role string, action Action, resource Resource, own Ownership) bool {
	switch resource {
	case ResourceStudentProfile:
		switch action {
		case ActionRead:
			if (role == "student" || role == "parent") && own.Own {
				return true
			}
			return role == "teacher" || role == "discipline" || role == "piket" ||
				role == "treasurer" || role == "admin_tu" || role == "headmaster"
		case ActionUpdate:
			return role == "admin_tu" || (role == "student" && own.Own)
		case ActionCreate, ActionDelete:
			return role == "admin_tu"
		}

	case ResourceFinance:
		switch action {
		case ActionRead:
			if (role == "student" || role == "parent") && own.Own {
				return true
			}
			return role == "treasurer" || role == "admin_tu" || role == "headmaster"
		case ActionCreate, ActionUpdate:
			return role == "treasurer"
		case ActionApprove:
			return role == "headmaster"
		}

	case ResourceAttendance:
		switch action {
		case ActionRead:
			if (role == "student" || role == "parent") && own.Own {
				return true
			}
			return role == "teacher" || role == "discipline" || role == "piket" ||
				role == "admin_tu" || role == "headmaster"
		case ActionUpdate:
			return role == "teacher" || role == "piket" || role == "discipline"
		case ActionSubmit:
			return (role == "student" || role == "parent") && own.Own
		}

	case ResourceExam:
		switch action {
		case ActionRead:
			return role == "teacher" || (role == "student" && own.Own)
		case ActionCreate, ActionUpdate, ActionDelete, ActionApprove:
			return role == "teacher"
		case ActionSubmit:
			return role == "student" && own.Own
		}

	case ResourceDiscipline:
		switch action {
		case ActionRead:
			if (role == "student" || role == "parent") && own.Own {
				return true
			}
			return role == "discipline" || role == "piket" || role == "teacher" || role == "headmaster"
		case ActionCreate:
			return role == "discipline" || role == "teacher" || role == "piket" || role == "student"
		case ActionUpdate, ActionDelete, ActionApprove:
			return role == "discipline"
		}

	case ResourcePkl:
		switch action {
		case ActionRead:
			if (role == "student" || role == "parent") && own.Own {
				return true
			}
			return role == "teacher" || role == "admin_tu" || role == "headmaster"
		case ActionCreate, ActionUpdate:
			return role == "admin_tu" || role == "teacher" || (role == "student" && own.Own)
		case ActionApprove:
			return role == "teacher"
		case ActionDelete:
			return role == "admin_tu"
		}

	case ResourceBkk:
		switch action {
		case ActionRead:
			if role == "student" && own.Own {
				return true
			}
			return role == "admin_tu" || role == "headmaster" || role == "teacher"
		case ActionCreate:
			return role == "admin_tu" || (role == "student" && own.Own)
		case ActionUpdate, ActionDelete:
			return role == "admin_tu"
		}

	case ResourceLetter:
		switch action {
		case ActionRead:
			if (role == "student" || role == "parent") && own.Own {
				return true
			}
			return role == "admin_tu" || role == "headmaster" || role == "discipline" || role == "teacher"
		case ActionCreate:
			return role == "admin_tu" || role == "discipline"
		case ActionApprove:
			return role == "headmaster"
		case ActionUpdate, ActionDelete:
			return role == "admin_tu"
		}

	case ResourceSchoolSettings:
		switch action {
		case ActionRead, ActionUpdate:
			return role == "admin_tu" || role == "headmaster"
		case ActionDelete:
			return role == "headmaster"
		}

	case ResourceBugReport:
		switch action {
		case ActionCreate:
			return role != ""
		case ActionRead:
			return role == "developer" || own.Own
		case ActionUpdate, ActionDelete:
			return role == "developer"
		}
	}

	return false
}
