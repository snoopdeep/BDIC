package store

import (
	"context"
	"fmt"
)

// StoredFile is a row of the files table.
type StoredFile struct {
	ID           string `json:"id"`
	StorageKey   string `json:"-"` // Never sent to a client: it is the path on disk.
	OriginalName string `json:"originalName"`
	MimeType     string `json:"mimeType"`
	SizeBytes    int64  `json:"sizeBytes"`
	Visibility   string `json:"visibility"`
	UploadedBy   string `json:"uploadedBy,omitempty"`
	CreatedAt    string `json:"createdAt"`
}

// File visibility. PUBLIC files are the ones the website shows — an approved
// gallery photograph, a downloadable form. Everything else is PRIVATE and only
// reachable through a handler that has checked the caller's role.
const (
	VisibilityPrivate = "PRIVATE"
	VisibilityPublic  = "PUBLIC"
)

// NewFile is the input for recording an upload.
type NewFile struct {
	StorageKey   string
	OriginalName string
	MimeType     string
	SizeBytes    int64
	Visibility   string
	UploadedBy   string
}

// CreateFile records an uploaded file and returns its id.
func (s *Store) CreateFile(ctx context.Context, input NewFile) (string, error) {
	if input.Visibility == "" {
		input.Visibility = VisibilityPrivate
	}

	const query = `
		INSERT INTO files (
			storage_key, original_name, mime_type, size_bytes, visibility, uploaded_by
		) VALUES ($1, $2, $3, $4, $5, NULLIF($6, '')::uuid)
		RETURNING id::text`

	var id string
	err := s.pool.QueryRow(ctx, query,
		input.StorageKey, input.OriginalName, input.MimeType,
		input.SizeBytes, input.Visibility, input.UploadedBy,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("record file: %w", err)
	}
	return id, nil
}

// GetFile reads a file record by id.
func (s *Store) GetFile(ctx context.Context, fileID string) (StoredFile, error) {
	const query = `
		SELECT id::text,
		       storage_key,
		       original_name,
		       mime_type,
		       size_bytes,
		       visibility,
		       COALESCE(uploaded_by::text, ''),
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF')
		  FROM files
		 WHERE id = $1::uuid`

	var file StoredFile
	err := s.pool.QueryRow(ctx, query, fileID).Scan(
		&file.ID, &file.StorageKey, &file.OriginalName, &file.MimeType,
		&file.SizeBytes, &file.Visibility, &file.UploadedBy, &file.CreatedAt,
	)
	if err != nil {
		return StoredFile{}, noRows(err)
	}
	return file, nil
}

// SetFileVisibility flips a file between private and public. Called when a
// gallery photograph is approved, which is the moment it becomes servable.
func (s *Store) SetFileVisibility(ctx context.Context, fileID, visibility string) error {
	const query = `UPDATE files SET visibility = $2 WHERE id = $1::uuid`
	tag, err := s.pool.Exec(ctx, query, fileID, visibility)
	if err != nil {
		return fmt.Errorf("set file visibility: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FileBelongsToStudent reports whether a file is one of a student's documents
// or their photograph.
//
// This is the check that keeps one parent out of another family's paperwork.
// The download handler calls it for every PRIVATE file a parent or student
// asks for; without it, a file id from anywhere would serve any document.
func (s *Store) FileBelongsToStudent(ctx context.Context, fileID, studentID string) (bool, error) {
	const query = `
		SELECT EXISTS (
		       SELECT 1 FROM student_documents
		        WHERE file_id = $1::uuid AND student_id = $2::uuid
		       UNION ALL
		       SELECT 1 FROM students
		        WHERE photo_file_id = $1::uuid AND id = $2::uuid
		)`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, fileID, studentID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check file ownership: %w", err)
	}
	return exists, nil
}

// FileBelongsToAnyGuardianChild is the parent-side version: a guardian may open
// a file belonging to any child linked to them, and no others.
func (s *Store) FileBelongsToAnyGuardianChild(ctx context.Context, fileID, guardianID string) (bool, error) {
	const query = `
		SELECT EXISTS (
		       SELECT 1
		         FROM student_documents sd
		         JOIN student_guardians sg ON sg.student_id = sd.student_id
		        WHERE sd.file_id = $1::uuid AND sg.guardian_id = $2::uuid
		       UNION ALL
		       SELECT 1
		         FROM students s
		         JOIN student_guardians sg ON sg.student_id = s.id
		        WHERE s.photo_file_id = $1::uuid AND sg.guardian_id = $2::uuid
		)`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, fileID, guardianID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check guardian file access: %w", err)
	}
	return exists, nil
}

// FileIsAttachedToPublishedNotice reports whether a file is an attachment on a
// notice that has actually been published, which is what lets any signed-in
// user open it.
func (s *Store) FileIsAttachedToPublishedNotice(ctx context.Context, fileID string) (bool, error) {
	const query = `
		SELECT EXISTS (
		       SELECT 1
		         FROM notice_attachments na
		         JOIN notices n ON n.id = na.notice_id
		        WHERE na.file_id = $1::uuid
		          AND n.status   = 'PUBLISHED'
		          AND n.publish_at <= now()
		)`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, fileID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check notice attachment: %w", err)
	}
	return exists, nil
}

// FileIsClassMaterial reports whether a file is homework or study material for
// a class, which a student or parent in that class may open.
func (s *Store) FileIsClassMaterial(ctx context.Context, fileID, studentID string) (bool, error) {
	const query = `
		SELECT EXISTS (
		       -- Homework attachment for a section this student is enrolled in.
		       SELECT 1
		         FROM homework_attachments ha
		         JOIN homework h     ON h.id = ha.homework_id
		         JOIN enrollments e   ON e.section_id = h.section_id
		                             AND e.session_id = h.session_id
		        WHERE ha.file_id   = $1::uuid
		          AND e.student_id = $2::uuid
		       UNION ALL
		       -- Study material for this student's class.
		       SELECT 1
		         FROM study_materials sm
		         JOIN enrollments e ON e.class_id   = sm.class_id
		                           AND e.session_id = sm.session_id
		        WHERE sm.file_id   = $1::uuid
		          AND e.student_id = $2::uuid
		       UNION ALL
		       -- The student's own submission.
		       SELECT 1
		         FROM homework_submissions hs
		        WHERE hs.file_id    = $1::uuid
		          AND hs.student_id = $2::uuid
		)`

	var exists bool
	if err := s.pool.QueryRow(ctx, query, fileID, studentID).Scan(&exists); err != nil {
		return false, fmt.Errorf("check class material access: %w", err)
	}
	return exists, nil
}
