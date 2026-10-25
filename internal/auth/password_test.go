package auth

import "testing"

func TestPasswordHashAndVerify(t *testing.T) {
	hash, err := HashPassword("Rahasia#123")
	if err != nil {
		t.Fatalf("hash: %v", err)
	}
	if hash == "Rahasia#123" {
		t.Fatal("password was stored in plaintext")
	}
	if !VerifyPassword("Rahasia#123", hash) {
		t.Fatal("correct password rejected")
	}
	if VerifyPassword("Rahasia#124", hash) {
		t.Fatal("wrong password accepted")
	}
	if VerifyPassword("Rahasia#123", "bukan-hash") {
		t.Fatal("malformed hash accepted")
	}
}

func TestPasswordHashUsesUniqueSalts(t *testing.T) {
	first, err := HashPassword("sama-persis")
	if err != nil {
		t.Fatal(err)
	}
	second, err := HashPassword("sama-persis")
	if err != nil {
		t.Fatal(err)
	}
	if first == second {
		t.Fatal("two hashes of the same password should not be identical")
	}
}
