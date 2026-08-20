package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/metaads"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/planning"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/database"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/logging"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/telemetry"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/providerconfigs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/publishing"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/rendering"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/video"
)

func main() {
	if err := run(); err != nil {
		slog.Error("worker stopped", "error", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	logger := logging.New(cfg.LogLevel)
	slog.SetDefault(logger)
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	pool, err := database.Open(ctx, cfg.DatabaseURL)
	if err != nil {
		return err
	}
	defer pool.Close()
	shutdownTelemetry, err := telemetry.Configure(ctx, cfg.OTLP.Endpoint, "studio-worker", cfg.Environment)
	if err != nil {
		return err
	}
	defer func() {
		shutdownContext, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = shutdownTelemetry(shutdownContext)
	}()

	workers := river.NewWorkers()
	river.AddWorker(workers, &jobs.MaintenanceWorker{Queries: db.New(pool), Pool: pool})
	secretCipher, err := cryptox.New(cfg.EncryptionKey)
	if err != nil {
		return err
	}
	providerResolver := providerconfigs.NewService(pool, secretCipher)
	river.AddWorker(workers, &jobs.MediaMetadataWorker{Pool: pool, Resolver: providerResolver, TempDir: cfg.WorkerTempDir})
	river.AddWorker(workers, &planning.Worker{Pool: pool, Resolver: providerResolver})
	jobEnqueuer, err := jobs.NewEnqueuer(pool)
	if err != nil {
		return err
	}
	river.AddWorker(workers, &video.SubmitWorker{Pool: pool, Resolver: providerResolver, Enqueuer: jobEnqueuer})
	river.AddWorker(workers, &video.StatusWorker{Pool: pool, Resolver: providerResolver, Enqueuer: jobEnqueuer})
	river.AddWorker(workers, &video.DownloadWorker{Pool: pool, Resolver: providerResolver, Enqueuer: jobEnqueuer, Config: cfg})
	river.AddWorker(workers, &video.TranscriptionWorker{Pool: pool, Resolver: providerResolver, Enqueuer: jobEnqueuer})
	river.AddWorker(workers, &video.QualityCheckWorker{Pool: pool})
	river.AddWorker(workers, &rendering.Worker{Pool: pool, Resolver: providerResolver, RendererSecret: cfg.Renderer.SharedSecret})
	river.AddWorker(workers, &publishing.Worker{Pool: pool, Resolver: providerResolver, Cipher: secretCipher})
	river.AddWorker(workers, &metaads.ActionWorker{Pool: pool, Resolver: providerResolver, Cipher: secretCipher, Enqueuer: jobEnqueuer})
	river.AddWorker(workers, &metaads.MetricsWorker{Pool: pool, Resolver: providerResolver, Cipher: secretCipher})
	queues := make(map[string]river.QueueConfig, len(jobs.RequiredQueues))
	for _, queue := range jobs.RequiredQueues {
		queues[queue] = river.QueueConfig{MaxWorkers: workerCount(queue)}
	}
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{
		Queues:  queues,
		Workers: workers,
		Logger:  logger,
		Plugins: telemetry.RiverPlugins(),
		PeriodicJobs: []*river.PeriodicJob{
			river.NewPeriodicJob(
				river.PeriodicInterval(time.Hour),
				func() (river.JobArgs, *river.InsertOpts) {
					return jobs.MaintenanceArgs{}, &river.InsertOpts{Queue: jobs.QueueMaintenance, MaxAttempts: 5}
				},
				&river.PeriodicJobOpts{RunOnStart: false},
			),
		},
	})
	if err != nil {
		return fmt.Errorf("create River client: %w", err)
	}
	if err := client.Start(ctx); err != nil {
		return fmt.Errorf("start River client: %w", err)
	}
	logger.Info("River worker started", "queues", jobs.RequiredQueues)
	<-ctx.Done()
	shutdownContext, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	defer cancel()
	return client.Stop(shutdownContext)
}

func workerCount(queue string) int {
	switch queue {
	case jobs.QueueSeedanceSubmit, jobs.QueueMetaAds, jobs.QueueSocialPublish:
		return 2
	case jobs.QueueRender, jobs.QueueSeedanceDownload, jobs.QueueMediaProcessing:
		return 3
	case jobs.QueueMaintenance:
		return 1
	default:
		return 5
	}
}
