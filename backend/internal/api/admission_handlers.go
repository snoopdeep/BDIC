package api

import (
	"errors"
	"net/http"

	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

// registerAdmissionRoutes wires all admission endpoints: forms, applications,
// and decisions. Parents/students see only their own; office staff see all.
func (s *Server) registerAdmissionRoutes(mux *http.ServeMux) {
	// ---- Open: no token required -------------------------------------------
	// Public forms and submission are rate-limited but unauthenticated,
	// to let a parent apply without an account yet.
	mux.HandleFunc("GET /api/v1/public/admissions/forms", s.handlePublicAdmissionForms)
	mux.HandleFunc("POST /api/v1/public/admissions/apply", s.handlePublicSubmitApplication)

	// ---- Any signed-in user ------------------------------------------------
	// My applications: a student or parent sees their own; staff see all.
	mux.Handle("GET /api/v1/admissions/my-applications",
		s.signedIn(s.handleGetMyApplications))

	// Single application fetch with role check
	mux.Handle("GET /api/v1/admissions/{id}",
		s.signedIn(s.handleGetApplication))

	// Admission decisions: similar scoping as my-applications
	mux.Handle("GET /api/v1/admissions/decisions",
		s.signedIn(s.handleGetDecisions))

	// ---- Office staff only -------------------------------------------------
	// These will be added later: update status, schedule test, etc.
	// For now, the endpoints requested are read-only for staff.
}

// applicationFormResponse wraps the application forms for public consumption.
type applicationFormResponse struct {
	Forms []map[string]any `json:"forms"`
}

// handlePublicAdmissionForms serves the admission forms for the current session.
// These include class names and required documents, so a parent knows what to
// supply when applying. This is a public endpoint, no auth required.
func (s *Server) handlePublicAdmissionForms(w http.ResponseWriter, r *http.Request) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	forms, err := s.store.GetApplicationForms(r.Context(), sessionID)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	// Transform to a cleaner response format
	result := make([]map[string]any, len(forms))
	for i, form := range forms {
		result[i] = map[string]any{
			"classId":           form.ClassID,
			"className":         form.ClassName,
			"classNameHi":       form.ClassNameHI,
			"level":             form.Level,
			"hasStream":         form.HasStream,
			"requiredDocuments": form.RequiredDocuments,
		}
	}

	httpx.JSON(w, http.StatusOK, applicationFormResponse{Forms: result})
}

// applyRequest holds the data from the public admission application form.
type applyRequest struct {
	SessionID        string   `json:"sessionId"`
	ApplicantName    string   `json:"applicantName"`
	DateOfBirth      *string  `json:"dateOfBirth"`
	Gender           *string  `json:"gender"`
	Category         *string  `json:"category"`
	Religion         *string  `json:"religion"`
	ClassAppliedID   string   `json:"classAppliedId"`
	StreamID         *string  `json:"streamId"`
	FatherName       string   `json:"fatherName"`
	MotherName       *string  `json:"motherName"`
	GuardianName     *string  `json:"guardianName"`
	GuardianRelation *string  `json:"guardianRelation"`
	GuardianPhone    string   `json:"guardianPhone"`
	GuardianAltPhone *string  `json:"guardianAltPhone"`
	GuardianEmail    *string  `json:"guardianEmail"`
	AddressLine      *string  `json:"addressLine"`
	Village          *string  `json:"village"`
	District         *string  `json:"district"`
	State            string   `json:"state"`
	Pincode          *string  `json:"pincode"`
	PreviousSchool   *string  `json:"previousSchool"`
	LastClassPassed  *string  `json:"lastClassPassed"`
	LastClassPercent *float64 `json:"lastClassPercent"`
}

// applyResponse is what we return after a successful application.
type applyResponse struct {
	ApplicationID string `json:"applicationId"`
	ApplicationNo string `json:"applicationNo"`
	LookupToken   string `json:"lookupToken"`
	MessageEn     string `json:"messageEn"`
	MessageHi     string `json:"messageHi"`
}

// handlePublicSubmitApplication receives a new admission application from the
// public form. The applicant gets an application number and a lookup token,
// which together let them check their status without signing in.
func (s *Server) handlePublicSubmitApplication(w http.ResponseWriter, r *http.Request) {
	var input applyRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Rate limit this endpoint: a family of three children is normal,
	// but a bot trying to flood the system is not.
	clientIP := auth.ClientIP(r, s.cfg.IsProduction())
	if !s.publicLimiter.Allow("apply:" + clientIP) {
		httpx.Fail(w, r, httpx.ErrTooManyRequests())
		return
	}

	// Basic validation
	if input.ApplicantName == "" || input.FatherName == "" || input.GuardianPhone == "" || input.ClassAppliedID == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Applicant name, father's name, phone, and class are required.",
			"आवेदक का नाम, पिता का नाम, फोन और कक्षा आवश्यक हैं।"))
		return
	}

	if input.SessionID == "" {
		var err error
		input.SessionID, err = s.currentSessionID(r)
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
	}

	// Default state if not provided
	if input.State == "" {
		input.State = "Uttar Pradesh"
	}

	// Number-series periods are the school-year name (for example "2026-27"),
	// not the session UUID. Looking it up here also rejects an invalid supplied
	// session before a number can be consumed.
	session, err := s.store.GetAcademicSession(r.Context(), input.SessionID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	// Generate the application number and lookup token.
	appNo, err := s.store.NextNumber(r.Context(), store.SeriesApplicationNo, session.Name)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	lookupToken, err := generateResetCode() // Use same token generation; it's secure
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	// Create the application
	app := store.AdmissionApplication{
		ApplicationNo:    appNo,
		SessionID:        input.SessionID,
		ApplicantName:    input.ApplicantName,
		DateOfBirth:      input.DateOfBirth,
		Gender:           input.Gender,
		Category:         input.Category,
		Religion:         input.Religion,
		ClassAppliedID:   input.ClassAppliedID,
		StreamID:         input.StreamID,
		FatherName:       input.FatherName,
		MotherName:       input.MotherName,
		GuardianName:     input.GuardianName,
		GuardianRelation: input.GuardianRelation,
		GuardianPhone:    input.GuardianPhone,
		GuardianAltPhone: input.GuardianAltPhone,
		GuardianEmail:    input.GuardianEmail,
		AddressLine:      input.AddressLine,
		Village:          input.Village,
		District:         input.District,
		State:            input.State,
		Pincode:          input.Pincode,
		PreviousSchool:   input.PreviousSchool,
		LastClassPassed:  input.LastClassPassed,
		LastClassPercent: input.LastClassPercent,
		Status:           "SUBMITTED",
		LookupToken:      hashResetCode(lookupToken),
	}

	id, err := s.store.CreateApplication(r.Context(), app)
	if err != nil {
		if errors.Is(err, store.ErrConflict) {
			httpx.Fail(w, r, httpx.NewError(http.StatusConflict, "APPLICATION_EXISTS",
				"An application with that number already exists.",
				"उस नंबर का आवेदन पहले से मौजूद है।").WithInternal(err))
			return
		}
		httpx.Fail(w, r, storeError(err))
		return
	}

	httpx.JSON(w, http.StatusCreated, applyResponse{
		ApplicationID: id,
		ApplicationNo: appNo,
		LookupToken:   lookupToken,
		MessageEn:     "Application submitted. Your application number is " + appNo + ". Save it and the code to check your status.",
		MessageHi:     "आवेदन जमा कर दिया गया। आपका आवेदन संख्या " + appNo + " है। अपनी स्थिति जांचने के लिए इसे और कोड को सहेजें।",
	})
}

// handleGetMyApplications returns applications for the current user.
// Students/parents see only applications associated with their children (not yet implemented).
// Staff see all applications.
func (s *Server) handleGetMyApplications(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	apps, err := s.store.GetUserApplications(r.Context(), identity.UserID, identity.Role)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": apps,
	})
}

// handleGetApplication returns a single application by ID with role-based access.
// The actual authorization check should happen in the store query; here we just
// fetch and return what the query deems appropriate for the user's role.
func (s *Server) handleGetApplication(w http.ResponseWriter, r *http.Request) {
	applicationID, err := pathUUID(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	identity := auth.MustFromContext(r.Context())

	app, err := s.store.GetApplication(r.Context(), applicationID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	// Authorization: staff see all; students/parents see nothing (for now,
	// until we link applications to user accounts). This is a simple implementation.
	if !identity.IsStaff() {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	// Hide the lookup token in the response
	app.LookupToken = ""

	httpx.JSON(w, http.StatusOK, app)
}

// handleGetDecisions returns all admission decisions visible to the caller.
// Students/parents see only decisions for their own applications (not yet scoped).
// Staff see all decisions.
func (s *Server) handleGetDecisions(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	decisions, err := s.store.GetAdmissionDecisions(r.Context(), identity.UserID, identity.Role)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": decisions,
	})
}
