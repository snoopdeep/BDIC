package store

import (
	"context"
	"fmt"
)

// TimetableEntry is the weekly timetable as a family, teacher, or office user
// needs to see it. Substitute fields are populated only for the requested date.
type TimetableEntry struct {
	ID                    string `json:"id"`
	SectionID             string `json:"sectionId"`
	ClassName             string `json:"className"`
	SectionName           string `json:"sectionName"`
	DayOfWeek             int    `json:"dayOfWeek"`
	PeriodID              string `json:"periodId"`
	PeriodName            string `json:"periodName"`
	StartTime             string `json:"startTime"`
	EndTime               string `json:"endTime"`
	SubjectID             string `json:"subjectId"`
	SubjectName           string `json:"subjectName"`
	StaffID               string `json:"staffId"`
	TeacherName           string `json:"teacherName"`
	SubstituteStaffID     string `json:"substituteStaffId,omitempty"`
	SubstituteTeacherName string `json:"substituteTeacherName,omitempty"`
	Room                  string `json:"room"`
}

type NewTimetableEntry struct {
	SessionID string
	SectionID string
	DayOfWeek int
	PeriodID  string
	SubjectID string
	StaffID   string
	Room      string
}

// ListTimetableForSection reads one weekly grid. Supplying date turns on the
// substitution overlay; it does not modify the underlying regular timetable.
func (s *Store) ListTimetableForSection(ctx context.Context, sessionID, sectionID, date string) ([]TimetableEntry, error) {
	const query = `
		SELECT te.id::text, te.section_id::text, c.name_en, sec.name,
		       te.day_of_week, p.id::text, p.name_en, p.start_time::text, p.end_time::text,
		       sub.id::text, sub.name_en, st.id::text, st.full_name_en,
		       COALESCE(replacement.id::text, ''), COALESCE(replacement.full_name_en, ''),
		       COALESCE(te.room, '')
		  FROM timetable_entries te
		  JOIN sections sec ON sec.id = te.section_id
		  JOIN classes c ON c.id = sec.class_id
		  JOIN periods p ON p.id = te.period_id
		  JOIN subjects sub ON sub.id = te.subject_id
		  JOIN staff st ON st.id = te.staff_id
		  LEFT JOIN substitutions replacement_assignment
		    ON replacement_assignment.timetable_entry_id = te.id
		   AND replacement_assignment.date = NULLIF($3, '')::date
		  LEFT JOIN staff replacement ON replacement.id = replacement_assignment.substitute_staff_id
		 WHERE te.session_id = $1::uuid AND te.section_id = $2::uuid
		 ORDER BY te.day_of_week, p.position`
	return s.listTimetable(ctx, query, sessionID, sectionID, date)
}

func (s *Store) ListTimetableForStaff(ctx context.Context, sessionID, staffID, date string) ([]TimetableEntry, error) {
	const query = `
		SELECT te.id::text, te.section_id::text, c.name_en, sec.name,
		       te.day_of_week, p.id::text, p.name_en, p.start_time::text, p.end_time::text,
		       sub.id::text, sub.name_en, st.id::text, st.full_name_en,
		       COALESCE(replacement.id::text, ''), COALESCE(replacement.full_name_en, ''),
		       COALESCE(te.room, '')
		  FROM timetable_entries te
		  JOIN sections sec ON sec.id = te.section_id
		  JOIN classes c ON c.id = sec.class_id
		  JOIN periods p ON p.id = te.period_id
		  JOIN subjects sub ON sub.id = te.subject_id
		  JOIN staff st ON st.id = te.staff_id
		  LEFT JOIN substitutions replacement_assignment
		    ON replacement_assignment.timetable_entry_id = te.id
		   AND replacement_assignment.date = NULLIF($3, '')::date
		  LEFT JOIN staff replacement ON replacement.id = replacement_assignment.substitute_staff_id
		 WHERE te.session_id = $1::uuid AND te.staff_id = $2::uuid
		 ORDER BY te.day_of_week, p.position`
	return s.listTimetable(ctx, query, sessionID, staffID, date)
}

func (s *Store) ListTimetableAll(ctx context.Context, sessionID, sectionID, date string, limit int) ([]TimetableEntry, error) {
	const query = `
		SELECT te.id::text, te.section_id::text, c.name_en, sec.name,
		       te.day_of_week, p.id::text, p.name_en, p.start_time::text, p.end_time::text,
		       sub.id::text, sub.name_en, st.id::text, st.full_name_en,
		       COALESCE(replacement.id::text, ''), COALESCE(replacement.full_name_en, ''),
		       COALESCE(te.room, '')
		  FROM timetable_entries te
		  JOIN sections sec ON sec.id = te.section_id
		  JOIN classes c ON c.id = sec.class_id
		  JOIN periods p ON p.id = te.period_id
		  JOIN subjects sub ON sub.id = te.subject_id
		  JOIN staff st ON st.id = te.staff_id
		  LEFT JOIN substitutions replacement_assignment
		    ON replacement_assignment.timetable_entry_id = te.id
		   AND replacement_assignment.date = NULLIF($3, '')::date
		  LEFT JOIN staff replacement ON replacement.id = replacement_assignment.substitute_staff_id
		 WHERE te.session_id = $1::uuid
		   AND (NULLIF($2, '') IS NULL OR te.section_id = NULLIF($2, '')::uuid)
		 ORDER BY c.sort_order, sec.sort_order, te.day_of_week, p.position
		 LIMIT $4`
	return s.listTimetable(ctx, query, sessionID, sectionID, date, limit)
}

func (s *Store) listTimetable(ctx context.Context, query string, args ...any) ([]TimetableEntry, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list timetable: %w", err)
	}
	defer rows.Close()
	items := []TimetableEntry{}
	for rows.Next() {
		var item TimetableEntry
		if err := rows.Scan(
			&item.ID, &item.SectionID, &item.ClassName, &item.SectionName,
			&item.DayOfWeek, &item.PeriodID, &item.PeriodName, &item.StartTime, &item.EndTime,
			&item.SubjectID, &item.SubjectName, &item.StaffID, &item.TeacherName,
			&item.SubstituteStaffID, &item.SubstituteTeacherName, &item.Room,
		); err != nil {
			return nil, fmt.Errorf("scan timetable: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate timetable: %w", err)
	}
	return items, nil
}

// ListTimetableForStudent scopes a learner to their current enrolled section.
func (s *Store) ListTimetableForStudent(ctx context.Context, sessionID, studentID, date string) ([]TimetableEntry, error) {
	const sectionQuery = `
		SELECT section_id::text FROM enrollments
		 WHERE session_id = $1::uuid AND student_id = $2::uuid AND status = 'ENROLLED'`
	var sectionID string
	if err := s.pool.QueryRow(ctx, sectionQuery, sessionID, studentID).Scan(&sectionID); err != nil {
		return nil, fmt.Errorf("find student timetable section: %w", noRows(err))
	}
	return s.ListTimetableForSection(ctx, sessionID, sectionID, date)
}

func (s *Store) ListTimetableForGuardian(ctx context.Context, sessionID, guardianID, date string) ([]TimetableEntry, error) {
	const query = `
		SELECT te.id::text, te.section_id::text, c.name_en, sec.name,
		       te.day_of_week, p.id::text, p.name_en, p.start_time::text, p.end_time::text,
		       sub.id::text, sub.name_en, st.id::text, st.full_name_en,
		       COALESCE(replacement.id::text, ''), COALESCE(replacement.full_name_en, ''),
		       COALESCE(te.room, '')
		  FROM timetable_entries te
		  JOIN sections sec ON sec.id = te.section_id
		  JOIN classes c ON c.id = sec.class_id
		  JOIN periods p ON p.id = te.period_id
		  JOIN subjects sub ON sub.id = te.subject_id
		  JOIN staff st ON st.id = te.staff_id
		  JOIN enrollments e ON e.section_id = te.section_id AND e.session_id = te.session_id AND e.status = 'ENROLLED'
		  JOIN student_guardians sg ON sg.student_id = e.student_id
		  LEFT JOIN substitutions replacement_assignment
		    ON replacement_assignment.timetable_entry_id = te.id
		   AND replacement_assignment.date = NULLIF($3, '')::date
		  LEFT JOIN staff replacement ON replacement.id = replacement_assignment.substitute_staff_id
		 WHERE te.session_id = $1::uuid AND sg.guardian_id = $2::uuid
		 ORDER BY c.sort_order, sec.sort_order, te.day_of_week, p.position`
	return s.listTimetable(ctx, query, sessionID, guardianID, date)
}

// CreateTimetableEntry validates that the subject is offered in the section's
// class/stream before relying on the database slot and teacher clash indexes.
func (s *Store) CreateTimetableEntry(ctx context.Context, input NewTimetableEntry) (string, error) {
	const offeredSubject = `
		SELECT EXISTS (
			SELECT 1
			  FROM sections sec
			  JOIN class_subjects cs ON cs.class_id = sec.class_id AND cs.session_id = $1::uuid
			 WHERE sec.id = $2::uuid AND cs.subject_id = $3::uuid
			   AND (cs.stream_id IS NULL OR cs.stream_id = sec.stream_id)
		)`
	var offered bool
	if err := s.pool.QueryRow(ctx, offeredSubject, input.SessionID, input.SectionID, input.SubjectID).Scan(&offered); err != nil {
		return "", fmt.Errorf("check timetable subject: %w", err)
	}
	if !offered {
		return "", fmt.Errorf("create timetable entry: subject is not offered for this section: %w", ErrConflict)
	}

	const query = `
		INSERT INTO timetable_entries (session_id, section_id, day_of_week, period_id, subject_id, staff_id, room)
		VALUES ($1::uuid, $2::uuid, $3, $4::uuid, $5::uuid, $6::uuid, NULLIF($7, ''))
		RETURNING id::text`
	var id string
	if err := s.pool.QueryRow(ctx, query, input.SessionID, input.SectionID, input.DayOfWeek, input.PeriodID, input.SubjectID, input.StaffID, input.Room).Scan(&id); err != nil {
		if _, unique := isUniqueViolation(err); unique {
			return "", fmt.Errorf("create timetable entry: %w", ErrConflict)
		}
		return "", fmt.Errorf("create timetable entry: %w", err)
	}
	return id, nil
}

type NewSubstitution struct {
	Date              string
	TimetableEntryID  string
	SubstituteStaffID string
	Reason            string
	CreatedBy         string
}

// CreateSubstitution records a dated replacement. The regular schedule stays
// unchanged and the timetable's unique constraint gives each lesson/date one
// authoritative substitute.
func (s *Store) CreateSubstitution(ctx context.Context, input NewSubstitution) (string, error) {
	const query = `
		INSERT INTO substitutions (date, timetable_entry_id, substitute_staff_id, reason, created_by)
		SELECT $1::date, te.id, st.id, NULLIF($4, ''), $5::uuid
		  FROM timetable_entries te
		  JOIN staff st ON st.id = $3::uuid AND st.status = 'ACTIVE'
		 WHERE te.id = $2::uuid
		RETURNING id::text`
	var id string
	if err := s.pool.QueryRow(ctx, query, input.Date, input.TimetableEntryID, input.SubstituteStaffID, input.Reason, input.CreatedBy).Scan(&id); err != nil {
		if _, unique := isUniqueViolation(err); unique {
			return "", fmt.Errorf("create substitution: %w", ErrConflict)
		}
		return "", fmt.Errorf("create substitution: %w", noRows(err))
	}
	return id, nil
}
