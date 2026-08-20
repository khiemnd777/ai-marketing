package jobs

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type SocialPublishArgs struct {
	SocialPostID uuid.UUID `json:"socialPostId" river:"unique"`
}

func (SocialPublishArgs) Kind() string { return "meta.social.publish" }
func (e *Enqueuer) EnqueueSocialPublish(ctx context.Context, tx pgx.Tx, id uuid.UUID, scheduledAt *time.Time) (int64, error) {
	opts := &river.InsertOpts{Queue: QueueSocialPublish, MaxAttempts: 3, UniqueOpts: river.UniqueOpts{ByArgs: true, ByQueue: true}}
	if scheduledAt != nil {
		opts.ScheduledAt = *scheduledAt
	}
	result, err := e.client.InsertTx(ctx, tx, SocialPublishArgs{SocialPostID: id}, opts)
	if err != nil {
		return 0, fmt.Errorf("enqueue social publish: %w", err)
	}
	return result.Job.ID, nil
}

type MetaAdActionArgs struct {
	ActionID uuid.UUID `json:"actionId" river:"unique"`
}

func (MetaAdActionArgs) Kind() string { return "meta.ad.action" }
func (e *Enqueuer) EnqueueMetaAdAction(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	result, err := e.client.InsertTx(ctx, tx, MetaAdActionArgs{ActionID: id}, &river.InsertOpts{Queue: QueueMetaAds, MaxAttempts: 1, UniqueOpts: river.UniqueOpts{ByArgs: true, ByQueue: true}})
	if err != nil {
		return 0, fmt.Errorf("enqueue Meta ad action: %w", err)
	}
	return result.Job.ID, nil
}

type MetaMetricsSyncArgs struct {
	AdCampaignID uuid.UUID `json:"adCampaignId" river:"unique"`
}

func (MetaMetricsSyncArgs) Kind() string { return "meta.metrics.sync" }
func (e *Enqueuer) EnqueueMetaMetricsSync(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	result, err := e.client.InsertTx(ctx, tx, MetaMetricsSyncArgs{AdCampaignID: id}, &river.InsertOpts{Queue: QueueMetricsSync, MaxAttempts: 5, UniqueOpts: river.UniqueOpts{ByArgs: true, ByQueue: true, ByPeriod: time.Hour}})
	if err != nil {
		return 0, fmt.Errorf("enqueue Meta metrics sync: %w", err)
	}
	return result.Job.ID, nil
}
