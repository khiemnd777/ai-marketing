package jobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type AIPlanningArgs struct {
	GenerationJobID uuid.UUID `json:"generationJobId" river:"unique"`
	ClientID        uuid.UUID `json:"clientId"`
	WorkspaceID     uuid.UUID `json:"workspaceId"`
	CampaignID      uuid.UUID `json:"campaignId"`
	Operation       string    `json:"operation"`
}

func (AIPlanningArgs) Kind() string { return "ai.planning.generate" }

func (e *Enqueuer) EnqueueAIPlanning(ctx context.Context, tx pgx.Tx, args AIPlanningArgs) (int64, error) {
	result, err := e.client.InsertTx(ctx, tx, args, &river.InsertOpts{Queue: QueueAIContent, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByArgs: true, ByQueue: true}})
	if err != nil {
		return 0, fmt.Errorf("enqueue AI planning: %w", err)
	}
	return result.Job.ID, nil
}
