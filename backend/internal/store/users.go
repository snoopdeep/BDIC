package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"bdic/backend/internal/auth"
)

// The identity query is kept as three pieces — columns, source, and the
// password column — so the sign-in path can add the password hash to the select
// list without splicing it in after the FROM clause. Composing SQL by
// concatenating a complete SELECT is how that kind of mistake happens.
//
// The three LEFT JOINs resolve the row this login *is* in whichever table
// applies: staff for a teacher, students for a student, guardians for a parent.
// Each of those tables is unique on user_id, so the join cannot multiply rows.
const identityColumns = `
	       u.id::text,
	       u.role,
	       u.status,
	       u.locale,
	       u.full_name_en,
	       COALESCE(u.full_name_hi, ''),
	       COALESCE(u.email, ''),
	       COALESCE(u.phone, ''),
	       u.must_reset,
	       u.token_version,
	       COALESCE(st.id::text, ''),
	       COALESCE(s.id::text, ''),
	       COALESCE(g.id::text, '')`

const identityFrom = `
	  FROM users u
	  LEFT JOIN staff     st ON st.user_id = u.id
	  LEFT JOIN students  s  ON s.user_id  = u.id
	  LEFT JOIN guardians g  ON g.user_id  = u.id`

// identitySelect reads an Identity and nothing else.
const identitySelect = `SELECT` + identityColumns + identityFrom

// credentialsSelect reads an Identity plus the password hash to compare against.
const credentialsSelect = `SELECT` + identityColumns + `,
	       u.password_hash` + identityFrom

// LoadIdentity reads the signed-in user. It satisfies auth.IdentityLoader.
func (s *Store) LoadIdentity(ctx context.Context, userID string) (auth.Identity, error) {
	if !auth.IsUUID(userID) {
		return auth.Identity{}, auth.ErrIdentityNotFound
	}

	var identity auth.Identity
	err := s.pool.QueryRow(ctx, identitySelect+` WHERE u.id = $1::uuid`, userID).Scan(
		&identity.UserID,
		&identity.Role,
		&identity.Status,
		&identity.Locale,
		&identity.FullNameEN,
		&identity.FullNameHI,
		&identity.Email,
		&identity.Phone,
		&identity.MustReset,
		&identity.TokenVersion,
		&identity.StaffID,
		&identity.StudentID,
		&identity.GuardianID,
	)
	if err != nil {
		if noRows(err) == ErrNotFound {
			return auth.Identity{}, auth.ErrIdentityNotFound
		}
		return auth.Identity{}, fmt.Errorf("load identity: %w", err)
	}
	return identity, nil
}

// Credentials is what the sign-in handler needs: the identity plus the hash to
// compare against. The hash never leaves this package's caller.
type Credentials struct {
	Identity     auth.Identity
	PasswordHash string
}

// FindCredentials looks a user up by whatever they typed into the sign-in box.
//
// One box, four kinds of identifier, because a teacher thinks in terms of an
// email, a parent thinks in terms of their phone number, and a student thinks
// in terms of their admission number. Asking them to pick the right kind first
// is a needless step.
func (s *Store) FindCredentials(ctx context.Context, identifier string) (Credentials, error) {
	trimmed := strings.TrimSpace(identifier)
	if trimmed == "" {
		return Credentials{}, ErrNotFound
	}

	const where = `
		 WHERE u.status <> 'DISABLED'
		   AND (
		        lower(u.email)         = lower($1)
		     OR u.phone                = $1
		     OR upper(u.employee_code) = upper($1)
		     OR upper(u.admission_no)  = upper($1)
		   )`

	var creds Credentials
	err := s.pool.QueryRow(ctx, credentialsSelect+where, trimmed).Scan(
		&creds.Identity.UserID,
		&creds.Identity.Role,
		&creds.Identity.Status,
		&creds.Identity.Locale,
		&creds.Identity.FullNameEN,
		&creds.Identity.FullNameHI,
		&creds.Identity.Email,
		&creds.Identity.Phone,
		&creds.Identity.MustReset,
		&creds.Identity.TokenVersion,
		&creds.Identity.StaffID,
		&creds.Identity.StudentID,
		&creds.Identity.GuardianID,
		&creds.PasswordHash,
	)
	if err != nil {
		return Credentials{}, noRows(err)
	}
	return creds, nil
}

// FindCredentialsByUserID reads the credential row for a user already known by
// id. Used by the change-password path, which has an authenticated caller and
// must re-check their current password without guessing which of their four
// possible identifiers they sign in with.
func (s *Store) FindCredentialsByUserID(ctx context.Context, userID string) (Credentials, error) {
	if !auth.IsUUID(userID) {
		return Credentials{}, ErrNotFound
	}

	var creds Credentials
	err := s.pool.QueryRow(ctx, credentialsSelect+` WHERE u.id = $1::uuid`, userID).Scan(
		&creds.Identity.UserID,
		&creds.Identity.Role,
		&creds.Identity.Status,
		&creds.Identity.Locale,
		&creds.Identity.FullNameEN,
		&creds.Identity.FullNameHI,
		&creds.Identity.Email,
		&creds.Identity.Phone,
		&creds.Identity.MustReset,
		&creds.Identity.TokenVersion,
		&creds.Identity.StaffID,
		&creds.Identity.StudentID,
		&creds.Identity.GuardianID,
		&creds.PasswordHash,
	)
	if err != nil {
		return Credentials{}, noRows(err)
	}
	return creds, nil
}

// RecordSignIn stamps the successful sign-in and, when the stored hash was made
// with an older bcrypt cost, upgrades it. The plaintext is available for this
// one moment and never again, so this is where the upgrade has to happen.
func (s *Store) RecordSignIn(ctx context.Context, userID, upgradedHash string) error {
	if upgradedHash == "" {
		const query = `UPDATE users SET last_login_at = now() WHERE id = $1::uuid`
		_, err := s.pool.Exec(ctx, query, userID)
		return err
	}

	const query = `
		UPDATE users
		   SET last_login_at = now(),
		       password_hash = $2,
		       updated_at    = now()
		 WHERE id = $1::uuid`
	_, err := s.pool.Exec(ctx, query, userID, upgradedHash)
	return err
}

// NewUser is the input for creating a login.
type NewUser struct {
	Email        string
	Phone        string
	EmployeeCode string
	AdmissionNo  string
	PasswordHash string
	FullNameEN   string
	FullNameHI   string
	Role         string
	Locale       string
	MustReset    bool
}

// CreateUser inserts a login and returns its id.
func (s *Store) CreateUser(ctx context.Context, input NewUser) (string, error) {
	if input.Locale == "" {
		input.Locale = "hi"
	}

	const query = `
		INSERT INTO users (
			email, phone, employee_code, admission_no,
			password_hash, full_name_en, full_name_hi, role, locale, must_reset
		) VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
		RETURNING id::text`

	var id string
	err := s.pool.QueryRow(ctx, query,
		nilIfBlank(input.Email),
		nilIfBlank(input.Phone),
		nilIfBlank(input.EmployeeCode),
		nilIfBlank(input.AdmissionNo),
		input.PasswordHash,
		strings.TrimSpace(input.FullNameEN),
		nilIfBlank(input.FullNameHI),
		input.Role,
		input.Locale,
		input.MustReset,
	).Scan(&id)

	if err != nil {
		if constraint, ok := isUniqueViolation(err); ok {
			return "", fmt.Errorf("%w: %s", ErrConflict, friendlyUserConstraint(constraint))
		}
		return "", fmt.Errorf("create user: %w", err)
	}
	return id, nil
}

// SetPassword replaces a password and bumps token_version, which signs the user
// out of every device they were already signed in on. That is the behaviour a
// password change should have.
func (s *Store) SetPassword(ctx context.Context, userID, passwordHash string, mustReset bool) error {
	const query = `
		UPDATE users
		   SET password_hash = $2,
		       must_reset    = $3,
		       token_version = token_version + 1,
		       updated_at    = now()
		 WHERE id = $1::uuid`

	tag, err := s.pool.Exec(ctx, query, userID, passwordHash, mustReset)
	if err != nil {
		return fmt.Errorf("set password: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetUserStatus suspends or restores an account, bumping token_version so a
// suspension takes effect immediately rather than when the token expires.
func (s *Store) SetUserStatus(ctx context.Context, userID, status string) error {
	const query = `
		UPDATE users
		   SET status        = $2,
		       token_version = token_version + 1,
		       updated_at    = now()
		 WHERE id = $1::uuid`

	tag, err := s.pool.Exec(ctx, query, userID, status)
	if err != nil {
		return fmt.Errorf("set user status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetLocale remembers the language a user chose, so they are never shown the
// other one again.
func (s *Store) SetLocale(ctx context.Context, userID, locale string) error {
	const query = `UPDATE users SET locale = $2, updated_at = now() WHERE id = $1::uuid`
	tag, err := s.pool.Exec(ctx, query, userID, locale)
	if err != nil {
		return fmt.Errorf("set locale: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CountUsers reports how many logins exist, used to decide whether the database
// is empty enough to need the bootstrap administrator.
func (s *Store) CountUsers(ctx context.Context) (int, error) {
	var count int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM users`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

// PasswordResetCode stores the hash of a one-time code. The code itself is sent
// to the user and never persisted, so this table is useless to an attacker.
type PasswordResetCode struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	Attempts  int
}

// CreatePasswordResetCode records a reset code hash.
//
// The lifetime is passed as a whole number of minutes into make_interval
// rather than as a Go duration string, because Postgres does not parse Go's
// "15m0s" formatting.
func (s *Store) CreatePasswordResetCode(ctx context.Context, userID, codeHash string, ttl time.Duration) error {
	minutes := int(ttl.Minutes())
	if minutes < 1 {
		minutes = 1
	}

	const query = `
		INSERT INTO password_reset_codes (user_id, code_hash, expires_at)
		VALUES ($1::uuid, $2, now() + make_interval(mins => $3))`
	_, err := s.pool.Exec(ctx, query, userID, codeHash, minutes)
	if err != nil {
		return fmt.Errorf("create reset code: %w", err)
	}
	return nil
}

// ConsumePasswordResetCode checks a code and marks it used. A code works once.
func (s *Store) ConsumePasswordResetCode(ctx context.Context, userID, codeHash string) error {
	const query = `
		UPDATE password_reset_codes
		   SET consumed_at = now()
		 WHERE id = (
		       SELECT id
		         FROM password_reset_codes
		        WHERE user_id     = $1::uuid
		          AND code_hash   = $2
		          AND consumed_at IS NULL
		          AND expires_at  > now()
		        ORDER BY created_at DESC
		        LIMIT 1
		 )`
	tag, err := s.pool.Exec(ctx, query, userID, codeHash)
	if err != nil {
		return fmt.Errorf("consume reset code: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// friendlyUserConstraint turns a constraint name into something a person can
// act on, without exposing the schema.
func friendlyUserConstraint(constraint string) string {
	switch constraint {
	case "users_email_unique":
		return "that email address is already registered"
	case "users_phone_unique":
		return "that phone number is already registered"
	case "users_employee_code_unique":
		return "that employee code is already in use"
	case "users_admission_no_unique":
		return "that admission number already has a login"
	default:
		return "that record already exists"
	}
}

// nilIfBlank converts an empty string to a SQL NULL, which is what the partial
// unique indexes on users expect: many users with no email, but no two users
// with the same one.
func nilIfBlank(value string) *string {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
