package api

import (
	"net/http"
	"time"

	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
)

// registerDashboardRoutes registers all dashboard endpoints. These are
// role-appropriate summaries shown when a user signs in.
func (s *Server) registerDashboardRoutes(mux *http.ServeMux) {
	// Dashboard endpoints require authentication but are open to all roles,
	// each seeing only data appropriate to them.
	mux.Handle("GET /api/v1/dashboard/summary",
		s.signedIn(s.handleDashboardSummary))
	mux.Handle("GET /api/v1/dashboard/announcements",
		s.signedIn(s.handleDashboardAnnouncements))
	mux.Handle("GET /api/v1/dashboard/quick-links",
		s.signedIn(s.handleDashboardQuickLinks))
}

// handleDashboardSummary returns role-appropriate summary statistics.
// - Students: next class, attendance percentage, fees due, recent marks
// - Parents: their children's class, attendance, fees status
// - Teachers: their classes, pending assignments, today's schedule
// - Admin/Principal: enrollment stats, pending approvals, recent activity
func (s *Server) handleDashboardSummary(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	switch {
	case identity.HasRole(auth.RoleStudent):
		s.handleStudentDashboardSummary(w, r, identity)
	case identity.HasRole(auth.RoleParent):
		s.handleParentDashboardSummary(w, r, identity)
	case identity.HasRole(auth.RoleTeacher):
		s.handleTeacherDashboardSummary(w, r, identity)
	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice, auth.RoleAccounts):
		s.handleAdminDashboardSummary(w, r, identity)
	default:
		httpx.Fail(w, r, httpx.ErrUnauthorized())
	}
}

// handleStudentDashboardSummary returns next class, attendance %, fees due, recent marks
func (s *Server) handleStudentDashboardSummary(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Get student enrollment for this session
	enrollment, err := s.store.GetStudentEnrollmentV2(r.Context(), identity.StudentID, sessionID)
	if err != nil {
		// Student may not be enrolled this session
		httpx.JSON(w, http.StatusOK, map[string]any{
			"role": "STUDENT",
			"data": map[string]any{
				"nextClass":         nil,
				"attendancePercent": 0,
				"feesDueAmount":     0,
				"recentMarks":       []any{},
			},
		})
		return
	}

	// Get next class on the timetable today
	today := time.Now().Weekday()
	dayOfWeek := int(today) // 0 = Sunday, 1 = Monday, etc.; we use 1-6 for Mon-Sat
	if dayOfWeek == 0 {
		dayOfWeek = 7 // Treat Sunday as 7 (no school)
	}

	sectionID, _ := enrollment["section_id"].(string)

	var nextClass map[string]any
	timetableEntry, err := s.store.GetNextTimetableEntry(r.Context(), sessionID, sectionID, dayOfWeek, time.Now())
	if err == nil && timetableEntry != nil {
		nextClass = map[string]any{
			"subjectName": timetableEntry["subject_name"],
			"startTime":   timetableEntry["start_time"],
			"endTime":     timetableEntry["end_time"],
			"room":        timetableEntry["room"],
			"teacherName": timetableEntry["teacher_name"],
		}
	}

	// Get attendance percentage for this session
	attendanceStats, _ := s.store.GetStudentAttendanceStats(r.Context(), identity.StudentID, sessionID)
	attendancePercent := 0.0
	if attendanceStats != nil {
		if total, ok := attendanceStats["total_days"].(int); ok && total > 0 {
			if present, ok := attendanceStats["present_days"].(int); ok {
				attendancePercent = float64(present) / float64(total) * 100
			}
		}
	}

	// Get outstanding fees (invoices with status DUE or PARTIAL)
	var feesDueAmount int64
	outstanding, _ := s.store.GetStudentOutstandingFees(r.Context(), identity.StudentID, sessionID)
	if outstanding != nil {
		if amount, ok := outstanding["due_amount_paise"].(int64); ok {
			feesDueAmount = amount
		}
	}

	// Get recent marks from the latest exam
	var recentMarks []map[string]any
	marks, _ := s.store.GetStudentRecentMarks(r.Context(), identity.StudentID, sessionID, 5)
	if marks != nil {
		for _, mark := range marks {
			recentMarks = append(recentMarks, map[string]any{
				"examName":      mark["exam_name"],
				"subjectName":   mark["subject_name"],
				"marksObtained": mark["marks_obtained"],
				"maxMarks":      mark["max_marks"],
				"percentage":    mark["percentage"],
				"grade":         mark["grade"],
			})
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"role": "STUDENT",
		"data": map[string]any{
			"nextClass":         nextClass,
			"attendancePercent": attendancePercent,
			"feesDueAmount":     feesDueAmount, // In paise
			"recentMarks":       recentMarks,
		},
	})
}

// handleParentDashboardSummary returns children's class, attendance, fees status
func (s *Server) handleParentDashboardSummary(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Get all children linked to this parent
	children, err := s.store.GetGuardianChildren(r.Context(), identity.GuardianID)
	if err != nil {
		children = []map[string]any{} // No children found
	}

	var childrenSummary []map[string]any

	for _, child := range children {
		childID, ok := child["student_id"].(string)
		if !ok {
			continue
		}

		childName, _ := child["full_name"].(string)

		// Get enrollment for this session
		enrollment, err := s.store.GetStudentEnrollmentV2(r.Context(), childID, sessionID)
		if err != nil {
			continue // Skip if not enrolled
		}

		// Get class info
		classID, _ := enrollment["class_id"].(string)
		classInfo, _ := s.store.GetClassInfo(r.Context(), classID)

		// Get attendance percentage
		attendanceStats, _ := s.store.GetStudentAttendanceStats(r.Context(), childID, sessionID)
		attendancePercent := 0.0
		if attendanceStats != nil {
			if total, ok := attendanceStats["total_days"].(int); ok && total > 0 {
				if present, ok := attendanceStats["present_days"].(int); ok {
					attendancePercent = float64(present) / float64(total) * 100
				}
			}
		}

		// Get fees status
		outstanding, _ := s.store.GetStudentOutstandingFees(r.Context(), childID, sessionID)
		feesStatus := "PAID"
		feesDue := int64(0)
		if outstanding != nil {
			if amount, ok := outstanding["due_amount_paise"].(int64); ok && amount > 0 {
				feesStatus = "PENDING"
				feesDue = amount
			}
		}

		className := ""
		if classInfo != nil {
			className, _ = classInfo["class_name"].(string)
		}

		childrenSummary = append(childrenSummary, map[string]any{
			"childName":         childName,
			"class":             className,
			"attendancePercent": attendancePercent,
			"feesStatus":        feesStatus,
			"feesDueAmount":     feesDue, // In paise
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"role": "PARENT",
		"data": map[string]any{
			"children": childrenSummary,
		},
	})
}

// handleTeacherDashboardSummary returns their classes, pending assignments, today's schedule
func (s *Server) handleTeacherDashboardSummary(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Get sections taught by this teacher this session
	sections, err := s.store.GetTeacherSections(r.Context(), identity.StaffID, sessionID)
	if err != nil {
		sections = []map[string]any{} // No sections assigned
	}

	var classesInfo []map[string]any
	for _, section := range sections {
		_, ok := section["section_id"].(string)
		if !ok {
			continue
		}

		sectionName, _ := section["section_name"].(string)
		className, _ := section["class_name"].(string)
		enrollmentCount, _ := section["enrollment_count"].(int)

		classesInfo = append(classesInfo, map[string]any{
			"className":       className,
			"sectionName":     sectionName,
			"enrollmentCount": enrollmentCount,
		})
	}

	// Get pending assignments (due in future or recently due)
	pendingAssignments, _ := s.store.GetTeacherPendingAssignments(r.Context(), identity.StaffID, sessionID)
	var assignments []map[string]any
	if pendingAssignments != nil {
		for _, hw := range pendingAssignments {
			title, _ := hw["title"].(string)
			sectionName, _ := hw["section_name"].(string)
			dueDate, _ := hw["due_date"].(string)
			subjectName, _ := hw["subject_name"].(string)

			assignments = append(assignments, map[string]any{
				"title":   title,
				"subject": subjectName,
				"section": sectionName,
				"dueDate": dueDate,
			})
		}
	}

	// Get today's schedule
	today := time.Now().Weekday()
	dayOfWeek := int(today)
	if dayOfWeek == 0 {
		dayOfWeek = 7 // Sunday, no school
	}

	todaySchedule, _ := s.store.GetTeacherTodaySchedule(r.Context(), identity.StaffID, sessionID, dayOfWeek)
	var schedule []map[string]any
	if todaySchedule != nil {
		for _, period := range todaySchedule {
			periodName, _ := period["period_name"].(string)
			sectionName, _ := period["section_name"].(string)
			subjectName, _ := period["subject_name"].(string)
			startTime, _ := period["start_time"].(string)
			endTime, _ := period["end_time"].(string)
			room, _ := period["room"].(string)

			schedule = append(schedule, map[string]any{
				"period":    periodName,
				"section":   sectionName,
				"subject":   subjectName,
				"startTime": startTime,
				"endTime":   endTime,
				"room":      room,
			})
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"role": "TEACHER",
		"data": map[string]any{
			"classes":            classesInfo,
			"pendingAssignments": assignments,
			"todaySchedule":      schedule,
		},
	})
}

// handleAdminDashboardSummary returns enrollment stats, pending approvals, recent activity
func (s *Server) handleAdminDashboardSummary(w http.ResponseWriter, r *http.Request, identity auth.Identity) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Get enrollment statistics
	enrollmentStats, _ := s.store.GetEnrollmentStats(r.Context(), sessionID)
	totalEnrolled := 0
	classCount := 0
	if enrollmentStats != nil {
		if total, ok := enrollmentStats["total_enrolled"].(int); ok {
			totalEnrolled = total
		}
		if classes, ok := enrollmentStats["class_count"].(int); ok {
			classCount = classes
		}
	}

	// Get pending approvals (student concessions, attendance corrections, etc.)
	pendingConcessions, _ := s.store.GetPendingConcessions(r.Context(), sessionID)
	pendingAttendanceCorrections, _ := s.store.GetPendingAttendanceCorrections(r.Context())
	totalPending := 0
	if pendingConcessions != nil {
		if count, ok := pendingConcessions["pending_count"].(int); ok {
			totalPending += count
		}
	}
	if pendingAttendanceCorrections != nil {
		if count, ok := pendingAttendanceCorrections["pending_count"].(int); ok {
			totalPending += count
		}
	}

	// Get recent activity (last 5 actions from audit log)
	recentActivity, _ := s.store.GetRecentActivity(r.Context(), 5)
	var activity []map[string]any
	if recentActivity != nil {
		for _, log := range recentActivity {
			entityType, _ := log["entity_type"].(string)
			action, _ := log["action"].(string)
			summary, _ := log["summary"].(string)
			timestamp, _ := log["timestamp"].(string)

			activity = append(activity, map[string]any{
				"entityType": entityType,
				"action":     action,
				"summary":    summary,
				"timestamp":  timestamp,
			})
		}
	}

	// Get outstanding fees across all students
	totalFeesPending, _ := s.store.GetTotalOutstandingFees(r.Context(), sessionID)
	outstandingFees := int64(0)
	if totalFeesPending != nil {
		if amount, ok := totalFeesPending["total_paise"].(int64); ok {
			outstandingFees = amount
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"role": "ADMIN",
		"data": map[string]any{
			"enrollmentStats": map[string]any{
				"totalEnrolled": totalEnrolled,
				"classCount":    classCount,
			},
			"pendingApprovals": totalPending,
			"outstandingFees":  outstandingFees, // In paise
			"recentActivity":   activity,
		},
	})
}

// handleDashboardAnnouncements returns announcements relevant to the signed-in user
func (s *Server) handleDashboardAnnouncements(w http.ResponseWriter, r *http.Request) {
	_ = auth.MustFromContext(r.Context()) // Ensure authentication, but announcements are global
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Get published notices that are not yet expired
	announcements, err := s.store.GetPublishedNotices(r.Context(), sessionID)
	if err != nil {
		announcements = []map[string]any{}
	}

	var filtered []map[string]any

	for _, notice := range announcements {
		// Filter announcements by relevance to this user's role
		// For now, return all published announcements visible to the user's role
		titleEN, _ := notice["title_en"].(string)
		titleHI, _ := notice["title_hi"].(string)
		bodyEN, _ := notice["body_en"].(string)
		bodyHI, _ := notice["body_hi"].(string)
		publishedAt, _ := notice["published_at"].(string)

		filtered = append(filtered, map[string]any{
			"titleEn":     titleEN,
			"titleHi":     titleHI,
			"bodyEn":      bodyEN,
			"bodyHi":      bodyHI,
			"publishedAt": publishedAt,
		})
	}

	// Limit to 10 most recent
	if len(filtered) > 10 {
		filtered = filtered[:10]
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"announcements": filtered,
	})
}

// handleDashboardQuickLinks returns role-appropriate quick links
func (s *Server) handleDashboardQuickLinks(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	var quickLinks []map[string]any

	switch {
	case identity.HasRole(auth.RoleStudent):
		quickLinks = []map[string]any{
			{
				"label":   "My Attendance",
				"labelHi": "मेरी हाजिरी",
				"path":    "/attendance",
				"icon":    "check-circle",
			},
			{
				"label":   "My Marks",
				"labelHi": "मेरे अंक",
				"path":    "/marks",
				"icon":    "book",
			},
			{
				"label":   "Fees Status",
				"labelHi": "शुल्क स्थिति",
				"path":    "/fees",
				"icon":    "money",
			},
			{
				"label":   "Assignments",
				"labelHi": "असाइनमेंट",
				"path":    "/assignments",
				"icon":    "tasks",
			},
			{
				"label":   "Timetable",
				"labelHi": "समय सारणी",
				"path":    "/timetable",
				"icon":    "calendar",
			},
		}

	case identity.HasRole(auth.RoleParent):
		quickLinks = []map[string]any{
			{
				"label":   "My Children",
				"labelHi": "मेरे बच्चे",
				"path":    "/children",
				"icon":    "users",
			},
			{
				"label":   "Attendance",
				"labelHi": "हाजिरी",
				"path":    "/attendance",
				"icon":    "check-circle",
			},
			{
				"label":   "Fees & Receipts",
				"labelHi": "शुल्क और रसीदें",
				"path":    "/fees",
				"icon":    "money",
			},
			{
				"label":   "Messages",
				"labelHi": "संदेश",
				"path":    "/messages",
				"icon":    "envelope",
			},
		}

	case identity.HasRole(auth.RoleTeacher):
		quickLinks = []map[string]any{
			{
				"label":   "My Classes",
				"labelHi": "मेरी कक्षाएं",
				"path":    "/classes",
				"icon":    "book",
			},
			{
				"label":   "Attendance",
				"labelHi": "हाजिरी",
				"path":    "/attendance",
				"icon":    "check-circle",
			},
			{
				"label":   "Marks Entry",
				"labelHi": "अंक प्रविष्टि",
				"path":    "/marks",
				"icon":    "pencil",
			},
			{
				"label":   "Assignments",
				"labelHi": "असाइनमेंट",
				"path":    "/assignments",
				"icon":    "tasks",
			},
			{
				"label":   "Timetable",
				"labelHi": "समय सारणी",
				"path":    "/timetable",
				"icon":    "calendar",
			},
		}

	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice, auth.RoleAccounts):
		quickLinks = []map[string]any{
			{
				"label":   "Students",
				"labelHi": "छात्र",
				"path":    "/students",
				"icon":    "users",
			},
			{
				"label":   "Staff",
				"labelHi": "कर्मचारी",
				"path":    "/staff",
				"icon":    "people",
			},
			{
				"label":   "Classes",
				"labelHi": "कक्षाएं",
				"path":    "/classes",
				"icon":    "book",
			},
			{
				"label":   "Fees Management",
				"labelHi": "शुल्क प्रबंधन",
				"path":    "/fees-management",
				"icon":    "money",
			},
			{
				"label":   "Announcements",
				"labelHi": "घोषणाएं",
				"path":    "/announcements",
				"icon":    "megaphone",
			},
			{
				"label":   "Reports",
				"labelHi": "रिपोर्ट",
				"path":    "/reports",
				"icon":    "chart",
			},
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"quickLinks": quickLinks,
	})
}
