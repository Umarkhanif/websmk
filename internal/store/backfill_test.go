package store

import "testing"

func TestBackfillHelpers(t *testing.T) {
	if got := textAny("abc"); got != "abc" {
		t.Fatalf("textAny = %q", got)
	}
	if got := textAny(42); got != "" {
		t.Fatalf("textAny(number) = %q", got)
	}
	if got := orText(nil, "NONE"); got != "NONE" {
		t.Fatalf("orText = %q", got)
	}
	if got := orText("PAID", "UNPAID"); got != "PAID" {
		t.Fatalf("orText = %q", got)
	}
	if got := numberAny(float64(350000)); got != 350000 {
		t.Fatalf("numberAny = %v", got)
	}
	if got := numberAny(true); got != 0 {
		t.Fatalf("numberAny(bool) = %v", got)
	}
	list := asListAny([]any{map[string]any{"id": "std-1"}})
	if len(list) != 1 || asMapAny(list[0])["id"] != "std-1" {
		t.Fatalf("asListAny/asMapAny = %v", list)
	}
}
