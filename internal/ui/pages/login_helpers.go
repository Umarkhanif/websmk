package pages

func loginTabClass(tab, target string) string {
	if tab == target {
		return "login-tab is-active"
	}
	return "login-tab"
}

func loginTabSelected(tab, target string) string {
	if tab == target {
		return "true"
	}
	return "false"
}
