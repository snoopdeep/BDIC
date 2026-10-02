package store

import (
	"context"
	"fmt"
)

// Dates are read out of Postgres already formatted as YYYY-MM-DD text rather
// than as time.Time. A school date is a calendar day, not an instant, and
// carrying it as a timestamp is how a date ends up a day out after a time-zone
// conversion somewhere between the database and a parent's phone.

// School is the single row describing the institution.
type School struct {
	ID             string `json:"id"`
	NameEN         string `json:"nameEn"`
	NameHI         string `json:"nameHi"`
	ShortName      string `json:"shortName"`
	AffiliationNo  string `json:"affiliationNo"`
	UDISECode      string `json:"udiseCode"`
	Board          string `json:"board"`
	AddressEN      string `json:"addressEn"`
	AddressHI      string `json:"addressHi"`
	Village        string `json:"village"`
	District       string `json:"district"`
	State          string `json:"state"`
	Pincode        string `json:"pincode"`
	PhonePrimary   string `json:"phonePrimary"`
	PhoneSecondary string `json:"phoneSecondary"`
	Email          string `json:"email"`
	OfficeHoursEN  string `json:"officeHoursEn"`
	OfficeHoursHI  string `json:"officeHoursHi"`
	PrincipalName  string `json:"principalName"`
	MapEmbedURL    string `json:"mapEmbedUrl"`
	InstagramURL   string `json:"instagramUrl"`
	FacebookURL    string `json:"facebookUrl"`
	GooglePlaceURL string `json:"googlePlaceUrl"`
	LogoFileID     string `json:"logoFileId,omitempty"`
}

// GetSchool reads the school row. There is exactly one, enforced by a unique
// index, so this never has to choose.
func (s *Store) GetSchool(ctx context.Context) (School, error) {
	const query = `
		SELECT id::text,
		       name_en,
		       name_hi,
		       short_name,
		       COALESCE(affiliation_no, ''),
		       COALESCE(udise_code, ''),
		       board,
		       COALESCE(address_en, ''),
		       COALESCE(address_hi, ''),
		       COALESCE(village, ''),
		       COALESCE(district, ''),
		       COALESCE(state, ''),
		       COALESCE(pincode, ''),
		       COALESCE(phone_primary, ''),
		       COALESCE(phone_secondary, ''),
		       COALESCE(email, ''),
		       COALESCE(office_hours_en, ''),
		       COALESCE(office_hours_hi, ''),
		       COALESCE(principal_name, ''),
		       COALESCE(map_embed_url, ''),
		       COALESCE(instagram_url, ''),
		       COALESCE(facebook_url, ''),
		       COALESCE(google_place_url, ''),
		       COALESCE(logo_file_id::text, '')
		  FROM schools
		 LIMIT 1`

	var school School
	err := s.pool.QueryRow(ctx, query).Scan(
		&school.ID, &school.NameEN, &school.NameHI, &school.ShortName,
		&school.AffiliationNo, &school.UDISECode, &school.Board,
		&school.AddressEN, &school.AddressHI, &school.Village,
		&school.District, &school.State, &school.Pincode,
		&school.PhonePrimary, &school.PhoneSecondary, &school.Email,
		&school.OfficeHoursEN, &school.OfficeHoursHI, &school.PrincipalName,
		&school.MapEmbedURL, &school.InstagramURL, &school.FacebookURL,
		&school.GooglePlaceURL, &school.LogoFileID,
	)
	if err != nil {
		return School{}, fmt.Errorf("get school: %w", noRows(err))
	}
	return school, nil
}

// UpdateSchool saves the editable school details.
func (s *Store) UpdateSchool(ctx context.Context, school School) error {
	const query = `
		UPDATE schools
		   SET name_en          = $2,
		       name_hi          = $3,
		       short_name       = $4,
		       affiliation_no   = NULLIF($5, ''),
		       udise_code       = NULLIF($6, ''),
		       board            = $7,
		       address_en       = NULLIF($8, ''),
		       address_hi       = NULLIF($9, ''),
		       village          = NULLIF($10, ''),
		       district         = NULLIF($11, ''),
		       state            = NULLIF($12, ''),
		       pincode          = NULLIF($13, ''),
		       phone_primary    = NULLIF($14, ''),
		       phone_secondary  = NULLIF($15, ''),
		       email            = NULLIF($16, ''),
		       office_hours_en  = NULLIF($17, ''),
		       office_hours_hi  = NULLIF($18, ''),
		       principal_name   = NULLIF($19, ''),
		       map_embed_url    = NULLIF($20, ''),
		       instagram_url    = NULLIF($21, ''),
		       facebook_url     = NULLIF($22, ''),
		       google_place_url = NULLIF($23, ''),
		       updated_at       = now()
		 WHERE id = $1::uuid`

	tag, err := s.pool.Exec(ctx, query,
		school.ID, school.NameEN, school.NameHI, school.ShortName,
		school.AffiliationNo, school.UDISECode, school.Board,
		school.AddressEN, school.AddressHI, school.Village,
		school.District, school.State, school.Pincode,
		school.PhonePrimary, school.PhoneSecondary, school.Email,
		school.OfficeHoursEN, school.OfficeHoursHI, school.PrincipalName,
		school.MapEmbedURL, school.InstagramURL, school.FacebookURL,
		school.GooglePlaceURL,
	)
	if err != nil {
		return fmt.Errorf("update school: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AcademicSession is one school year.
type AcademicSession struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	StartDate string `json:"startDate"`
	EndDate   string `json:"endDate"`
	IsCurrent bool   `json:"isCurrent"`
}

// CurrentSession returns the session marked current. A partial unique index
// guarantees there is at most one, and the seed guarantees there is one.
func (s *Store) CurrentSession(ctx context.Context) (AcademicSession, error) {
	const query = `
		SELECT id::text,
		       name,
		       to_char(start_date, 'YYYY-MM-DD'),
		       to_char(end_date,   'YYYY-MM-DD'),
		       is_current
		  FROM academic_sessions
		 WHERE is_current
		 LIMIT 1`

	var session AcademicSession
	err := s.pool.QueryRow(ctx, query).Scan(
		&session.ID, &session.Name, &session.StartDate, &session.EndDate, &session.IsCurrent)
	if err != nil {
		return AcademicSession{}, fmt.Errorf("current session: %w", noRows(err))
	}
	return session, nil
}

// GetAcademicSession loads one academic session by ID. Receipt and certificate
// number series use its human-readable name as their period, rather than an
// opaque UUID that an accounts clerk cannot reconcile with a fee book.
func (s *Store) GetAcademicSession(ctx context.Context, sessionID string) (AcademicSession, error) {
	const query = `
		SELECT id::text,
		       name,
		       to_char(start_date, 'YYYY-MM-DD'),
		       to_char(end_date,   'YYYY-MM-DD'),
		       is_current
		  FROM academic_sessions
		 WHERE id = $1::uuid`

	var session AcademicSession
	err := s.pool.QueryRow(ctx, query, sessionID).Scan(
		&session.ID, &session.Name, &session.StartDate, &session.EndDate, &session.IsCurrent)
	if err != nil {
		return AcademicSession{}, fmt.Errorf("get academic session: %w", noRows(err))
	}
	return session, nil
}

// ListSessions returns every session, newest first.
func (s *Store) ListSessions(ctx context.Context) ([]AcademicSession, error) {
	const query = `
		SELECT id::text,
		       name,
		       to_char(start_date, 'YYYY-MM-DD'),
		       to_char(end_date,   'YYYY-MM-DD'),
		       is_current
		  FROM academic_sessions
		 ORDER BY start_date DESC`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	sessions := []AcademicSession{}
	for rows.Next() {
		var item AcademicSession
		if err := rows.Scan(&item.ID, &item.Name, &item.StartDate, &item.EndDate, &item.IsCurrent); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}
		sessions = append(sessions, item)
	}
	return sessions, rows.Err()
}

// Stream is an Intermediate stream: Science, Commerce, or Arts.
type Stream struct {
	ID     string `json:"id"`
	Code   string `json:"code"`
	NameEN string `json:"nameEn"`
	NameHI string `json:"nameHi"`
}

// ListStreams returns the streams in display order.
func (s *Store) ListStreams(ctx context.Context) ([]Stream, error) {
	const query = `
		SELECT id::text, code, name_en, name_hi
		  FROM streams
		 ORDER BY sort_order, name_en`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list streams: %w", err)
	}
	defer rows.Close()

	streams := []Stream{}
	for rows.Next() {
		var item Stream
		if err := rows.Scan(&item.ID, &item.Code, &item.NameEN, &item.NameHI); err != nil {
			return nil, fmt.Errorf("scan stream: %w", err)
		}
		streams = append(streams, item)
	}
	return streams, rows.Err()
}

// Section is one division of a class, with its stream where the class has one.
type Section struct {
	ID         string `json:"id"`
	ClassID    string `json:"classId"`
	Name       string `json:"name"`
	StreamID   string `json:"streamId,omitempty"`
	StreamName string `json:"streamName,omitempty"`
	Capacity   int    `json:"capacity"`
	Enrolled   int    `json:"enrolled"`
}

// Class is one class, with its sections attached.
type Class struct {
	ID        string    `json:"id"`
	Code      string    `json:"code"`
	NameEN    string    `json:"nameEn"`
	NameHI    string    `json:"nameHi"`
	Level     int       `json:"level"`
	HasStream bool      `json:"hasStream"`
	Sections  []Section `json:"sections"`
}

// ListClasses returns every class with its sections and each section's current
// head count, which is what the office needs to answer "are there seats left".
func (s *Store) ListClasses(ctx context.Context, sessionID string) ([]Class, error) {
	const classQuery = `
		SELECT id::text, code, name_en, name_hi, level, has_stream
		  FROM classes
		 ORDER BY sort_order, level`

	rows, err := s.pool.Query(ctx, classQuery)
	if err != nil {
		return nil, fmt.Errorf("list classes: %w", err)
	}
	defer rows.Close()

	classes := []Class{}
	index := map[string]int{}
	for rows.Next() {
		var item Class
		if err := rows.Scan(&item.ID, &item.Code, &item.NameEN, &item.NameHI, &item.Level, &item.HasStream); err != nil {
			return nil, fmt.Errorf("scan class: %w", err)
		}
		item.Sections = []Section{}
		index[item.ID] = len(classes)
		classes = append(classes, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// The enrolment count is a correlated subquery rather than a join with a
	// GROUP BY, so a section with nobody in it still comes back with zero
	// instead of disappearing.
	const sectionQuery = `
		SELECT sec.id::text,
		       sec.class_id::text,
		       sec.name,
		       COALESCE(sec.stream_id::text, ''),
		       COALESCE(str.name_en, ''),
		       sec.capacity,
		       (SELECT count(*)
		          FROM enrollments e
		         WHERE e.section_id = sec.id
		           AND e.session_id = $1::uuid
		           AND e.status     = 'ENROLLED')::int
		  FROM sections sec
		  LEFT JOIN streams str ON str.id = sec.stream_id
		 ORDER BY sec.sort_order, sec.name`

	sectionRows, err := s.pool.Query(ctx, sectionQuery, sessionID)
	if err != nil {
		return nil, fmt.Errorf("list sections: %w", err)
	}
	defer sectionRows.Close()

	for sectionRows.Next() {
		var item Section
		if err := sectionRows.Scan(
			&item.ID, &item.ClassID, &item.Name,
			&item.StreamID, &item.StreamName, &item.Capacity, &item.Enrolled,
		); err != nil {
			return nil, fmt.Errorf("scan section: %w", err)
		}
		if position, ok := index[item.ClassID]; ok {
			classes[position].Sections = append(classes[position].Sections, item)
		}
	}
	return classes, sectionRows.Err()
}

// Subject is one teaching subject.
type Subject struct {
	ID         string `json:"id"`
	Code       string `json:"code"`
	NameEN     string `json:"nameEn"`
	NameHI     string `json:"nameHi"`
	IsLanguage bool   `json:"isLanguage"`
}

// ListSubjects returns every subject the school offers.
func (s *Store) ListSubjects(ctx context.Context) ([]Subject, error) {
	const query = `
		SELECT id::text, code, name_en, name_hi, is_language
		  FROM subjects
		 ORDER BY sort_order, name_en`

	rows, err := s.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list subjects: %w", err)
	}
	defer rows.Close()

	subjects := []Subject{}
	for rows.Next() {
		var item Subject
		if err := rows.Scan(&item.ID, &item.Code, &item.NameEN, &item.NameHI, &item.IsLanguage); err != nil {
			return nil, fmt.Errorf("scan subject: %w", err)
		}
		subjects = append(subjects, item)
	}
	return subjects, rows.Err()
}

// ClassSubject links a subject to a class, optionally to one stream within it.
type ClassSubject struct {
	ID            string `json:"id"`
	ClassID       string `json:"classId"`
	StreamID      string `json:"streamId,omitempty"`
	SubjectID     string `json:"subjectId"`
	SubjectNameEN string `json:"subjectNameEn"`
	SubjectNameHI string `json:"subjectNameHi"`
	IsCompulsory  bool   `json:"isCompulsory"`
	WeeklyPeriods int    `json:"weeklyPeriods"`
}

// ListClassSubjects returns the subjects a class runs this session.
func (s *Store) ListClassSubjects(ctx context.Context, sessionID, classID string) ([]ClassSubject, error) {
	const query = `
		SELECT cs.id::text,
		       cs.class_id::text,
		       COALESCE(cs.stream_id::text, ''),
		       cs.subject_id::text,
		       sub.name_en,
		       sub.name_hi,
		       cs.is_compulsory,
		       cs.weekly_periods
		  FROM class_subjects cs
		  JOIN subjects sub ON sub.id = cs.subject_id
		 WHERE cs.session_id = $1::uuid
		   -- NULLIF, not a bare cast: with $2 typed as text for the '' test,
		   -- a plain $2::uuid would fail on '' if Postgres ever evaluated that
		   -- side of the OR.
		   AND ($2 = '' OR cs.class_id = NULLIF($2, '')::uuid)
		 ORDER BY sub.sort_order, sub.name_en`

	rows, err := s.pool.Query(ctx, query, sessionID, classID)
	if err != nil {
		return nil, fmt.Errorf("list class subjects: %w", err)
	}
	defer rows.Close()

	items := []ClassSubject{}
	for rows.Next() {
		var item ClassSubject
		if err := rows.Scan(
			&item.ID, &item.ClassID, &item.StreamID, &item.SubjectID,
			&item.SubjectNameEN, &item.SubjectNameHI,
			&item.IsCompulsory, &item.WeeklyPeriods,
		); err != nil {
			return nil, fmt.Errorf("scan class subject: %w", err)
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

// GetSetting reads one settings row as raw JSON.
func (s *Store) GetSetting(ctx context.Context, key string) ([]byte, error) {
	var value []byte
	err := s.pool.QueryRow(ctx, `SELECT value FROM settings WHERE key = $1`, key).Scan(&value)
	if err != nil {
		return nil, noRows(err)
	}
	return value, nil
}

// SetSetting writes one settings row.
func (s *Store) SetSetting(ctx context.Context, key string, value []byte, actorUserID string) error {
	const query = `
		INSERT INTO settings (key, value, updated_by, updated_at)
		VALUES ($1, $2::jsonb, NULLIF($3, '')::uuid, now())
		ON CONFLICT (key) DO UPDATE
		   SET value      = excluded.value,
		       updated_by = excluded.updated_by,
		       updated_at = now()`
	_, err := s.pool.Exec(ctx, query, key, value, actorUserID)
	if err != nil {
		return fmt.Errorf("set setting %s: %w", key, err)
	}
	return nil
}

// AllSettings returns every setting as a map of raw JSON values, which the
// frontend reads once at start-up.
func (s *Store) AllSettings(ctx context.Context) (map[string][]byte, error) {
	rows, err := s.pool.Query(ctx, `SELECT key, value FROM settings`)
	if err != nil {
		return nil, fmt.Errorf("all settings: %w", err)
	}
	defer rows.Close()

	result := map[string][]byte{}
	for rows.Next() {
		var key string
		var value []byte
		if err := rows.Scan(&key, &value); err != nil {
			return nil, fmt.Errorf("scan setting: %w", err)
		}
		result[key] = value
	}
	return result, rows.Err()
}
