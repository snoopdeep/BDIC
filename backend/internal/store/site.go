package store

import (
	"context"
	"fmt"
)

// PublicNotice is the intentionally small, public-safe representation of a
// notice. Audience details and acknowledgements stay inside the portal.
type PublicNotice struct {
	ID          string
	Title       string
	Description string
	CreatedDate string
}

func (s *Store) ListPublicNotices(ctx context.Context, limit, offset int) ([]PublicNotice, error) {
	const query = `
		SELECT id::text, title_en, body_en,
		       to_char(publish_at AT TIME ZONE 'Asia/Kolkata', 'YYYY-MM-DD')
		  FROM notices
		 WHERE is_public
		   AND status = 'PUBLISHED'
		   AND publish_at <= now()
		   AND (expire_at IS NULL OR expire_at > now())
		 ORDER BY publish_at DESC, created_at DESC
		 LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public notices: %w", err)
	}
	defer rows.Close()

	items := []PublicNotice{}
	for rows.Next() {
		var item PublicNotice
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.CreatedDate); err != nil {
			return nil, fmt.Errorf("scan public notice: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public notices: %w", err)
	}
	return items, nil
}

// PublicEvent contains only calendar information which the school marks as
// public. Attendance, consent, and participant information are never exposed.
type PublicEvent struct {
	ID          string
	Title       string
	Description string
	StartDate   string
	EndDate     string
	StartTime   string
	VenueEn     string
	VenueHi     string
	Category    string
	Published   bool
}

// PublicGalleryAlbum is a consent-safe preview of a gallery collection. Image
// bytes are served only after an administrator has approved an uploaded file,
// so an empty album is still useful to visitors without inventing photographs.
type PublicGalleryAlbum struct {
	ID          string
	Title       string
	Description string
	Category    string
	EventDate   string
	CreatedDate string
}

func (s *Store) ListPublicGalleryAlbums(ctx context.Context, limit, offset int) ([]PublicGalleryAlbum, error) {
	const query = `
		SELECT id::text, title_en, COALESCE(description_en, ''), category,
		       COALESCE(to_char(event_date, 'YYYY-MM-DD'), ''),
		       to_char(created_at AT TIME ZONE 'Asia/Kolkata', 'YYYY-MM-DD')
		  FROM gallery_albums
		 WHERE is_public
		 ORDER BY event_date DESC NULLS LAST, sort_order, created_at DESC
		 LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public gallery albums: %w", err)
	}
	defer rows.Close()

	items := []PublicGalleryAlbum{}
	for rows.Next() {
		var item PublicGalleryAlbum
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Category, &item.EventDate, &item.CreatedDate); err != nil {
			return nil, fmt.Errorf("scan public gallery album: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public gallery albums: %w", err)
	}
	return items, nil
}

func (s *Store) ListPublicEvents(ctx context.Context, limit, offset int) ([]PublicEvent, error) {
	const query = `
		SELECT id::text, title_en, COALESCE(description_en, ''),
		       to_char(start_date, 'YYYY-MM-DD'),
		       COALESCE(to_char(end_date, 'YYYY-MM-DD'), ''),
		       COALESCE(to_char(start_time, 'HH24:MI'), ''),
		       COALESCE(venue_en, ''), COALESCE(venue_hi, ''), category, is_public
		  FROM events
		 WHERE is_public AND start_date >= CURRENT_DATE
		 ORDER BY start_date, start_time NULLS LAST, created_at DESC
		 LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public events: %w", err)
	}
	defer rows.Close()

	items := []PublicEvent{}
	for rows.Next() {
		var item PublicEvent
		if err := rows.Scan(
			&item.ID, &item.Title, &item.Description, &item.StartDate, &item.EndDate,
			&item.StartTime, &item.VenueEn, &item.VenueHi, &item.Category, &item.Published,
		); err != nil {
			return nil, fmt.Errorf("scan public event: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public events: %w", err)
	}
	return items, nil
}

// PublicAchievement is a public-school-story record; it contains no student
// identifier or photograph URL, so there is no route around photo consent.
type PublicAchievement struct {
	ID          string
	Title       string
	Description string
	StudentName string
	Year        int
	Category    string
	MarksOrRank string
	CreatedDate string
}

func (s *Store) ListPublicAchievements(ctx context.Context, limit, offset int) ([]PublicAchievement, error) {
	const query = `
		SELECT a.id::text, a.title_en, COALESCE(a.detail_en, ''),
		       COALESCE(a.student_name, st.full_name_en, ''), a.year, a.category,
		       COALESCE(a.marks_or_rank, ''), to_char(a.created_at AT TIME ZONE 'Asia/Kolkata', 'YYYY-MM-DD')
		  FROM achievements a
		  LEFT JOIN students st ON st.id = a.student_id
		 ORDER BY a.year DESC, a.sort_order, a.created_at DESC
		 LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public achievements: %w", err)
	}
	defer rows.Close()

	items := []PublicAchievement{}
	for rows.Next() {
		var item PublicAchievement
		if err := rows.Scan(
			&item.ID, &item.Title, &item.Description, &item.StudentName, &item.Year,
			&item.Category, &item.MarksOrRank, &item.CreatedDate,
		); err != nil {
			return nil, fmt.Errorf("scan public achievement: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public achievements: %w", err)
	}
	return items, nil
}

// PublicFaculty only includes employees who have explicitly opted into the
// public directory. Contact and qualification fields are school-approved data.
type PublicFaculty struct {
	ID             string
	FullName       string
	Designation    string
	Department     string
	Email          string
	Phone          string
	Qualifications string
}

func (s *Store) ListPublicFaculty(ctx context.Context, limit, offset int) ([]PublicFaculty, error) {
	const query = `
		SELECT id::text, full_name_en, COALESCE(designation_en, ''),
		       COALESCE(department, ''), COALESCE(email, ''), COALESCE(phone, ''),
		       COALESCE(qualification, '')
		  FROM staff
		 WHERE show_on_website AND status = 'ACTIVE'
		 ORDER BY full_name_en, employee_code
		 LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public faculty: %w", err)
	}
	defer rows.Close()

	items := []PublicFaculty{}
	for rows.Next() {
		var item PublicFaculty
		if err := rows.Scan(
			&item.ID, &item.FullName, &item.Designation, &item.Department,
			&item.Email, &item.Phone, &item.Qualifications,
		); err != nil {
			return nil, fmt.Errorf("scan public faculty member: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public faculty: %w", err)
	}
	return items, nil
}

type PublicFacility struct {
	ID          string
	Title       string
	Description string
	Icon        string
}

func (s *Store) ListPublicFacilities(ctx context.Context, limit, offset int) ([]PublicFacility, error) {
	const query = `
		SELECT id::text, title_en, COALESCE(description_en, ''), COALESCE(icon, '')
		  FROM facilities
		 WHERE is_published
		 ORDER BY sort_order, title_en
		 LIMIT $1 OFFSET $2`

	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list public facilities: %w", err)
	}
	defer rows.Close()

	items := []PublicFacility{}
	for rows.Next() {
		var item PublicFacility
		if err := rows.Scan(&item.ID, &item.Title, &item.Description, &item.Icon); err != nil {
			return nil, fmt.Errorf("scan public facility: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate public facilities: %w", err)
	}
	return items, nil
}
