package auth

import (
	"strings"
	"testing"
	"time"
)

const testUserID = "3f2a9c1e-5b7d-4e8a-9c21-7d4e5f6a8b90"

func TestIssueAndVerifyRoundTrip(t *testing.T) {
	signer := NewSigner([]byte("a-signing-key-long-enough-for-tests"), time.Hour)

	token, expiry, err := signer.Issue(testUserID, RoleTeacher, 3)
	if err != nil {
		t.Fatalf("Issue returned an error: %v", err)
	}
	if !expiry.After(time.Now()) {
		t.Fatalf("expiry %v is not in the future", expiry)
	}

	claims, err := signer.Verify(token)
	if err != nil {
		t.Fatalf("Verify rejected a token we just issued: %v", err)
	}
	if claims.UserID != testUserID {
		t.Errorf("UserID = %q, want %q", claims.UserID, testUserID)
	}
	if claims.Role != RoleTeacher {
		t.Errorf("Role = %q, want %q", claims.Role, RoleTeacher)
	}
	if claims.TokenVersion != 3 {
		t.Errorf("TokenVersion = %d, want 3", claims.TokenVersion)
	}
}

// A token signed with one key must not verify with another. This is the whole
// point of the signature, so it is worth asserting explicitly.
func TestVerifyRejectsAnotherKeysToken(t *testing.T) {
	issuer := NewSigner([]byte("the-original-signing-key-for-tests"), time.Hour)
	attacker := NewSigner([]byte("a-completely-different-key-here!!"), time.Hour)

	token, _, err := issuer.Issue(testUserID, RoleSuperAdmin, 1)
	if err != nil {
		t.Fatalf("Issue returned an error: %v", err)
	}

	if _, err := attacker.Verify(token); err == nil {
		t.Fatal("Verify accepted a token signed with a different key")
	}
}

func TestVerifyRejectsTamperedPayload(t *testing.T) {
	signer := NewSigner([]byte("a-signing-key-long-enough-for-tests"), time.Hour)

	token, _, err := signer.Issue(testUserID, RoleStudent, 1)
	if err != nil {
		t.Fatalf("Issue returned an error: %v", err)
	}

	payload, signature, _ := strings.Cut(token, ".")

	// Flip one character of the payload, keeping the original signature. This
	// is the shape of a privilege-escalation attempt: edit the role claim and
	// hope nobody checks.
	edited := []byte(payload)
	if edited[0] == 'A' {
		edited[0] = 'B'
	} else {
		edited[0] = 'A'
	}

	if _, err := signer.Verify(string(edited) + "." + signature); err == nil {
		t.Fatal("Verify accepted a token whose payload had been edited")
	}
}

func TestVerifyRejectsExpiredToken(t *testing.T) {
	// A negative lifetime produces a token that expired before it existed.
	signer := NewSigner([]byte("a-signing-key-long-enough-for-tests"), -time.Minute)

	token, _, err := signer.Issue(testUserID, RoleParent, 1)
	if err != nil {
		t.Fatalf("Issue returned an error: %v", err)
	}

	if _, err := signer.Verify(token); err == nil {
		t.Fatal("Verify accepted an expired token")
	}
}

func TestVerifyRejectsMalformedTokens(t *testing.T) {
	signer := NewSigner([]byte("a-signing-key-long-enough-for-tests"), time.Hour)

	cases := map[string]string{
		"empty":           "",
		"no separator":    "abcdef",
		"empty payload":   ".signature",
		"empty signature": "payload.",
		"three parts":     "a.b.c",
		"not base64":      "!!!.###",
	}

	for name, token := range cases {
		t.Run(name, func(t *testing.T) {
			if _, err := signer.Verify(token); err == nil {
				t.Errorf("Verify accepted %q", token)
			}
		})
	}
}

func TestBearerToken(t *testing.T) {
	cases := []struct {
		header string
		want   string
	}{
		{"Bearer abc123", "abc123"},
		{"bearer abc123", "abc123"},   // case-insensitive scheme, per RFC 6750
		{"Bearer  abc123 ", "abc123"}, // surrounding space trimmed
		{"Basic abc123", ""},
		{"abc123", ""},
		{"Bearer", ""},
		{"", ""},
	}

	for _, testCase := range cases {
		if got := BearerToken(testCase.header); got != testCase.want {
			t.Errorf("BearerToken(%q) = %q, want %q", testCase.header, got, testCase.want)
		}
	}
}

func TestIsUUID(t *testing.T) {
	valid := []string{
		"3f2a9c1e-5b7d-4e8a-9c21-7d4e5f6a8b90",
		"00000000-0000-0000-0000-000000000000",
		"FFFFFFFF-FFFF-FFFF-FFFF-FFFFFFFFFFFF",
	}
	for _, value := range valid {
		if !IsUUID(value) {
			t.Errorf("IsUUID(%q) = false, want true", value)
		}
	}

	invalid := []string{
		"",
		"not-a-uuid",
		"3f2a9c1e5b7d4e8a9c217d4e5f6a8b90",               // no hyphens
		"3f2a9c1e-5b7d-4e8a-9c21-7d4e5f6a8b9",            // too short
		"3f2a9c1e-5b7d-4e8a-9c21-7d4e5f6a8b901",          // too long
		"3f2a9c1g-5b7d-4e8a-9c21-7d4e5f6a8b90",           // 'g' is not hex
		"3f2a9c1e_5b7d_4e8a_9c21_7d4e5f6a8b90",           // wrong separator
		"'; DROP TABLE students; --                    ", // the reason this check exists
	}
	for _, value := range invalid {
		if IsUUID(value) {
			t.Errorf("IsUUID(%q) = true, want false", value)
		}
	}
}

func TestIsValidRole(t *testing.T) {
	for _, role := range AllRoles {
		if !IsValidRole(role) {
			t.Errorf("IsValidRole(%q) = false, want true", role)
		}
	}
	for _, role := range []string{"", "admin", "super_admin", "ROOT", "Teacher"} {
		if IsValidRole(role) {
			t.Errorf("IsValidRole(%q) = true, want false", role)
		}
	}
}

func TestIdentityPermissions(t *testing.T) {
	cases := []struct {
		role           string
		isStaff        bool
		manageSchool   bool
		manageStudents bool
		manageFees     bool
		approveResults bool
	}{
		{RoleSuperAdmin, true, true, true, true, true},
		{RolePrincipal, true, true, true, true, true},
		{RoleOffice, true, false, true, false, false},
		{RoleAccounts, true, false, false, true, false},
		{RoleTeacher, true, false, false, false, false},
		{RoleStudent, false, false, false, false, false},
		{RoleParent, false, false, false, false, false},
	}

	for _, testCase := range cases {
		t.Run(testCase.role, func(t *testing.T) {
			identity := Identity{Role: testCase.role}

			if got := identity.IsStaff(); got != testCase.isStaff {
				t.Errorf("IsStaff() = %v, want %v", got, testCase.isStaff)
			}
			if got := identity.CanManageSchool(); got != testCase.manageSchool {
				t.Errorf("CanManageSchool() = %v, want %v", got, testCase.manageSchool)
			}
			if got := identity.CanManageStudents(); got != testCase.manageStudents {
				t.Errorf("CanManageStudents() = %v, want %v", got, testCase.manageStudents)
			}
			if got := identity.CanManageFees(); got != testCase.manageFees {
				t.Errorf("CanManageFees() = %v, want %v", got, testCase.manageFees)
			}
			if got := identity.CanApproveResults(); got != testCase.approveResults {
				t.Errorf("CanApproveResults() = %v, want %v", got, testCase.approveResults)
			}
		})
	}
}
