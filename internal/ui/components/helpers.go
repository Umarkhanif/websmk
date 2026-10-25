package components

import "wesmk/internal/ui"

func mobileNavClass(item ui.NavItem, active string) string {
	if item.ID == active {
		return "mobile-nav-item is-active"
	}
	return "mobile-nav-item"
}

func mobileMoreClass(items []ui.NavItem, active string) string {
	if ui.MoreActive(items, active) {
		return "mobile-nav-item is-active"
	}
	return "mobile-nav-item"
}
