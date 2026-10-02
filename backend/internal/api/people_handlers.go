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

func (s *Server) registerPeopleRoutes(mux *http.ServeMux) {
	mux.Handle("GET /api/v1/people/students", s.restricted(s.handleListDirectoryStudents,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice, auth.RoleTeacher))
	mux.Handle("GET /api/v1/people/staff", s.restricted(s.handleListDirectoryStaff,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice))
	mux.Handle("POST /api/v1/people/students", s.restricted(s.handleCreateStudentProfile,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice))
	mux.Handle("POST /api/v1/people/staff", s.restricted(s.handleCreateStaffProfile,
		auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice))
}

func (s *Server) handleListDirectoryStudents(w http.ResponseWriter, r *http.Request) {
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	limit, offset := paginate(r)
	items, err := s.store.ListDirectoryStudents(r.Context(), sessionID, limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items), "limit": limit, "offset": offset})
}

func (s *Server) handleListDirectoryStaff(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	items, err := s.store.ListDirectoryStaff(r.Context(), limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"items": items, "count": len(items), "limit": limit, "offset": offset})
}

type staffAccountRequest struct {
	Enabled           bool   `json:"enabled"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	TemporaryPassword string `json:"temporaryPassword"`
	Role              string `json:"role"`
}

type createStaffProfileRequest struct {
	EmployeeCode  string              `json:"employeeCode"`
	FullNameEN    string              `json:"fullNameEn"`
	FullNameHI    string              `json:"fullNameHi"`
	DesignationEN string              `json:"designationEn"`
	DesignationHI string              `json:"designationHi"`
	Department    string              `json:"department"`
	Qualification string              `json:"qualification"`
	Phone         string              `json:"phone"`
	Email         string              `json:"email"`
	Account       staffAccountRequest `json:"account"`
}

func (s *Server) handleCreateStaffProfile(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	var input createStaffProfileRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	input.EmployeeCode = strings.TrimSpace(input.EmployeeCode)
	input.FullNameEN = strings.TrimSpace(input.FullNameEN)
	input.FullNameHI = strings.TrimSpace(input.FullNameHI)
	input.Email = strings.TrimSpace(input.Email)
	input.Phone = strings.TrimSpace(input.Phone)
	if input.EmployeeCode == "" || input.FullNameEN == "" || len(input.EmployeeCode) > 60 || len(input.FullNameEN) > 200 {
		httpx.Fail(w, r, httpx.ErrBadRequest("Employee code and full name are required.", "कर्मचारी कोड और पूरा नाम आवश्यक है।"))
		return
	}
	var login *store.NewUser
	if input.Account.Enabled {
		if !identity.CanManageSchool() {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		role := strings.ToUpper(strings.TrimSpace(input.Account.Role))
		if role == "" {
			role = auth.RoleTeacher
		}
		if role != auth.RoleTeacher && role != auth.RoleOffice && role != auth.RoleAccounts && role != auth.RolePrincipal {
			httpx.Fail(w, r, httpx.ErrBadRequest("Staff account role must be TEACHER, OFFICE, ACCOUNTS, or PRINCIPAL.", "स्टाफ खाते की भूमिका TEACHER, OFFICE, ACCOUNTS या PRINCIPAL होनी चाहिए।").WithField("account.role", "Invalid role"))
			return
		}
		if role == auth.RolePrincipal && !identity.HasRole(auth.RoleSuperAdmin) {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		hash, err := auth.HashPassword(input.Account.TemporaryPassword)
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		login = &store.NewUser{
			Email: strings.TrimSpace(input.Account.Email), Phone: strings.TrimSpace(input.Account.Phone), PasswordHash: hash,
			FullNameEN: input.FullNameEN, FullNameHI: input.FullNameHI, Role: role, Locale: "hi", MustReset: true,
		}
	}
	ctx, cancel := contextWithTimeout(r, 15*time.Second)
	defer cancel()
	staffID, userID, err := s.store.CreateStaffProfile(ctx, store.NewStaffProfile{
		EmployeeCode: input.EmployeeCode, FullNameEN: input.FullNameEN, FullNameHI: input.FullNameHI,
		DesignationEN: strings.TrimSpace(input.DesignationEN), DesignationHI: strings.TrimSpace(input.DesignationHI),
		Department: strings.TrimSpace(input.Department), Qualification: strings.TrimSpace(input.Qualification),
		Phone: input.Phone, Email: input.Email, Login: login,
	})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionCreate, EntityType: "staff", EntityID: staffID, Summary: "Created staff profile: " + input.FullNameEN, AfterState: map[string]any{"staffId": staffID, "userId": userID, "employeeCode": input.EmployeeCode}})
	httpx.JSON(w, http.StatusCreated, map[string]string{"staffId": staffID, "userId": userID})
}

type portalAccountRequest struct {
	Enabled           bool   `json:"enabled"`
	Email             string `json:"email"`
	Phone             string `json:"phone"`
	TemporaryPassword string `json:"temporaryPassword"`
}

type createStudentProfileRequest struct {
	AdmissionNo string `json:"admissionNo"`
	FullNameEN  string `json:"fullNameEn"`
	FullNameHI  string `json:"fullNameHi"`
	DateOfBirth string `json:"dateOfBirth"`
	Gender      string `json:"gender"`
	Category    string `json:"category"`
	AddressLine string `json:"addressLine"`
	Village     string `json:"village"`
	District    string `json:"district"`
	State       string `json:"state"`
	Pincode     string `json:"pincode"`
	FatherName  string `json:"fatherName"`
	MotherName  string `json:"motherName"`
	ClassID     string `json:"classId"`
	SectionID   string `json:"sectionId"`
	StreamID    string `json:"streamId"`
	Guardian    struct {
		FullNameEN string               `json:"fullNameEn"`
		FullNameHI string               `json:"fullNameHi"`
		Relation   string               `json:"relation"`
		Phone      string               `json:"phone"`
		AltPhone   string               `json:"altPhone"`
		Email      string               `json:"email"`
		Occupation string               `json:"occupation"`
		Address    string               `json:"address"`
		Account    portalAccountRequest `json:"account"`
	} `json:"guardian"`
	StudentAccount portalAccountRequest `json:"studentAccount"`
}

func (s *Server) handleCreateStudentProfile(w http.ResponseWriter, r *http.Request) {
	var input createStudentProfileRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}
	input.AdmissionNo = strings.TrimSpace(input.AdmissionNo)
	input.FullNameEN = strings.TrimSpace(input.FullNameEN)
	input.Guardian.FullNameEN = strings.TrimSpace(input.Guardian.FullNameEN)
	input.Guardian.Relation = strings.ToUpper(strings.TrimSpace(input.Guardian.Relation))
	input.Guardian.Phone = strings.TrimSpace(input.Guardian.Phone)
	if input.AdmissionNo == "" || input.FullNameEN == "" || !auth.IsUUID(input.ClassID) || !auth.IsUUID(input.SectionID) || input.Guardian.FullNameEN == "" || input.Guardian.Phone == "" || (input.Guardian.Relation != "FATHER" && input.Guardian.Relation != "MOTHER" && input.Guardian.Relation != "GUARDIAN" && input.Guardian.Relation != "OTHER") {
		httpx.Fail(w, r, httpx.ErrBadRequest("Student, class, section, and primary guardian details are required.", "विद्यार्थी, कक्षा, सेक्शन और प्राथमिक अभिभावक का विवरण आवश्यक है।"))
		return
	}
	if input.StreamID != "" && !auth.IsUUID(input.StreamID) {
		httpx.Fail(w, r, httpx.ErrBadRequest("The stream is not valid.", "स्ट्रीम मान्य नहीं है।").WithField("streamId", "Invalid id"))
		return
	}
	if input.DateOfBirth != "" {
		if _, err := time.Parse("2006-01-02", input.DateOfBirth); err != nil {
			httpx.Fail(w, r, httpx.ErrBadRequest("Use YYYY-MM-DD for date of birth.", "जन्म तिथि के लिए YYYY-MM-DD दें।").WithField("dateOfBirth", "Invalid date"))
			return
		}
	}
	var studentLogin *store.NewUser
	if input.StudentAccount.Enabled {
		hash, err := auth.HashPassword(input.StudentAccount.TemporaryPassword)
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		studentLogin = &store.NewUser{Email: strings.TrimSpace(input.StudentAccount.Email), Phone: strings.TrimSpace(input.StudentAccount.Phone), PasswordHash: hash, FullNameEN: input.FullNameEN, FullNameHI: strings.TrimSpace(input.FullNameHI), Role: auth.RoleStudent, Locale: "hi", MustReset: true}
	}
	var guardianLogin *store.NewUser
	if input.Guardian.Account.Enabled {
		hash, err := auth.HashPassword(input.Guardian.Account.TemporaryPassword)
		if err != nil {
			httpx.Fail(w, r, err)
			return
		}
		guardianLogin = &store.NewUser{Email: strings.TrimSpace(input.Guardian.Account.Email), Phone: strings.TrimSpace(input.Guardian.Account.Phone), PasswordHash: hash, FullNameEN: input.Guardian.FullNameEN, FullNameHI: strings.TrimSpace(input.Guardian.FullNameHI), Role: auth.RoleParent, Locale: "hi", MustReset: true}
	}
	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}
	ctx, cancel := contextWithTimeout(r, 15*time.Second)
	defer cancel()
	result, err := s.store.CreateStudentProfile(ctx, store.NewStudentProfile{
		AdmissionNo: input.AdmissionNo, FullNameEN: input.FullNameEN, FullNameHI: strings.TrimSpace(input.FullNameHI), DateOfBirth: strings.TrimSpace(input.DateOfBirth),
		Gender: strings.ToUpper(strings.TrimSpace(input.Gender)), Category: strings.ToUpper(strings.TrimSpace(input.Category)), AddressLine: strings.TrimSpace(input.AddressLine),
		Village: strings.TrimSpace(input.Village), District: strings.TrimSpace(input.District), State: strings.TrimSpace(input.State), Pincode: strings.TrimSpace(input.Pincode),
		FatherName: strings.TrimSpace(input.FatherName), MotherName: strings.TrimSpace(input.MotherName), SessionID: sessionID, ClassID: input.ClassID, SectionID: input.SectionID, StreamID: input.StreamID,
		Guardian:     store.NewGuardian{FullNameEN: input.Guardian.FullNameEN, FullNameHI: strings.TrimSpace(input.Guardian.FullNameHI), Relation: input.Guardian.Relation, Phone: input.Guardian.Phone, AltPhone: strings.TrimSpace(input.Guardian.AltPhone), Email: strings.TrimSpace(input.Guardian.Email), Occupation: strings.TrimSpace(input.Guardian.Occupation), Address: strings.TrimSpace(input.Guardian.Address), Login: guardianLogin},
		StudentLogin: studentLogin,
	})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	s.audit.Record(ctx, r, audit.Entry{Action: audit.ActionCreate, EntityType: "student", EntityID: result.StudentID, Summary: "Created student profile: " + input.FullNameEN, AfterState: result})
	httpx.JSON(w, http.StatusCreated, result)
}
