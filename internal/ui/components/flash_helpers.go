package components

func flashClass(kind string) string {
	switch kind {
	case "success":
		return "flash flash-success"
	case "warning":
		return "flash flash-warning"
	case "danger":
		return "flash flash-danger"
	default:
		return "flash flash-info"
	}
}

func flashIcon(kind string) string {
	switch kind {
	case "success":
		return "check"
	case "warning", "danger":
		return "alert"
	default:
		return "help"
	}
}
