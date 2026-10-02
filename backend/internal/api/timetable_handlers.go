package api

import (
	"net/http"
	"strings"
	"time"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

func (s *Server) registerTimetableRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/timetable", s.signedIn(s.handleListTimetable))
	mux.Handle("POST /api/v1/timetable", s.restricted(s.handleCreateTimetableEntry, auth.RoleSuperAdmin, auth.RolePrincipal))
	mux.Handle("POST /api/v1/timetable/substitutions", s.restricted(s.handleCreateSubstitution,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice))
}

func (s *Server) handleListTimetable(w http.ResponseWriter, r *http.Request) {
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
	date := strings.TrimSpace(r.URL.Query().Get("date"))
	if date != "" {
		if _, err := time.Parse("2006-01-02", date); err != nil {
			httpx.Fail(w, r, httpx.ErrBadRequest("Use YYYY-MM-DD for the date.", "दिनांक के लिए YYYY-MM-DD दें।").WithField("date", "Invalid date"))
			return
		}
	}
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	var items []store.TimetableEntry
	switch {
	case identity.HasRole(auth.RoleStudent):
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListTimetableForStudent(ctx, sessionID, identity.StudentID, date)
	case identity.HasRole(auth.RoleParent):
		if identity.GuardianID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListTimetableForGuardian(ctx, sessionID, identity.GuardianID, date)
	case identity.HasRole(auth.RoleTeacher):
		if identity.StaffID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListTimetableForStaff(ctx, sessionID, identity.StaffID, date)
	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice):
		items, err = s.store.ListTimetableAll(ctx, sessionID, sectionID, date, 200)
	default:
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}
	if err != nil {
		if err == store.ErrNotFound {
			httpx.JSON(w, http.StatusOK, map[string]any{"items": []store.TimetableEntry{}})
			return
		}
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

type createTimetableEntryRequest struct {
	SectionID string `json:"sectionId"`
	DayOfWeek int    `json:"dayOfWeek"`
	PeriodID  string `json:"periodId"`
	SubjectID string `json:"subjectId"`
	StaffID   string `json:"staffId"`
	Room      string `json:"room"`
}

func (s *Server) handleCreateTimetableEntry(w http.ResponseWriter, r *http.Request) {
	var input createTimetableEntryRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if !auth.IsUUID(input.SectionID) || !auth.IsUUID(input.PeriodID) || !auth.IsUUID(input.SubjectID) || !auth.IsUUID(input.StaffID) || input.DayOfWeek < 1 || input.DayOfWeek > 6 || len(input.Room) > 100 {
		httpx.Fail(w, r, httpx.ErrBadRequest("Provide a section, Monday-to-Saturday day, period, subject, and active teacher.", "सेक्शन, सोमवार-शनिवार का दिन, कालांश, विषय और सक्रिय शिक्षक दें।"))
		return
	}
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	id, err := s.store.CreateTimetableEntry(ctx, store.NewTimetableEntry{
		SessionID: sessionID, SectionID: input.SectionID, DayOfWeek: input.DayOfWeek,
		PeriodID: input.PeriodID, SubjectID: input.SubjectID, StaffID: input.StaffID, Room: strings.TrimSpace(input.Room),
	})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionCreate, EntityType: "timetable_entry", EntityID: id, Summary: "Created timetable entry", AfterState: input})
	httpx.JSON(w, http.StatusCreated, map[string]string{"id": id})
}

type createSubstitutionRequest struct {
	Date              string `json:"date"`
	TimetableEntryID  string `json:"timetableEntryId"`
	SubstituteStaffID string `json:"substituteStaffId"`
	Reason            string `json:"reason"`
}

func (s *Server) handleCreateSubstitution(w http.ResponseWriter, r *http.Request) {
	var input createSubstitutionRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if _, err := time.Parse("2006-01-02", input.Date); err != nil || !auth.IsUUID(input.TimetableEntryID) || !auth.IsUUID(input.SubstituteStaffID) || len(input.Reason) > 500 {
		httpx.Fail(w, r, httpx.ErrBadRequest("Provide a valid date, timetable lesson, active substitute, and short reason.", "मान्य दिनांक, समय-सारणी पाठ, सक्रिय विकल्प और संक्षिप्त कारण दें।"))
		return
	}
	identity := auth.MustFromContext(r.Context())
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	id, err := s.store.CreateSubstitution(ctx, store.NewSubstitution{
		Date: input.Date, TimetableEntryID: input.TimetableEntryID, SubstituteStaffID: input.SubstituteStaffID,
		Reason: strings.TrimSpace(input.Reason), CreatedBy: identity.UserID,
	})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionCreate, EntityType: "substitution", EntityID: id, Summary: "Created timetable substitution", AfterState: input})
	httpx.JSON(w, http.StatusCreated, map[string]string{"id": id})
}
