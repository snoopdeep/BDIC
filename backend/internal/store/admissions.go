package store

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ApplicationForm represents the public form template for admissions.
type ApplicationForm struct {
	ClassID           string         `json:"classId"`
	ClassName         string         `json:"className"`
	ClassNameHI       string         `json:"classNameHi"`
	Level             int            `json:"level"`
	HasStream         bool           `json:"hasStream"`
	RequiredDocuments []DocumentType `json:"requiredDocuments"`
}

// DocumentType represents a required document type for admission.
type DocumentType struct {
	ID        string `json:"id"`
	DocType   string `json:"docType"`
	LabelEN   string `json:"labelEn"`
	LabelHI   string `json:"labelHi"`
	Mandatory bool   `json:"mandatory"`
	SortOrder int    `json:"sortOrder"`
}

// AdmissionApplication represents an admission application record.
type AdmissionApplication struct {
	ID                 string     `json:"id"`
	ApplicationNo      string     `json:"applicationNo"`
	SessionID          string     `json:"sessionId"`
	ApplicantName      string     `json:"applicantName"`
	DateOfBirth        *string    `json:"dateOfBirth"`
	Gender             *string    `json:"gender"`
	Category           *string    `json:"category"`
	Religion           *string    `json:"religion"`
	ClassAppliedID     string     `json:"classAppliedId"`
	StreamID           *string    `json:"streamId"`
	FatherName         string     `json:"fatherName"`
	MotherName         *string    `json:"motherName"`
	GuardianName       *string    `json:"guardianName"`
	GuardianRelation   *string    `json:"guardianRelation"`
	GuardianPhone      string     `json:"guardianPhone"`
	GuardianAltPhone   *string    `json:"guardianAltPhone"`
	GuardianEmail      *string    `json:"guardianEmail"`
	AddressLine        *string    `json:"addressLine"`
	Village            *string    `json:"village"`
	District           *string    `json:"district"`
	State              string     `json:"state"`
	Pincode            *string    `json:"pincode"`
	PreviousSchool     *string    `json:"previousSchool"`
	LastClassPassed    *string    `json:"lastClassPassed"`
	LastClassPercent   *float64   `json:"lastClassPercent"`
	PhotoFileID        *string    `json:"photoFileId"`
	Status             string     `json:"status"`
	StatusReason       *string    `json:"statusReason"`
	TestDate           *string    `json:"testDate"`
	TestTime           *string    `json:"testTime"`
	TestVenue          *string    `json:"testVenue"`
	DecidedByUserID    *string    `json:"decidedByUserId"`
	DecidedAt          *time.Time `json:"decidedAt"`
	ConvertedStudentID *string    `json:"convertedStudentId"`
	SubmittedAt        time.Time  `json:"submittedAt"`
	UpdatedAt          time.Time  `json:"updatedAt"`
	LookupToken        string     `json:"-"` // Never exposed in JSON
}

// AdmissionDecision represents an admission decision record.
type AdmissionDecision struct {
	ID             string     `json:"id"`
	ApplicationNo  string     `json:"applicationNo"`
	ApplicantName  string     `json:"applicantName"`
	ClassAppliedID string     `json:"classAppliedId"`
	Status         string     `json:"status"`
	TestDate       *string    `json:"testDate"`
	TestTime       *string    `json:"testTime"`
	TestVenue      *string    `json:"testVenue"`
	DecidedAt      *time.Time `json:"decidedAt"`
	StatusReason   *string    `json:"statusReason"`
}

// GetApplicationForms returns all application forms for the current session
// with their required documents.
func (s *Store) GetApplicationForms(ctx context.Context, sessionID string) ([]ApplicationForm, error) {
	query := `
		SELECT
			c.id,
			c.name_en,
			c.name_hi,
			c.level,
			c.has_stream,
			ard.id,
			ard.doc_type,
			ard.label_en,
			ard.label_hi,
			ard.mandatory,
			ard.sort_order
		FROM classes c
		JOIN academic_sessions s ON c.school_id = s.school_id
		LEFT JOIN admission_required_documents ard ON ard.class_id = c.id
		WHERE s.id = $1::uuid
		ORDER BY c.level, c.code, COALESCE(ard.sort_order, 0)
	`

	rows, err := s.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("query application forms: %w", err)
	}
	defer rows.Close()

	formMap := make(map[string]*ApplicationForm)
	forms := make([]*ApplicationForm, 0)

	for rows.Next() {
		var classID, className, classNameHI string
		var level int
		var hasStream bool
		var docID, docType, labelEN, labelHI *string
		var mandatory *bool
		var sortOrder *int

		if err := rows.Scan(&classID, &className, &classNameHI, &level, &hasStream,
			&docID, &docType, &labelEN, &labelHI, &mandatory, &sortOrder); err != nil {
			return nil, fmt.Errorf("scan application form: %w", err)
		}

		// Get or create the form entry
		if _, exists := formMap[classID]; !exists {
			form := &ApplicationForm{
				ClassID:           classID,
				ClassName:         className,
				ClassNameHI:       classNameHI,
				Level:             level,
				HasStream:         hasStream,
				RequiredDocuments: []DocumentType{},
			}
			formMap[classID] = form
			forms = append(forms, form)
		}

		// Add document if one exists
		if docID != nil && docType != nil {
			doc := DocumentType{
				ID:        *docID,
				DocType:   *docType,
				LabelEN:   *labelEN,
				LabelHI:   *labelHI,
				Mandatory: *mandatory,
				SortOrder: *sortOrder,
			}
			formMap[classID].RequiredDocuments = append(formMap[classID].RequiredDocuments, doc)
		}
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate application forms: %w", err)
	}

	result := make([]ApplicationForm, 0, len(forms))
	for _, form := range forms {
		result = append(result, *form)
	}
	return result, nil
}

// GetUserApplications returns all admission applications for a specific user,
// scoped by their role. Students/Parents see only their own; Staff see all.
func (s *Store) GetUserApplications(ctx context.Context, userID, role string) ([]AdmissionApplication, error) {
	var query string

	// Staff can see all applications; students/parents see only their own
	if role == "STUDENT" || role == "PARENT" {
		// For now, return empty as we don't have linkage between users and applications
		// until they're converted to students. This would be enhanced with a parent/guardian lookup.
		query = `
			SELECT
				id, application_no, session_id, applicant_name, date_of_birth,
				gender, category, religion, class_applied_id, stream_id,
				father_name, mother_name, guardian_name, guardian_relation,
				guardian_phone, guardian_alt_phone, guardian_email,
				address_line, village, district, state, pincode,
				previous_school, last_class_passed, last_class_percent,
				photo_file_id, status, status_reason, test_date, test_time,
				test_venue, decided_by, decided_at, converted_student_id,
				submitted_at, updated_at, lookup_token
			FROM admission_applications
			WHERE 1=0
			ORDER BY submitted_at DESC
		`
	} else {
		// Staff see all
		query = `
			SELECT
				id, application_no, session_id, applicant_name, date_of_birth,
				gender, category, religion, class_applied_id, stream_id,
				father_name, mother_name, guardian_name, guardian_relation,
				guardian_phone, guardian_alt_phone, guardian_email,
				address_line, village, district, state, pincode,
				previous_school, last_class_passed, last_class_percent,
				photo_file_id, status, status_reason, test_date, test_time,
				test_venue, decided_by, decided_at, converted_student_id,
				submitted_at, updated_at, lookup_token
			FROM admission_applications
			ORDER BY submitted_at DESC
		`
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query user applications: %w", err)
	}
	defer rows.Close()

	return scanApplications(rows)
}

// GetApplication returns a single admission application by ID.
// The caller must have verified authorization.
func (s *Store) GetApplication(ctx context.Context, applicationID string) (AdmissionApplication, error) {
	query := `
		SELECT
			id, application_no, session_id, applicant_name, date_of_birth,
			gender, category, religion, class_applied_id, stream_id,
			father_name, mother_name, guardian_name, guardian_relation,
			guardian_phone, guardian_alt_phone, guardian_email,
			address_line, village, district, state, pincode,
			previous_school, last_class_passed, last_class_percent,
			photo_file_id, status, status_reason, test_date, test_time,
			test_venue, decided_by, decided_at, converted_student_id,
			submitted_at, updated_at, lookup_token
		FROM admission_applications
		WHERE id = $1::uuid
	`

	var app AdmissionApplication
	err := s.pool.QueryRow(ctx, query, applicationID).Scan(
		&app.ID, &app.ApplicationNo, &app.SessionID, &app.ApplicantName, &app.DateOfBirth,
		&app.Gender, &app.Category, &app.Religion, &app.ClassAppliedID, &app.StreamID,
		&app.FatherName, &app.MotherName, &app.GuardianName, &app.GuardianRelation,
		&app.GuardianPhone, &app.GuardianAltPhone, &app.GuardianEmail,
		&app.AddressLine, &app.Village, &app.District, &app.State, &app.Pincode,
		&app.PreviousSchool, &app.LastClassPassed, &app.LastClassPercent,
		&app.PhotoFileID, &app.Status, &app.StatusReason, &app.TestDate, &app.TestTime,
		&app.TestVenue, &app.DecidedByUserID, &app.DecidedAt, &app.ConvertedStudentID,
		&app.SubmittedAt, &app.UpdatedAt, &app.LookupToken,
	)
	return app, noRows(err)
}

// CreateApplication inserts a new admission application.
func (s *Store) CreateApplication(ctx context.Context, app AdmissionApplication) (string, error) {
	query := `
		INSERT INTO admission_applications (
			application_no, session_id, applicant_name, date_of_birth,
			gender, category, religion, class_applied_id, stream_id,
			father_name, mother_name, guardian_name, guardian_relation,
			guardian_phone, guardian_alt_phone, guardian_email,
			address_line, village, district, state, pincode,
			previous_school, last_class_passed, last_class_percent,
			photo_file_id, status, lookup_token
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13,
			$14, $15, $16, $17, $18, $19, $20, $21, $22, $23, $24, $25, $26
		)
		RETURNING id
	`

	var id string
	err := s.pool.QueryRow(ctx, query,
		app.ApplicationNo, app.SessionID, app.ApplicantName, app.DateOfBirth,
		app.Gender, app.Category, app.Religion, app.ClassAppliedID, app.StreamID,
		app.FatherName, app.MotherName, app.GuardianName, app.GuardianRelation,
		app.GuardianPhone, app.GuardianAltPhone, app.GuardianEmail,
		app.AddressLine, app.Village, app.District, app.State, app.Pincode,
		app.PreviousSchool, app.LastClassPassed, app.LastClassPercent,
		app.PhotoFileID, app.Status, app.LookupToken,
	).Scan(&id)

	if err != nil {
		if constraint, ok := isUniqueViolation(err); ok {
			if constraint == "admission_applications_no_unique" {
				return "", ErrConflict
			}
		}
		if _, ok := isForeignKeyViolation(err); ok {
			return "", fmt.Errorf("invalid class or stream reference: %w", err)
		}
		return "", fmt.Errorf("insert application: %w", err)
	}

	return id, nil
}

// GetAdmissionDecisions returns all decisions (applications with decided status).
// Students/Parents see only their own; Staff see all.
func (s *Store) GetAdmissionDecisions(ctx context.Context, userID, role string) ([]AdmissionDecision, error) {
	var query string

	if role == "STUDENT" || role == "PARENT" {
		// For now, return empty as we don't have linkage
		query = `
			SELECT
				id, application_no, applicant_name, class_applied_id,
				status, test_date, test_time, test_venue, decided_at, status_reason
			FROM admission_applications
			WHERE 1=0
			AND status IN ('SELECTED', 'WAITLISTED', 'REJECTED', 'ADMITTED')
			ORDER BY decided_at DESC
		`
	} else {
		// Staff see all decided applications
		query = `
			SELECT
				id, application_no, applicant_name, class_applied_id,
				status, test_date, test_time, test_venue, decided_at, status_reason
			FROM admission_applications
			WHERE status IN ('SELECTED', 'WAITLISTED', 'REJECTED', 'ADMITTED', 'FEE_PENDING')
			ORDER BY decided_at DESC
		`
	}

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query admission decisions: %w", err)
	}
	defer rows.Close()

	decisions := []AdmissionDecision{}
	for rows.Next() {
		var d AdmissionDecision
		if err := rows.Scan(
			&d.ID, &d.ApplicationNo, &d.ApplicantName, &d.ClassAppliedID,
			&d.Status, &d.TestDate, &d.TestTime, &d.TestVenue, &d.DecidedAt, &d.StatusReason,
		); err != nil {
			return nil, fmt.Errorf("scan decision: %w", err)
		}
		decisions = append(decisions, d)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate decisions: %w", err)
	}

	return decisions, nil
}

// scanApplications is a helper to scan application rows into a slice.
func scanApplications(rows pgx.Rows) ([]AdmissionApplication, error) {
	apps := []AdmissionApplication{}
	for rows.Next() {
		var app AdmissionApplication
		if err := rows.Scan(
			&app.ID, &app.ApplicationNo, &app.SessionID, &app.ApplicantName, &app.DateOfBirth,
			&app.Gender, &app.Category, &app.Religion, &app.ClassAppliedID, &app.StreamID,
			&app.FatherName, &app.MotherName, &app.GuardianName, &app.GuardianRelation,
			&app.GuardianPhone, &app.GuardianAltPhone, &app.GuardianEmail,
			&app.AddressLine, &app.Village, &app.District, &app.State, &app.Pincode,
			&app.PreviousSchool, &app.LastClassPassed, &app.LastClassPercent,
			&app.PhotoFileID, &app.Status, &app.StatusReason, &app.TestDate, &app.TestTime,
			&app.TestVenue, &app.DecidedByUserID, &app.DecidedAt, &app.ConvertedStudentID,
			&app.SubmittedAt, &app.UpdatedAt, &app.LookupToken,
		); err != nil {
			return nil, fmt.Errorf("scan application: %w", err)
		}
		apps = append(apps, app)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate applications: %w", err)
	}

	return apps, nil
}
