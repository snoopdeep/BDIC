// Package audit records who changed what.
//
// Every write that a school could later be asked to account for goes through
// here: a mark changed, a receipt cancelled, attendance edited after the day,
// a concession approved, a photograph published. The table is append-only —
// nothing in the application updates or deletes a row in it.
package audit

import (
	"context"
	"encoding/json"
	"log/slog"
	"net"
	"net/http"

	"github.com/jackc/pgx/v5/pgxpool"

	"bdic/backend/internal/auth"
)

// Actions. Kept as constants so a report can group by them reliably.
const (
	ActionCreate            = "CREATE"
	ActionUpdate            = "UPDATE"
	ActionDelete            = "DELETE"
	ActionLogin             = "LOGIN"
	ActionLoginFailed       = "LOGIN_FAILED"
	ActionLogout            = "LOGOUT"
	ActionPasswordReset     = "PASSWORD_RESET"
	ActionApprove           = "APPROVE"
	ActionReject            = "REJECT"
	ActionPublish           = "PUBLISH"
	ActionCancel            = "CANCEL"
	ActionAttendanceCorrect = "ATTENDANCE_CORRECT"
	ActionMarksEnter        = "MARKS_ENTER"
	ActionMarksSubmit       = "MARKS_SUBMIT"
	ActionResultPublish     = "RESULT_PUBLISH"
	ActionFeeCollect        = "FEE_COLLECT"
	ActionReceiptCancel     = "RECEIPT_CANCEL"
	ActionConcessionApprove = "CONCESSION_APPROVE"
	ActionAdmissionDecide   = "ADMISSION_DECIDE"
	ActionPhotoApprove      = "PHOTO_APPROVE"
	ActionExport            = "EXPORT"
)

// Entry is one audit record.
type Entry struct {
	Action      string
	EntityType  string
	EntityID    string
	Summary     string
	BeforeState any
	AfterState  any
}

// Recorder writes audit entries.
type Recorder struct {
	pool   *pgxpool.Pool
	logger *slog.Logger
}

// NewRecorder builds a Recorder.
func NewRecorder(pool *pgxpool.Pool, logger *slog.Logger) *Recorder {
	return &Recorder{pool: pool, logger: logger}
}

// Record writes an audit entry, taking the actor from the request context.
//
// A failure to write the audit row is logged but does not fail the request the
// user made. That is a deliberate trade: losing one audit row is bad, but
// refusing to save a teacher's attendance because the audit insert failed is
// worse, and the loss is visible in the log either way.
func (rec *Recorder) Record(ctx context.Context, r *http.Request, entry Entry) {
	var actorID, actorRole *string
	if identity, ok := auth.FromContext(ctx); ok {
		id := identity.UserID
		role := identity.Role
		actorID = &id
		actorRole = &role
	}

	var ip *string
	var userAgent *string
	if r != nil {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			if net.ParseIP(host) != nil {
				ip = &host
			}
		}
		if agent := r.UserAgent(); agent != "" {
			// Trim so one absurd header cannot bloat the table.
			if len(agent) > 512 {
				agent = agent[:512]
			}
			userAgent = &agent
		}
	}

	beforeJSON, err := marshalOrNil(entry.BeforeState)
	if err != nil {
		rec.logger.Warn("audit before_state not serialisable", "error", err, "action", entry.Action)
	}
	afterJSON, err := marshalOrNil(entry.AfterState)
	if err != nil {
		rec.logger.Warn("audit after_state not serialisable", "error", err, "action", entry.Action)
	}

	const query = `
		INSERT INTO audit_log (
			actor_user_id, actor_role, action, entity_type, entity_id,
			summary, before_state, after_state, ip_address, user_agent
		) VALUES (
			$1::uuid, $2, $3, $4, $5,
			$6, $7::jsonb, $8::jsonb, $9::inet, $10
		)`

	_, execErr := rec.pool.Exec(ctx, query,
		actorID, actorRole, entry.Action, entry.EntityType, nullIfEmpty(entry.EntityID),
		nullIfEmpty(entry.Summary), beforeJSON, afterJSON, ip, userAgent,
	)
	if execErr != nil {
		rec.logger.Error("audit write failed",
			"error", execErr,
			"action", entry.Action,
			"entityType", entry.EntityType,
			"entityId", entry.EntityID)
	}
}

// RecordLoginFailure notes a failed sign-in attempt. There is no actor, because
// nobody successfully identified themselves; the identifier that was tried is
// recorded in the summary so repeated attempts against one account are visible.
func (rec *Recorder) RecordLoginFailure(ctx context.Context, r *http.Request, identifier, reason string) {
	rec.Record(ctx, r, Entry{
		Action:     ActionLoginFailed,
		EntityType: "user",
		Summary:    "Failed sign-in for " + redactIdentifier(identifier) + ": " + reason,
	})
}

func marshalOrNil(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}

func nullIfEmpty(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

// redactIdentifier keeps enough of an email or phone number to recognise a
// pattern of attempts without writing the whole thing into the audit table.
func redactIdentifier(identifier string) string {
	runes := []rune(identifier)
	if len(runes) <= 4 {
		return "****"
	}
	return string(runes[:2]) + "****" + string(runes[len(runes)-2:])
}
