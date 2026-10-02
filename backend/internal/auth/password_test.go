package auth

import (
	"testing"
	"time"
)

func TestHashAndVerifyPassword(t *testing.T) {
	const plain = "chandauli-2026-school"

	hash, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}

	// The plaintext must not be recoverable from, or even visible in, the hash.
	if hash == plain {
		t.Fatal("HashPassword returned the plaintext")
	}
	if len(hash) < 50 {
		t.Fatalf("hash looks too short to be bcrypt: %q", hash)
	}

	if !VerifyPassword(hash, plain) {
		t.Error("VerifyPassword rejected the correct password")
	}
	if VerifyPassword(hash, plain+"x") {
		t.Error("VerifyPassword accepted a wrong password")
	}
	if VerifyPassword(hash, "") {
		t.Error("VerifyPassword accepted an empty password")
	}
}

// bcrypt salts every hash, so the same password hashed twice must produce two
// different strings. Otherwise a stolen table would reveal which users share a
// password.
func TestHashPasswordIsSalted(t *testing.T) {
	const plain = "chandauli-2026-school"

	first, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}
	second, err := HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}

	if first == second {
		t.Error("the same password hashed twice produced the same hash")
	}
	if !VerifyPassword(first, plain) || !VerifyPassword(second, plain) {
		t.Error("a salted hash failed to verify")
	}
}

func TestVerifyPasswordRejectsGarbageHash(t *testing.T) {
	for _, hash := range []string{"", "not-a-hash", "$2a$12$tooshort"} {
		if VerifyPassword(hash, "anything") {
			t.Errorf("VerifyPassword accepted the malformed hash %q", hash)
		}
	}
}

func TestCheckPasswordStrength(t *testing.T) {
	accepted := []string{
		"chandauli2026",
		"Bhagwan-Das-9",
		"a1b2c3d4e5f6",
	}
	for _, password := range accepted {
		if err := CheckPasswordStrength(password); err != nil {
			t.Errorf("CheckPasswordStrength(%q) rejected a reasonable password: %v", password, err)
		}
	}

	rejected := map[string]string{
		"short":            "abc1",
		"no digits":        "abcdefghijkl",
		"no letters":       "123456789012",
		"common":           "password123",
		"common uppercase": "Password123",
		"school default":   "BDIC@2026",
		"empty":            "",
	}
	for name, password := range rejected {
		t.Run(name, func(t *testing.T) {
			if err := CheckPasswordStrength(password); err == nil {
				t.Errorf("CheckPasswordStrength(%q) accepted a weak password", password)
			}
		})
	}
}

// HashPassword must apply the strength rules, so a weak password cannot slip in
// through a code path that forgot to check it first.
func TestHashPasswordEnforcesStrength(t *testing.T) {
	if _, err := HashPassword("abc"); err == nil {
		t.Error("HashPassword accepted a password that fails the strength rules")
	}
}

func TestNeedsRehash(t *testing.T) {
	hash, err := HashPassword("chandauli2026")
	if err != nil {
		t.Fatalf("HashPassword returned an error: %v", err)
	}
	if NeedsRehash(hash) {
		t.Error("a hash we just made at the current cost was marked as needing a rehash")
	}

	// A cost-10 hash (bcrypt's default, below ours) should be upgraded.
	const costTenHash = "$2a$10$N9qo8uLOickgx2ZMRZoMye1VdLLpOa3mPPI/mj6KKPQXe1L2Cs2Sq"
	if !NeedsRehash(costTenHash) {
		t.Error("a cost-10 hash was not marked as needing a rehash")
	}

	if NeedsRehash("not-a-hash") {
		t.Error("a malformed hash was marked as needing a rehash")
	}
}

func TestLimiterAllowsUpToTheLimitThenBlocks(t *testing.T) {
	limiter := NewLimiter(3, time.Minute)

	for attempt := 1; attempt <= 3; attempt++ {
		if !limiter.Allow("someone") {
			t.Fatalf("attempt %d was blocked but should have been allowed", attempt)
		}
	}
	if limiter.Allow("someone") {
		t.Error("the fourth attempt was allowed past a limit of three")
	}

	// A different key has its own budget, so one person being locked out does
	// not lock out everybody else.
	if !limiter.Allow("someone-else") {
		t.Error("a different key was blocked by another key's attempts")
	}
}

func TestLimiterResetClearsAKey(t *testing.T) {
	limiter := NewLimiter(1, time.Minute)

	if !limiter.Allow("teacher") {
		t.Fatal("the first attempt was blocked")
	}
	if limiter.Allow("teacher") {
		t.Fatal("the second attempt was allowed past a limit of one")
	}

	// This is what a successful sign-in does, so two typos followed by the
	// right password does not leave the user locked out.
	limiter.Reset("teacher")

	if !limiter.Allow("teacher") {
		t.Error("the key was still blocked after Reset")
	}
}

func TestLimiterWindowExpires(t *testing.T) {
	limiter := NewLimiter(1, 20*time.Millisecond)

	if !limiter.Allow("parent") {
		t.Fatal("the first attempt was blocked")
	}
	if limiter.Allow("parent") {
		t.Fatal("the second attempt was allowed inside the window")
	}

	time.Sleep(40 * time.Millisecond)

	if !limiter.Allow("parent") {
		t.Error("the key was still blocked after the window had passed")
	}
}
