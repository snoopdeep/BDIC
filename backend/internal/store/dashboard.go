package store

import (
	"context"
	"fmt"
	"time"
)

// GetStudentEnrollment returns the enrollment record for a student in a specific session.
func (s *Store) GetStudentEnrollment(ctx context.Context, studentID, sessionID string) (map[string]any, error) {
	return s.GetStudentEnrollmentV2(ctx, studentID, sessionID)
}

// GetStudentEnrollment returns the enrollment record for a student in a specific session.
func (s *Store) GetStudentEnrollmentV2(ctx context.Context, studentID, sessionID string) (map[string]any, error) {
	const query = `
		SELECT id::text,
		       student_id::text,
		       session_id::text,
		       class_id::text,
		       section_id::text,
		       COALESCE(stream_id::text, ''),
		       COALESCE(roll_no, 0),
		       status
		  FROM enrollments
		 WHERE student_id = $1::uuid
		   AND session_id = $2::uuid
		 LIMIT 1`

	var id, studentIDOut, sessionIDOut, classID, sectionID, streamID, status string
	var rollNo int

	err := s.pool.QueryRow(ctx, query, studentID, sessionID).Scan(
		&id, &studentIDOut, &sessionIDOut, &classID, &sectionID, &streamID, &rollNo, &status,
	)
	if err != nil {
		return nil, fmt.Errorf("get student enrollment: %w", noRows(err))
	}

	return map[string]any{
		"id":         id,
		"student_id": studentIDOut,
		"session_id": sessionIDOut,
		"class_id":   classID,
		"section_id": sectionID,
		"stream_id":  streamID,
		"roll_no":    rollNo,
		"status":     status,
	}, nil
}

// GetNextTimetableEntry returns the next timetable entry for a section on a given day after a certain time.
func (s *Store) GetNextTimetableEntry(ctx context.Context, sessionID, sectionID string, dayOfWeek int, afterTime time.Time) (map[string]any, error) {
	const query = `
		SELECT tt.id::text,
		       subj.name_en,
		       per.start_time::text,
		       per.end_time::text,
		       COALESCE(tt.room, ''),
		       st.full_name_en
		  FROM timetable_entries tt
		  JOIN subjects subj ON subj.id = tt.subject_id
		  JOIN periods per ON per.id = tt.period_id
		  JOIN staff st ON st.id = tt.staff_id
		 WHERE tt.session_id = $1::uuid
		   AND tt.section_id = $2::uuid
		   AND tt.day_of_week = $3
		   AND per.start_time > $4::time
		 ORDER BY per.start_time ASC
		 LIMIT 1`

	var id, subjectName, startTime, endTime, room, teacherName string
	err := s.pool.QueryRow(ctx, query, sessionID, sectionID, dayOfWeek, afterTime.Format("15:04:05")).Scan(
		&id, &subjectName, &startTime, &endTime, &room, &teacherName,
	)
	if err != nil {
		return nil, fmt.Errorf("get next timetable entry: %w", err)
	}

	return map[string]any{
		"id":           id,
		"subject_name": subjectName,
		"start_time":   startTime,
		"end_time":     endTime,
		"room":         room,
		"teacher_name": teacherName,
	}, nil
}

// GetStudentAttendanceStats returns attendance statistics for a student in a session.
func (s *Store) GetStudentAttendanceStats(ctx context.Context, studentID, sessionID string) (map[string]any, error) {
	const query = `
		SELECT COUNT(*) FILTER (WHERE status = 'PRESENT') as present_days,
		       COUNT(CASE WHEN status IN ('PRESENT', 'ABSENT', 'LATE', 'HALF_DAY') THEN 1 END) as total_days,
		       COUNT(*) FILTER (WHERE status IN ('LEAVE', 'MEDICAL')) as leave_days
		  FROM attendance
		 WHERE student_id = $1::uuid
		   AND session_id = $2::uuid`

	var presentDays, totalDays, leaveDays int
	err := s.pool.QueryRow(ctx, query, studentID, sessionID).Scan(
		&presentDays, &totalDays, &leaveDays,
	)
	if err != nil {
		return nil, fmt.Errorf("get student attendance stats: %w", err)
	}

	return map[string]any{
		"present_days": presentDays,
		"total_days":   totalDays,
		"leave_days":   leaveDays,
	}, nil
}

// GetStudentOutstandingFees returns the total outstanding fees for a student in a session.
func (s *Store) GetStudentOutstandingFees(ctx context.Context, studentID, sessionID string) (map[string]any, error) {
	const query = `
		SELECT COALESCE(SUM(gross_paise - paid_paise + late_fee_paise - concession_paise), 0)::bigint as due_amount_paise,
		       COUNT(*) as due_invoice_count
		  FROM invoices
		 WHERE student_id = $1::uuid
		   AND session_id = $2::uuid
		   AND status IN ('DUE', 'PARTIAL')`

	var dueAmount int64
	var invoiceCount int
	err := s.pool.QueryRow(ctx, query, studentID, sessionID).Scan(
		&dueAmount, &invoiceCount,
	)
	if err != nil {
		return nil, fmt.Errorf("get student outstanding fees: %w", err)
	}

	return map[string]any{
		"due_amount_paise":  dueAmount,
		"due_invoice_count": invoiceCount,
	}, nil
}

// GetStudentRecentMarks returns the recent marks for a student in a session.
func (s *Store) GetStudentRecentMarks(ctx context.Context, studentID, sessionID string, limit int) ([]map[string]any, error) {
	const query = `
		SELECT ex.name_en,
		       subj.name_en,
		       m.marks_obtained,
		       exs.max_marks,
		       CASE WHEN exs.max_marks > 0
		            THEN ROUND((m.marks_obtained / exs.max_marks) * 100, 2)
		            ELSE 0 END as percentage,
		       COALESCE(gs.grade, '—')
		  FROM marks m
		  JOIN exam_subjects exs ON exs.id = m.exam_subject_id
		  JOIN exams ex ON ex.id = exs.exam_id
		  JOIN subjects subj ON subj.id = exs.subject_id
		  LEFT JOIN grading_scales gs ON gs.session_id = ex.session_id
		    AND m.marks_obtained >= gs.min_percent
		    AND m.marks_obtained <= gs.max_percent
		 WHERE m.student_id = $1::uuid
		   AND ex.session_id = $2::uuid
		   AND ex.status = 'PUBLISHED'
		   AND m.special_status = 'NONE'
		 ORDER BY ex.created_at DESC, subj.name_en ASC
		 LIMIT $3`

	rows, err := s.pool.Query(ctx, query, studentID, sessionID, limit)
	if err != nil {
		return nil, fmt.Errorf("get student recent marks: %w", err)
	}
	defer rows.Close()

	var marks []map[string]any
	for rows.Next() {
		var examName, subjectName, grade string
		var marksObtained, percentage float64
		var maxMarks int

		if err := rows.Scan(&examName, &subjectName, &marksObtained, &maxMarks, &percentage, &grade); err != nil {
			return nil, fmt.Errorf("scan mark: %w", err)
		}

		marks = append(marks, map[string]any{
			"exam_name":      examName,
			"subject_name":   subjectName,
			"marks_obtained": marksObtained,
			"max_marks":      maxMarks,
			"percentage":     percentage,
			"grade":          grade,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate marks: %w", err)
	}

	return marks, nil
}

// GetGuardianChildren returns all children linked to a guardian.
func (s *Store) GetGuardianChildren(ctx context.Context, guardianID string) ([]map[string]any, error) {
	const query = `
		SELECT st.id::text,
		       st.full_name_en,
		       st.admission_no
		  FROM students st
		  JOIN student_guardians sg ON sg.student_id = st.id
		 WHERE sg.guardian_id = $1::uuid
		   AND st.status = 'ACTIVE'
		 ORDER BY st.full_name_en`

	rows, err := s.pool.Query(ctx, query, guardianID)
	if err != nil {
		return nil, fmt.Errorf("get guardian children: %w", err)
	}
	defer rows.Close()

	var children []map[string]any
	for rows.Next() {
		var studentID, fullName, admissionNo string
		if err := rows.Scan(&studentID, &fullName, &admissionNo); err != nil {
			return nil, fmt.Errorf("scan child: %w", err)
		}

		children = append(children, map[string]any{
			"student_id":   studentID,
			"full_name":    fullName,
			"admission_no": admissionNo,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate children: %w", err)
	}

	return children, nil
}

// GetClassInfo returns basic class information.
func (s *Store) GetClassInfo(ctx context.Context, classID string) (map[string]any, error) {
	const query = `
		SELECT cl.id::text,
		       cl.name_en,
		       cl.code,
		       cl.level
		  FROM classes cl
		 WHERE cl.id = $1::uuid
		 LIMIT 1`

	var id, name, code string
	var level int
	err := s.pool.QueryRow(ctx, query, classID).Scan(&id, &name, &code, &level)
	if err != nil {
		return nil, fmt.Errorf("get class info: %w", noRows(err))
	}

	return map[string]any{
		"id":         id,
		"class_name": name,
		"code":       code,
		"level":      level,
	}, nil
}

// GetTeacherSections returns all sections taught by a teacher in a session.
func (s *Store) GetTeacherSections(ctx context.Context, staffID, sessionID string) ([]map[string]any, error) {
	const query = `
		SELECT DISTINCT sec.id::text,
		       sec.name,
		       cl.name_en,
		       COUNT(en.id) as enrollment_count
		  FROM section_teachers st_teach
		  JOIN sections sec ON sec.id = st_teach.section_id
		  JOIN classes cl ON cl.id = sec.class_id
		  LEFT JOIN enrollments en ON en.section_id = sec.id AND en.session_id = $2::uuid
		 WHERE st_teach.staff_id = $1::uuid
		   AND st_teach.session_id = $2::uuid
		 GROUP BY sec.id, sec.name, cl.name_en
		 ORDER BY cl.name_en, sec.name`

	rows, err := s.pool.Query(ctx, query, staffID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get teacher sections: %w", err)
	}
	defer rows.Close()

	var sections []map[string]any
	for rows.Next() {
		var sectionID, sectionName, className string
		var enrollmentCount int
		if err := rows.Scan(&sectionID, &sectionName, &className, &enrollmentCount); err != nil {
			return nil, fmt.Errorf("scan section: %w", err)
		}

		sections = append(sections, map[string]any{
			"section_id":       sectionID,
			"section_name":     sectionName,
			"class_name":       className,
			"enrollment_count": enrollmentCount,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sections: %w", err)
	}

	return sections, nil
}

// GetTeacherPendingAssignments returns homework assigned to a teacher's sections with future due dates.
func (s *Store) GetTeacherPendingAssignments(ctx context.Context, staffID, sessionID string) ([]map[string]any, error) {
	const query = `
		SELECT hw.id::text,
		       hw.title,
		       sec.name,
		       hw.due_date::text,
		       subj.name_en,
		       hw.status
		  FROM homework hw
		  JOIN sections sec ON sec.id = hw.section_id
		  JOIN subjects subj ON subj.id = hw.subject_id
		 WHERE hw.staff_id = $1::uuid
		   AND hw.session_id = $2::uuid
		   AND hw.due_date >= CURRENT_DATE
		   AND hw.status IN ('DRAFT', 'PUBLISHED')
		 ORDER BY hw.due_date ASC
		 LIMIT 10`

	rows, err := s.pool.Query(ctx, query, staffID, sessionID)
	if err != nil {
		return nil, fmt.Errorf("get teacher pending assignments: %w", err)
	}
	defer rows.Close()

	var assignments []map[string]any
	for rows.Next() {
		var id, title, sectionName, dueDate, subjectName, status string
		if err := rows.Scan(&id, &title, &sectionName, &dueDate, &subjectName, &status); err != nil {
			return nil, fmt.Errorf("scan assignment: %w", err)
		}

		assignments = append(assignments, map[string]any{
			"id":           id,
			"title":        title,
			"section_name": sectionName,
			"due_date":     dueDate,
			"subject_name": subjectName,
			"status":       status,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate assignments: %w", err)
	}

	return assignments, nil
}

// GetTeacherTodaySchedule returns timetable entries for a teacher on a specific day.
func (s *Store) GetTeacherTodaySchedule(ctx context.Context, staffID, sessionID string, dayOfWeek int) ([]map[string]any, error) {
	const query = `
		SELECT per.name_en,
		       sec.name,
		       subj.name_en,
		       per.start_time::text,
		       per.end_time::text,
		       COALESCE(tt.room, '')
		  FROM timetable_entries tt
		  JOIN periods per ON per.id = tt.period_id
		  JOIN sections sec ON sec.id = tt.section_id
		  JOIN subjects subj ON subj.id = tt.subject_id
		 WHERE tt.staff_id = $1::uuid
		   AND tt.session_id = $2::uuid
		   AND tt.day_of_week = $3
		 ORDER BY per.start_time ASC`

	rows, err := s.pool.Query(ctx, query, staffID, sessionID, dayOfWeek)
	if err != nil {
		return nil, fmt.Errorf("get teacher today schedule: %w", err)
	}
	defer rows.Close()

	var schedule []map[string]any
	for rows.Next() {
		var periodName, sectionName, subjectName, startTime, endTime, room string
		if err := rows.Scan(&periodName, &sectionName, &subjectName, &startTime, &endTime, &room); err != nil {
			return nil, fmt.Errorf("scan schedule entry: %w", err)
		}

		schedule = append(schedule, map[string]any{
			"period_name":  periodName,
			"section_name": sectionName,
			"subject_name": subjectName,
			"start_time":   startTime,
			"end_time":     endTime,
			"room":         room,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate schedule: %w", err)
	}

	return schedule, nil
}

// GetEnrollmentStats returns enrollment statistics for a session.
func (s *Store) GetEnrollmentStats(ctx context.Context, sessionID string) (map[string]any, error) {
	const query = `
		SELECT COUNT(DISTINCT en.id) as total_enrolled,
		       COUNT(DISTINCT cl.id) as class_count
		  FROM enrollments en
		  JOIN classes cl ON cl.id = en.class_id
		 WHERE en.session_id = $1::uuid
		   AND en.status = 'ENROLLED'`

	var totalEnrolled, classCount int
	err := s.pool.QueryRow(ctx, query, sessionID).Scan(&totalEnrolled, &classCount)
	if err != nil {
		return nil, fmt.Errorf("get enrollment stats: %w", err)
	}

	return map[string]any{
		"total_enrolled": totalEnrolled,
		"class_count":    classCount,
	}, nil
}

// GetPendingConcessions returns pending student concession requests.
func (s *Store) GetPendingConcessions(ctx context.Context, sessionID string) (map[string]any, error) {
	const query = `
		SELECT COUNT(*) as pending_count
		  FROM student_concessions
		 WHERE session_id = $1::uuid
		   AND status = 'PENDING'`

	var pendingCount int
	err := s.pool.QueryRow(ctx, query, sessionID).Scan(&pendingCount)
	if err != nil {
		return nil, fmt.Errorf("get pending concessions: %w", err)
	}

	return map[string]any{
		"pending_count": pendingCount,
	}, nil
}

// GetPendingAttendanceCorrections returns pending attendance correction requests.
func (s *Store) GetPendingAttendanceCorrections(ctx context.Context) (map[string]any, error) {
	const query = `
		SELECT COUNT(*) as pending_count
		  FROM attendance_corrections
		 WHERE status = 'PENDING'`

	var pendingCount int
	err := s.pool.QueryRow(ctx, query).Scan(&pendingCount)
	if err != nil {
		return nil, fmt.Errorf("get pending attendance corrections: %w", err)
	}

	return map[string]any{
		"pending_count": pendingCount,
	}, nil
}

// GetRecentActivity returns recent audit log entries.
func (s *Store) GetRecentActivity(ctx context.Context, limit int) ([]map[string]any, error) {
	const query = `
		SELECT entity_type,
		       action,
		       summary,
		       created_at::text
		  FROM audit_logs
		 ORDER BY created_at DESC
		 LIMIT $1`

	rows, err := s.pool.Query(ctx, query, limit)
	if err != nil {
		return nil, fmt.Errorf("get recent activity: %w", err)
	}
	defer rows.Close()

	var activity []map[string]any
	for rows.Next() {
		var entityType, action, summary, timestamp string
		if err := rows.Scan(&entityType, &action, &summary, &timestamp); err != nil {
			return nil, fmt.Errorf("scan activity: %w", err)
		}

		activity = append(activity, map[string]any{
			"entity_type": entityType,
			"action":      action,
			"summary":     summary,
			"timestamp":   timestamp,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate activity: %w", err)
	}

	return activity, nil
}

// GetTotalOutstandingFees returns total outstanding fees across all students in a session.
func (s *Store) GetTotalOutstandingFees(ctx context.Context, sessionID string) (map[string]any, error) {
	const query = `
		SELECT COALESCE(SUM(gross_paise - paid_paise + late_fee_paise - concession_paise), 0)::bigint as total_paise
		  FROM invoices
		 WHERE session_id = $1::uuid
		   AND status IN ('DUE', 'PARTIAL')`

	var totalPaise int64
	err := s.pool.QueryRow(ctx, query, sessionID).Scan(&totalPaise)
	if err != nil {
		return nil, fmt.Errorf("get total outstanding fees: %w", err)
	}

	return map[string]any{
		"total_paise": totalPaise,
	}, nil
}

// GetPublishedNotices returns published notices for the dashboard.
func (s *Store) GetPublishedNotices(ctx context.Context, sessionID string) ([]map[string]any, error) {
	const query = `
		SELECT title_en,
		       COALESCE(title_hi, ''),
		       body_en,
		       COALESCE(body_hi, ''),
		       publish_at::text
		  FROM notices
		 WHERE status = 'PUBLISHED'
		   AND (expire_at IS NULL OR expire_at > NOW())
		   AND publish_at <= NOW()
		 ORDER BY publish_at DESC
		 LIMIT 20`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get published notices: %w", err)
	}
	defer rows.Close()

	var notices []map[string]any
	for rows.Next() {
		var titleEN, titleHI, bodyEN, bodyHI, publishedAt string
		if err := rows.Scan(&titleEN, &titleHI, &bodyEN, &bodyHI, &publishedAt); err != nil {
			return nil, fmt.Errorf("scan notice: %w", err)
		}

		notices = append(notices, map[string]any{
			"title_en":     titleEN,
			"title_hi":     titleHI,
			"body_en":      bodyEN,
			"body_hi":      bodyHI,
			"published_at": publishedAt,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate notices: %w", err)
	}

	return notices, nil
}
