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

func (s *Server) registerCertificateRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/certificates", s.signedIn(s.handleListCertificates))
	mux.Handle("POST /api/v1/certificates", s.restricted(s.handleIssueCertificate,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice))
}

func (s *Server) handleListCertificates(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	studentID, err := queryUUID(r, "studentId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	var items []store.Certificate
	switch {
	case identity.HasRole(auth.RoleStudent):
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListCertificatesForStudent(ctx, identity.StudentID, limit, offset)
	case identity.HasRole(auth.RoleParent):
		if identity.GuardianID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		items, err = s.store.ListCertificatesForGuardian(ctx, identity.GuardianID, limit, offset)
	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice):
		items, err = s.store.ListCertificatesAll(ctx, studentID, limit, offset)
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

type issueCertificateRequest struct {
	StudentID string         `json:"studentId"`
	Kind      string         `json:"kind"`
	Payload   map[string]any `json:"payload"`
}

func (s *Server) handleIssueCertificate(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	var input issueCertificateRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	input.Kind = strings.ToUpper(strings.TrimSpace(input.Kind))
	if !auth.IsUUID(input.StudentID) || (input.Kind != "BONAFIDE" && input.Kind != "CHARACTER" && input.Kind != "TRANSFER" && input.Kind != "ATTENDANCE") {
		httpx.Fail(w, r, httpx.ErrBadRequest("Choose a student and certificate type: BONAFIDE, CHARACTER, TRANSFER, or ATTENDANCE.", "विद्यार्थी और प्रमाण पत्र प्रकार BONAFIDE, CHARACTER, TRANSFER या ATTENDANCE चुनें।"))
		return
	}
	if input.Payload == nil {
		input.Payload = map[string]any{}
	}
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()
	session, err := s.store.GetAcademicSession(ctx, sessionID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	certificate, err := s.store.IssueCertificate(ctx, store.IssueCertificateInput{
		StudentID: input.StudentID, Kind: input.Kind, SessionName: session.Name, IssuedBy: identity.UserID, Payload: input.Payload,
	})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionCreate, EntityType: "certificate", EntityID: certificate.ID, Summary: "Issued " + certificate.Kind + " certificate " + certificate.SerialNo, AfterState: certificate})
	httpx.JSON(w, http.StatusCreated, map[string]any{"certificate": certificate})
}
