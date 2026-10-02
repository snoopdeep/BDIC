package auth

import "context"

// Roles. These strings are also the users.role CHECK constraint in the
// database, so the two cannot drift apart without a migration.
const (
	RoleSuperAdmin = "SUPER_ADMIN"
	RolePrincipal  = "PRINCIPAL"
	RoleOffice     = "OFFICE"
	RoleAccounts   = "ACCOUNTS"
	RoleTeacher    = "TEACHER"
	RoleStudent    = "STUDENT"
	RoleParent     = "PARENT"
)

// AllRoles is the full set, used for validation.
var AllRoles = []string{
	RoleSuperAdmin, RolePrincipal, RoleOffice,
	RoleAccounts, RoleTeacher, RoleStudent, RoleParent,
}

// IsValidRole reports whether a role string is one the system knows.
func IsValidRole(role string) bool {
	for _, candidate := range AllRoles {
		if candidate == role {
			return true
		}
	}
	return false
}

// Identity is the signed-in user as the handlers see them. It is loaded from
// the database on every authenticated request, not taken from the token, so a
// suspended account stops working immediately rather than when its token
// happens to expire.
type Identity struct {
	UserID       string `json:"userId"`
	Role         string `json:"role"`
	Status       string `json:"status"`
	Locale       string `json:"locale"`
	FullNameEN   string `json:"fullNameEn"`
	FullNameHI   string `json:"fullNameHi"`
	Email        string `json:"email,omitempty"`
	Phone        string `json:"phone,omitempty"`
	MustReset    bool   `json:"mustReset"`
	TokenVersion int    `json:"-"`

	// The row this user *is*, in whichever table applies to their role. These
	// are what every scoped query filters on: a teacher's own sections, a
	// student's own record, a guardian's own children.
	StaffID    string `json:"staffId,omitempty"`
	StudentID  string `json:"studentId,omitempty"`
	GuardianID string `json:"guardianId,omitempty"`
}

// HasRole reports whether the identity holds any of the given roles.
func (i Identity) HasRole(roles ...string) bool {
	for _, role := range roles {
		if i.Role == role {
			return true
		}
	}
	return false
}

// IsStaff covers everyone employed by the school, as opposed to a student or a
// parent. Used for "is this person on the inside" checks.
func (i Identity) IsStaff() bool {
	return i.HasRole(RoleSuperAdmin, RolePrincipal, RoleOffice, RoleAccounts, RoleTeacher)
}

// CanManageSchool covers the roles that may change the school's structure:
// sessions, classes, subjects, fee heads, staff.
func (i Identity) CanManageSchool() bool {
	return i.HasRole(RoleSuperAdmin, RolePrincipal)
}

// CanManageStudents covers admissions and student records.
func (i Identity) CanManageStudents() bool {
	return i.HasRole(RoleSuperAdmin, RolePrincipal, RoleOffice)
}

// CanManageFees covers fee structures, collection, and receipts.
func (i Identity) CanManageFees() bool {
	return i.HasRole(RoleSuperAdmin, RolePrincipal, RoleAccounts)
}

// CanApproveResults is deliberately narrow. Publishing a result to parents is
// the Principal's act, not the office's.
func (i Identity) CanApproveResults() bool {
	return i.HasRole(RoleSuperAdmin, RolePrincipal)
}

// CanPublishNotice covers who may send a notice without further approval.
func (i Identity) CanPublishNotice() bool {
	return i.HasRole(RoleSuperAdmin, RolePrincipal, RoleOffice)
}

// IdentityLoader reads an Identity from storage. Declared here so the auth
// middleware does not import the store package, which would be a cycle.
type IdentityLoader interface {
	LoadIdentity(ctx context.Context, userID string) (Identity, error)
}

// contextKey is unexported so nothing outside this package can put a forged
// Identity into a request context.
type contextKey struct{}

var identityKey contextKey

// WithIdentity returns a context carrying the signed-in user.
func WithIdentity(ctx context.Context, identity Identity) context.Context {
	return context.WithValue(ctx, identityKey, identity)
}

// FromContext returns the signed-in user, and false on an unauthenticated
// request. Handlers behind RequireAuth can rely on the second value being true.
func FromContext(ctx context.Context) (Identity, bool) {
	identity, ok := ctx.Value(identityKey).(Identity)
	return identity, ok
}

// MustFromContext returns the signed-in user. It is only safe behind
// RequireAuth, which is the only thing that puts an Identity in the context.
func MustFromContext(ctx context.Context) Identity {
	identity, ok := FromContext(ctx)
	if !ok {
		panic("auth: no identity in context; handler is not behind RequireAuth")
	}
	return identity
}
