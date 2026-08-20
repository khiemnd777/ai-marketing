package jobs

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/riverqueue/river"
)

type SeedanceSubmitArgs struct {
	GenerationTaskID uuid.UUID `json:"generationTaskId" river:"unique"`
}

func (SeedanceSubmitArgs) Kind() string { return "seedance.task.submit" }

type SeedanceStatusArgs struct {
	GenerationTaskID uuid.UUID `json:"generationTaskId" river:"unique"`
}

func (SeedanceStatusArgs) Kind() string { return "seedance.task.status" }

type SeedanceDownloadArgs struct {
	GenerationTaskID uuid.UUID `json:"generationTaskId" river:"unique"`
}

func (SeedanceDownloadArgs) Kind() string { return "seedance.output.download" }

type TranscriptionArgs struct {
	GenerationTaskID uuid.UUID `json:"generationTaskId" river:"unique"`
}

func (TranscriptionArgs) Kind() string { return "scene.transcribe" }

type QualityCheckArgs struct {
	GenerationTaskID uuid.UUID `json:"generationTaskId" river:"unique"`
}

func (QualityCheckArgs) Kind() string { return "scene.quality-check" }

func (e *Enqueuer) EnqueueSeedanceSubmit(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	return e.insertVideo(ctx, tx, SeedanceSubmitArgs{GenerationTaskID: id}, QueueSeedanceSubmit, 1)
}

func (e *Enqueuer) EnqueueSeedanceStatus(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	return e.insertVideo(ctx, tx, SeedanceStatusArgs{GenerationTaskID: id}, QueueSeedanceStatus, 120)
}

func (e *Enqueuer) EnqueueSeedanceDownload(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	return e.insertVideo(ctx, tx, SeedanceDownloadArgs{GenerationTaskID: id}, QueueSeedanceDownload, 5)
}

func (e *Enqueuer) EnqueueTranscription(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	return e.insertVideo(ctx, tx, TranscriptionArgs{GenerationTaskID: id}, QueueTranscription, 3)
}

func (e *Enqueuer) EnqueueQualityCheck(ctx context.Context, tx pgx.Tx, id uuid.UUID) (int64, error) {
	return e.insertVideo(ctx, tx, QualityCheckArgs{GenerationTaskID: id}, QueueQualityCheck, 3)
}

func (e *Enqueuer) insertVideo(ctx context.Context, tx pgx.Tx, args river.JobArgs, queue string, attempts int) (int64, error) {
	result, err := e.client.InsertTx(ctx, tx, args, &river.InsertOpts{Queue: queue, MaxAttempts: attempts, UniqueOpts: river.UniqueOpts{ByArgs: true, ByQueue: true}})
	if err != nil {
		return 0, fmt.Errorf("enqueue %s: %w", queue, err)
	}
	return result.Job.ID, nil
}
