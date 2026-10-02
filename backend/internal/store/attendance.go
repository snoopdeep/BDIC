package store

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// AttendanceRecord is the family-safe attendance history. Notes are included
// only for the learner whose record has been authorised by the query.
type AttendanceRecord struct {
	ID          string `json:"id"`
	StudentID   string `json:"studentId"`
	AdmissionNo string `json:"admissionNo"`
	StudentName string `json:"studentName"`
	SectionID   string `json:"sectionId"`
	SectionName string `json:"sectionName"`
	Date        string `json:"date"`
	Status      string `json:"status"`
	Note        string `json:"note"`
}

// AttendanceRosterItem is one entry in the daily teacher register. A missing
// status has not yet been marked; it is not silently interpreted as absent.
type AttendanceRosterItem struct {
	StudentID   string `json:"studentId"`
	AdmissionNo string `json:"admissionNo"`
	StudentName string `json:"studentName"`
	RollNo      int    `json:"rollNo,omitempty"`
	Status      string `json:"status,omitempty"`
	Note        string `json:"note,omitempty"`
}

type AttendanceEntryInput struct {
	StudentID string
	Status    string
	Note      string
}

// TeacherMayTakeAttendance enforces assignment in storage, rather than relying
// on a client-side timetable screen. A section teacher or a teacher scheduled
// for the section may take the register.
func (s *Store) TeacherMayTakeAttendance(ctx context.Context, sessionID, sectionID, staffID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM section_teachers
			 WHERE session_id = $1::uuid AND section_id = $2::uuid AND staff_id = $3::uuid
			UNION ALL
			SELECT 1 FROM timetable_entries
			 WHERE session_id = $1::uuid AND section_id = $2::uuid AND staff_id = $3::uuid
		)`
	var allowed bool
	if err := s.pool.QueryRow(ctx, query, sessionID, sectionID, staffID).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check attendance permission: %w", err)
	}
	return allowed, nil
}

// ListAttendanceRoster loads an enrolled section together with its entries for
// one school day. It never pulls old attendance from a different section.
func (s *Store) ListAttendanceRoster(ctx context.Context, sessionID, sectionID, date string) ([]AttendanceRosterItem, error) {
	const query = `
		SELECT st.id::text, st.admission_no, st.full_name_en, COALESCE(e.roll_no, 0),
		       COALESCE(a.status, ''), COALESCE(a.note, '')
		  FROM enrollments e
		  JOIN students st ON st.id = e.student_id
		  LEFT JOIN attendance a ON a.student_id = st.id AND a.date = $3::date
		 WHERE e.session_id = $1::uuid
		   AND e.section_id = $2::uuid
		   AND e.status = 'ENROLLED'
		 ORDER BY e.roll_no NULLS LAST, st.full_name_en`
	rows, err := s.pool.Query(ctx, query, sessionID, sectionID, date)
	if err != nil {
		return nil, fmt.Errorf("list attendance roster: %w", err)
	}
	defer rows.Close()
	items := []AttendanceRosterItem{}
	for rows.Next() {
		var item AttendanceRosterItem
		if err := rows.Scan(&item.StudentID, &item.AdmissionNo, &item.StudentName, &item.RollNo, &item.Status, &item.Note); err != nil {
			return nil, fmt.Errorf("scan attendance roster: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attendance roster: %w", err)
	}
	return items, nil
}

// SaveSameDayAttendance records a complete section register and its summary
// atomically. It is intentionally limited to the current school day; edits to
// a previous date must go through the correction workflow so the original
// status remains auditable.
func (s *Store) SaveSameDayAttendance(ctx context.Context, sessionID, sectionID, date, markedBy string, entries []AttendanceEntryInput) error {
	return s.InTx(ctx, func(tx pgx.Tx) error {
		present, absent, other := 0, 0, 0
		for _, entry := range entries {
			const enrolled = `
				SELECT EXISTS (
					SELECT 1 FROM enrollments
					 WHERE session_id = $1::uuid AND section_id = $2::uuid
					   AND student_id = $3::uuid AND status = 'ENROLLED'
				)`
			var isEnrolled bool
			if err := tx.QueryRow(ctx, enrolled, sessionID, sectionID, entry.StudentID).Scan(&isEnrolled); err != nil {
				return fmt.Errorf("check attendance enrollment: %w", err)
			}
			if !isEnrolled {
				return fmt.Errorf("save attendance: %w", ErrNotFound)
			}

			const upsert = `
				INSERT INTO attendance (session_id, student_id, section_id, date, status, marked_by, note)
				VALUES ($1::uuid, $2::uuid, $3::uuid, $4::date, $5, $6::uuid, NULLIF($7, ''))
				ON CONFLICT (student_id, date) DO UPDATE
				   SET status = EXCLUDED.status, note = EXCLUDED.note,
				       marked_by = EXCLUDED.marked_by, marked_at = now()
				 WHERE attendance.session_id = EXCLUDED.session_id
				   AND attendance.section_id = EXCLUDED.section_id`
			tag, err := tx.Exec(ctx, upsert, sessionID, entry.StudentID, sectionID, date, entry.Status, markedBy, entry.Note)
			if err != nil {
				return fmt.Errorf("save attendance entry: %w", err)
			}
			if tag.RowsAffected() == 0 {
				return fmt.Errorf("save attendance: student already has an entry in another section: %w", ErrConflict)
			}

			switch entry.Status {
			case "PRESENT":
				present++
			case "ABSENT":
				absent++
			default:
				other++
			}
		}

		const submission = `
			INSERT INTO attendance_submissions (session_id, section_id, date, marked_by, present, absent, other)
			VALUES ($1::uuid, $2::uuid, $3::date, $4::uuid, $5, $6, $7)
			ON CONFLICT (section_id, date) DO UPDATE
			   SET marked_by = EXCLUDED.marked_by, present = EXCLUDED.present,
			       absent = EXCLUDED.absent, other = EXCLUDED.other, submitted_at = now()`
		if _, err := tx.Exec(ctx, submission, sessionID, sectionID, date, markedBy, present, absent, other); err != nil {
			return fmt.Errorf("save attendance submission: %w", err)
		}
		return nil
	})
}

func (s *Store) ListAttendanceForStudent(ctx context.Context, sessionID, studentID string, limit, offset int) ([]AttendanceRecord, error) {
	const query = `
		SELECT a.id::text, a.student_id::text, st.admission_no, st.full_name_en,
		       a.section_id::text, sec.name, to_char(a.date, 'YYYY-MM-DD'), a.status, COALESCE(a.note, '')
		  FROM attendance a
		  JOIN students st ON st.id = a.student_id
		  JOIN sections sec ON sec.id = a.section_id
		 WHERE a.session_id = $1::uuid AND a.student_id = $2::uuid
		 ORDER BY a.date DESC
		 LIMIT $3 OFFSET $4`
	return s.listAttendance(ctx, query, sessionID, studentID, limit, offset)
}

func (s *Store) ListAttendanceForGuardian(ctx context.Context, sessionID, guardianID string, limit, offset int) ([]AttendanceRecord, error) {
	const query = `
		SELECT a.id::text, a.student_id::text, st.admission_no, st.full_name_en,
		       a.section_id::text, sec.name, to_char(a.date, 'YYYY-MM-DD'), a.status, COALESCE(a.note, '')
		  FROM attendance a
		  JOIN students st ON st.id = a.student_id
		  JOIN student_guardians sg ON sg.student_id = st.id
		  JOIN sections sec ON sec.id = a.section_id
		 WHERE a.session_id = $1::uuid AND sg.guardian_id = $2::uuid
		 ORDER BY a.date DESC, st.full_name_en
		 LIMIT $3 OFFSET $4`
	return s.listAttendance(ctx, query, sessionID, guardianID, limit, offset)
}

func (s *Store) ListAttendanceAll(ctx context.Context, sessionID, sectionID string, limit, offset int) ([]AttendanceRecord, error) {
	const query = `
		SELECT a.id::text, a.student_id::text, st.admission_no, st.full_name_en,
		       a.section_id::text, sec.name, to_char(a.date, 'YYYY-MM-DD'), a.status, COALESCE(a.note, '')
		  FROM attendance a
		  JOIN students st ON st.id = a.student_id
		  JOIN sections sec ON sec.id = a.section_id
		 WHERE a.session_id = $1::uuid
		   AND (NULLIF($2, '') IS NULL OR a.section_id = NULLIF($2, '')::uuid)
		 ORDER BY a.date DESC, st.full_name_en
		 LIMIT $3 OFFSET $4`
	return s.listAttendance(ctx, query, sessionID, sectionID, limit, offset)
}

// ListAttendanceForTeacher lists attendance only for sections a teacher is
// assigned to. The EXISTS checks avoid duplicate rows when a teacher is both a
// section teacher and appears in the timetable for the same section.
func (s *Store) ListAttendanceForTeacher(ctx context.Context, sessionID, staffID, sectionID string, limit, offset int) ([]AttendanceRecord, error) {
	const query = `
		SELECT a.id::text, a.student_id::text, st.admission_no, st.full_name_en,
		       a.section_id::text, sec.name, to_char(a.date, 'YYYY-MM-DD'), a.status, COALESCE(a.note, '')
		  FROM attendance a
		  JOIN students st ON st.id = a.student_id
		  JOIN sections sec ON sec.id = a.section_id
		 WHERE a.session_id = $1::uuid
		   AND (NULLIF($3, '')::uuid IS NULL OR a.section_id = NULLIF($3, '')::uuid)
		   AND (
		       EXISTS (
		           SELECT 1 FROM section_teachers section_teacher
		            WHERE section_teacher.session_id = a.session_id
		              AND section_teacher.section_id = a.section_id
		              AND section_teacher.staff_id = $2::uuid
		       )
		       OR EXISTS (
		           SELECT 1 FROM timetable_entries timetable
		            WHERE timetable.session_id = a.session_id
		              AND timetable.section_id = a.section_id
		              AND timetable.staff_id = $2::uuid
		       )
		   )
		 ORDER BY a.date DESC, sec.name, st.full_name_en
		 LIMIT $4 OFFSET $5`

	rows, err := s.pool.Query(ctx, query, sessionID, staffID, sectionID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list teacher attendance: %w", err)
	}
	defer rows.Close()

	items := []AttendanceRecord{}
	for rows.Next() {
		var item AttendanceRecord
		if err := rows.Scan(
			&item.ID, &item.StudentID, &item.AdmissionNo, &item.StudentName,
			&item.SectionID, &item.SectionName, &item.Date, &item.Status, &item.Note,
		); err != nil {
			return nil, fmt.Errorf("scan teacher attendance: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teacher attendance: %w", err)
	}
	return items, nil
}

func (s *Store) listAttendance(ctx context.Context, query, firstID, secondID string, limit, offset int) ([]AttendanceRecord, error) {
	rows, err := s.pool.Query(ctx, query, firstID, secondID, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list attendance: %w", err)
	}
	defer rows.Close()
	items := []AttendanceRecord{}
	for rows.Next() {
		var item AttendanceRecord
		if err := rows.Scan(&item.ID, &item.StudentID, &item.AdmissionNo, &item.StudentName,
			&item.SectionID, &item.SectionName, &item.Date, &item.Status, &item.Note); err != nil {
			return nil, fmt.Errorf("scan attendance: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate attendance: %w", err)
	}
	return items, nil
}
