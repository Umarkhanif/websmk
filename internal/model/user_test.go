package model

import (
	"encoding/json"
	"testing"
)

func TestNormalizePhone(t *testing.T) {
	cases := map[string]string{
		"+62 812-3456-7800": "081234567800",
		"081234567800":      "081234567800",
		"6281234567800":     "081234567800",
		"":                  "",
	}
	for input, expected := range cases {
		if actual := NormalizePhone(input); actual != expected {
			t.Errorf("NormalizePhone(%q) = %q, want %q", input, actual, expected)
		}
	}
}

func TestUserJSONMatchesClientShape(t *testing.T) {
	user := User{
		ID:               "usr-student-1",
		Username:         "0058471290",
		FullName:         "Muhammad Budi Santoso",
		Role:             "student",
		PhoneWA:          "081234567801",
		AvatarURL:        "https://example.test/avatar.png",
		StudentProfileID: "std-budi-01",
		Attributes:       map[string]any{"isKetuaKelas": true},
		PasswordHash:     "scrypt-should-never-leak",
	}
	encoded, err := json.Marshal(user)
	if err != nil {
		t.Fatal(err)
	}

	var decoded map[string]any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}

	if decoded["username_nisn_nip"] != user.Username {
		t.Fatalf("missing username_nisn_nip: %s", encoded)
	}
	if decoded["studentProfileId"] != user.StudentProfileID {
		t.Fatalf("missing studentProfileId: %s", encoded)
	}
	if decoded["isKetuaKelas"] != true {
		t.Fatalf("attributes not merged: %s", encoded)
	}
	if _, leaked := decoded["PasswordHash"]; leaked {
		t.Fatalf("password hash leaked in JSON: %s", encoded)
	}
	if _, leaked := decoded["password_hash"]; leaked {
		t.Fatalf("password hash leaked in JSON: %s", encoded)
	}
}
