package api

import (
	"encoding/json"
	"net/http"
	"strings"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
)

// handlePublicSchool serves the school's identity to the public website.
//
// Only the details the school wants on a signboard: name, address, phone,
// office hours, social links. Nothing about any person.
func (s *Server) handlePublicSchool(w http.ResponseWriter, r *http.Request) {
	school, err := s.store.GetSchool(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	session, sessionErr := s.store.CurrentSession(r.Context())
	sessionName := ""
	if sessionErr == nil {
		sessionName = session.Name
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"nameEn":         school.NameEN,
		"nameHi":         school.NameHI,
		"shortName":      school.ShortName,
		"board":          school.Board,
		"affiliationNo":  school.AffiliationNo,
		"udiseCode":      school.UDISECode,
		"addressEn":      school.AddressEN,
		"addressHi":      school.AddressHI,
		"village":        school.Village,
		"district":       school.District,
		"state":          school.State,
		"pincode":        school.Pincode,
		"phonePrimary":   school.PhonePrimary,
		"phoneSecondary": school.PhoneSecondary,
		"email":          school.Email,
		"officeHoursEn":  school.OfficeHoursEN,
		"officeHoursHi":  school.OfficeHoursHI,
		"principalName":  school.PrincipalName,
		"mapEmbedUrl":    school.MapEmbedURL,
		"instagramUrl":   school.InstagramURL,
		"facebookUrl":    school.FacebookURL,
		"googlePlaceUrl": school.GooglePlaceURL,
		"academicYear":   sessionName,
	})
}

// handlePublicStructure serves the classes, streams, and subjects to the public
// Academics and Admissions pages.
//
// Section head counts are stripped: how full a class is, is the school's
// business, not a visitor's.
func (s *Server) handlePublicStructure(w http.ResponseWriter, r *http.Request) {
	session, err := s.store.CurrentSession(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	classes, err := s.store.ListClasses(r.Context(), session.ID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	streams, err := s.store.ListStreams(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	subjects, err := s.store.ListSubjects(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	type publicClass struct {
		ID        string `json:"id"`
		Code      string `json:"code"`
		NameEN    string `json:"nameEn"`
		NameHI    string `json:"nameHi"`
		Level     int    `json:"level"`
		HasStream bool   `json:"hasStream"`
	}

	publicClasses := make([]publicClass, 0, len(classes))
	for _, item := range classes {
		publicClasses = append(publicClasses, publicClass{
			ID:        item.ID,
			Code:      item.Code,
			NameEN:    item.NameEN,
			NameHI:    item.NameHI,
			Level:     item.Level,
			HasStream: item.HasStream,
		})
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"academicYear": session.Name,
		"classes":      publicClasses,
		"streams":      streams,
		"subjects":     subjects,
	})
}

// handleGetSchool returns the full school record to a signed-in user.
func (s *Server) handleGetSchool(w http.ResponseWriter, r *http.Request) {
	school, err := s.store.GetSchool(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, school)
}

// handleUpdateSchool saves the school's details. Management only.
func (s *Server) handleUpdateSchool(w http.ResponseWriter, r *http.Request) {
	existing, err := s.store.GetSchool(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	// Start from what is stored and overwrite with what was sent, so a partial
	// update does not blank out fields the form did not include.
	updated := existing
	if err := httpx.DecodeJSON(w, r, &updated); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	// The id is not editable, whatever the body said.
	updated.ID = existing.ID

	if strings.TrimSpace(updated.NameEN) == "" || strings.TrimSpace(updated.NameHI) == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"The school name is required in both English and Hindi.",
			"विद्यालय का नाम अंग्रेज़ी और हिंदी दोनों में आवश्यक है।"))
		return
	}

	if err := s.store.UpdateSchool(r.Context(), updated); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:      audit.ActionUpdate,
		EntityType:  "school",
		EntityID:    updated.ID,
		Summary:     "School details updated",
		BeforeState: existing,
		AfterState:  updated,
	})

	httpx.JSON(w, http.StatusOK, updated)
}

// handleListSessions returns every academic year.
func (s *Server) handleListSessions(w http.ResponseWriter, r *http.Request) {
	sessions, err := s.store.ListSessions(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": sessions})
}

// handleListClasses returns classes with their sections and head counts.
func (s *Server) handleListClasses(w http.ResponseWriter, r *http.Request) {
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

	// A student or a parent has no business seeing how full every section is,
	// so the counts are removed for them while the names stay.
	identity := auth.MustFromContext(r.Context())
	if !identity.IsStaff() {
		for classIndex := range classes {
			for sectionIndex := range classes[classIndex].Sections {
				classes[classIndex].Sections[sectionIndex].Enrolled = 0
				classes[classIndex].Sections[sectionIndex].Capacity = 0
			}
		}
	}

	httpx.JSON(w, http.StatusOK, map[string]any{"items": classes})
}

// handleListSubjects returns every subject.
func (s *Server) handleListSubjects(w http.ResponseWriter, r *http.Request) {
	subjects, err := s.store.ListSubjects(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": subjects})
}

// handleListClassSubjects returns the subjects one class runs this session.
func (s *Server) handleListClassSubjects(w http.ResponseWriter, r *http.Request) {
	classID, err := pathUUID(r, "classId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	items, err := s.store.ListClassSubjects(r.Context(), sessionID, classID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items})
}

// staffOnlySettings are the settings a student or parent does not get to read.
//
// Most settings are harmless and the portal needs them to render correctly.
// These few reveal internal policy thresholds, so they stay with staff.
var staffOnlySettings = map[string]bool{
	"attendance.lowThresholdPercent": true,
	"fees.lateFeeRule":               true,
	"notices.approvalRequired":       true,
}

// handleListSettings returns the settings the caller is allowed to read.
func (s *Server) handleListSettings(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	raw, err := s.store.AllSettings(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	result := make(map[string]json.RawMessage, len(raw))
	for key, value := range raw {
		if staffOnlySettings[key] && !identity.IsStaff() {
			continue
		}
		result[key] = json.RawMessage(value)
	}

	httpx.JSON(w, http.StatusOK, result)
}

// handleUpdateSetting writes one setting. Management only.
func (s *Server) handleUpdateSetting(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	key := strings.TrimSpace(r.PathValue("key"))
	if key == "" || len(key) > 120 {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"That setting name is not valid.",
			"वह सेटिंग नाम मान्य नहीं है।"))
		return
	}

	var value json.RawMessage
	if err := httpx.DecodeJSON(w, r, &value); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	if len(value) == 0 || !json.Valid(value) {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Send the setting's value as JSON.",
			"सेटिंग का मान JSON में भेजें।"))
		return
	}

	previous, _ := s.store.GetSetting(r.Context(), key)

	if err := s.store.SetSetting(r.Context(), key, value, identity.UserID); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:      audit.ActionUpdate,
		EntityType:  "setting",
		EntityID:    key,
		Summary:     "Setting " + key + " changed",
		BeforeState: json.RawMessage(previous),
		AfterState:  value,
	})

	httpx.JSON(w, http.StatusOK, map[string]any{"key": key, "value": value})
}
