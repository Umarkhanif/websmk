package pages

func loginTabClass(tab, target string) string {
	if tab == target {
		return "login-tab is-active"
	}
	return "login-tab"
}

func loginTabCurrent(tab, target string) string {
	if tab == target {
		return "page"
	}
	return "false"
}
