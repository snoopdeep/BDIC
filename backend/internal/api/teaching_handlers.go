package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/files"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

// registerTeachingRoutes registers all endpoints for the teaching module.
// Teachers can manage their classes and study materials.
func (s *Server) registerTeachingRoutes(mux *http.ServeMux) {
	// ---- Teachers only ------------------------------------------------
	// List classes taught by the current teacher
	mux.Handle("GET /api/v1/teaching/my-classes",
		s.restricted(s.handleListMyClasses, auth.RoleTeacher))

	// List students in a class (class must be taught by this teacher)
	mux.Handle("GET /api/v1/teaching/classes/{classId}/students",
		s.restricted(s.handleListClassStudents, auth.RoleTeacher))

	// ---- Study materials: anyone can read, teachers/admins can write -----
	// List study materials for current user's context
	mux.Handle("GET /api/v1/teaching/study-material",
		s.signedIn(s.handleListStudyMaterials))

	// Upload study material (teachers and admins only)
	mux.Handle("POST /api/v1/teaching/study-material",
		s.restricted(s.handleUploadStudyMaterial, auth.RoleTeacher, auth.RolePrincipal, auth.RoleSuperAdmin))

	// Download study material (teachers and admins can see all; students see their class materials)
	mux.Handle("GET /api/v1/teaching/study-material/{materialId}",
		s.signedIn(s.handleDownloadStudyMaterial))

	// Delete study material (owner or admin)
	mux.Handle("DELETE /api/v1/teaching/study-material/{materialId}",
		s.restricted(s.handleDeleteStudyMaterial, auth.RoleTeacher, auth.RolePrincipal, auth.RoleSuperAdmin))
}

// ============================================================================
// Classes Endpoints
// ============================================================================

// handleListMyClasses returns all classes taught by the current teacher.
func (s *Server) handleListMyClasses(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	// Only teachers have a staff ID
	if identity.StaffID == "" {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	classes, err := s.store.GetTeacherClasses(ctx, sessionID, identity.StaffID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"classes": classes,
	})
}

// handleListClassStudents returns all students in a specific class.
// The teacher must be the class teacher for this class.
func (s *Server) handleListClassStudents(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	if identity.StaffID == "" {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

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

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	// Verify the teacher teaches this class in the current session
	isTeacher, err := s.store.IsTeacherForClass(ctx, sessionID, classID, identity.StaffID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	if !isTeacher {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	// Get all students in the class
	students, err := s.store.GetClassStudents(ctx, sessionID, classID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"classId":  classID,
		"students": students,
	})
}

// ============================================================================
// Study Materials Endpoints
// ============================================================================

// handleListStudyMaterials returns available study materials based on user role.
// - Admins and principals: see all materials in the current session
// - Teachers: see materials for their classes
// - Students: see materials for their own class
func (s *Server) handleListStudyMaterials(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	var materials []interface{}
	var queryErr error

	switch {
	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal):
		// Admins see all materials in the session
		materials, queryErr = s.store.ListAllStudyMaterials(ctx, sessionID)

	case identity.HasRole(auth.RoleTeacher):
		// Teachers see materials for classes they teach
		if identity.StaffID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		materials, queryErr = s.store.ListTeacherStudyMaterials(ctx, sessionID, identity.StaffID)

	case identity.HasRole(auth.RoleStudent):
		// Students see materials for their enrolled class
		if identity.StudentID == "" {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
		materials, queryErr = s.store.ListStudentStudyMaterials(ctx, sessionID, identity.StudentID)

	default:
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	if queryErr != nil {
		httpx.Fail(w, r, storeError(queryErr))
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{
		"materials": materials,
	})
}

// uploadStudyMaterialRequest is the input for uploading study material.
type uploadStudyMaterialRequest struct {
	ClassID     string `json:"classId"`
	SubjectID   string `json:"subjectId"`
	TitleEN     string `json:"titleEn"`
	TitleHI     string `json:"titleHi"`
	Description string `json:"description"`
	Category    string `json:"category"` // NOTES, SYLLABUS, WORKSHEET, PREVIOUS_PAPER, VIDEO, REFERENCE
	LinkURL     string `json:"linkUrl"`  // Optional: external link
}

// handleUploadStudyMaterial uploads a study material file or link.
// Only teachers and admins can upload.
func (s *Server) handleUploadStudyMaterial(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	if !identity.IsStaff() {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	sessionID, err := s.currentSessionID(r)
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	var input uploadStudyMaterialRequest
	if err := httpx.DecodeJSON(w, r, &input); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	// Validate required fields
	if input.ClassID == "" || input.SubjectID == "" || input.TitleEN == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Provide the class, subject, and title.",
			"क्लास, विषय और शीर्षक प्रदान करें।").
			WithField("classId", "Required").
			WithField("subjectId", "Required").
			WithField("titleEn", "Required"))
		return
	}

	// Validate category
	validCategories := map[string]bool{
		"NOTES": true, "SYLLABUS": true, "WORKSHEET": true,
		"PREVIOUS_PAPER": true, "VIDEO": true, "REFERENCE": true,
	}
	if input.Category == "" {
		input.Category = "NOTES"
	}
	if !validCategories[input.Category] {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Choose a valid category: NOTES, SYLLABUS, WORKSHEET, PREVIOUS_PAPER, VIDEO, or REFERENCE.",
			"एक वैध श्रेणी चुनें।").
			WithField("category", "Invalid category"))
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	// Verify class and subject exist and are accessible to this teacher
	if identity.HasRole(auth.RoleTeacher) {
		// Teacher can only upload for classes they teach
		isTeacher, err := s.store.IsTeacherForClass(ctx, sessionID, input.ClassID, identity.StaffID)
		if err != nil {
			httpx.Fail(w, r, storeError(err))
			return
		}
		if !isTeacher {
			httpx.Fail(w, r, httpx.ErrForbidden())
			return
		}
	}

	var fileID string
	if input.LinkURL == "" {
		// File upload mode
		r.Body = http.MaxBytesReader(w, r.Body, s.cfg.MaxUploadBytes)

		if err := r.ParseMultipartForm(4 << 20); err != nil {
			httpx.Fail(w, r, httpx.NewError(http.StatusRequestEntityTooLarge, "UPLOAD_TOO_LARGE",
				fmt.Sprintf("That file is too large. The limit is %d MB.", s.cfg.MaxUploadBytes>>20),
				fmt.Sprintf("यह फ़ाइल बहुत बड़ी है। अधिकतम %d MB.", s.cfg.MaxUploadBytes>>20)).
				WithInternal(err))
			return
		}
		defer func() {
			if r.MultipartForm != nil {
				_ = r.MultipartForm.RemoveAll()
			}
		}()

		upload, header, err := r.FormFile("file")
		if err != nil {
			httpx.Fail(w, r, httpx.ErrBadRequest(
				"Choose a file to upload.",
				"अपलोड करने के लिए एक फ़ाइल चुनें।").WithInternal(err))
			return
		}
		defer upload.Close()

		// Sniff the MIME type
		head := make([]byte, 512)
		read, err := io.ReadFull(upload, head)
		if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}
		head = head[:read]

		detected := http.DetectContentType(head)
		baseType := strings.TrimSpace(strings.Split(detected, ";")[0])

		if !files.AllowedMIMETypes[baseType] {
			httpx.Fail(w, r, httpx.ErrBadRequest(
				"That kind of file is not accepted. Use a photograph, a PDF, or a spreadsheet.",
				"यह प्रकार की फ़ाइल स्वीकार नहीं है।").
				WithField("file", "Unsupported type: "+baseType))
			return
		}

		if _, err := upload.Seek(0, io.SeekStart); err != nil {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}

		key, err := files.NewKey("study-material", header.Filename)
		if err != nil {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}

		if err := s.storage.Save(key, upload); err != nil {
			httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
			return
		}

		fileID, err = s.store.CreateFile(r.Context(), store.NewFile{
			StorageKey:   key,
			OriginalName: safeFilename(header.Filename),
			MimeType:     baseType,
			SizeBytes:    header.Size,
			Visibility:   store.VisibilityPrivate,
			UploadedBy:   identity.UserID,
		})
		if err != nil {
			_ = s.storage.Delete(key)
			httpx.Fail(w, r, storeError(err))
			return
		}
	} else {
		// Link mode: validate the URL
		input.LinkURL = strings.TrimSpace(input.LinkURL)
		if !strings.HasPrefix(input.LinkURL, "http://") && !strings.HasPrefix(input.LinkURL, "https://") {
			httpx.Fail(w, r, httpx.ErrBadRequest(
				"The URL must start with http:// or https://.",
				"URL को http:// या https:// से शुरू होना चाहिए।").
				WithField("linkUrl", "Invalid URL"))
			return
		}
	}

	// Create the study material record
	materialID, err := s.store.CreateStudyMaterial(r.Context(), store.NewStudyMaterial{
		SessionID:   sessionID,
		ClassID:     input.ClassID,
		SubjectID:   input.SubjectID,
		TitleEN:     strings.TrimSpace(input.TitleEN),
		TitleHI:     strings.TrimSpace(input.TitleHI),
		Description: strings.TrimSpace(input.Description),
		Category:    input.Category,
		FileID:      fileID,
		LinkURL:     input.LinkURL,
		UploadedBy:  identity.UserID,
	})
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionCreate,
		EntityType: "study_material",
		EntityID:   materialID,
		Summary:    "Uploaded study material: " + input.TitleEN,
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":        materialID,
		"classId":   input.ClassID,
		"subjectId": input.SubjectID,
		"titleEn":   input.TitleEN,
		"titleHi":   input.TitleHI,
		"category":  input.Category,
	})
}

// handleDownloadStudyMaterial serves a study material file or redirects to a link.
func (s *Server) handleDownloadStudyMaterial(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	materialID, err := pathUUID(r, "materialId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	material, err := s.store.GetStudyMaterial(ctx, materialID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	// Check access permissions
	canAccess := false
	switch {
	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal):
		// Admins can access all
		canAccess = true

	case identity.HasRole(auth.RoleTeacher):
		// Teachers can access materials for classes they teach
		if identity.StaffID != "" {
			isTeacher, err := s.store.IsTeacherForClass(ctx, material.SessionID, material.ClassID, identity.StaffID)
			if err == nil && isTeacher {
				canAccess = true
			}
		}

	case identity.HasRole(auth.RoleStudent):
		// Students can access materials for their own class
		if identity.StudentID != "" {
			inClass, err := s.store.IsStudentInClass(ctx, material.SessionID, material.ClassID, identity.StudentID)
			if err == nil && inClass {
				canAccess = true
			}
		}
	}

	if !canAccess {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	// If it's a link, redirect
	if material.LinkURL != "" {
		http.Redirect(w, r, material.LinkURL, http.StatusSeeOther)
		return
	}

	// Otherwise serve the file
	if material.FileID == "" {
		httpx.Fail(w, r, httpx.ErrNotFound())
		return
	}

	file, err := s.store.GetFile(ctx, material.FileID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	reader, err := s.storage.Open(file.StorageKey)
	if err != nil {
		if errors.Is(err, files.ErrNotStored) {
			s.logger.Error("study material file not in storage",
				"fileId", material.FileID, "key", file.StorageKey)
			httpx.Fail(w, r, httpx.ErrNotFound().WithInternal(err))
			return
		}
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", file.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", file.SizeBytes))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	// Use inline for PDFs so they display, attachment for others
	disposition := "attachment"
	if file.MimeType == "application/pdf" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`%s; filename="%s"`, disposition, safeFilename(file.OriginalName)))
	if _, err := io.Copy(w, reader); err != nil {
		s.logger.Warn("study material download interrupted",
			"materialId", materialID, "error", err)
	}
}

// handleDeleteStudyMaterial deletes a study material.
// Only the uploader or an admin can delete.
func (s *Server) handleDeleteStudyMaterial(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	materialID, err := pathUUID(r, "materialId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	material, err := s.store.GetStudyMaterial(ctx, materialID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	// Only the uploader or an admin can delete
	isOwner := material.UploadedBy == identity.UserID
	isAdmin := identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal)

	if !isOwner && !isAdmin {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	// Delete associated file if it exists
	if material.FileID != "" {
		file, err := s.store.GetFile(ctx, material.FileID)
		if err == nil {
			_ = s.storage.Delete(file.StorageKey)
		}
	}

	// Delete the study material record
	if err := s.store.DeleteStudyMaterial(ctx, materialID); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionDelete,
		EntityType: "study_material",
		EntityID:   materialID,
		Summary:    "Deleted study material: " + material.TitleEN,
	})

	httpx.NoContent(w)
}
