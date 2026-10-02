package store

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
)

type Homework struct {
	ID              string `json:"id"`
	SectionID       string `json:"sectionId"`
	ClassName       string `json:"className"`
	SectionName     string `json:"sectionName"`
	SubjectID       string `json:"subjectId"`
	SubjectName     string `json:"subjectName"`
	StaffID         string `json:"staffId"`
	TeacherName     string `json:"teacherName"`
	Title           string `json:"title"`
	Instructions    string `json:"instructions"`
	DueDate         string `json:"dueDate"`
	AllowUpload     bool   `json:"allowUpload"`
	Status          string `json:"status"`
	AttachmentCount int    `json:"attachmentCount"`
}

type HomeworkAttachmentInput struct {
	FileID  string
	LinkURL string
	Label   string
}

type NewHomework struct {
	SessionID     string
	SectionID     string
	SubjectID     string
	StaffID       string
	Title         string
	Instructions  string
	DueDate       string
	AllowUpload   bool
	Status        string
	Attachments   []HomeworkAttachmentInput
	CreatorUserID string
}

// TeacherMayPublishHomework verifies the teacher has that subject in the
// section timetable. A section teacher may also publish if their subject
// allocation explicitly lists the subject.
func (s *Store) TeacherMayPublishHomework(ctx context.Context, sessionID, sectionID, subjectID, staffID string) (bool, error) {
	const query = `
		SELECT EXISTS (
			SELECT 1 FROM timetable_entries
			 WHERE session_id = $1::uuid AND section_id = $2::uuid
			   AND subject_id = $3::uuid AND staff_id = $4::uuid
			UNION ALL
			SELECT 1 FROM section_teachers st
			 JOIN staff_subjects ss ON ss.staff_id = st.staff_id AND ss.subject_id = $3::uuid
			 WHERE st.session_id = $1::uuid AND st.section_id = $2::uuid AND st.staff_id = $4::uuid
		)`
	var allowed bool
	if err := s.pool.QueryRow(ctx, query, sessionID, sectionID, subjectID, staffID).Scan(&allowed); err != nil {
		return false, fmt.Errorf("check homework permission: %w", err)
	}
	return allowed, nil
}

// CreateHomework writes the work and its references atomically. Every file is
// required to belong to the publisher, preventing a teacher from attaching an
// arbitrary private file ID obtained elsewhere in the application.
func (s *Store) CreateHomework(ctx context.Context, input NewHomework) (string, error) {
	if len(input.Attachments) > 10 {
		return "", fmt.Errorf("create homework: too many attachments: %w", ErrConflict)
	}
	var homeworkID string
	err := s.InTx(ctx, func(tx pgx.Tx) error {
		for _, attachment := range input.Attachments {
			if attachment.FileID != "" {
				const ownFile = `SELECT EXISTS (SELECT 1 FROM files WHERE id = $1::uuid AND uploaded_by = $2::uuid)`
				var owned bool
				if err := tx.QueryRow(ctx, ownFile, attachment.FileID, input.CreatorUserID).Scan(&owned); err != nil {
					return fmt.Errorf("check homework attachment: %w", err)
				}
				if !owned {
					return fmt.Errorf("create homework: attachment is not owned by publisher: %w", ErrConflict)
				}
			}
		}

		const insertHomework = `
			INSERT INTO homework (session_id, section_id, subject_id, staff_id, title, instructions, due_date, allow_upload, status)
			VALUES ($1::uuid, $2::uuid, $3::uuid, $4::uuid, $5, $6, $7::date, $8, $9)
			RETURNING id::text`
		if err := tx.QueryRow(ctx, insertHomework, input.SessionID, input.SectionID, input.SubjectID, input.StaffID,
			input.Title, input.Instructions, input.DueDate, input.AllowUpload, input.Status).Scan(&homeworkID); err != nil {
			return fmt.Errorf("insert homework: %w", err)
		}
		for _, attachment := range input.Attachments {
			const insertAttachment = `
				INSERT INTO homework_attachments (homework_id, file_id, link_url, label)
				VALUES ($1::uuid, NULLIF($2, '')::uuid, NULLIF($3, ''), NULLIF($4, ''))`
			if _, err := tx.Exec(ctx, insertAttachment, homeworkID, attachment.FileID, attachment.LinkURL, attachment.Label); err != nil {
				return fmt.Errorf("insert homework attachment: %w", err)
			}
		}
		return nil
	})
	if err != nil {
		if _, unique := isUniqueViolation(err); unique {
			return "", fmt.Errorf("create homework: %w", ErrConflict)
		}
		return "", err
	}
	return homeworkID, nil
}

func (s *Store) ListHomeworkAll(ctx context.Context, sessionID string, limit, offset int) ([]Homework, error) {
	const query = homeworkSelect + `
		 WHERE h.session_id = $1::uuid
		 GROUP BY h.id, sec.id, c.id, sub.id, st.id
		 ORDER BY h.due_date DESC, h.created_at DESC
		 LIMIT $2 OFFSET $3`
	return s.listHomework(ctx, query, sessionID, limit, offset)
}

func (s *Store) ListHomeworkForStaff(ctx context.Context, sessionID, staffID string, limit, offset int) ([]Homework, error) {
	const query = homeworkSelect + `
		 WHERE h.session_id = $1::uuid AND h.staff_id = $2::uuid
		 GROUP BY h.id, sec.id, c.id, sub.id, st.id
		 ORDER BY h.due_date DESC, h.created_at DESC
		 LIMIT $3 OFFSET $4`
	return s.listHomework(ctx, query, sessionID, staffID, limit, offset)
}

func (s *Store) ListHomeworkForStudent(ctx context.Context, sessionID, studentID string, limit, offset int) ([]Homework, error) {
	const query = homeworkSelect + `
		 JOIN enrollments e ON e.section_id = h.section_id AND e.session_id = h.session_id
		 WHERE h.session_id = $1::uuid AND e.student_id = $2::uuid AND e.status = 'ENROLLED' AND h.status = 'PUBLISHED'
		 GROUP BY h.id, sec.id, c.id, sub.id, st.id
		 ORDER BY h.due_date DESC, h.created_at DESC
		 LIMIT $3 OFFSET $4`
	return s.listHomework(ctx, query, sessionID, studentID, limit, offset)
}

func (s *Store) ListHomeworkForGuardian(ctx context.Context, sessionID, guardianID string, limit, offset int) ([]Homework, error) {
	const query = homeworkSelect + `
		 JOIN enrollments e ON e.section_id = h.section_id AND e.session_id = h.session_id
		 JOIN student_guardians sg ON sg.student_id = e.student_id
		 WHERE h.session_id = $1::uuid AND sg.guardian_id = $2::uuid AND e.status = 'ENROLLED' AND h.status = 'PUBLISHED'
		 GROUP BY h.id, sec.id, c.id, sub.id, st.id
		 ORDER BY h.due_date DESC, h.created_at DESC
		 LIMIT $3 OFFSET $4`
	return s.listHomework(ctx, query, sessionID, guardianID, limit, offset)
}

const homeworkSelect = `
	SELECT h.id::text, h.section_id::text, c.name_en, sec.name, h.subject_id::text,
	       sub.name_en, h.staff_id::text, st.full_name_en, h.title, h.instructions,
	       to_char(h.due_date, 'YYYY-MM-DD'), h.allow_upload, h.status,
	       count(DISTINCT ha.id)::int
	  FROM homework h
	  JOIN sections sec ON sec.id = h.section_id
	  JOIN classes c ON c.id = sec.class_id
	  JOIN subjects sub ON sub.id = h.subject_id
	  JOIN staff st ON st.id = h.staff_id
	  LEFT JOIN homework_attachments ha ON ha.homework_id = h.id`

func (s *Store) listHomework(ctx context.Context, query string, args ...any) ([]Homework, error) {
	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list homework: %w", err)
	}
	defer rows.Close()
	items := []Homework{}
	for rows.Next() {
		var item Homework
		if err := rows.Scan(&item.ID, &item.SectionID, &item.ClassName, &item.SectionName, &item.SubjectID,
			&item.SubjectName, &item.StaffID, &item.TeacherName, &item.Title, &item.Instructions,
			&item.DueDate, &item.AllowUpload, &item.Status, &item.AttachmentCount); err != nil {
			return nil, fmt.Errorf("scan homework: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate homework: %w", err)
	}
	return items, nil
}

type HomeworkSubmission struct {
	ID          string     `json:"id"`
	HomeworkID  string     `json:"homeworkId"`
	StudentID   string     `json:"studentId"`
	StudentName string     `json:"studentName"`
	FileID      string     `json:"fileId,omitempty"`
	Note        string     `json:"note"`
	Status      string     `json:"status"`
	Remark      string     `json:"remark"`
	ReviewedAt  *time.Time `json:"reviewedAt,omitempty"`
	SubmittedAt time.Time  `json:"submittedAt"`
}

// SubmitHomework verifies the learner belongs to the work's section and that
// file uploads were enabled before inserting or replacing their own draft.
func (s *Store) SubmitHomework(ctx context.Context, homeworkID, studentID, userID, fileID, note string) (string, error) {
	if fileID != "" {
		const ownsFile = `SELECT EXISTS (SELECT 1 FROM files WHERE id = $1::uuid AND uploaded_by = $2::uuid)`
		var owned bool
		if err := s.pool.QueryRow(ctx, ownsFile, fileID, userID).Scan(&owned); err != nil {
			return "", fmt.Errorf("check submission file: %w", err)
		}
		if !owned {
			return "", fmt.Errorf("submit homework: file is not owned by student: %w", ErrConflict)
		}
	}
	const query = `
		INSERT INTO homework_submissions (homework_id, student_id, file_id, note)
		SELECT h.id, $2::uuid, NULLIF($3, '')::uuid, NULLIF($4, '')
		  FROM homework h
		  JOIN enrollments e ON e.section_id = h.section_id AND e.session_id = h.session_id
		 WHERE h.id = $1::uuid AND e.student_id = $2::uuid AND e.status = 'ENROLLED'
		   AND h.status = 'PUBLISHED' AND h.allow_upload
		ON CONFLICT (homework_id, student_id) DO UPDATE
		   SET file_id = EXCLUDED.file_id, note = EXCLUDED.note, status = 'SUBMITTED',
		       remark = NULL, reviewed_by = NULL, reviewed_at = NULL, submitted_at = now()
		 WHERE homework_submissions.reviewed_at IS NULL
		RETURNING id::text`
	var id string
	if err := s.pool.QueryRow(ctx, query, homeworkID, studentID, fileID, note).Scan(&id); err != nil {
		if err == pgx.ErrNoRows {
			return "", fmt.Errorf("submit homework: not available or already reviewed: %w", ErrConflict)
		}
		return "", fmt.Errorf("submit homework: %w", err)
	}
	return id, nil
}

// ReviewHomeworkSubmission records teacher feedback without modifying the
// original student work. Only the homework owner (or a manager) may call it.
func (s *Store) ReviewHomeworkSubmission(ctx context.Context, submissionID, reviewerUserID, reviewerStaffID, status, remark string, canManage bool) error {
	const owned = `
		UPDATE homework_submissions hs
		   SET status = $4, remark = NULLIF($5, ''), reviewed_by = $2::uuid, reviewed_at = now()
		  FROM homework h
		 WHERE hs.id = $1::uuid AND h.id = hs.homework_id
		   AND ($6 OR h.staff_id = $3::uuid)`
	tag, err := s.pool.Exec(ctx, owned, submissionID, reviewerUserID, reviewerStaffID, status, strings.TrimSpace(remark), canManage)
	if err != nil {
		return fmt.Errorf("review homework submission: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
