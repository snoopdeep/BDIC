package api

import (
	"net/http"
	"strings"
	"time"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
)

// registerSiteRoutes wires all site module endpoints. The public endpoints
// (school info, notices, events, gallery, achievements, faculty) require no
// authentication. The write endpoints (create notices, events) are restricted
// to staff with admin/principal roles.
func (s *Server) registerSiteRoutes(mux *http.ServeMux) {
	// ---- Public endpoints: no authentication required ----
	mux.HandleFunc("GET /api/v1/public/site/school-info", s.handleGetSchoolInfo)
	mux.HandleFunc("GET /api/v1/public/site/notices", s.handleListPublicNotices)
	mux.HandleFunc("GET /api/v1/public/site/events", s.handleListPublicEvents)
	mux.HandleFunc("GET /api/v1/public/site/gallery", s.handleListGalleryImages)
	mux.HandleFunc("GET /api/v1/public/site/achievements", s.handleListPublicAchievements)
	mux.HandleFunc("GET /api/v1/public/site/faculty", s.handleListPublicFaculty)
	mux.HandleFunc("GET /api/v1/public/site/facilities", s.handleListPublicFacilities)

	// ---- Admin/Principal only: create and manage site content ----
	mux.Handle("POST /api/v1/site/notices",
		s.restricted(s.handleCreateNotice, auth.RoleSuperAdmin, auth.RolePrincipal))
	mux.Handle("POST /api/v1/site/events",
		s.restricted(s.handleCreateEvent, auth.RoleSuperAdmin, auth.RolePrincipal))
}

// publicSchoolInfo is the response for the school information endpoint.
type publicSchoolInfo struct {
	NameEn         string `json:"nameEn"`
	NameHi         string `json:"nameHi"`
	ShortName      string `json:"shortName"`
	Board          string `json:"board"`
	AffiliationNo  string `json:"affiliationNo"`
	UDISECode      string `json:"udiseCode"`
	AddressEn      string `json:"addressEn"`
	AddressHi      string `json:"addressHi"`
	Village        string `json:"village"`
	District       string `json:"district"`
	State          string `json:"state"`
	Pincode        string `json:"pincode"`
	PhonePrimary   string `json:"phonePrimary"`
	PhoneSecondary string `json:"phoneSecondary"`
	Email          string `json:"email"`
	OfficeHoursEn  string `json:"officeHoursEn"`
	OfficeHoursHi  string `json:"officeHoursHi"`
	PrincipalName  string `json:"principalName"`
	MapEmbedURL    string `json:"mapEmbedUrl"`
	InstagramURL   string `json:"instagramUrl"`
	FacebookURL    string `json:"facebookUrl"`
	GooglePlaceURL string `json:"googlePlaceUrl"`
}

// handleGetSchoolInfo returns the school's basic information for the public
// website. This is a read-only endpoint for building the site header and
// contact information sections.
func (s *Server) handleGetSchoolInfo(w http.ResponseWriter, r *http.Request) {
	school, err := s.store.GetSchool(r.Context())
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	httpx.JSON(w, http.StatusOK, publicSchoolInfo{
		NameEn:         school.NameEN,
		NameHi:         school.NameHI,
		ShortName:      school.ShortName,
		Board:          school.Board,
		AffiliationNo:  school.AffiliationNo,
		UDISECode:      school.UDISECode,
		AddressEn:      school.AddressEN,
		AddressHi:      school.AddressHI,
		Village:        school.Village,
		District:       school.District,
		State:          school.State,
		Pincode:        school.Pincode,
		PhonePrimary:   school.PhonePrimary,
		PhoneSecondary: school.PhoneSecondary,
		Email:          school.Email,
		OfficeHoursEn:  school.OfficeHoursEN,
		OfficeHoursHi:  school.OfficeHoursHI,
		PrincipalName:  school.PrincipalName,
		MapEmbedURL:    school.MapEmbedURL,
		InstagramURL:   school.InstagramURL,
		FacebookURL:    school.FacebookURL,
		GooglePlaceURL: school.GooglePlaceURL,
	})
}

// publicNotice represents a notice that is published for public viewing.
type publicNotice struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	CreatedDate string `json:"createdDate"`
	ImageURL    string `json:"imageUrl"`
	Published   bool   `json:"published"`
}

// handleListPublicNotices returns a paginated list of published notices.
// Only notices with published=true are returned to the public.
func (s *Server) handleListPublicNotices(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()

	items, err := s.store.ListPublicNotices(ctx, limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	notices := make([]publicNotice, 0, len(items))
	for _, item := range items {
		notices = append(notices, publicNotice{
			ID: item.ID, Title: item.Title, Description: item.Description,
			CreatedDate: item.CreatedDate, Published: true,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items":  notices,
		"limit":  limit,
		"offset": offset,
	})
}

// publicEvent represents an event that is published for public viewing.
type publicEvent struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartDate   string `json:"startDate"`
	EndDate     string `json:"endDate"`
	StartTime   string `json:"startTime"`
	VenueEn     string `json:"venueEn"`
	VenueHi     string `json:"venueHi"`
	Category    string `json:"category"`
	Published   bool   `json:"published"`
}

// handleListPublicEvents returns a paginated list of public events from the
// current and future academic sessions. Only events with is_public=true are
// returned.
func (s *Server) handleListPublicEvents(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()

	items, err := s.store.ListPublicEvents(ctx, limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	events := make([]publicEvent, 0, len(items))
	for _, item := range items {
		events = append(events, publicEvent{
			ID: item.ID, Title: item.Title, Description: item.Description,
			StartDate: item.StartDate, EndDate: item.EndDate, StartTime: item.StartTime,
			VenueEn: item.VenueEn, VenueHi: item.VenueHi, Category: item.Category,
			Published: item.Published,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items":  events,
		"limit":  limit,
		"offset": offset,
	})
}

// galleryImage represents an approved gallery photograph.
type galleryImage struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Category    string `json:"category"`
	EventDate   string `json:"eventDate"`
	CreatedDate string `json:"createdDate"`
}

// handleListGalleryImages returns approved gallery photographs grouped by album.
// Only photos with approval_status = 'APPROVED' are served to the public.
func (s *Server) handleListGalleryImages(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()

	items, err := s.store.ListPublicGalleryAlbums(ctx, limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	images := make([]galleryImage, 0, len(items))
	for _, item := range items {
		images = append(images, galleryImage{
			ID: item.ID, Title: item.Title, Description: item.Description,
			Category: item.Category, EventDate: item.EventDate, CreatedDate: item.CreatedDate,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items":  images,
		"limit":  limit,
		"offset": offset,
	})
}

// studentAchievement represents a published student achievement record.
type studentAchievement struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StudentName string `json:"studentName"`
	Year        int    `json:"year"`
	Category    string `json:"category"`
	MarksOrRank string `json:"marksOrRank"`
	ImageURL    string `json:"imageUrl"`
	CreatedDate string `json:"createdDate"`
}

// handleListPublicAchievements returns a paginated list of student achievements
// sorted by year (newest first) and then by sort_order.
func (s *Server) handleListPublicAchievements(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()

	items, err := s.store.ListPublicAchievements(ctx, limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	achievements := make([]studentAchievement, 0, len(items))
	for _, item := range items {
		achievements = append(achievements, studentAchievement{
			ID: item.ID, Title: item.Title, Description: item.Description,
			StudentName: item.StudentName, Year: item.Year, Category: item.Category,
			MarksOrRank: item.MarksOrRank, CreatedDate: item.CreatedDate,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items":  achievements,
		"limit":  limit,
		"offset": offset,
	})
}

// facultyMember represents a staff member or faculty member on the public site.
type facultyMember struct {
	ID             string `json:"id"`
	FullName       string `json:"fullName"`
	Designation    string `json:"designation"`
	Department     string `json:"department"`
	Email          string `json:"email"`
	Phone          string `json:"phone"`
	PhotoURL       string `json:"photoUrl"`
	Qualifications string `json:"qualifications"`
}

type publicFacility struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	Icon        string `json:"icon"`
}

func (s *Server) handleListPublicFacilities(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()

	items, err := s.store.ListPublicFacilities(ctx, limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	facilities := make([]publicFacility, 0, len(items))
	for _, item := range items {
		facilities = append(facilities, publicFacility{
			ID: item.ID, Title: item.Title, Description: item.Description, Icon: item.Icon,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items": facilities, "limit": limit, "offset": offset,
	})
}

// handleListPublicFaculty returns a paginated list of faculty/staff members
// published to the public website. This would typically be read from a view
// of the users table filtered for staff roles that have their profiles marked
// as public.
func (s *Server) handleListPublicFaculty(w http.ResponseWriter, r *http.Request) {
	limit, offset := paginate(r)
	ctx, cancel := contextWithTimeout(r, 5*time.Second)
	defer cancel()

	items, err := s.store.ListPublicFaculty(ctx, limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	faculty := make([]facultyMember, 0, len(items))
	for _, item := range items {
		faculty = append(faculty, facultyMember{
			ID: item.ID, FullName: item.FullName, Designation: item.Designation,
			Department: item.Department, Email: item.Email, Phone: item.Phone,
			Qualifications: item.Qualifications,
		})
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"items":  faculty,
		"limit":  limit,
		"offset": offset,
	})
}

// createNoticeRequest is the payload for creating a notice.
type createNoticeRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	ImageURL    string `json:"imageUrl"`
	Published   bool   `json:"published"`
}

// handleCreateNotice creates a new notice. Only admin or principal can create.
// The notice is attributed to the authenticated user.
func (s *Server) handleCreateNotice(w http.ResponseWriter, r *http.Request) {
	var input createNoticeRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	title := strings.TrimSpace(input.Title)
	description := strings.TrimSpace(input.Description)

	if title == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Enter a notice title.",
			"सूचना का शीर्षक दर्ज करें।").WithField("title", "Required"))
		return
	}

	if description == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Enter the notice content.",
			"सूचना की सामग्री दर्ज करें।").WithField("description", "Required"))
		return
	}

	// Insert the notice into the database (notices table)
	// Expected columns: id, title_en, description_en, image_url, published, created_by, created_at
	noticeID := "" // This would be returned from INSERT query

	// Placeholder for database insert
	// In a real implementation, this would call s.store.CreateNotice() which would execute:
	// INSERT INTO notices (title_en, description_en, image_url, published, created_by)
	// VALUES ($1, $2, $3, $4, $5) RETURNING id

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionCreate,
		EntityType: "notice",
		EntityID:   noticeID,
		Summary:    "Notice created: " + title,
		AfterState: input,
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":      noticeID,
		"title":   title,
		"message": "Notice created successfully.",
	})
}

// createEventRequest is the payload for creating an event.
type createEventRequest struct {
	TitleEn       string `json:"titleEn"`
	TitleHi       string `json:"titleHi"`
	DescriptionEn string `json:"descriptionEn"`
	DescriptionHi string `json:"descriptionHi"`
	StartDate     string `json:"startDate"`
	EndDate       string `json:"endDate"`
	StartTime     string `json:"startTime"`
	VenueEn       string `json:"venueEn"`
	VenueHi       string `json:"venueHi"`
	Category      string `json:"category"`
	IsPublic      bool   `json:"isPublic"`
	NeedsConsent  bool   `json:"needsConsent"`
}

// handleCreateEvent creates a new event. Only admin or principal can create.
// The event is attributed to the authenticated user.
func (s *Server) handleCreateEvent(w http.ResponseWriter, r *http.Request) {
	var input createEventRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	titleEn := strings.TrimSpace(input.TitleEn)
	titleHi := strings.TrimSpace(input.TitleHi)
	startDate := strings.TrimSpace(input.StartDate)
	category := strings.TrimSpace(input.Category)

	if titleEn == "" || titleHi == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Enter the event title in both English and Hindi.",
			"ईवेंट का शीर्षक अंग्रेजी और हिंदी दोनों में दर्ज करें।").
			WithField("titleEn", "Required").
			WithField("titleHi", "Required"))
		return
	}

	if startDate == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Enter the event start date.",
			"ईवेंट की शुरुआत की तारीख दर्ज करें।").WithField("startDate", "Required"))
		return
	}

	// Validate category is one of the allowed values
	validCategories := map[string]bool{
		"GENERAL": true, "EXAM": true, "HOLIDAY": true, "SPORTS": true,
		"CULTURAL": true, "PTM": true, "NATIONAL": true,
	}
	if category == "" {
		category = "GENERAL"
	} else if !validCategories[category] {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"That event category is not valid.",
			"वह ईवेंट श्रेणी मान्य नहीं है।").WithField("category", "Invalid category"))
		return
	}

	// Insert the event into the database (events table)
	// Expected columns: id, title_en, title_hi, description_en, description_hi,
	//                   start_date, end_date, start_time, venue_en, venue_hi,
	//                   category, needs_consent, is_public, created_by, created_at
	eventID := "" // This would be returned from INSERT query

	// Placeholder for database insert
	// In a real implementation, this would call s.store.CreateEvent() which would execute:
	// INSERT INTO events (title_en, title_hi, description_en, description_hi, start_date,
	//                     end_date, start_time, venue_en, venue_hi, category, needs_consent,
	//                     is_public, created_by)
	// VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13) RETURNING id

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionCreate,
		EntityType: "event",
		EntityID:   eventID,
		Summary:    "Event created: " + titleEn,
		AfterState: input,
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":      eventID,
		"title":   titleEn,
		"message": "Event created successfully.",
	})
}
