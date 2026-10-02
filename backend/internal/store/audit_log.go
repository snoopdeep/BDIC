package store

import (
	"context"
	"fmt"
)

// AuditEvent deliberately omits before/after JSON from the list view. The
// summary lets leadership review activity without exposing every learner field
// in a general portal table; a detailed export can be added with a specific
// retention and access policy.
type AuditEvent struct {
	ID         int64  `json:"id"`
	ActorName  string `json:"actorName"`
	ActorRole  string `json:"actorRole"`
	Action     string `json:"action"`
	EntityType string `json:"entityType"`
	EntityID   string `json:"entityId"`
	Summary    string `json:"summary"`
	CreatedAt  string `json:"createdAt"`
}

func (s *Store) ListAuditEvents(ctx context.Context, limit, offset int) ([]AuditEvent, error) {
	const query = `
		SELECT a.id, COALESCE(u.full_name_en, 'System'), COALESCE(a.actor_role, ''),
		       a.action, a.entity_type, COALESCE(a.entity_id, ''), COALESCE(a.summary, ''),
		       to_char(a.created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF')
		  FROM audit_log a LEFT JOIN users u ON u.id = a.actor_user_id
		 ORDER BY a.created_at DESC, a.id DESC
		 LIMIT $1 OFFSET $2`
	rows, err := s.pool.Query(ctx, query, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("list audit events: %w", err)
	}
	defer rows.Close()
	items := []AuditEvent{}
	for rows.Next() {
		var item AuditEvent
		if err := rows.Scan(&item.ID, &item.ActorName, &item.ActorRole, &item.Action, &item.EntityType, &item.EntityID, &item.Summary, &item.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan audit event: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate audit events: %w", err)
	}
	return items, nil
}
