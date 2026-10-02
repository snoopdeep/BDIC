package auth

import (
	"errors"
	"net/http"

	"bdic/backend/internal/httpx"
)

// Middleware turns a bearer token into an Identity on the request context.
type Middleware struct {
	signer *Signer
	loader IdentityLoader
}

// NewMiddleware builds the authentication middleware.
func NewMiddleware(signer *Signer, loader IdentityLoader) *Middleware {
	return &Middleware{signer: signer, loader: loader}
}

// RequireAuth rejects a request without a valid token, and otherwise loads the
// user from the database and attaches them to the context.
//
// The database is read on every request rather than trusting the token alone.
// It costs one indexed lookup and it means a suspended account, a reset
// password, or a changed role takes effect on the very next request.
func (m *Middleware) RequireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := BearerToken(r.Header.Get("Authorization"))
		if token == "" {
			httpx.Fail(w, r, httpx.ErrUnauthorized())
			return
		}

		claims, err := m.signer.Verify(token)
		if err != nil {
			httpx.Fail(w, r, httpx.ErrUnauthorized().WithInternal(err))
			return
		}

		identity, err := m.loader.LoadIdentity(r.Context(), claims.UserID)
		if err != nil {
			// A token for a user who no longer exists is simply unauthorized.
			httpx.Fail(w, r, httpx.ErrUnauthorized().WithInternal(err))
			return
		}

		if identity.Status != "ACTIVE" {
			httpx.Fail(w, r, httpx.NewError(http.StatusForbidden, "ACCOUNT_INACTIVE",
				"This account has been suspended. Please contact the school office.",
				"यह खाता निलंबित कर दिया गया है। कृपया विद्यालय कार्यालय से संपर्क करें।"))
			return
		}

		// The revocation check. A password reset or a role change bumps
		// token_version, and every token issued before that stops working.
		if identity.TokenVersion != claims.TokenVersion {
			httpx.Fail(w, r, httpx.NewError(http.StatusUnauthorized, "TOKEN_REVOKED",
				"Your session has ended. Please sign in again.",
				"आपका सत्र समाप्त हो गया है। कृपया दोबारा साइन इन करें।"))
			return
		}

		// Note that identity.Role came from the database, not from the token.
		// The token's role claim is informational, so a role the office changes
		// applies on the very next request.
		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
	})
}

// RequireRole wraps a handler so only the listed roles reach it.
//
// This is the coarse gate. Row-level scoping — a teacher seeing only their own
// sections, a parent only their own children — is done in the queries, because
// a role check alone cannot express it.
func RequireRole(roles ...string) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			identity, ok := FromContext(r.Context())
			if !ok {
				httpx.Fail(w, r, httpx.ErrUnauthorized())
				return
			}
			if !identity.HasRole(roles...) {
				httpx.Fail(w, r, httpx.ErrForbidden())
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// RequireStaff allows anyone employed by the school and nobody else.
func RequireStaff(next http.Handler) http.Handler {
	return RequireRole(
		RoleSuperAdmin, RolePrincipal, RoleOffice, RoleAccounts, RoleTeacher,
	)(next)
}

// OptionalAuth attaches an Identity when a valid token is present and
// otherwise lets the request through unauthenticated.
//
// Used by the public endpoints that show a little more to a signed-in user —
// the notice board, for instance, which shows public notices to a visitor and
// class notices to a parent.
func (m *Middleware) OptionalAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := BearerToken(r.Header.Get("Authorization"))
		if token == "" {
			next.ServeHTTP(w, r)
			return
		}
		claims, err := m.signer.Verify(token)
		if err != nil {
			next.ServeHTTP(w, r)
			return
		}
		identity, err := m.loader.LoadIdentity(r.Context(), claims.UserID)
		if err != nil || identity.Status != "ACTIVE" || identity.TokenVersion != claims.TokenVersion {
			next.ServeHTTP(w, r)
			return
		}
		next.ServeHTTP(w, r.WithContext(WithIdentity(r.Context(), identity)))
	})
}

// ErrIdentityNotFound is what an IdentityLoader returns for a user id that is
// no longer in the users table.
var ErrIdentityNotFound = errors.New("identity not found")
