package api

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"

	"bdic/backend/internal/audit"
	"bdic/backend/internal/auth"
	"bdic/backend/internal/files"
	"bdic/backend/internal/httpx"
	"bdic/backend/internal/store"
)

func (s *Server) registerFileRoutes(mux *http.ServeMux) {
	// Staff upload school material. Students may upload only a pending homework
	// submission; parents' document upload goes through the admission endpoints.
	mux.Handle("POST /api/v1/files", s.signedIn(s.handleUploadFile))
	mux.Handle("GET /api/v1/files/{fileId}", s.signedIn(s.handleDownloadFile))

	// Public files: approved gallery photographs, downloadable forms, the
	// school logo. Only files explicitly marked PUBLIC are served here.
	mux.HandleFunc("GET /api/v1/public/files/{fileId}", s.handlePublicFile)
}

// handleUploadFile accepts one file and records it.
//
// The returned id is what other endpoints reference — a student's photograph, a
// homework attachment, a gallery image. Uploading and attaching are two steps
// on purpose: the upload can be retried on a bad connection without creating a
// half-made homework row.
func (s *Server) handleUploadFile(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	if !identity.IsStaff() && !identity.HasRole(auth.RoleStudent) {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	// Cap the body before parsing, so an enormous upload is refused rather
	// than buffered.
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

	category := strings.TrimSpace(r.FormValue("category"))
	if category == "" {
		category = "misc"
	}
	if identity.HasRole(auth.RoleStudent) && category != "homework-submission" {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	/*
		Sniff the type from the first 512 bytes rather than trusting the
		Content-Type the browser sent. A renamed .php or .html arriving as
		"image/jpeg" would otherwise be recorded as an image and later served
		back to another user with that type.
	*/
	head := make([]byte, 512)
	read, err := io.ReadFull(upload, head)
	if err != nil && !errors.Is(err, io.ErrUnexpectedEOF) && !errors.Is(err, io.EOF) {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	head = head[:read]

	detected := http.DetectContentType(head)
	// DetectContentType appends a charset for text types; compare the type only.
	baseType := strings.TrimSpace(strings.Split(detected, ";")[0])

	if !files.AllowedMIMETypes[baseType] {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"That kind of file is not accepted. Use a photograph, a PDF, or a spreadsheet.",
			"यह प्रकार की फ़ाइल स्वीकार नहीं है। फोटो, PDF या स्प्रेडशीट का उपयोग करें।").
			WithField("file", "Unsupported type: "+baseType))
		return
	}

	if _, err := upload.Seek(0, io.SeekStart); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	key, err := files.NewKey(category, header.Filename)
	if err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	if err := s.storage.Save(key, upload); err != nil {
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}

	fileID, err := s.store.CreateFile(r.Context(), store.NewFile{
		StorageKey:   key,
		OriginalName: safeFilename(header.Filename),
		MimeType:     baseType,
		SizeBytes:    header.Size,
		Visibility:   store.VisibilityPrivate,
		UploadedBy:   identity.UserID,
	})
	if err != nil {
		// Do not leave an orphan on disk that nothing references.
		_ = s.storage.Delete(key)
		httpx.Fail(w, r, storeError(err))
		return
	}

	s.audit.Record(r.Context(), r, audit.Entry{
		Action:     audit.ActionCreate,
		EntityType: "file",
		EntityID:   fileID,
		Summary:    "Uploaded " + safeFilename(header.Filename),
	})

	httpx.JSON(w, http.StatusCreated, map[string]any{
		"id":           fileID,
		"originalName": safeFilename(header.Filename),
		"mimeType":     baseType,
		"sizeBytes":    header.Size,
	})
}

// handleDownloadFile serves a private file, after checking the caller may see it.
//
// This is the function that keeps one family's documents away from another.
// Staff see everything; a student sees their own record and their class's
// material; a parent sees the same for each child linked to them, and nothing
// else. An id from anywhere else is a 404, not a 403 — a 403 would confirm the
// file exists.
func (s *Server) handleDownloadFile(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	fileID, err := pathUUID(r, "fileId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	record, err := s.store.GetFile(r.Context(), fileID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	allowed, err := s.mayReadFile(r, identity, record)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	if !allowed {
		httpx.Fail(w, r, httpx.ErrNotFound())
		return
	}

	s.serveFile(w, r, record)
}

// handlePublicFile serves a file the school has explicitly published. No token
// required, and nothing that is not marked PUBLIC is served here.
func (s *Server) handlePublicFile(w http.ResponseWriter, r *http.Request) {
	fileID, err := pathUUID(r, "fileId")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	record, err := s.store.GetFile(r.Context(), fileID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	if record.Visibility != store.VisibilityPublic {
		httpx.Fail(w, r, httpx.ErrNotFound())
		return
	}

	// Public assets can be cached hard: the key is random, so a changed file
	// is a new key and a new URL.
	w.Header().Set("Cache-Control", "public, max-age=604800, immutable")
	s.serveFile(w, r, record)
}

// mayReadFile decides whether this caller may open this file.
func (s *Server) mayReadFile(r *http.Request, identity auth.Identity, record store.StoredFile) (bool, error) {
	if record.Visibility == store.VisibilityPublic {
		return true, nil
	}

	// Staff are trusted with the school's own records. Which staff can reach
	// which screen is gated at the route; a file id they hold came from a
	// screen they were allowed to open.
	if identity.IsStaff() {
		return true, nil
	}

	// A notice attachment is readable by anyone signed in, once published.
	published, err := s.store.FileIsAttachedToPublishedNotice(r.Context(), record.ID)
	if err != nil {
		return false, err
	}
	if published {
		return true, nil
	}

	switch identity.Role {
	case auth.RoleStudent:
		if identity.StudentID == "" {
			return false, nil
		}
		own, err := s.store.FileBelongsToStudent(r.Context(), record.ID, identity.StudentID)
		if err != nil {
			return false, err
		}
		if own {
			return true, nil
		}
		return s.store.FileIsClassMaterial(r.Context(), record.ID, identity.StudentID)

	case auth.RoleParent:
		if identity.GuardianID == "" {
			return false, nil
		}
		return s.store.FileBelongsToAnyGuardianChild(r.Context(), record.ID, identity.GuardianID)
	}

	return false, nil
}

func (s *Server) serveFile(w http.ResponseWriter, r *http.Request, record store.StoredFile) {
	reader, err := s.storage.Open(record.StorageKey)
	if err != nil {
		if errors.Is(err, files.ErrNotStored) {
			// The row exists but the bytes do not. Worth a log line: it means
			// a backup restored the database without the uploads.
			s.logger.Error("file record has no stored contents",
				"fileId", record.ID, "key", record.StorageKey)
			httpx.Fail(w, r, httpx.ErrNotFound().WithInternal(err))
			return
		}
		httpx.Fail(w, r, httpx.ErrInternal().WithInternal(err))
		return
	}
	defer reader.Close()

	w.Header().Set("Content-Type", record.MimeType)
	w.Header().Set("Content-Length", fmt.Sprintf("%d", record.SizeBytes))
	w.Header().Set("X-Content-Type-Options", "nosniff")

	/*
		inline for images and PDFs, so a parent tapping a document sees it
		rather than downloading it. attachment for everything else, which
		makes a spreadsheet save instead of rendering.

		The filename is quoted and stripped of quotes and control characters:
		an unescaped one can forge extra header fields.
	*/
	disposition := "attachment"
	if files.IsImage(record.MimeType) || record.MimeType == "application/pdf" {
		disposition = "inline"
	}
	w.Header().Set("Content-Disposition",
		fmt.Sprintf(`%s; filename="%s"`, disposition, safeFilename(record.OriginalName)))

	if _, err := io.Copy(w, reader); err != nil {
		// The response has begun, so there is nothing to send but a log line.
		s.logger.Warn("file download interrupted", "fileId", record.ID, "error", err)
	}
}

// safeFilename strips anything that could break out of a quoted header value
// or a filesystem path.
func safeFilename(name string) string {
	cleaned := strings.Map(func(r rune) rune {
		switch {
		case r < 32, r == 127: // control characters
			return -1
		case r == '"', r == '\\', r == '/', r == ';', r == '\n', r == '\r':
			return '_'
		default:
			return r
		}
	}, name)

	cleaned = strings.TrimSpace(cleaned)
	if cleaned == "" {
		return "download"
	}
	if len(cleaned) > 120 {
		return cleaned[:120]
	}
	return cleaned
}
