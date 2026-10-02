package store

import (
	"context"
	"fmt"
)

// AcademicRecord represents a student's academic performance record.
type AcademicRecord struct {
	ID            string  `json:"id"`
	StudentID     string  `json:"studentId"`
	Subject       string  `json:"subject"`
	AcademicYear  string  `json:"academicYear"`
	Session       string  `json:"session"`
	MarksObtained float64 `json:"marksObtained"`
	MaxMarks      float64 `json:"maxMarks"`
	Grade         string  `json:"grade"`
	Percentage    float64 `json:"percentage"`
	RecordType    string  `json:"recordType"`
}

// CreateAcademicRecordInput is the input for creating an academic record.
type CreateAcademicRecordInput struct {
	StudentID     string
	Subject       string
	AcademicYear  string
	Session       string
	MarksObtained float64
	MaxMarks      float64
	Grade         string
	RecordType    string
	CreatedBy     string
}

// GetAcademicRecords returns all academic records for a student.
//
// The records are returned in reverse chronological order (most recent first).
// If no records are found, returns an empty slice with no error.
func (s *Store) GetAcademicRecords(ctx context.Context, studentID string) ([]AcademicRecord, error) {
	const query = `
		SELECT id::text,
		       student_id::text,
		       subject,
		       academic_year,
		       session,
		       marks_obtained,
		       max_marks,
		       grade,
		       CASE
		           WHEN max_marks > 0
		           THEN ROUND((marks_obtained / max_marks) * 100, 2)
		           ELSE 0
		       END as percentage,
		       record_type
		  FROM academic_records
		 WHERE student_id = $1::uuid
		 ORDER BY academic_year DESC, session DESC, created_at DESC`

	rows, err := s.pool.Query(ctx, query, studentID)
	if err != nil {
		return nil, fmt.Errorf("query academic records: %w", err)
	}
	defer rows.Close()

	records := []AcademicRecord{}
	for rows.Next() {
		var item AcademicRecord
		if err := rows.Scan(
			&item.ID, &item.StudentID, &item.Subject, &item.AcademicYear,
			&item.Session, &item.MarksObtained, &item.MaxMarks, &item.Grade,
			&item.Percentage, &item.RecordType,
		); err != nil {
			return nil, fmt.Errorf("scan academic record: %w", err)
		}
		records = append(records, item)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate academic records: %w", err)
	}

	return records, nil
}

// GetAcademicRecordsForGuardian returns academic records only for children
// linked to one guardian. It deliberately scopes in SQL so a parent cannot
// turn a client-side student id into access to another family's records.
func (s *Store) GetAcademicRecordsForGuardian(ctx context.Context, guardianID string) ([]AcademicRecord, error) {
	const query = `
		SELECT ar.id::text,
		       ar.student_id::text,
		       ar.subject,
		       ar.academic_year,
		       ar.session,
		       ar.marks_obtained,
		       ar.max_marks,
		       ar.grade,
		       CASE
		           WHEN ar.max_marks > 0
		           THEN ROUND((ar.marks_obtained / ar.max_marks) * 100, 2)
		           ELSE 0
		       END AS percentage,
		       ar.record_type
		  FROM academic_records ar
		  JOIN student_guardians sg ON sg.student_id = ar.student_id
		 WHERE sg.guardian_id = $1::uuid
		 ORDER BY ar.academic_year DESC, ar.session DESC, ar.created_at DESC`

	rows, err := s.pool.Query(ctx, query, guardianID)
	if err != nil {
		return nil, fmt.Errorf("query guardian academic records: %w", err)
	}
	defer rows.Close()

	records := []AcademicRecord{}
	for rows.Next() {
		var item AcademicRecord
		if err := rows.Scan(
			&item.ID, &item.StudentID, &item.Subject, &item.AcademicYear,
			&item.Session, &item.MarksObtained, &item.MaxMarks, &item.Grade,
			&item.Percentage, &item.RecordType,
		); err != nil {
			return nil, fmt.Errorf("scan guardian academic record: %w", err)
		}
		records = append(records, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate guardian academic records: %w", err)
	}
	return records, nil
}

// GetTranscript returns a complete transcript for a student, including all
// academic records and summary information.
//
// The transcript includes:
// - Student details (name, admission number)
// - All academic records organized by year/session
// - Overall performance statistics
func (s *Store) GetTranscript(ctx context.Context, studentID string) (map[string]interface{}, error) {
	// First get the student details
	studentQuery := `
		SELECT id::text, admission_no, full_name_en, full_name_hi
		  FROM students
		 WHERE id = $1::uuid
		 LIMIT 1`

	var studentID_out, admissionNo, nameEN, nameHI string
	err := s.pool.QueryRow(ctx, studentQuery, studentID).Scan(
		&studentID_out, &admissionNo, &nameEN, &nameHI,
	)
	if err != nil {
		return nil, fmt.Errorf("query student: %w", noRows(err))
	}

	// Get academic records
	records, err := s.GetAcademicRecords(ctx, studentID)
	if err != nil {
		return nil, err
	}

	// Build transcript response
	transcript := map[string]interface{}{
		"studentId":    studentID_out,
		"admissionNo":  admissionNo,
		"fullNameEn":   nameEN,
		"fullNameHi":   nameHI,
		"records":      records,
		"totalRecords": len(records),
	}

	// Calculate aggregate statistics if records exist
	if len(records) > 0 {
		totalMarks := 0.0
		totalMaxMarks := 0.0
		for _, r := range records {
			totalMarks += r.MarksObtained
			totalMaxMarks += r.MaxMarks
		}

		overallPercentage := 0.0
		if totalMaxMarks > 0 {
			overallPercentage = (totalMarks / totalMaxMarks) * 100
		}

		transcript["totalMarksObtained"] = totalMarks
		transcript["totalMaxMarks"] = totalMaxMarks
		transcript["overallPercentage"] = fmt.Sprintf("%.2f", overallPercentage)
	}

	return transcript, nil
}

// CreateAcademicRecord inserts a new academic record for a student.
//
// Returns the created record's ID. The student must exist; if not, an error
// is returned.
func (s *Store) CreateAcademicRecord(ctx context.Context, input CreateAcademicRecordInput) (string, error) {
	const query = `
		INSERT INTO academic_records (
			student_id,
			subject,
			academic_year,
			session,
			marks_obtained,
			max_marks,
			grade,
			record_type,
			created_by,
			created_at
		)
		VALUES (
			$1::uuid,
			$2,
			$3,
			$4,
			$5,
			$6,
			$7,
			$8,
			$9::uuid,
			now()
		)
		RETURNING id::text`

	var recordID string
	err := s.pool.QueryRow(
		ctx, query,
		input.StudentID,
		input.Subject,
		input.AcademicYear,
		input.Session,
		input.MarksObtained,
		input.MaxMarks,
		input.Grade,
		input.RecordType,
		input.CreatedBy,
	).Scan(&recordID)

	if err != nil {
		// Check if it's a foreign key violation (student doesn't exist)
		if _, ok := isForeignKeyViolation(err); ok {
			return "", fmt.Errorf("create academic record: student not found: %w", ErrNotFound)
		}
		return "", fmt.Errorf("create academic record: %w", err)
	}

	return recordID, nil
}
