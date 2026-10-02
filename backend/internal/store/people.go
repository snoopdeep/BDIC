package store

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// DirectoryStudent is the minimum safe student information that office staff
// need in a directory. Sensitive details such as medical notes and Aadhaar
// fragments never leave the student record endpoint.
type DirectoryStudent struct {
	ID          string `json:"id"`
	AdmissionNo string `json:"admissionNo"`
	FullNameEN  string `json:"fullNameEn"`
	FullNameHI  string `json:"fullNameHi"`
	ClassName   string `json:"className"`
	SectionName string `json:"sectionName"`
	Status      string `json:"status"`
}

func (s *Store) ListDirectoryStudents(ctx context.Context, sessionID string, limit, offset int) ([]DirectoryStudent, error) {
	const query = `
		SELECT st.id::text, st.admission_no, st.full_name_en, COALESCE(st.full_name_hi, ''),
		       COALESCE(c.name_en, ''), COALESCE(sec.name, ''), st.status
		  FROM students st
		  LEFT JOIN enrollments e ON e.student_id = st.id AND e.session_id = $1::uuid
		  LEFT JOIN classes c ON c.id = e.class_id
		  LEFT JOIN sections sec ON sec.id = e.section_id
		 ORDER BY st.full_name_en, st.admission_no
		 LIMIT $2 OFFSET $3`

	rows, err := s.pool.Query(ctx, query, sessionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list students: %w", err)
	}
	defer rows.Close()
	items := []DirectoryStudent{}
	for rows.Next() {
		var item DirectoryStudent
		if err := rows.Scan(&item.ID, &item.AdmissionNo, &item.FullNameEN, &item.FullNameHI, &item.ClassName, &item.SectionName, &item.Status); err != nil {
			return nil, fmt.Errorf("scan student: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate students: %w", err)
	}
	return items, nil
}

type DirectoryStaff struct {
	ID            string `json:"id"`
	EmployeeCode  string `json:"employeeCode"`
	FullNameEN    string `json:"fullNameEn"`
	FullNameHI    string `json:"fullNameHi"`
	DesignationEN string `json:"designationEn"`
	Department    string `json:"department"`
	Status        string `json:"status"`
}

func (s *Store) ListDirectoryStaff(ctx context.Context, limit, offset int) ([]DirectoryStaff, error) {
	const query = `
		SELECT id::text, employee_code, full_name_en, COALESCE(full_name_hi, ''),
		       COALESCE(designation_en, ''), COALESCE(department, ''), status
		  FROM staff
		 ORDER BY full_name_en, employee_code
		 LIMIT $1 OFFSET $2`
	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list staff: %w", err)
	}
	defer rows.Close()
	items := []DirectoryStaff{}
	for rows.Next() {
		var item DirectoryStaff
		if err := rows.Scan(&item.ID, &item.EmployeeCode, &item.FullNameEN, &item.FullNameHI, &item.DesignationEN, &item.Department, &item.Status); err != nil {
			return nil, fmt.Errorf("scan staff: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate staff: %w", err)
	}
	return items, nil
}

// NewStaffProfile creates the personnel record and, optionally, its matching
// portal account in one transaction. A staff row without a login is valid for
// non-digital employees; a login without the profile is never created.
type NewStaffProfile struct {
	EmployeeCode  string
	FullNameEN    string
	FullNameHI    string
	DesignationEN string
	DesignationHI string
	Department    string
	Qualification string
	Phone         string
	Email         string
	Login         *NewUser
}

func (s *Store) CreateStaffProfile(ctx context.Context, input NewStaffProfile) (staffID, userID string, err error) {
	err = s.InTx(ctx, func(tx pgx.Tx) error {
		if input.Login != nil {
			input.Login.EmployeeCode = input.EmployeeCode
			if input.Login.FullNameEN == "" {
				input.Login.FullNameEN = input.FullNameEN
			}
			if input.Login.FullNameHI == "" {
				input.Login.FullNameHI = input.FullNameHI
			}
			if input.Login.Email == "" {
				input.Login.Email = input.Email
			}
			if input.Login.Phone == "" {
				input.Login.Phone = input.Phone
			}
			var createErr error
			userID, createErr = createUserTx(ctx, tx, *input.Login)
			if createErr != nil {
				return createErr
			}
		}

		const query = `
			INSERT INTO staff (
				user_id, employee_code, full_name_en, full_name_hi,
				designation_en, designation_hi, department, qualification, phone, email
			) VALUES (
				NULLIF($1, '')::uuid, $2, $3, NULLIF($4, ''),
				NULLIF($5, ''), NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, '')
			)
			RETURNING id::text`
		if err := tx.QueryRow(ctx, query,
			userID, strings.TrimSpace(input.EmployeeCode), strings.TrimSpace(input.FullNameEN), strings.TrimSpace(input.FullNameHI),
			strings.TrimSpace(input.DesignationEN), strings.TrimSpace(input.DesignationHI), strings.TrimSpace(input.Department),
			strings.TrimSpace(input.Qualification), strings.TrimSpace(input.Phone), strings.TrimSpace(input.Email),
		).Scan(&staffID); err != nil {
			return fmt.Errorf("create staff profile: %w", err)
		}
		return nil
	})
	if err != nil {
		if _, unique := isUniqueViolation(err); unique {
			return "", "", fmt.Errorf("create staff profile: %w", ErrConflict)
		}
		return "", "", err
	}
	return staffID, userID, nil
}

// NewGuardian is the parent/guardian component of a student onboarding. A
// guardian is a separate record because siblings can legitimately share it.
type NewGuardian struct {
	FullNameEN string
	FullNameHI string
	Relation   string
	Phone      string
	AltPhone   string
	Email      string
	Occupation string
	Address    string
	Login      *NewUser
}

// NewStudentProfile is the minimum complete admission record: student,
// guardian relationship, and current-session enrollment. Optional account
// objects create student and parent logins in the same transaction.
type NewStudentProfile struct {
	AdmissionNo  string
	FullNameEN   string
	FullNameHI   string
	DateOfBirth  string
	Gender       string
	Category     string
	AddressLine  string
	Village      string
	District     string
	State        string
	Pincode      string
	FatherName   string
	MotherName   string
	SessionID    string
	ClassID      string
	SectionID    string
	StreamID     string
	Guardian     NewGuardian
	StudentLogin *NewUser
}

type StudentProfileResult struct {
	StudentID      string `json:"studentId"`
	GuardianID     string `json:"guardianId"`
	StudentUserID  string `json:"studentUserId,omitempty"`
	GuardianUserID string `json:"guardianUserId,omitempty"`
}

func (s *Store) CreateStudentProfile(ctx context.Context, input NewStudentProfile) (StudentProfileResult, error) {
	var result StudentProfileResult
	err := s.InTx(ctx, func(tx pgx.Tx) error {
		if input.StudentLogin != nil {
			input.StudentLogin.AdmissionNo = input.AdmissionNo
			input.StudentLogin.Role = "STUDENT"
			if input.StudentLogin.FullNameEN == "" {
				input.StudentLogin.FullNameEN = input.FullNameEN
			}
			if input.StudentLogin.FullNameHI == "" {
				input.StudentLogin.FullNameHI = input.FullNameHI
			}
			var err error
			result.StudentUserID, err = createUserTx(ctx, tx, *input.StudentLogin)
			if err != nil {
				return err
			}
		}
		if input.Guardian.Login != nil {
			input.Guardian.Login.Phone = firstNonBlank(input.Guardian.Login.Phone, input.Guardian.Phone)
			input.Guardian.Login.Role = "PARENT"
			if input.Guardian.Login.FullNameEN == "" {
				input.Guardian.Login.FullNameEN = input.Guardian.FullNameEN
			}
			if input.Guardian.Login.FullNameHI == "" {
				input.Guardian.Login.FullNameHI = input.Guardian.FullNameHI
			}
			if input.Guardian.Login.Email == "" {
				input.Guardian.Login.Email = input.Guardian.Email
			}
			var err error
			result.GuardianUserID, err = createUserTx(ctx, tx, *input.Guardian.Login)
			if err != nil {
				return err
			}
		}

		const studentQuery = `
			INSERT INTO students (
				admission_no, user_id, full_name_en, full_name_hi, date_of_birth, gender, category,
				address_line, village, district, state, pincode, father_name, mother_name
			) VALUES (
				$1, NULLIF($2, '')::uuid, $3, NULLIF($4, ''), NULLIF($5, '')::date, NULLIF($6, ''), NULLIF($7, ''),
				NULLIF($8, ''), NULLIF($9, ''), NULLIF($10, ''), NULLIF($11, ''), NULLIF($12, ''), NULLIF($13, ''), NULLIF($14, '')
			)
			RETURNING id::text`
		if err := tx.QueryRow(ctx, studentQuery,
			strings.TrimSpace(input.AdmissionNo), result.StudentUserID, strings.TrimSpace(input.FullNameEN), strings.TrimSpace(input.FullNameHI),
			strings.TrimSpace(input.DateOfBirth), strings.TrimSpace(input.Gender), strings.TrimSpace(input.Category),
			strings.TrimSpace(input.AddressLine), strings.TrimSpace(input.Village), strings.TrimSpace(input.District), strings.TrimSpace(input.State), strings.TrimSpace(input.Pincode),
			strings.TrimSpace(input.FatherName), strings.TrimSpace(input.MotherName),
		).Scan(&result.StudentID); err != nil {
			return fmt.Errorf("create student profile: %w", err)
		}

		const guardianQuery = `
			INSERT INTO guardians (user_id, full_name_en, full_name_hi, relation, phone, alt_phone, email, occupation, address)
			VALUES (NULLIF($1, '')::uuid, $2, NULLIF($3, ''), $4, $5, NULLIF($6, ''), NULLIF($7, ''), NULLIF($8, ''), NULLIF($9, ''))
			RETURNING id::text`
		if err := tx.QueryRow(ctx, guardianQuery,
			result.GuardianUserID, strings.TrimSpace(input.Guardian.FullNameEN), strings.TrimSpace(input.Guardian.FullNameHI),
			strings.TrimSpace(input.Guardian.Relation), strings.TrimSpace(input.Guardian.Phone), strings.TrimSpace(input.Guardian.AltPhone),
			strings.TrimSpace(input.Guardian.Email), strings.TrimSpace(input.Guardian.Occupation), strings.TrimSpace(input.Guardian.Address),
		).Scan(&result.GuardianID); err != nil {
			return fmt.Errorf("create guardian profile: %w", err)
		}
		if _, err := tx.Exec(ctx, `INSERT INTO student_guardians (student_id, guardian_id, is_primary) VALUES ($1::uuid, $2::uuid, true)`, result.StudentID, result.GuardianID); err != nil {
			return fmt.Errorf("link guardian: %w", err)
		}

		const enrollmentQuery = `
			INSERT INTO enrollments (student_id, session_id, class_id, section_id, stream_id)
			SELECT $1::uuid, $2::uuid, sec.class_id, sec.id, sec.stream_id
			  FROM sections sec
			 WHERE sec.id = $3::uuid AND sec.class_id = $4::uuid
			   AND ((NULLIF($5, '') IS NULL AND sec.stream_id IS NULL) OR sec.stream_id = NULLIF($5, '')::uuid)
			RETURNING id::text`
		var enrollmentID string
		if err := tx.QueryRow(ctx, enrollmentQuery, result.StudentID, input.SessionID, input.SectionID, input.ClassID, input.StreamID).Scan(&enrollmentID); err != nil {
			return fmt.Errorf("create enrollment: %w", noRows(err))
		}
		return nil
	})
	if err != nil {
		if _, unique := isUniqueViolation(err); unique {
			return StudentProfileResult{}, fmt.Errorf("create student profile: %w", ErrConflict)
		}
		return StudentProfileResult{}, err
	}
	return result, nil
}

func createUserTx(ctx context.Context, tx pgx.Tx, input NewUser) (string, error) {
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
	err := tx.QueryRow(ctx, query,
		nilIfBlank(input.Email), nilIfBlank(input.Phone), nilIfBlank(input.EmployeeCode), nilIfBlank(input.AdmissionNo),
		input.PasswordHash, strings.TrimSpace(input.FullNameEN), nilIfBlank(input.FullNameHI), input.Role, input.Locale, input.MustReset,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("create portal account: %w", err)
	}
	return id, nil
}

func firstNonBlank(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
