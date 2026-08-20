package audit

import (
	"context"
	"encoding/json"
	"net/netip"

	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
)

type Event struct {
	ActorID     uuid.NullUUID
	Action      string
	EntityType  string
	EntityID    uuid.NullUUID
	ClientID    uuid.NullUUID
	WorkspaceID uuid.NullUUID
	RequestID   string
	IPAddress   *netip.Addr
	UserAgent   string
	Outcome     string
	Reason      string
	Before      any
	After       any
	Metadata    map[string]any
}

func Record(ctx context.Context, queries *db.Queries, event Event) error {
	metadata, err := json.Marshal(event.Metadata)
	if err != nil {
		return err
	}
	before, err := optionalJSON(event.Before)
	if err != nil {
		return err
	}
	after, err := optionalJSON(event.After)
	if err != nil {
		return err
	}
	var entityType *string
	if event.EntityType != "" {
		entityType = &event.EntityType
	}
	var reason *string
	if event.Reason != "" {
		reason = &event.Reason
	}
	_, err = queries.InsertAuditLog(ctx, db.InsertAuditLogParams{
		ActorInternalUserID: event.ActorID,
		Action:              event.Action,
		EntityType:          entityType,
		EntityID:            event.EntityID,
		ClientID:            event.ClientID,
		WorkspaceID:         event.WorkspaceID,
		RequestID:           event.RequestID,
		IpAddress:           event.IPAddress,
		UserAgent:           event.UserAgent,
		Outcome:             event.Outcome,
		Reason:              reason,
		BeforeData:          before,
		AfterData:           after,
		Metadata:            metadata,
	})
	return err
}

func optionalJSON(value any) ([]byte, error) {
	if value == nil {
		return nil, nil
	}
	return json.Marshal(value)
}
