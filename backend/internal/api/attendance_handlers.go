package api

import (
	"context"
	"net/http"
	"strings"
	"time"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

func (s *Server) registerAttendanceRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/attendance", s.signedIn(s.handleListAttendance))
	mux.Handle("GET /api/v1/attendance/sections/{sectionId}/roster", s.restricted(s.handleAttendanceRoster,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice, auth.RoleTeacher))
	mux.Handle("PUT /api/v1/attendance/sections/{sectionId}", s.restricted(s.handleSaveAttendance,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice, auth.RoleTeacher))
}

func (s *Server) handleListAttendance(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	sectionID, err := queryUUID(r, "sectionId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	var items []store.AttendanceRecord
	switch {
	case identity.HasRole(auth.RoleStudent):
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListAttendanceForStudent(ctx, sessionID, identity.StudentID, limit, offset)
	case identity.HasRole(auth.RoleParent):
		if identity.GuardianID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListAttendanceForGuardian(ctx, sessionID, identity.GuardianID, limit, offset)
	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice):
		items, err = s.store.ListAttendanceAll(ctx, sessionID, sectionID, limit, offset)
	case identity.HasRole(auth.RoleTeacher):
		if identity.StaffID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		// The normal teacher view has no filter and therefore contains only
		// assigned sections. An optional section filter remains safe because the
		// store scopes it with the same assignment checks.
		items, err = s.store.ListAttendanceForTeacher(ctx, sessionID, identity.StaffID, sectionID, limit, offset)
	default:
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items), "limit": limit, "offset": offset})
}

func (s *Server) attendanceAuthorised(ctx context.Context, sessionID, sectionID string, identity auth.Identity) (bool, error) {
	if identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice) {
		return true, nil
	}
	if !identity.HasRole(auth.RoleTeacher) || identity.StaffID == "" {
		return false, nil
	}
	return s.store.TeacherMayTakeAttendance(ctx, sessionID, sectionID, identity.StaffID)
}

func (s *Server) handleAttendanceRoster(w http.ResponseWriter, r *http.Request) {
	sectionID, err := pathUUID(r, "sectionId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	date, err := schoolDate(r.URL.Query().Get("date"))
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	if allowed, err := s.attendanceAuthorised(ctx, sessionID, sectionID, auth.MustFromContext(r.Context())); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	} else if !allowed {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}
	roster, err := s.store.ListAttendanceRoster(ctx, sessionID, sectionID, date)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"sectionId": sectionID, "date": date, "items": roster})
}

type saveAttendanceRequest struct {
	Date    string `json:"date"`
	Entries []struct {
		StudentID string `json:"studentId"`
		Status    string `json:"status"`
		Note      string `json:"note"`
	} `json:"entries"`
}

func (s *Server) handleSaveAttendance(w http.ResponseWriter, r *http.Request) {
	sectionID, err := pathUUID(r, "sectionId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	var input saveAttendanceRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	date, err := schoolDate(input.Date)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	today := time.Now().In(time.FixedZone("IST", 5*60*60+30*60)).Format("2006-01-02")
	if date != today {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Past attendance must be changed through a correction request; future attendance cannot be marked.",
			"पुरानी उपस्थिति सुधार अनुरोध से बदली जाती है; भविष्य की उपस्थिति दर्ज नहीं की जा सकती।"))
		return
	}
	if len(input.Entries) == 0 || len(input.Entries) > 200 {
		httpx.Fail(w, r, httpx.ErrBadRequest("Add every student in this section to the register.", "इस सेक्शन के सभी विद्यार्थियों को रजिस्टर में जोड़ें।").WithField("entries", "Required"))
		return
	}

	validStatuses := map[string]bool{"PRESENT": true, "ABSENT": true, "LATE": true, "LEAVE": true, "MEDICAL": true, "HALF_DAY": true}
	seen := make(map[string]bool, len(input.Entries))
	entries := make([]store.AttendanceEntryInput, 0, len(input.Entries))
	for _, entry := range input.Entries {
		entry.StudentID = strings.TrimSpace(entry.StudentID)
		entry.Status = strings.ToUpper(strings.TrimSpace(entry.Status))
		if !auth.IsUUID(entry.StudentID) || seen[entry.StudentID] || !validStatuses[entry.Status] || len(entry.Note) > 500 {
			httpx.Fail(w, r, httpx.ErrBadRequest("Every entry needs a unique student and a valid attendance status.", "हर प्रविष्टि में अलग विद्यार्थी और मान्य उपस्थिति स्थिति चाहिए।").WithField("entries", "Invalid entry"))
			return
		}
		seen[entry.StudentID] = true
		entries = append(entries, store.AttendanceEntryInput{StudentID: entry.StudentID, Status: entry.Status, Note: strings.TrimSpace(entry.Note)})
	}

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	identity := auth.MustFromContext(r.Context())
	ctx, cancel := contextWithTimeout(r, 15*time.Second)
	defer cancel()
	if allowed, err := s.attendanceAuthorised(ctx, sessionID, sectionID, identity); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	} else if !allowed {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}
	if err := s.store.SaveSameDayAttendance(ctx, sessionID, sectionID, date, identity.UserID, entries); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{
		Action: audit.ActionUpdate, EntityType: "attendance_submission", EntityID: sectionID,
		Summary:    "Saved attendance for " + date,
		AfterState: map[string]any{"sectionId": sectionID, "date": date, "students": len(entries)},
	})
	httpx.JSON(w, http.StatusOK, map[string]any{"sectionId": sectionID, "date": date, "saved": len(entries)})
}

func schoolDate(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Now().In(time.FixedZone("IST", 5*60*60+30*60)).Format("2006-01-02"), nil
	}
	date, err := time.Parse("2006-01-02", value)
	if err != nil {
		return "", httpx.ErrBadRequest("Use YYYY-MM-DD for the date.", "दिनांक के लिए YYYY-MM-DD दें।").WithField("date", "Invalid date")
	}
	return date.Format("2006-01-02"), nil
}
