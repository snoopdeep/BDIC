package api

import (
	"errors"
	"fmt"
	"net/http"
	"time"

	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

// registerExamRoutes registers all exam-related routes.
func (s *Server) registerExamRoutes(mux *http.ServeMux) {
	// ---- Any signed-in user ------------------------------------------------
	// Students and parents can fetch exam schedule and their own results.
	mux.Handle("GET /api/v1/exams/schedule", s.signedIn(s.handleGetExamSchedule))
	mux.Handle("GET /api/v1/exams/results", s.signedIn(s.handleGetMyExamResults))
	mux.Handle("GET /api/v1/exams/admit-cards", s.signedIn(s.handleGetAdmitCard))

	// ---- Teachers and above -------------------------------------------------
	// Teachers can view results for their class and submit/update results.
	mux.Handle("GET /api/v1/exams/{examId}/results",
		s.restricted(s.handleGetExamResults, auth.RoleTeacher, auth.RolePrincipal, auth.RoleSuperAdmin))
	mux.Handle("GET /api/v1/exams/{examId}/answer-sheets",
		s.restricted(s.handleListAnswerSheets, auth.RoleTeacher, auth.RolePrincipal, auth.RoleSuperAdmin))
	mux.Handle("POST /api/v1/exams/results",
		s.restricted(s.handleCreateUpdateExamResult, auth.RoleTeacher, auth.RolePrincipal, auth.RoleSuperAdmin))
}

// ExamScheduleItem represents a single exam in the schedule.
type ExamScheduleItem struct {
	ID        string `json:"id"`
	ExamName  string `json:"examName"`
	ExamDate  string `json:"examDate"`
	StartTime string `json:"startTime"`
	EndTime   string `json:"endTime"`
	ClassID   string `json:"classId"`
	Subject   string `json:"subject"`
	MaxMarks  int    `json:"maxMarks"`
	PassMarks int    `json:"passMarks"`
}

// handleGetExamSchedule fetches exam schedules for the current user.
// Students see exams for their class; teachers see exams for their classes.
func (s *Server) handleGetExamSchedule(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	identity := auth.MustFromContext(r.Context())
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	limit, offset := paginate(r)

	// Build query based on user role
	query := `
		SELECT
			es.id,
			e.name_en,
			es.exam_date,
			es.start_time,
			-- Default end_time calculation: start_time + 3 hours
			COALESCE(es.start_time + interval '3 hours', null) as end_time,
			es.class_id,
			subj.name_en,
			es.max_marks,
			es.pass_marks
		FROM exam_subjects es
		JOIN exams e ON e.id = es.exam_id
		JOIN classes c ON c.id = es.class_id
		JOIN subjects subj ON subj.id = es.subject_id
		WHERE e.session_id = $1::uuid
		AND e.status IN ('SCHEDULED', 'MARKS_OPEN', 'APPROVED', 'PUBLISHED')
		AND es.exam_date >= CURRENT_DATE
	`

	args := []interface{}{sessionID}
	argNum := 2

	// Filter by class for students
	if identity.HasRole(auth.RoleStudent) && identity.StudentID != "" {
		query += ` AND es.class_id IN (
			SELECT class_id FROM enrollments
			WHERE student_id = $` + fmt.Sprintf("%d", argNum) + `::uuid
		)`
		args = append(args, identity.StudentID)
		argNum++
	}

	// Filter by teacher's classes
	if identity.HasRole(auth.RoleTeacher) && identity.StaffID != "" {
		query += ` AND es.class_id IN (
			SELECT sec.class_id
			  FROM section_teachers section_teacher
			  JOIN sections sec ON sec.id = section_teacher.section_id
			 WHERE section_teacher.session_id = $1::uuid
			   AND section_teacher.staff_id = $` + fmt.Sprintf("%d", argNum) + `::uuid
			UNION
			SELECT sec.class_id
			  FROM timetable_entries timetable
			  JOIN sections sec ON sec.id = timetable.section_id
			 WHERE timetable.session_id = $1::uuid
			   AND timetable.staff_id = $` + fmt.Sprintf("%d", argNum) + `::uuid
		)`
		args = append(args, identity.StaffID)
		argNum++
	}

	query += ` ORDER BY es.exam_date ASC
		LIMIT $` + fmt.Sprintf("%d", argNum) + ` OFFSET $` + fmt.Sprintf("%d", argNum+1)
	args = append(args, limit, offset)

	rows, err := s.store.Pool().Query(ctx, query, args...)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	defer rows.Close()

	items := []ExamScheduleItem{}
	for rows.Next() {
		var item ExamScheduleItem
		var examDate *time.Time
		var startTime *time.Time
		var endTime *time.Time

		err := rows.Scan(
			&item.ID,
			&item.ExamName,
			&examDate,
			&startTime,
			&endTime,
			&item.ClassID,
			&item.Subject,
			&item.MaxMarks,
			&item.PassMarks,
		)
		if err != nil {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}

		if examDate != nil {
			item.ExamDate = examDate.Format("2006-01-02")
		}
		if startTime != nil {
			item.StartTime = startTime.Format("15:04")
		}
		if endTime != nil {
			item.EndTime = endTime.Format("15:04")
		}

		items = append(items, item)
	}

	if err = rows.Err(); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM exam_subjects es
		JOIN exams e ON e.id = es.exam_id
		WHERE e.session_id = $1::uuid
		AND e.status IN ('SCHEDULED', 'MARKS_OPEN', 'APPROVED', 'PUBLISHED')
		AND es.exam_date >= CURRENT_DATE
	`
	countArgs := []interface{}{sessionID}
	countArgNum := 2

	if identity.HasRole(auth.RoleStudent) && identity.StudentID != "" {
		countQuery += ` AND es.class_id IN (
			SELECT class_id FROM enrollments
			WHERE student_id = $` + fmt.Sprintf("%d", countArgNum) + `::uuid
		)`
		countArgs = append(countArgs, identity.StudentID)
		countArgNum++
	}

	if identity.HasRole(auth.RoleTeacher) && identity.StaffID != "" {
		countQuery += ` AND es.class_id IN (
			SELECT sec.class_id
			  FROM section_teachers section_teacher
			  JOIN sections sec ON sec.id = section_teacher.section_id
			 WHERE section_teacher.session_id = $1::uuid
			   AND section_teacher.staff_id = $` + fmt.Sprintf("%d", countArgNum) + `::uuid
			UNION
			SELECT sec.class_id
			  FROM timetable_entries timetable
			  JOIN sections sec ON sec.id = timetable.section_id
			 WHERE timetable.session_id = $1::uuid
			   AND timetable.staff_id = $` + fmt.Sprintf("%d", countArgNum) + `::uuid
		)`
		countArgs = append(countArgs, identity.StaffID)
		countArgNum++
	}

	var total int
	err = s.store.Pool().QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	httpx.JSON(w, http.StatusOK, httpx.NewPage(items, total, limit, offset))
}

// ExamResult represents exam results for a student.
type ExamResult struct {
	StudentID     string   `json:"studentId"`
	ExamID        string   `json:"examId"`
	ExamName      string   `json:"examName"`
	MarksObtained *float32 `json:"marksObtained"`
	TotalMarks    int      `json:"totalMarks"`
	Grade         *string  `json:"grade"`
	SpecialStatus string   `json:"specialStatus"`
}

// handleGetMyExamResults fetches results for a student's own record or for
// every child linked to a parent. Staff use the explicit exam-scoped endpoint.
func (s *Server) handleGetMyExamResults(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	identity := auth.MustFromContext(r.Context())
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	limit, offset := paginate(r)

	var scopeSQL string
	var scopeID string
	switch {
	case identity.HasRole(auth.RoleStudent) && identity.StudentID != "":
		scopeSQL = "AND m.student_id = $2::uuid"
		scopeID = identity.StudentID
	case identity.HasRole(auth.RoleParent) && identity.GuardianID != "":
		scopeSQL = `AND m.student_id IN (
			SELECT student_id FROM student_guardians WHERE guardian_id = $2::uuid
		)`
		scopeID = identity.GuardianID
	default:
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	query := `
		SELECT
			m.student_id,
			e.id,
			e.name_en,
			m.marks_obtained,
			es.max_marks,
			rc.grade,
			m.special_status
		FROM marks m
		JOIN exam_subjects es ON es.id = m.exam_subject_id
		JOIN exams e ON e.id = es.exam_id
		LEFT JOIN report_cards rc ON rc.exam_id = e.id AND rc.student_id = m.student_id
		WHERE e.session_id = $1::uuid
		` + scopeSQL + `
		AND e.status IN ('APPROVED', 'PUBLISHED')
		ORDER BY e.created_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.store.Pool().Query(ctx, query, sessionID, scopeID, limit, offset)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	defer rows.Close()

	items := []ExamResult{}
	for rows.Next() {
		var result ExamResult
		err := rows.Scan(
			&result.StudentID,
			&result.ExamID,
			&result.ExamName,
			&result.MarksObtained,
			&result.TotalMarks,
			&result.Grade,
			&result.SpecialStatus,
		)
		if err != nil {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}
		items = append(items, result)
	}

	if err = rows.Err(); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	// Get total count
	var total int
	countQuery := `
		SELECT COUNT(*)
		FROM marks m
		JOIN exam_subjects es ON es.id = m.exam_subject_id
		JOIN exams e ON e.id = es.exam_id
		WHERE e.session_id = $1::uuid
		` + scopeSQL + `
		AND e.status IN ('APPROVED', 'PUBLISHED')
	`
	err = s.store.Pool().QueryRow(ctx, countQuery, sessionID, scopeID).Scan(&total)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	httpx.JSON(w, http.StatusOK, httpx.NewPage(items, total, limit, offset))
}

// handleGetExamResults fetches results for a specific exam.
// Teachers see results for their class; admins see all results.
func (s *Server) handleGetExamResults(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	identity := auth.MustFromContext(r.Context())

	examID, err := pathUUID(r, "examId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	limit, offset := paginate(r)

	query := `
		SELECT
			m.student_id,
			e.id,
			e.name_en,
			m.marks_obtained,
			es.max_marks,
			rc.grade,
			m.special_status
		FROM marks m
		JOIN exam_subjects es ON es.id = m.exam_subject_id
		JOIN exams e ON e.id = es.exam_id
		LEFT JOIN report_cards rc ON rc.exam_id = e.id AND rc.student_id = m.student_id
		WHERE e.id = $1::uuid
	`

	args := []interface{}{examID}
	argNum := 2

	// Filter by teacher's class
	if identity.HasRole(auth.RoleTeacher) && identity.StaffID != "" {
		query += ` AND es.class_id IN (
			SELECT DISTINCT class_id FROM teaching_assignments
			WHERE staff_id = $` + fmt.Sprintf("%d", argNum) + `::uuid
		)`
		args = append(args, identity.StaffID)
		argNum++
	}

	query += ` ORDER BY m.student_id ASC
		LIMIT $` + fmt.Sprintf("%d", argNum) + ` OFFSET $` + fmt.Sprintf("%d", argNum+1)
	args = append(args, limit, offset)

	rows, err := s.store.Pool().Query(ctx, query, args...)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	defer rows.Close()

	items := []ExamResult{}
	for rows.Next() {
		var result ExamResult
		err := rows.Scan(
			&result.StudentID,
			&result.ExamID,
			&result.ExamName,
			&result.MarksObtained,
			&result.TotalMarks,
			&result.Grade,
			&result.SpecialStatus,
		)
		if err != nil {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}
		items = append(items, result)
	}

	if err = rows.Err(); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM marks m
		JOIN exam_subjects es ON es.id = m.exam_subject_id
		JOIN exams e ON e.id = es.exam_id
		WHERE e.id = $1::uuid
	`
	countArgs := []interface{}{examID}
	countArgNum := 2

	if identity.HasRole(auth.RoleTeacher) && identity.StaffID != "" {
		countQuery += ` AND es.class_id IN (
			SELECT DISTINCT class_id FROM teaching_assignments
			WHERE staff_id = $` + fmt.Sprintf("%d", countArgNum) + `::uuid
		)`
		countArgs = append(countArgs, identity.StaffID)
		countArgNum++
	}

	var total int
	err = s.store.Pool().QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	httpx.JSON(w, http.StatusOK, httpx.NewPage(items, total, limit, offset))
}

// AnswerSheet represents a student's answer sheet for an exam.
type AnswerSheet struct {
	ID            string   `json:"id"`
	StudentID     string   `json:"studentId"`
	StudentName   string   `json:"studentName"`
	ExamID        string   `json:"examId"`
	MarksObtained *float32 `json:"marksObtained"`
	TotalMarks    int      `json:"totalMarks"`
	LockedAt      *string  `json:"lockedAt"`
	LockedBy      *string  `json:"lockedBy"`
}

// handleListAnswerSheets lists student answer sheets for an exam.
func (s *Server) handleListAnswerSheets(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	identity := auth.MustFromContext(r.Context())

	examID, err := pathUUID(r, "examId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	limit, offset := paginate(r)

	query := `
		SELECT
			m.id,
			m.student_id,
			COALESCE(u.full_name_en, s.admission_number),
			e.id,
			m.marks_obtained,
			es.max_marks,
			es.locked_at,
			es.locked_by
		FROM marks m
		JOIN exam_subjects es ON es.id = m.exam_subject_id
		JOIN exams e ON e.id = es.exam_id
		JOIN students s ON s.id = m.student_id
		JOIN users u ON u.id = s.user_id
		WHERE e.id = $1::uuid
	`

	args := []interface{}{examID}
	argNum := 2

	// Filter by teacher's class
	if identity.HasRole(auth.RoleTeacher) && identity.StaffID != "" {
		query += ` AND es.class_id IN (
			SELECT DISTINCT class_id FROM teaching_assignments
			WHERE staff_id = $` + fmt.Sprintf("%d", argNum) + `::uuid
		)`
		args = append(args, identity.StaffID)
		argNum++
	}

	query += ` ORDER BY u.full_name_en ASC
		LIMIT $` + fmt.Sprintf("%d", argNum) + ` OFFSET $` + fmt.Sprintf("%d", argNum+1)
	args = append(args, limit, offset)

	rows, err := s.store.Pool().Query(ctx, query, args...)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	defer rows.Close()

	items := []AnswerSheet{}
	for rows.Next() {
		var sheet AnswerSheet
		var lockedAt *time.Time
		err := rows.Scan(
			&sheet.ID,
			&sheet.StudentID,
			&sheet.StudentName,
			&sheet.ExamID,
			&sheet.MarksObtained,
			&sheet.TotalMarks,
			&lockedAt,
			&sheet.LockedBy,
		)
		if err != nil {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}

		if lockedAt != nil {
			timeStr := lockedAt.Format(time.RFC3339)
			sheet.LockedAt = &timeStr
		}

		items = append(items, sheet)
	}

	if err = rows.Err(); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	// Get total count
	countQuery := `
		SELECT COUNT(*)
		FROM marks m
		JOIN exam_subjects es ON es.id = m.exam_subject_id
		JOIN exams e ON e.id = es.exam_id
		WHERE e.id = $1::uuid
	`
	countArgs := []interface{}{examID}
	countArgNum := 2

	if identity.HasRole(auth.RoleTeacher) && identity.StaffID != "" {
		countQuery += ` AND es.class_id IN (
			SELECT DISTINCT class_id FROM teaching_assignments
			WHERE staff_id = $` + fmt.Sprintf("%d", countArgNum) + `::uuid
		)`
		countArgs = append(countArgs, identity.StaffID)
		countArgNum++
	}

	var total int
	err = s.store.Pool().QueryRow(ctx, countQuery, countArgs...).Scan(&total)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	httpx.JSON(w, http.StatusOK, httpx.NewPage(items, total, limit, offset))
}

// CreateUpdateExamResultRequest represents the payload for creating/updating exam results.
type CreateUpdateExamResultRequest struct {
	ExamSubjectID string   `json:"examSubjectId"`
	StudentID     string   `json:"studentId"`
	MarksObtained *float32 `json:"marksObtained"`
	SpecialStatus string   `json:"specialStatus"`
	Remark        *string  `json:"remark"`
}

// handleCreateUpdateExamResult creates or updates exam results.
// Only teachers for their class, and admins can do this.
func (s *Server) handleCreateUpdateExamResult(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	identity := auth.MustFromContext(r.Context())

	var payload CreateUpdateExamResultRequest
	if err := httpx.DecodeJSON(w, r, &payload); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Validate input
	if payload.ExamSubjectID == "" || payload.StudentID == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"examSubjectId and studentId are required.",
			"examSubjectId और studentId आवश्यक हैं।"))
		return
	}

	if !auth.IsUUID(payload.ExamSubjectID) || !auth.IsUUID(payload.StudentID) {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Invalid identifier format.",
			"अमान्य पहचानकर्ता प्रारूप।"))
		return
	}

	// Validate special status
	validStatuses := map[string]bool{
		"NONE":           true,
		"ABSENT":         true,
		"EXEMPT":         true,
		"MEDICAL":        true,
		"REEXAM_PENDING": true,
	}

	if payload.SpecialStatus == "" {
		payload.SpecialStatus = "NONE"
	}

	if !validStatuses[payload.SpecialStatus] {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Invalid special status.",
			"अमान्य विशेष स्थिति।"))
		return
	}

	// For NONE status, marks must be provided
	if payload.SpecialStatus == "NONE" && payload.MarksObtained == nil {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Marks are required when special status is NONE.",
			"जब विशेष स्थिति NONE हो तो अंक आवश्यक हैं।"))
		return
	}

	// For non-NONE status, marks must not be provided
	if payload.SpecialStatus != "NONE" && payload.MarksObtained != nil {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Marks must be null when special status is not NONE.",
			"जब विशेष स्थिति NONE नहीं हो तो अंक null होने चाहिए।"))
		return
	}

	// Verify teacher has access to this exam subject (if teacher)
	if identity.HasRole(auth.RoleTeacher) && identity.StaffID != "" {
		query := `
			SELECT 1 FROM exam_subjects es
			WHERE es.id = $1::uuid
			AND es.class_id IN (
				SELECT DISTINCT class_id FROM teaching_assignments
				WHERE staff_id = $2::uuid
			)
		`
		var exists int
		err := s.store.Pool().QueryRow(ctx, query, payload.ExamSubjectID, identity.StaffID).Scan(&exists)
		if err != nil {
			if errors.Is(err, store.ErrNotFound) {
				httpx.Fail(w, r, httpx.ErrForbidden())
				return
			}
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}
	}

	// Insert or update the mark
	insertQuery := `
		INSERT INTO marks (exam_subject_id, student_id, marks_obtained, special_status, entered_by, entered_at)
		VALUES ($1::uuid, $2::uuid, $3, $4, $5::uuid, now())
		ON CONFLICT (exam_subject_id, student_id)
		DO UPDATE SET
			marks_obtained = $3,
			special_status = $4,
			entered_by = $5::uuid,
			entered_at = now(),
			remark = $6
		RETURNING id, marks_obtained, special_status
	`

	var id string
	var marksObtained *float32
	var specialStatus string

	err := s.store.Pool().QueryRow(ctx, insertQuery,
		payload.ExamSubjectID,
		payload.StudentID,
		payload.MarksObtained,
		payload.SpecialStatus,
		identity.UserID,
		payload.Remark,
	).Scan(&id, &marksObtained, &specialStatus)

	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	response := map[string]interface{}{
		"id":            id,
		"marksObtained": marksObtained,
		"specialStatus": specialStatus,
	}

	httpx.JSON(w, http.StatusOK, response)
}

// AdmitCard represents an admit card for an exam.
type AdmitCard struct {
	ExamID          string  `json:"examId"`
	ExamName        string  `json:"examName"`
	StudentID       string  `json:"studentId"`
	StudentName     string  `json:"studentName"`
	ClassName       string  `json:"className"`
	RollNumber      *int    `json:"rollNumber"`
	ExamDate        string  `json:"examDate"`
	StartTime       string  `json:"startTime"`
	EndTime         string  `json:"endTime"`
	RoomNumber      *string `json:"roomNumber"`
	InvigilatorName *string `json:"invigilatorName"`
}

// handleGetAdmitCard fetches and downloads admit cards.
// Students can only download their own admit cards.
func (s *Server) handleGetAdmitCard(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	identity := auth.MustFromContext(r.Context())

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Only students can download admit cards (for themselves)
	if !identity.HasRole(auth.RoleStudent) || identity.StudentID == "" {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	examID := r.URL.Query().Get("examId")
	if examID != "" && !auth.IsUUID(examID) {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Invalid exam ID format.",
			"अमान्य परीक्षा ID प्रारूप।"))
		return
	}

	// Build query for admit cards
	query := `
		SELECT
			e.id,
			e.name_en,
			s.id,
			u.full_name_en,
			c.name_en,
			e.student_enrollment_number,
			es.exam_date,
			es.start_time,
			es.start_time + interval '3 hours' as end_time,
			NULL::text as room_number,
			inv.user_id
		FROM exams e
		JOIN exam_subjects es ON es.exam_id = e.id
		JOIN classes c ON c.id = es.class_id
		JOIN students s ON s.id = ANY(
			SELECT student_id FROM enrollments WHERE class_id = c.id
		)
		JOIN users u ON u.id = s.user_id
		LEFT JOIN invigilators inv ON inv.exam_subject_id = es.id
		WHERE e.session_id = $1::uuid
		AND s.id = $2::uuid
		AND e.status IN ('SCHEDULED', 'MARKS_OPEN', 'APPROVED', 'PUBLISHED')
		AND es.exam_date >= CURRENT_DATE
	`

	args := []interface{}{sessionID, identity.StudentID}

	if examID != "" {
		query += ` AND e.id = $3::uuid`
		args = append(args, examID)
	}

	query += ` ORDER BY es.exam_date ASC`

	rows, err := s.store.Pool().Query(ctx, query, args...)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	defer rows.Close()

	cards := []AdmitCard{}
	for rows.Next() {
		var card AdmitCard
		var examDate *time.Time
		var startTime *time.Time
		var endTime *time.Time
		var invigilatorID *string

		err := rows.Scan(
			&card.ExamID,
			&card.ExamName,
			&card.StudentID,
			&card.StudentName,
			&card.ClassName,
			&card.RollNumber,
			&examDate,
			&startTime,
			&endTime,
			&card.RoomNumber,
			&invigilatorID,
		)
		if err != nil {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}

		if examDate != nil {
			card.ExamDate = examDate.Format("2006-01-02")
		}
		if startTime != nil {
			card.StartTime = startTime.Format("15:04")
		}
		if endTime != nil {
			card.EndTime = endTime.Format("15:04")
		}

		// Fetch invigilator name if available
		if invigilatorID != nil {
			invQuery := `SELECT full_name_en FROM users WHERE id = $1::uuid`
			var invName string
			err := s.store.Pool().QueryRow(ctx, invQuery, *invigilatorID).Scan(&invName)
			if err == nil {
				card.InvigilatorName = &invName
			}
		}

		cards = append(cards, card)
	}

	if err = rows.Err(); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	if len(cards) == 0 {
		httpx.Fail(w, r, httpx.ErrNotFound())
		return
	}

	// Return admit cards as JSON
	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"admitCards": cards,
	})
}
