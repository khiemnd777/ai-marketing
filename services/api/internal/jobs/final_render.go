package jobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type FinalRenderArgs struct {
	RenderJobID uuid.UUID `json:"renderJobId" river:"unique"`
}

func (FinalRenderArgs) Kind() string { return "video.final-render" }

func (e *Enqueuer) EnqueueFinalRender(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	result, err := e.client.InsertTx(ctx, tx, FinalRenderArgs{RenderJobID: id}, &river.InsertOpts{Queue: QueueRender, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true, ByQueue: true}})
	if err != nil {
		return 0, fmt.Errorf("enqueue final render: %w", err)
	}
	return result.Job.ID, nil
}
