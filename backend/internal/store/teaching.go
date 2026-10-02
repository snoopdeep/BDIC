package store

import (
	"context"
	"fmt"
)

// TeacherClass represents a class taught by a teacher.
type TeacherClass struct {
	ID        string `json:"id"`
	ClassID   string `json:"classId"`
	Code      string `json:"code"`
	NameEN    string `json:"nameEn"`
	NameHI    string `json:"nameHi"`
	Level     int    `json:"level"`
	CreatedAt string `json:"createdAt"`
}

// GetTeacherClasses returns all classes taught by a specific teacher in a session.
func (s *Store) GetTeacherClasses(ctx context.Context, sessionID, staffID string) ([]TeacherClass, error) {
	const query = `
		SELECT DISTINCT
		       c.id::text,
		       c.code,
		       c.name_en,
		       c.name_hi,
		       c.level,
		       NOW()::text
		  FROM timetable_entries te
		  JOIN classes c ON c.id = (
		       SELECT class_id FROM sections WHERE id = te.section_id
		  )
		 WHERE te.session_id = $1::uuid
		   AND te.staff_id = $2::uuid
		UNION
		SELECT DISTINCT
		       c.id::text,
		       c.code,
		       c.name_en,
		       c.name_hi,
		       c.level,
		       NOW()::text
		  FROM section_teachers st
		  JOIN sections s ON s.id = st.section_id
		  JOIN classes c ON c.id = s.class_id
		 WHERE st.session_id = $1::uuid
		   AND st.staff_id = $2::uuid
		 ORDER BY name_en`

	rows, err := s.pool.Query(ctx, query, sessionID, staffID)
	if err != nil {
		return nil, fmt.Errorf("get teacher classes: %w", err)
	}
	defer rows.Close()

	var classes []TeacherClass
	for rows.Next() {
		var class TeacherClass
		if err := rows.Scan(&class.ID, &class.Code, &class.NameEN, &class.NameHI, &class.Level, &class.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan teacher class: %w", err)
		}
		classes = append(classes, class)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate teacher classes: %w", err)
	}

	return classes, nil
}

// IsTeacherForClass checks if a staff member teaches a specific class in a session.
func (s *Store) IsTeacherForClass(ctx context.Context, sessionID, classID, staffID string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM timetable_entries te
			 WHERE te.session_id = $1::uuid
			   AND te.staff_id = $3::uuid
			   AND te.section_id IN (
			       SELECT id FROM sections WHERE class_id = $2::uuid
			   )
			UNION ALL
			SELECT 1 FROM section_teachers st
			 WHERE st.session_id = $1::uuid
			   AND st.staff_id = $3::uuid
			   AND st.section_id IN (
			       SELECT id FROM sections WHERE class_id = $2::uuid
			   )
		)`

	var exists bool
	err := s.pool.QueryRow(ctx, query, sessionID, classID, staffID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is teacher for class: %w", noRows(err))
	}
	return exists, nil
}

// ClassStudent represents a student in a class.
type ClassStudent struct {
	ID           string `json:"id"`
	StudentID    string `json:"studentId"`
	AdmissionNo  string `json:"admissionNo"`
	FullNameEN   string `json:"fullNameEn"`
	FullNameHI   string `json:"fullNameHi"`
	RollNo       int    `json:"rollNo,omitempty"`
	EnrollmentID string `json:"enrollmentId"`
}

// GetClassStudents returns all students enrolled in a class for a session.
func (s *Store) GetClassStudents(ctx context.Context, sessionID, classID string) ([]ClassStudent, error) {
	const query = `
		SELECT st.id::text,
		       e.id::text,
		       st.admission_no,
		       st.full_name_en,
		       COALESCE(st.full_name_hi, ''),
		       e.roll_no
		  FROM enrollments e
		  JOIN students st ON st.id = e.student_id
		 WHERE e.session_id = $1::uuid
		   AND e.class_id = $2::uuid
		   AND e.status = 'ENROLLED'
		 ORDER BY e.roll_no NULLS LAST, st.full_name_en`

	rows, err := s.pool.Query(ctx, query, sessionID, classID)
	if err != nil {
		return nil, fmt.Errorf("get class students: %w", err)
	}
	defer rows.Close()

	var students []ClassStudent
	for rows.Next() {
		var student ClassStudent
		if err := rows.Scan(&student.StudentID, &student.EnrollmentID, &student.AdmissionNo,
			&student.FullNameEN, &student.FullNameHI, &student.RollNo); err != nil {
			return nil, fmt.Errorf("scan class student: %w", err)
		}
		student.ID = student.StudentID
		students = append(students, student)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate class students: %w", err)
	}

	return students, nil
}

// NewStudyMaterial is the input for creating a study material.
type NewStudyMaterial struct {
	SessionID   string
	ClassID     string
	SubjectID   string
	TitleEN     string
	TitleHI     string
	Description string
	Category    string
	FileID      string
	LinkURL     string
	UploadedBy  string
}

// StudyMaterialRecord is the full study material record from the database.
type StudyMaterialRecord struct {
	ID          string
	SessionID   string
	ClassID     string
	SubjectID   string
	TitleEN     string
	TitleHI     string
	Description string
	Category    string
	FileID      string
	LinkURL     string
	UploadedBy  string
	CreatedAt   string
}

// CreateStudyMaterial inserts a new study material record.
func (s *Store) CreateStudyMaterial(ctx context.Context, input NewStudyMaterial) (string, error) {
	const query = `
		INSERT INTO study_materials
		  (session_id, class_id, subject_id, title_en, title_hi, description, category, file_id, link_url, uploaded_by)
		VALUES
		  ($1::uuid, $2::uuid, $3::uuid, $4, $5, $6, $7, NULLIF($8, '')::uuid, NULLIF($9, ''), $10::uuid)
		RETURNING id::text`

	var materialID string
	err := s.pool.QueryRow(ctx, query,
		input.SessionID, input.ClassID, input.SubjectID,
		input.TitleEN, input.TitleHI, input.Description,
		input.Category, input.FileID, input.LinkURL, input.UploadedBy).
		Scan(&materialID)

	if err != nil {
		return "", fmt.Errorf("create study material: %w", err)
	}
	return materialID, nil
}

// GetStudyMaterial retrieves a study material by ID.
func (s *Store) GetStudyMaterial(ctx context.Context, materialID string) (StudyMaterialRecord, error) {
	const query = `
		SELECT id::text,
		       session_id::text,
		       class_id::text,
		       subject_id::text,
		       title_en,
		       COALESCE(title_hi, ''),
		       COALESCE(description, ''),
		       category,
		       COALESCE(file_id::text, ''),
		       COALESCE(link_url, ''),
		       COALESCE(uploaded_by::text, ''),
		       created_at::text
		  FROM study_materials
		 WHERE id = $1::uuid`

	var material StudyMaterialRecord
	err := s.pool.QueryRow(ctx, query, materialID).Scan(
		&material.ID, &material.SessionID, &material.ClassID, &material.SubjectID,
		&material.TitleEN, &material.TitleHI, &material.Description, &material.Category,
		&material.FileID, &material.LinkURL, &material.UploadedBy, &material.CreatedAt)

	if err != nil {
		return StudyMaterialRecord{}, fmt.Errorf("get study material: %w", noRows(err))
	}
	return material, nil
}

// ListAllStudyMaterials returns all study materials in a session (admin view).
func (s *Store) ListAllStudyMaterials(ctx context.Context, sessionID string) ([]interface{}, error) {
	const query = `
		SELECT sm.id::text,
		       sm.class_id::text,
		       sm.subject_id::text,
		       sm.title_en,
		       COALESCE(sm.title_hi, ''),
		       COALESCE(sm.description, ''),
		       sm.category,
		       COALESCE(f.original_name, ''),
		       COALESCE(sm.link_url, ''),
		       COALESCE(sm.uploaded_by::text, ''),
		       sm.created_at::text
		  FROM study_materials sm
		  LEFT JOIN files f ON f.id = sm.file_id
		 WHERE sm.session_id = $1::uuid
		 ORDER BY 11 DESC`

	rows, err := s.pool.Query(ctx, query, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list all study materials: %w", err)
	}
	defer rows.Close()

	var materials []interface{}
	for rows.Next() {
		var m map[string]interface{}
		var id, classID, subjectID, titleEn, titleHi, description, category, fileName, linkURL, uploadedBy, createdAt string

		if err := rows.Scan(&id, &classID, &subjectID, &titleEn, &titleHi, &description, &category, &fileName, &linkURL, &uploadedBy, &createdAt); err != nil {
			return nil, fmt.Errorf("scan study material: %w", err)
		}

		m = map[string]interface{}{
			"id":          id,
			"classId":     classID,
			"subjectId":   subjectID,
			"titleEn":     titleEn,
			"titleHi":     titleHi,
			"description": description,
			"category":    category,
			"fileName":    fileName,
			"linkUrl":     linkURL,
			"uploadedBy":  uploadedBy,
			"createdAt":   createdAt,
		}
		materials = append(materials, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate study materials: %w", err)
	}

	return materials, nil
}

// ListTeacherStudyMaterials returns study materials for classes taught by a teacher.
func (s *Store) ListTeacherStudyMaterials(ctx context.Context, sessionID, staffID string) ([]interface{}, error) {
	const query = `
		SELECT DISTINCT
		       sm.id::text,
		       sm.class_id::text,
		       sm.subject_id::text,
		       sm.title_en,
		       COALESCE(sm.title_hi, ''),
		       COALESCE(sm.description, ''),
		       sm.category,
		       COALESCE(f.original_name, ''),
		       COALESCE(sm.link_url, ''),
		       COALESCE(sm.uploaded_by::text, ''),
		       sm.created_at::text
		  FROM study_materials sm
		  LEFT JOIN files f ON f.id = sm.file_id
		 WHERE sm.session_id = $1::uuid
		   AND sm.class_id IN (
		       SELECT DISTINCT s.class_id
		         FROM timetable_entries te
		         JOIN sections s ON s.id = te.section_id
		        WHERE te.session_id = $1::uuid
		          AND te.staff_id = $2::uuid
		       UNION
		       SELECT DISTINCT s.class_id
		         FROM section_teachers st
		         JOIN sections s ON s.id = st.section_id
		        WHERE st.session_id = $1::uuid
		          AND st.staff_id = $2::uuid
		   )
		 ORDER BY 11 DESC`

	rows, err := s.pool.Query(ctx, query, sessionID, staffID)
	if err != nil {
		return nil, fmt.Errorf("list teacher study materials: %w", err)
	}
	defer rows.Close()

	var materials []interface{}
	for rows.Next() {
		var m map[string]interface{}
		var id, classID, subjectID, titleEn, titleHi, description, category, fileName, linkURL, uploadedBy, createdAt string

		if err := rows.Scan(&id, &classID, &subjectID, &titleEn, &titleHi, &description, &category, &fileName, &linkURL, &uploadedBy, &createdAt); err != nil {
			return nil, fmt.Errorf("scan study material: %w", err)
		}

		m = map[string]interface{}{
			"id":          id,
			"classId":     classID,
			"subjectId":   subjectID,
			"titleEn":     titleEn,
			"titleHi":     titleHi,
			"description": description,
			"category":    category,
			"fileName":    fileName,
			"linkUrl":     linkURL,
			"uploadedBy":  uploadedBy,
			"createdAt":   createdAt,
		}
		materials = append(materials, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate study materials: %w", err)
	}

	return materials, nil
}

// ListStudentStudyMaterials returns study materials for the student's class.
func (s *Store) ListStudentStudyMaterials(ctx context.Context, sessionID, studentID string) ([]interface{}, error) {
	const query = `
		SELECT sm.id::text,
		       sm.class_id::text,
		       sm.subject_id::text,
		       sm.title_en,
		       COALESCE(sm.title_hi, ''),
		       COALESCE(sm.description, ''),
		       sm.category,
		       COALESCE(f.original_name, ''),
		       COALESCE(sm.link_url, ''),
		       COALESCE(sm.uploaded_by::text, ''),
		       sm.created_at::text
		  FROM study_materials sm
		  LEFT JOIN files f ON f.id = sm.file_id
		 WHERE sm.session_id = $1::uuid
		   AND sm.class_id = (
		       SELECT class_id FROM enrollments
		        WHERE session_id = $1::uuid
		          AND student_id = $2::uuid
		          AND status = 'ENROLLED'
		   )
		 ORDER BY sm.created_at DESC`

	rows, err := s.pool.Query(ctx, query, sessionID, studentID)
	if err != nil {
		return nil, fmt.Errorf("list student study materials: %w", err)
	}
	defer rows.Close()

	var materials []interface{}
	for rows.Next() {
		var m map[string]interface{}
		var id, classID, subjectID, titleEn, titleHi, description, category, fileName, linkURL, uploadedBy, createdAt string

		if err := rows.Scan(&id, &classID, &subjectID, &titleEn, &titleHi, &description, &category, &fileName, &linkURL, &uploadedBy, &createdAt); err != nil {
			return nil, fmt.Errorf("scan study material: %w", err)
		}

		m = map[string]interface{}{
			"id":          id,
			"classId":     classID,
			"subjectId":   subjectID,
			"titleEn":     titleEn,
			"titleHi":     titleHi,
			"description": description,
			"category":    category,
			"fileName":    fileName,
			"linkUrl":     linkURL,
			"uploadedBy":  uploadedBy,
			"createdAt":   createdAt,
		}
		materials = append(materials, m)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate study materials: %w", err)
	}

	return materials, nil
}

// DeleteStudyMaterial removes a study material record.
func (s *Store) DeleteStudyMaterial(ctx context.Context, materialID string) error {
	const query = `DELETE FROM study_materials WHERE id = $1::uuid`

	result, err := s.pool.Exec(ctx, query, materialID)
	if err != nil {
		return fmt.Errorf("delete study material: %w", err)
	}

	if result.RowsAffected() == 0 {
		return ErrNotFound
	}

	return nil
}

// IsStudentInClass checks if a student is enrolled in a specific class in a session.
func (s *Store) IsStudentInClass(ctx context.Context, sessionID, classID, studentID string) (bool, error) {
	const query = `
		SELECT EXISTS(
			SELECT 1 FROM enrollments
			 WHERE session_id = $1::uuid
			   AND class_id = $2::uuid
			   AND student_id = $3::uuid
			   AND status = 'ENROLLED'
		)`

	var exists bool
	err := s.pool.QueryRow(ctx, query, sessionID, classID, studentID).Scan(&exists)
	if err != nil {
		return false, fmt.Errorf("is student in class: %w", noRows(err))
	}
	return exists, nil
}
