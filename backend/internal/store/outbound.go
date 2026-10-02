package store

import (
	"context"
	"encoding/json"
	"fmt"
)

// Channels for outbound_messages.
const (
	ChannelSMS      = "SMS"
	ChannelWhatsApp = "WHATSAPP"
	ChannelEmail    = "EMAIL"
	ChannelPush     = "PUSH"
)

// OutboundDraft is a message the system would send if a provider were
// connected.
//
// Nothing in this build sends anything. Rows are written with status DRAFT so
// the school can open the Outbox screen and read, word for word, what every
// parent would have received. When the school opens an SMS or WhatsApp account
// and the credentials are filled into .env, a dispatcher picks DRAFT rows up.
type OutboundDraft struct {
	Channel         string
	Recipient       string
	RecipientUserID string
	TemplateKey     string
	Subject         string
	Body            string
	Payload         map[string]any
	RelatedType     string
	RelatedID       string
}

// QueueOutbound records one message as a draft and returns its id.
func (s *Store) QueueOutbound(ctx context.Context, draft OutboundDraft) (string, error) {
	payload := draft.Payload
	if payload == nil {
		payload = map[string]any{}
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return "", fmt.Errorf("encode outbound payload: %w", err)
	}

	const query = `
		INSERT INTO outbound_messages (
			channel, recipient, recipient_user_id, template_key,
			subject, body, payload, status, related_type, related_id
		) VALUES (
			$1, $2, NULLIF($3, '')::uuid, NULLIF($4, ''),
			NULLIF($5, ''), $6, $7::jsonb, 'DRAFT', NULLIF($8, ''), NULLIF($9, '')
		)
		RETURNING id::text`

	var id string
	err = s.pool.QueryRow(ctx, query,
		draft.Channel, draft.Recipient, draft.RecipientUserID, draft.TemplateKey,
		draft.Subject, draft.Body, encoded, draft.RelatedType, draft.RelatedID,
	).Scan(&id)
	if err != nil {
		return "", fmt.Errorf("queue outbound message: %w", err)
	}
	return id, nil
}

// OutboundMessage is a row of the outbox as the school reads it.
type OutboundMessage struct {
	ID          string `json:"id"`
	Channel     string `json:"channel"`
	Recipient   string `json:"recipient"`
	TemplateKey string `json:"templateKey,omitempty"`
	Subject     string `json:"subject,omitempty"`
	Body        string `json:"body"`
	Status      string `json:"status"`
	RelatedType string `json:"relatedType,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

// ListOutbound returns the outbox newest first, optionally filtered by channel
// and status.
func (s *Store) ListOutbound(ctx context.Context, channel, status string, limit, offset int) ([]OutboundMessage, int, error) {
	const countQuery = `
		SELECT count(*)
		  FROM outbound_messages
		 WHERE ($1 = '' OR channel = $1)
		   AND ($2 = '' OR status  = $2)`

	var total int
	if err := s.pool.QueryRow(ctx, countQuery, channel, status).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count outbound: %w", err)
	}

	const query = `
		SELECT id::text,
		       channel,
		       recipient,
		       COALESCE(template_key, ''),
		       COALESCE(subject, ''),
		       body,
		       status,
		       COALESCE(related_type, ''),
		       to_char(created_at, 'YYYY-MM-DD"T"HH24:MI:SSOF')
		  FROM outbound_messages
		 WHERE ($1 = '' OR channel = $1)
		   AND ($2 = '' OR status  = $2)
		 ORDER BY created_at DESC
		 LIMIT $3 OFFSET $4`

	rows, err := s.pool.Query(ctx, query, channel, status, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("list outbound: %w", err)
	}
	defer rows.Close()

	items := []OutboundMessage{}
	for rows.Next() {
		var item OutboundMessage
		if err := rows.Scan(
			&item.ID, &item.Channel, &item.Recipient, &item.TemplateKey,
			&item.Subject, &item.Body, &item.Status, &item.RelatedType, &item.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan outbound: %w", err)
		}
		items = append(items, item)
	}
	return items, total, rows.Err()
}
