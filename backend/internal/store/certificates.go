package store

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

type Certificate struct {
	ID          string `json:"id"`
	StudentID   string `json:"studentId"`
	AdmissionNo string `json:"admissionNo"`
	StudentName string `json:"studentName"`
	Kind        string `json:"kind"`
	SerialNo    string `json:"serialNo"`
	IssuedOn    string `json:"issuedOn"`
	Cancelled   bool   `json:"cancelled"`
}

type IssueCertificateInput struct {
	StudentID   string
	Kind        string
	SessionName string
	IssuedBy    string
	Payload     map[string]any
}

func (s *Store) ListCertificatesAll(ctx context.Context, studentID string, limit, offset int) ([]Certificate, error) {
	const query = `
		SELECT c.id::text, c.student_id::text, st.admission_no, st.full_name_en,
		       c.kind, c.serial_no, to_char(c.issued_on, 'YYYY-MM-DD'), c.cancelled_at IS NOT NULL
		  FROM certificates c JOIN students st ON st.id = c.student_id
		 WHERE NULLIF($1, '') IS NULL OR c.student_id = NULLIF($1, '')::uuid
		 ORDER BY c.issued_on DESC, c.created_at DESC
		 LIMIT $2 OFFSET $3`
	return s.listCertificates(ctx, query, studentID, limit, offset)
}

func (s *Store) ListCertificatesForStudent(ctx context.Context, studentID string, limit, offset int) ([]Certificate, error) {
	const query = `
		SELECT c.id::text, c.student_id::text, st.admission_no, st.full_name_en,
		       c.kind, c.serial_no, to_char(c.issued_on, 'YYYY-MM-DD'), c.cancelled_at IS NOT NULL
		  FROM certificates c JOIN students st ON st.id = c.student_id
		 WHERE c.student_id = $1::uuid
		 ORDER BY c.issued_on DESC, c.created_at DESC
		 LIMIT $2 OFFSET $3`
	return s.listCertificates(ctx, query, studentID, limit, offset)
}

func (s *Store) ListCertificatesForGuardian(ctx context.Context, guardianID string, limit, offset int) ([]Certificate, error) {
	const query = `
		SELECT c.id::text, c.student_id::text, st.admission_no, st.full_name_en,
		       c.kind, c.serial_no, to_char(c.issued_on, 'YYYY-MM-DD'), c.cancelled_at IS NOT NULL
		  FROM certificates c
		  JOIN students st ON st.id = c.student_id
		  JOIN student_guardians sg ON sg.student_id = st.id
		 WHERE sg.guardian_id = $1::uuid
		 ORDER BY c.issued_on DESC, c.created_at DESC
		 LIMIT $2 OFFSET $3`
	return s.listCertificates(ctx, query, guardianID, limit, offset)
}

func (s *Store) listCertificates(ctx context.Context, query string, args ...any) ([]Certificate, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list certificates: %w", err)
	}
	defer rows.Close()
	items := []Certificate{}
	for rows.Next() {
		var item Certificate
		if err := rows.Scan(&item.ID, &item.StudentID, &item.AdmissionNo, &item.StudentName, &item.Kind, &item.SerialNo, &item.IssuedOn, &item.Cancelled); err != nil {
			return nil, fmt.Errorf("scan certificate: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate certificates: %w", err)
	}
	return items, nil
}

// IssueCertificate consumes a school-series number and stores the immutable
// payload at the same time. Printing is a presentation concern; the issued
// record is what makes a later reprint exactly match the original certificate.
func (s *Store) IssueCertificate(ctx context.Context, input IssueCertificateInput) (Certificate, error) {
	scope, ok := certificateSeries(input.Kind)
	if !ok {
		return Certificate{}, fmt.Errorf("issue certificate: %w", ErrConflict)
	}
	payload, err := json.Marshal(input.Payload)
	if err != nil {
		return Certificate{}, fmt.Errorf("encode certificate payload: %w", err)
	}
	var result Certificate
	err = s.InTx(ctx, func(tx pgx.Tx) error {
		serialNo, err := AllocateNumber(ctx, tx, scope, input.SessionName)
		if err != nil {
			return err
		}
		// Resolve student display fields after the insert inside the same
		// transaction, so the returned receipt remains self-contained.
		var id string
		const insert = `
			INSERT INTO certificates (student_id, kind, serial_no, issued_by, payload)
			VALUES ($1::uuid, $2, $3, $4::uuid, $5::jsonb)
			RETURNING id::text, to_char(issued_on, 'YYYY-MM-DD')`
		if err := tx.QueryRow(ctx, insert, input.StudentID, input.Kind, serialNo, input.IssuedBy, payload).Scan(&id, &result.IssuedOn); err != nil {
			return fmt.Errorf("insert certificate: %w", err)
		}
		const student = `SELECT admission_no, full_name_en FROM students WHERE id = $1::uuid`
		if err := tx.QueryRow(ctx, student, input.StudentID).Scan(&result.AdmissionNo, &result.StudentName); err != nil {
			return fmt.Errorf("read certificate student: %w", noRows(err))
		}
		result.ID, result.StudentID, result.Kind, result.SerialNo = id, input.StudentID, input.Kind, serialNo
		return nil
	})
	if err != nil {
		if _, unique := isUniqueViolation(err); unique {
			return Certificate{}, fmt.Errorf("issue certificate: %w", ErrConflict)
		}
		return Certificate{}, err
	}
	return result, nil
}

func certificateSeries(kind string) (string, bool) {
	switch kind {
	case "BONAFIDE":
		return SeriesCertBonafide, true
	case "CHARACTER":
		return SeriesCertCharacter, true
	case "TRANSFER":
		return SeriesCertTransfer, true
	case "ATTENDANCE":
		return SeriesCertAttendance, true
	default:
		return "", false
	}
}
