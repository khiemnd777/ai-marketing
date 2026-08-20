package auth

import "testing"

func TestPasswordRoundTrip(t *testing.T) {
	params := Argon2Params{Memory: 8 * 1024, Iterations: 1, Parallelism: 1, SaltLength: 16, KeyLength: 32}
	encoded, err := HashPassword("correct horse battery staple", params)
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	valid, err := VerifyPassword("correct horse battery staple", encoded)
	if err != nil || !valid {
		t.Fatalf("VerifyPassword() = %v, %v", valid, err)
	}
	valid, err = VerifyPassword("incorrect password", encoded)
	if err != nil || valid {
		t.Fatalf("VerifyPassword(wrong) = %v, %v", valid, err)
	}
}

func TestPasswordRejectsMalformedHash(t *testing.T) {
	if _, err := VerifyPassword("password", "not-a-hash"); err == nil {
		t.Fatal("expected malformed hash error")
	}
}
