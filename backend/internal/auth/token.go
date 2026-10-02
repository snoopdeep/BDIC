package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"
)

// Claims are the contents of an access token.
//
// TokenVersion is the revocation mechanism: bumping users.token_version
// invalidates every token that user already holds, which is what happens when
// a password is reset or an account is suspended. There is no server-side
// session table to keep in step.
//
// UserID is carried as a string rather than a parsed UUID: the database is the
// only thing that needs to understand the format, and every query that uses it
// casts explicitly with ::uuid.
type Claims struct {
	UserID       string `json:"sub"`
	Role         string `json:"role"`
	TokenVersion int    `json:"ver"`
	IssuedAt     int64  `json:"iat"`
	ExpiresAt    int64  `json:"exp"`
}

// Signer issues and verifies access tokens.
type Signer struct {
	key []byte
	ttl time.Duration
}

// NewSigner builds a Signer. The key comes from AUTH_SIGNING_KEY, which the
// config package requires to be at least 32 characters.
func NewSigner(key []byte, ttl time.Duration) *Signer {
	return &Signer{key: key, ttl: ttl}
}

// TTL is how long a freshly issued token lasts.
func (s *Signer) TTL() time.Duration { return s.ttl }

// Issue returns a signed token for a user.
func (s *Signer) Issue(userID, role string, tokenVersion int) (string, time.Time, error) {
	now := time.Now()
	expiry := now.Add(s.ttl)

	payload, err := json.Marshal(Claims{
		UserID:       userID,
		Role:         role,
		TokenVersion: tokenVersion,
		IssuedAt:     now.Unix(),
		ExpiresAt:    expiry.Unix(),
	})
	if err != nil {
		return "", time.Time{}, fmt.Errorf("marshal claims: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + s.sign(encoded), expiry, nil
}

// ErrInvalidToken covers every reason a token is not acceptable. The caller
// deliberately cannot tell a forged signature from an expired one, because the
// client has nothing useful to do with the difference.
var ErrInvalidToken = errors.New("invalid or expired token")

// Verify checks a token's signature and expiry and returns its claims.
func (s *Signer) Verify(token string) (Claims, error) {
	encoded, signature, found := strings.Cut(token, ".")
	if !found || encoded == "" || signature == "" {
		return Claims{}, ErrInvalidToken
	}

	// Compare the signature before decoding the payload, so a forged token
	// never reaches the JSON parser.
	if !hmac.Equal([]byte(signature), []byte(s.sign(encoded))) {
		return Claims{}, ErrInvalidToken
	}

	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return Claims{}, ErrInvalidToken
	}

	var claims Claims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return Claims{}, ErrInvalidToken
	}
	if !IsUUID(claims.UserID) || claims.Role == "" {
		return Claims{}, ErrInvalidToken
	}
	if time.Now().Unix() >= claims.ExpiresAt {
		return Claims{}, ErrInvalidToken
	}
	return claims, nil
}

func (s *Signer) sign(payload string) string {
	mac := hmac.New(sha256.New, s.key)
	mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

// BearerToken pulls the token out of an Authorization header.
func BearerToken(header string) string {
	const prefix = "Bearer "
	if len(header) <= len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return ""
	}
	return strings.TrimSpace(header[len(prefix):])
}

// IsUUID reports whether a string is a canonical 8-4-4-4-12 hexadecimal UUID.
//
// Written out rather than pulled from a dependency, and checked before any id
// from a URL or a token reaches a query, so a malformed id is a 400 rather than
// a database error.
func IsUUID(value string) bool {
	if len(value) != 36 {
		return false
	}
	for i := 0; i < 36; i++ {
		c := value[i]
		switch i {
		case 8, 13, 18, 23:
			if c != '-' {
				return false
			}
		default:
			isHex := (c >= '0' && c <= '9') ||
				(c >= 'a' && c <= 'f') ||
				(c >= 'A' && c <= 'F')
			if !isHex {
				return false
			}
		}
	}
	return true
}
