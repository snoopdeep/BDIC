package auth

import (
	"errors"
	"fmt"
	"strings"
	"unicode"

	"golang.org/x/crypto/bcrypt"

	"bdic/backend/internal/httpx"
)

// bcryptCost is deliberately above bcrypt's default of 10. At 12 a hash takes
// roughly a quarter of a second on the kind of machine this runs on, which is
// invisible to someone signing in and expensive for someone guessing.
const bcryptCost = 12

// HashPassword returns the bcrypt hash of a password. The plaintext is never
// stored, logged, or returned anywhere.
func HashPassword(plain string) (string, error) {
	if err := CheckPasswordStrength(plain); err != nil {
		return "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(plain), bcryptCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// VerifyPassword reports whether the plaintext matches the stored hash.
//
// bcrypt.CompareHashAndPassword is constant-time for a given hash, so a wrong
// password takes the same time as a right one.
func VerifyPassword(hash, plain string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain)) == nil
}

// minPasswordLength is the floor. Length matters more than symbol variety, so
// the rule asks for length first.
const minPasswordLength = 10

// weakPasswords are the ones people actually choose at a school, in both the
// obvious English and the obvious local forms. Rejected outright.
var weakPasswords = map[string]bool{
	"password":    true,
	"password123": true,
	"12345678":    true,
	"123456789":   true,
	"1234567890":  true,
	"qwertyuiop":  true,
	"admin@1234":  true,
	"school@123":  true,
	"bdic@2026":   true,
	"bdic@1234":   true,
	"teacher@123": true,
	"principal1":  true,
}

// CheckPasswordStrength validates a new password before it is hashed.
func CheckPasswordStrength(plain string) error {
	if len([]rune(plain)) < minPasswordLength {
		return httpx.ErrBadRequest(
			fmt.Sprintf("Use at least %d characters.", minPasswordLength),
			fmt.Sprintf("कम से कम %d अक्षर रखें।", minPasswordLength)).
			WithField("password", "Too short")
	}
	if len(plain) > 128 {
		return httpx.ErrBadRequest(
			"That password is too long.",
			"यह पासवर्ड बहुत लंबा है।").
			WithField("password", "Too long")
	}
	if weakPasswords[strings.ToLower(strings.TrimSpace(plain))] {
		return httpx.ErrBadRequest(
			"That password is too easy to guess. Choose something else.",
			"यह पासवर्ड आसानी से अनुमान लगाया जा सकता है। कुछ और चुनें।").
			WithField("password", "Too common")
	}

	var hasLetter, hasDigit bool
	for _, r := range plain {
		switch {
		case unicode.IsLetter(r):
			hasLetter = true
		case unicode.IsDigit(r):
			hasDigit = true
		}
	}
	if !hasLetter || !hasDigit {
		return httpx.ErrBadRequest(
			"Include at least one letter and one number.",
			"कम से कम एक अक्षर और एक अंक शामिल करें।").
			WithField("password", "Needs a letter and a number")
	}
	return nil
}

// ErrNeedsRehash signals that a stored hash was made with an older cost and
// should be replaced on the next successful sign-in.
var ErrNeedsRehash = errors.New("password hash uses an outdated cost")

// NeedsRehash reports whether a stored hash should be upgraded. Called after a
// successful sign-in, when the plaintext is available for one moment.
func NeedsRehash(hash string) bool {
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false
	}
	return cost < bcryptCost
}
