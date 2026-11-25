package pages

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"wesmk/internal/ui"
)

func TestMagicLinkConfirmRequiresPost(t *testing.T) {
	var buffer bytes.Buffer
	page := ui.MagicLinkPage{Token: "token-uji", CSRFToken: "csrf-uji"}
	if err := MagicLink(page).Render(context.Background(), &buffer); err != nil {
		t.Fatalf("render: %v", err)
	}

	html := buffer.String()
	if !strings.Contains(html, `method="post"`) {
		t.Fatal("confirmation form must submit with POST")
	}
	if !strings.Contains(html, `action="/login/magic/token-uji"`) {
		t.Fatal("form must post back to the magic link route")
	}
	if !strings.Contains(html, `name="_csrf"`) {
		t.Fatal("confirmation form must carry the CSRF token")
	}
}
