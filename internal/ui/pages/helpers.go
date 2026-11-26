package pages

import (
	"fmt"
	"net/url"
)

func sectionTabClass(current, target string) string {
	if current == target {
		return "section-tab is-active"
	}
	return "section-tab"
}

func studentPageURL(query string, page int) string {
	values := url.Values{}
	values.Set("section", "students")
	if query != "" {
		values.Set("q", query)
	}
	values.Set("page", fmt.Sprintf("%d", page))
	return "/admin/master-data?" + values.Encode()
}

// meterWidth turns a 0-100 score into a CSS width and keeps bad data from
// breaking the layout.
func meterWidth(value int) string {
	if value < 0 {
		value = 0
	}
	if value > 100 {
		value = 100
	}
	return fmt.Sprintf("width:%d%%", value)
}

func attendanceBadgeClass(status string) string {
	switch status {
	case "HADIR_GERBANG":
		return "badge-success"
	case "SAKIT_ORTU", "IZIN_ORTU":
		return "badge-warning"
	case "ALPA_BOLOS":
		return "badge-danger"
	default:
		return "badge-neutral"
	}
}

func attendanceIcon(status string) string {
	switch status {
	case "HADIR_GERBANG":
		return "check"
	case "SAKIT_ORTU", "IZIN_ORTU", "ALPA_BOLOS":
		return "alert"
	default:
		return "clock"
	}
}

func statusBadgeClass(status string) string {
	switch status {
	case "VALID_FULL":
		return "badge-success"
	case "TASK_ONLY", "PENDING":
		return "badge-warning"
	case "EMPTY_CLASS":
		return "badge-danger"
	default:
		return "badge-neutral"
	}
}

func leaderButtonClass(current, target string) string {
	if current == target {
		return "button button-primary button-small"
	}
	return "button button-secondary button-small"
}

func quizStepClass(index int) string {
	if index == 0 {
		return "quiz-step is-active"
	}
	return "quiz-step"
}
