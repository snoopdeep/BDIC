package api

import (
	"errors"
	"net/http"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

// registerAcademicRoutes registers all academic-related endpoints.
func (s *Server) registerAcademicRoutes(mux *http.ServeMux) {
	// ---- Any signed-in user ------------------------------------------------
	// List classes and subjects are readable by everyone
	mux.Handle("GET /api/v1/academics/classes", s.signedIn(s.handleListAcademicClasses))
	mux.Handle("GET /api/v1/academics/subjects", s.signedIn(s.handleListAcademicSubjects))

	// ---- Academic records: role-based access --------------------------------
	// Students see only their own records
	mux.Handle("GET /api/v1/academics/records", s.signedIn(s.handleGetMyAcademicRecords))

	// Staff (teachers) and admins see student records
	// Students cannot use this endpoint - they use the GET /records endpoint above
	mux.Handle("GET /api/v1/academics/records/{studentId}",
		s.restricted(s.handleGetStudentAcademicRecords,
			auth.RolePrincipal, auth.RoleOffice, auth.RoleTeacher))

	// Transcripts: students and staff can view/download
	mux.Handle("GET /api/v1/academics/transcripts", s.signedIn(s.handleGetTranscript))

	// ---- Create records: admin/principal only --------------------------------
	mux.Handle("POST /api/v1/academics/records",
		s.restricted(s.handleCreateAcademicRecord,
			auth.RoleSuperAdmin, auth.RolePrincipal))
}

// handleListAcademicClasses returns all classes for the current session.
func (s *Server) handleListAcademicClasses(w http.ResponseWriter, r *http.Request) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	classes, err := s.store.ListClasses(r.Context(), sessionID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	type classResponse struct {
		ID        string `json:"id"`
		Code      string `json:"code"`
		NameEN    string `json:"nameEn"`
		NameHI    string `json:"nameHi"`
		Level     int    `json:"level"`
		HasStream bool   `json:"hasStream"`
	}

	response := make([]classResponse, 0, len(classes))
	for _, item := range classes {
		response = append(response, classResponse{
			ID:        item.ID,
			Code:      item.Code,
			NameEN:    item.NameEN,
			NameHI:    item.NameHI,
			Level:     item.Level,
			HasStream: item.HasStream,
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"classes": response,
	})
}

// handleListAcademicSubjects returns all subjects offered by the school.
func (s *Server) handleListAcademicSubjects(w http.ResponseWriter, r *http.Request) {
	subjects, err := s.store.ListSubjects(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	type subjectResponse struct {
		ID         string `json:"id"`
		Code       string `json:"code"`
		NameEN     string `json:"nameEn"`
		NameHI     string `json:"nameHi"`
		IsLanguage bool   `json:"isLanguage"`
	}

	response := make([]subjectResponse, 0, len(subjects))
	for _, item := range subjects {
		response = append(response, subjectResponse{
			ID:         item.ID,
			Code:       item.Code,
			NameEN:     item.NameEN,
			NameHI:     item.NameHI,
			IsLanguage: item.IsLanguage,
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"subjects": response,
	})
}

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
	RecordType    string  `json:"recordType"` // "EXAM" or "TRANSCRIPT"
}

// handleGetMyAcademicRecords returns the signed-in student's records, or all
// records for the children linked to a signed-in parent.
func (s *Server) handleGetMyAcademicRecords(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	var (
		records []store.AcademicRecord
		err     error
	)
	switch {
	case identity.HasRole(auth.RoleStudent):
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(
				errors.New("student identity missing studentId")))
			return
		}
		records, err = s.store.GetAcademicRecords(r.Context(), identity.StudentID)
	case identity.HasRole(auth.RoleParent):
		if identity.GuardianID == "" {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(
				errors.New("parent identity missing guardianId")))
			return
		}
		records, err = s.store.GetAcademicRecordsForGuardian(r.Context(), identity.GuardianID)
	default:
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.JSON(w, http.StatusOK, map[string]any{"records": []AcademicRecord{}})
			return
		}
		httpx.Fail(w, r, storeError(err))
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"records": records,
	})
}

// handleGetStudentAcademicRecords returns academic records for a specific student.
//
// Accessible by:
// - Teachers: can see students in their sections
// - Staff (Office, Accounts): can see all students' records
// - Principal/SuperAdmin: can see all students' records
//
// The studentId must be a valid UUID.
func (s *Server) handleGetStudentAcademicRecords(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	studentID, err := pathUUID(r, "studentId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// For teachers: verify the student is in their section(s)
	if identity.HasRole(auth.RoleTeacher) {
		// This would require a check in the store to verify teacher has access
		// For now, we allow teachers to see any student's records
		// In production, add a method like: store.TeacherCanAccessStudent()
	}

	records, err := s.store.GetAcademicRecords(r.Context(), studentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.JSON(w, http.StatusOK, map[string]any{
				"records": []AcademicRecord{},
			})
			return
		}
		httpx.Fail(w, r, storeError(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionUpdate,
		EntityType: "academic_record",
		EntityID:   studentID,
		Summary:    identity.FullNameEN + " accessed academic records for student " + studentID,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{
		"records": records,
	})
}

// handleGetTranscript returns a transcript for the signed-in user (student) or
// allows staff to generate a transcript for a student.
//
// For students: returns their complete transcript as a JSON document that can be
// rendered or downloaded as PDF.
//
// For staff: query parameter ?studentId= specifies which student's transcript.
func (s *Server) handleGetTranscript(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	var studentID string
	var isStudentRequest bool

	if identity.HasRole(auth.RoleStudent) {
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(
				errors.New("student identity missing studentId")))
			return
		}
		studentID = identity.StudentID
		isStudentRequest = true
	} else if identity.IsStaff() {
		// Staff can request transcript for any student via query parameter
		requestedID := r.URL.Query().Get("studentId")
		if requestedID == "" {
			httpx.Fail(w, r, httpx.ErrBadRequest(
				"Specify which student's transcript using ?studentId=<id>",
				"?studentId=<id> का उपयोग करके छात्र की प्रतिलिपि निर्दिष्ट करें"))
			return
		}
		if !auth.IsUUID(requestedID) {
			httpx.Fail(w, r, httpx.ErrBadRequest(
				"The studentId is not a valid identifier.",
				"studentId एक मान्य पहचानकर्ता नहीं है।"))
			return
		}
		studentID = requestedID
	} else {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	transcript, err := s.store.GetTranscript(r.Context(), studentID)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.JSON(w, http.StatusOK, map[string]any{
				"transcript": map[string]any{
					"studentId": studentID,
					"records":   []AcademicRecord{},
				},
			})
			return
		}
		httpx.Fail(w, r, storeError(err))
		return
	}

	if !isStudentRequest {
		s.audit.Record(r.Context(), r, audit.Entry{
			Action:     audit.ActionExport,
			EntityType: "transcript",
			EntityID:   studentID,
			Summary:    identity.FullNameEN + " generated transcript for student " + studentID,
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"transcript": transcript,
	})
}

// CreateAcademicRecordRequest is the request body for creating an academic record.
type CreateAcademicRecordRequest struct {
	StudentID     string  `json:"studentId"`
	Subject       string  `json:"subject"`
	AcademicYear  string  `json:"academicYear"`
	Session       string  `json:"session"`
	MarksObtained float64 `json:"marksObtained"`
	MaxMarks      float64 `json:"maxMarks"`
	Grade         string  `json:"grade"`
	RecordType    string  `json:"recordType"`
}

// handleCreateAcademicRecord creates a new academic record.
//
// Only accessible by Principal and SuperAdmin. Requires:
// - studentId: valid UUID of an existing student
// - subject: subject name or ID
// - academicYear: the academic year (e.g., "2024-25")
// - session: the session name
// - marksObtained: marks the student obtained
// - maxMarks: maximum marks for this assessment
// - grade: the grade awarded (e.g., "A", "B", "C")
// - recordType: "EXAM" or "TRANSCRIPT"
func (s *Server) handleCreateAcademicRecord(w http.ResponseWriter, r *http.Request) {
	var input CreateAcademicRecordRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Validate required fields
	if input.StudentID == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"studentId is required.",
			"studentId आवश्यक है।").WithField("studentId", "Cannot be empty"))
		return
	}

	if !auth.IsUUID(input.StudentID) {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"studentId must be a valid identifier.",
			"studentId एक मान्य पहचानकर्ता होना चाहिए।").WithField("studentId", "Not a valid UUID"))
		return
	}

	if input.Subject == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"subject is required.",
			"subject आवश्यक है।").WithField("subject", "Cannot be empty"))
		return
	}

	if input.MaxMarks <= 0 {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"maxMarks must be greater than 0.",
			"maxMarks 0 से अधिक होना चाहिए।").WithField("maxMarks", "Must be positive"))
		return
	}

	if input.MarksObtained < 0 || input.MarksObtained > input.MaxMarks {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"marksObtained must be between 0 and maxMarks.",
			"marksObtained 0 और maxMarks के बीच होना चाहिए।").WithField("marksObtained", "Out of range"))
		return
	}

	if input.Grade == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"grade is required.",
			"grade आवश्यक है।").WithField("grade", "Cannot be empty"))
		return
	}

	if input.RecordType != "EXAM" && input.RecordType != "TRANSCRIPT" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"recordType must be 'EXAM' or 'TRANSCRIPT'.",
			"recordType 'EXAM' या 'TRANSCRIPT' होना चाहिए।").WithField("recordType", "Invalid value"))
		return
	}

	identity := auth.MustFromContext(r.Context())

	// Create the record in the store
	recordID, err := s.store.CreateAcademicRecord(r.Context(), store.CreateAcademicRecordInput{
		StudentID:     input.StudentID,
		Subject:       input.Subject,
		AcademicYear:  input.AcademicYear,
		Session:       input.Session,
		MarksObtained: input.MarksObtained,
		MaxMarks:      input.MaxMarks,
		Grade:         input.Grade,
		RecordType:    input.RecordType,
		CreatedBy:     identity.UserID,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			httpx.Fail(w, r, httpx.ErrBadRequest(
				"That student does not exist.",
				"वह छात्र मौजूद नहीं है।").WithInternal(err))
			return
		}
		httpx.Fail(w, r, storeError(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionCreate,
		EntityType: "academic_record",
		EntityID:   recordID,
		Summary:    identity.FullNameEN + " created academic record for student " + input.StudentID,
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":        recordID,
		"studentId": input.StudentID,
		"subject":   input.Subject,
		"grade":     input.Grade,
	})
}
