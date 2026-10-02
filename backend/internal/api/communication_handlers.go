package api

import (
	"database/sql"
	"errors"
	"fmt"
	"net/http"
	"time"

	"bdic/backend/internal/auth"
	"bdic/backend/internal/httpx"
)

// registerCommunicationRoutes wires up all communication endpoints: messages,
// notifications, and the outbound queue (for admins viewing unsent messages).
func (s *Server) registerCommunicationRoutes(mux *http.ServeMux) {
	// ---- Any signed-in user ------------------------------------------------
	// Message threads and individual messages for signed-in users.
	mux.Handle("GET /api/v1/communication/messages", s.signedIn(s.handleListMessages))
	mux.Handle("GET /api/v1/communication/messages/{id}", s.signedIn(s.handleGetMessage))
	mux.Handle("POST /api/v1/communication/messages", s.signedIn(s.handleSendMessage))
	mux.Handle("DELETE /api/v1/communication/messages/{id}", s.signedIn(s.handleDeleteMessage))

	// Notifications for all signed-in users.
	mux.Handle("GET /api/v1/communication/notifications", s.signedIn(s.handleListNotifications))

	// ---- Staff and admin only -----------------------------------------------
	// Admins can view the outbound queue (draft messages waiting to be sent).
	mux.Handle("GET /api/v1/communication/outbox",
		s.restricted(s.handleGetOutbox, auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice))
}

// Message represents a single message in a thread.
type Message struct {
	ID           string     `json:"id"`
	ThreadID     string     `json:"threadId"`
	SenderUserID string     `json:"senderUserId"`
	SenderName   string     `json:"senderName"`
	Body         string     `json:"body"`
	ReadAt       *time.Time `json:"readAt,omitempty"`
	CreatedAt    time.Time  `json:"createdAt"`
}

// MessageThread represents a conversation between a teacher, student, and guardian.
type MessageThread struct {
	ID           string    `json:"id"`
	StudentID    string    `json:"studentId"`
	StudentName  string    `json:"studentName"`
	StaffID      string    `json:"staffId"`
	StaffName    string    `json:"staffName"`
	GuardianID   string    `json:"guardianId"`
	GuardianName string    `json:"guardianName"`
	Subject      *string   `json:"subject,omitempty"`
	Status       string    `json:"status"`
	Messages     []Message `json:"messages,omitempty"`
	CreatedAt    time.Time `json:"createdAt"`
	UpdatedAt    time.Time `json:"updatedAt"`
	UnreadCount  int       `json:"unreadCount"`
}

// Notification represents a notice or alert sent to the user.
type Notification struct {
	ID             string     `json:"id"`
	NoticeID       string     `json:"noticeId"`
	Title          string     `json:"title"`
	Body           string     `json:"body"`
	MessageType    string     `json:"messageType"` // "NOTICE", "ALERT", etc.
	AudienceKind   string     `json:"audienceKind"`
	ReadAt         *time.Time `json:"readAt,omitempty"`
	RequiresAck    bool       `json:"requiresAck"`
	AcknowledgedAt *time.Time `json:"acknowledgedAt,omitempty"`
	PublishedAt    time.Time  `json:"publishedAt"`
	ExpiredAt      *time.Time `json:"expiredAt,omitempty"`
}

// OutboundMessage represents a message queued to be sent via SMS, WhatsApp, Email, or Push.
type OutboundMessage struct {
	ID              string     `json:"id"`
	Channel         string     `json:"channel"`   // SMS, WHATSAPP, EMAIL, PUSH
	Recipient       string     `json:"recipient"` // Phone number or email
	RecipientUserID string     `json:"recipientUserId"`
	RecipientName   string     `json:"recipientName"`
	Subject         *string    `json:"subject,omitempty"` // For email
	Body            string     `json:"body"`
	MessageType     string     `json:"messageType,omitempty"` // Template key or custom
	Status          string     `json:"status"`                // DRAFT, QUEUED, SENT, FAILED, CANCELLED
	Provider        *string    `json:"provider,omitempty"`
	ProviderRef     *string    `json:"providerRef,omitempty"`
	Error           *string    `json:"error,omitempty"`
	Attempts        int        `json:"attempts"`
	RelatedType     *string    `json:"relatedType,omitempty"` // Notice, Message, etc.
	RelatedID       *string    `json:"relatedId,omitempty"`
	CreatedAt       time.Time  `json:"createdAt"`
	SentAt          *time.Time `json:"sentAt,omitempty"`
}

// sendMessageRequest is the payload for creating a new message.
type sendMessageRequest struct {
	ThreadID string `json:"threadId"`
	Body     string `json:"body"`
}

// handleListMessages retrieves all message threads for the current user.
// Teachers/staff see threads for their classes, students see their own threads,
// parents see threads for their children.
func (s *Server) handleListMessages(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	limit, offset := paginate(r)

	// Build query based on user role.
	query := `
		SELECT
			mt.id, mt.student_id, st.full_name_en, st.full_name_hi,
			mt.staff_id, sf.full_name_en, sf.full_name_hi,
			mt.guardian_id, gd.full_name_en, gd.full_name_hi,
			mt.subject, mt.status, mt.created_at, mt.updated_at,
			COALESCE(unread.count, 0) as unread_count
		FROM message_threads mt
		LEFT JOIN students st ON mt.student_id = st.id
		LEFT JOIN staff sf ON mt.staff_id = sf.id
		LEFT JOIN guardians gd ON mt.guardian_id = gd.id
		LEFT JOIN (
			SELECT thread_id, COUNT(*) as count
			FROM messages
			WHERE read_at IS NULL AND sender_user_id != $1
			GROUP BY thread_id
		) unread ON mt.id = unread.thread_id
	`
	args := []interface{}{identity.UserID}

	// Filter by user role.
	switch {
	case identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice):
		// Admins see all threads.
	case identity.HasRole(auth.RoleTeacher):
		// Teachers see threads where they are the staff member.
		query += " WHERE mt.staff_id = $2"
		args = append(args, identity.StaffID)
	case identity.HasRole(auth.RoleStudent):
		// Students see threads where they are the student.
		query += " WHERE mt.student_id = $2"
		args = append(args, identity.StudentID)
	case identity.HasRole(auth.RoleParent):
		// Parents see threads for their children.
		query += `
			WHERE mt.student_id IN (
				SELECT student_id FROM student_guardians
				WHERE guardian_id = $2
			)
		`
		args = append(args, identity.GuardianID)
	default:
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	query += fmt.Sprintf(
		" ORDER BY mt.updated_at DESC LIMIT $%d OFFSET $%d",
		len(args)+1,
		len(args)+2,
	)
	args = append(args, limit, offset)

	rows, err := s.store.Pool().Query(ctx, query, args...)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	defer rows.Close()

	threads := []MessageThread{}
	for rows.Next() {
		var thread MessageThread
		var studentNameEN, studentNameHI string
		var staffNameEN, staffNameHI string
		var guardianNameEN, guardianNameHI string

		err = rows.Scan(
			&thread.ID, &thread.StudentID, &studentNameEN, &studentNameHI,
			&thread.StaffID, &staffNameEN, &staffNameHI,
			&thread.GuardianID, &guardianNameEN, &guardianNameHI,
			&thread.Subject, &thread.Status, &thread.CreatedAt, &thread.UpdatedAt,
			&thread.UnreadCount,
		)
		if err != nil {
			httpx.Fail(w, r, storeError(err))
			return
		}

		// Use English name, fallback to Hindi if empty.
		thread.StudentName = studentNameEN
		if thread.StudentName == "" {
			thread.StudentName = studentNameHI
		}
		thread.StaffName = staffNameEN
		if thread.StaffName == "" {
			thread.StaffName = staffNameHI
		}
		thread.GuardianName = guardianNameEN
		if thread.GuardianName == "" {
			thread.GuardianName = guardianNameHI
		}

		threads = append(threads, thread)
	}

	if err = rows.Err(); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"threads": threads,
		"count":   len(threads),
		"limit":   limit,
		"offset":  offset,
	})
}

// handleGetMessage retrieves a single message and its entire thread.
func (s *Server) handleGetMessage(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	threadID, err := pathUUID(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	// Verify user has access to this thread.
	accessQuery := `
		SELECT mt.id
		FROM message_threads mt
		WHERE mt.id = $1 AND (
			mt.staff_id = $2
			OR mt.student_id IN (
				SELECT id FROM students WHERE users_id = $2
			)
			OR mt.guardian_id IN (
				SELECT id FROM guardians WHERE users_id = $2
			)
		)
	`

	var threadExists string
	err = s.store.Pool().QueryRow(ctx, accessQuery, threadID, identity.UserID).Scan(&threadExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Fail(w, r, httpx.ErrNotFound())
		} else {
			httpx.Fail(w, r, storeError(err))
		}
		return
	}

	// Fetch the thread with all messages.
	threadQuery := `
		SELECT
			mt.id, mt.student_id, st.full_name_en, st.full_name_hi,
			mt.staff_id, sf.full_name_en, sf.full_name_hi,
			mt.guardian_id, gd.full_name_en, gd.full_name_hi,
			mt.subject, mt.status, mt.created_at, mt.updated_at
		FROM message_threads mt
		LEFT JOIN students st ON mt.student_id = st.id
		LEFT JOIN staff sf ON mt.staff_id = sf.id
		LEFT JOIN guardians gd ON mt.guardian_id = gd.id
		WHERE mt.id = $1
	`

	var thread MessageThread
	var studentNameEN, studentNameHI string
	var staffNameEN, staffNameHI string
	var guardianNameEN, guardianNameHI string

	err = s.store.Pool().QueryRow(ctx, threadQuery, threadID).Scan(
		&thread.ID, &thread.StudentID, &studentNameEN, &studentNameHI,
		&thread.StaffID, &staffNameEN, &staffNameHI,
		&thread.GuardianID, &guardianNameEN, &guardianNameHI,
		&thread.Subject, &thread.Status, &thread.CreatedAt, &thread.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Fail(w, r, httpx.ErrNotFound())
		} else {
			httpx.Fail(w, r, storeError(err))
		}
		return
	}

	thread.StudentName = studentNameEN
	if thread.StudentName == "" {
		thread.StudentName = studentNameHI
	}
	thread.StaffName = staffNameEN
	if thread.StaffName == "" {
		thread.StaffName = staffNameHI
	}
	thread.GuardianName = guardianNameEN
	if thread.GuardianName == "" {
		thread.GuardianName = guardianNameHI
	}

	// Fetch all messages in this thread.
	messagesQuery := `
		SELECT
			m.id, m.thread_id, m.sender_user_id,
			COALESCE(u.full_name_en, u.full_name_hi, 'Unknown') as sender_name,
			m.body, m.read_at, m.created_at
		FROM messages m
		LEFT JOIN users u ON m.sender_user_id = u.id
		WHERE m.thread_id = $1
		ORDER BY m.created_at ASC
	`

	messageRows, err := s.store.Pool().Query(ctx, messagesQuery, threadID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	defer messageRows.Close()

	thread.Messages = []Message{}
	for messageRows.Next() {
		var msg Message
		err = messageRows.Scan(
			&msg.ID, &msg.ThreadID, &msg.SenderUserID, &msg.SenderName,
			&msg.Body, &msg.ReadAt, &msg.CreatedAt,
		)
		if err != nil {
			httpx.Fail(w, r, storeError(err))
			return
		}
		thread.Messages = append(thread.Messages, msg)
	}

	if err = messageRows.Err(); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	// Mark all messages as read for the current user.
	markReadQuery := `
		UPDATE messages
		SET read_at = NOW()
		WHERE thread_id = $1 AND sender_user_id != $2 AND read_at IS NULL
	`
	if _, err = s.store.Pool().Exec(ctx, markReadQuery, threadID, identity.UserID); err != nil {
		s.logger.Error("failed to mark messages as read", "error", err)
	}

	// Calculate unread count.
	unreadQuery := `
		SELECT COALESCE(COUNT(*), 0)
		FROM messages
		WHERE thread_id = $1 AND read_at IS NULL AND sender_user_id != $2
	`
	err = s.store.Pool().QueryRow(ctx, unreadQuery, threadID, identity.UserID).Scan(&thread.UnreadCount)
	if err != nil {
		s.logger.Error("failed to count unread messages", "error", err)
	}

	httpx.JSON(w, http.StatusOK, thread)
}

// handleSendMessage creates a new message in a thread. Only teachers, staff, and admin can send messages.
func (s *Server) handleSendMessage(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	// Only staff and admins can send messages.
	if !identity.HasRole(auth.RoleTeacher, auth.RoleSuperAdmin, auth.RolePrincipal, auth.RoleOffice) {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	var req sendMessageRequest
	if err := httpx.DecodeJSON(w, r, &req); err != nil {
		httpx.Fail(w, r, err)
		return
	}

	if req.ThreadID == "" || req.Body == "" {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"Thread ID and message body are required.",
			"थ्रेड ID और संदेश निकाय आवश्यक हैं।"))
		return
	}

	if !auth.IsUUID(req.ThreadID) {
		httpx.Fail(w, r, httpx.ErrBadRequest(
			"That identifier is not valid.",
			"यह पहचानकर्ता मान्य नहीं है।"))
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	// Verify the thread exists and the user is a participant.
	threadCheckQuery := `
		SELECT id FROM message_threads
		WHERE id = $1 AND staff_id = $2
	`

	var threadExists string
	err := s.store.Pool().QueryRow(ctx, threadCheckQuery, req.ThreadID, identity.StaffID).Scan(&threadExists)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Fail(w, r, httpx.ErrForbidden().WithInternal(err))
		} else {
			httpx.Fail(w, r, storeError(err))
		}
		return
	}

	// Insert the new message.
	insertQuery := `
		INSERT INTO messages (thread_id, sender_user_id, body, created_at)
		VALUES ($1, $2, $3, NOW())
		RETURNING id, thread_id, sender_user_id, body, read_at, created_at
	`

	var msg Message
	err = s.store.Pool().QueryRow(ctx, insertQuery, req.ThreadID, identity.UserID, req.Body).
		Scan(&msg.ID, &msg.ThreadID, &msg.SenderUserID, &msg.Body, &msg.ReadAt, &msg.CreatedAt)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	msg.SenderName = identity.FullNameEN
	if msg.SenderName == "" {
		msg.SenderName = identity.FullNameHI
	}

	// Update thread's updated_at timestamp.
	updateThreadQuery := `UPDATE message_threads SET updated_at = NOW() WHERE id = $1`
	if _, err = s.store.Pool().Exec(ctx, updateThreadQuery, req.ThreadID); err != nil {
		s.logger.Error("failed to update thread timestamp", "error", err)
	}

	httpx.JSON(w, http.StatusCreated, msg)
}

// handleDeleteMessage deletes a message. Only the sender or an admin can delete.
func (s *Server) handleDeleteMessage(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())

	messageID, err := pathUUID(r, "id")
	if err != nil {
		httpx.Fail(w, r, err)
		return
	}

	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	// Check if the user is the sender or an admin.
	query := `
		SELECT sender_user_id FROM messages WHERE id = $1
	`

	var senderID string
	err = s.store.Pool().QueryRow(ctx, query, messageID).Scan(&senderID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			httpx.Fail(w, r, httpx.ErrNotFound())
		} else {
			httpx.Fail(w, r, storeError(err))
		}
		return
	}

	if senderID != identity.UserID && !identity.HasRole(auth.RoleSuperAdmin, auth.RolePrincipal) {
		httpx.Fail(w, r, httpx.ErrForbidden())
		return
	}

	// Delete the message.
	deleteQuery := `DELETE FROM messages WHERE id = $1`
	result, err := s.store.Pool().Exec(ctx, deleteQuery, messageID)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	if result.RowsAffected() == 0 {
		httpx.Fail(w, r, httpx.ErrNotFound())
		return
	}

	httpx.NoContent(w)
}

// handleListNotifications retrieves all notices and notifications for the current user.
func (s *Server) handleListNotifications(w http.ResponseWriter, r *http.Request) {
	identity := auth.MustFromContext(r.Context())
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	limit, offset := paginate(r)

	// Fetch notices relevant to the user based on their role.
	query := `
		SELECT
			n.id, n.id as notice_id,
			COALESCE(n.title_en, n.title_hi) as title,
			COALESCE(n.body_en, n.body_hi) as body,
			'NOTICE' as message_type,
			n.audience_kind,
			nr.read_at,
			n.requires_ack,
			nr.acknowledged_at,
			n.publish_at,
			n.expire_at
		FROM notices n
		LEFT JOIN notice_receipts nr ON n.id = nr.notice_id AND nr.user_id = $1
		WHERE n.status = 'PUBLISHED'
		AND (
			n.is_public = true
			OR $1 IN (SELECT user_id FROM notice_receipts WHERE notice_id = n.id)
			OR n.audience_kind = 'WHOLE_SCHOOL'
			OR (n.audience_kind = 'ROLES' AND $2 = ANY(n.audience_roles))
		)
		AND (n.expire_at IS NULL OR n.expire_at > NOW())
		ORDER BY n.publish_at DESC
		LIMIT $3 OFFSET $4
	`

	rows, err := s.store.Pool().Query(ctx, query, identity.UserID, identity.Role, limit, offset)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	defer rows.Close()

	notifications := []Notification{}
	for rows.Next() {
		var notif Notification
		err = rows.Scan(
			&notif.ID, &notif.NoticeID,
			&notif.Title, &notif.Body, &notif.MessageType, &notif.AudienceKind,
			&notif.ReadAt, &notif.RequiresAck, &notif.AcknowledgedAt,
			&notif.PublishedAt, &notif.ExpiredAt,
		)
		if err != nil {
			httpx.Fail(w, r, storeError(err))
			return
		}
		notifications = append(notifications, notif)
	}

	if err = rows.Err(); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"notifications": notifications,
		"count":         len(notifications),
		"limit":         limit,
		"offset":        offset,
	})
}

// handleGetOutbox retrieves all outbound messages (draft, queued, sent, failed).
// Only accessible to staff and admin roles.
func (s *Server) handleGetOutbox(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := contextWithTimeout(r, 10*time.Second)
	defer cancel()

	limit, offset := paginate(r)

	// Optional filters.
	statusFilter := r.URL.Query().Get("status")   // DRAFT, QUEUED, SENT, FAILED, CANCELLED
	channelFilter := r.URL.Query().Get("channel") // SMS, WHATSAPP, EMAIL, PUSH

	query := `
		SELECT
			id, channel, recipient, recipient_user_id,
			subject, body, template_key,
			status, provider, provider_ref, error, attempts,
			related_type, related_id,
			created_at, sent_at
		FROM outbound_messages
		WHERE 1=1
	`

	args := []interface{}{}

	if statusFilter != "" {
		query += fmt.Sprintf(" AND status = $%d", len(args)+1)
		args = append(args, statusFilter)
	}

	if channelFilter != "" {
		query += fmt.Sprintf(" AND channel = $%d", len(args)+1)
		args = append(args, channelFilter)
	}

	query += fmt.Sprintf(
		" ORDER BY created_at DESC LIMIT $%d OFFSET $%d",
		len(args)+1,
		len(args)+2,
	)
	args = append(args, limit, offset)

	rows, err := s.store.Pool().Query(ctx, query, args...)
	if err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}
	defer rows.Close()

	outboundMessages := []OutboundMessage{}
	for rows.Next() {
		var msg OutboundMessage
		var recipientUserID sql.NullString
		var subject sql.NullString
		var provider sql.NullString
		var providerRef sql.NullString
		var errorMsg sql.NullString
		var relatedType sql.NullString
		var relatedID sql.NullString
		var sentAt sql.NullTime

		err = rows.Scan(
			&msg.ID, &msg.Channel, &msg.Recipient, &recipientUserID,
			&subject, &msg.Body, &msg.MessageType,
			&msg.Status, &provider, &providerRef, &errorMsg, &msg.Attempts,
			&relatedType, &relatedID,
			&msg.CreatedAt, &sentAt,
		)
		if err != nil {
			httpx.Fail(w, r, storeError(err))
			return
		}

		// Populate optional fields.
		if recipientUserID.Valid {
			msg.RecipientUserID = recipientUserID.String
		}
		if subject.Valid {
			msg.Subject = &subject.String
		}
		if provider.Valid {
			msg.Provider = &provider.String
		}
		if providerRef.Valid {
			msg.ProviderRef = &providerRef.String
		}
		if errorMsg.Valid {
			msg.Error = &errorMsg.String
		}
		if relatedType.Valid {
			msg.RelatedType = &relatedType.String
		}
		if relatedID.Valid {
			msg.RelatedID = &relatedID.String
		}
		if sentAt.Valid {
			msg.SentAt = &sentAt.Time
		}

		// Fetch recipient user name if available.
		if msg.RecipientUserID != "" {
			nameQuery := `
				SELECT COALESCE(full_name_en, full_name_hi, 'Unknown')
				FROM users WHERE id = $1
			`
			err = s.store.Pool().QueryRow(ctx, nameQuery, msg.RecipientUserID).Scan(&msg.RecipientName)
			if err != nil {
				msg.RecipientName = "Unknown"
			}
		}

		outboundMessages = append(outboundMessages, msg)
	}

	if err = rows.Err(); err != nil {
		httpx.Fail(w, r, storeError(err))
		return
	}

	httpx.JSON(w, http.StatusOK, map[string]interface{}{
		"messages": outboundMessages,
		"count":    len(outboundMessages),
		"limit":    limit,
		"offset":   offset,
	})
}
