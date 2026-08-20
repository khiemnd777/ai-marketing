package metrics

import (
	"context"
	"strconv"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus"
)

type HTTPMetrics struct {
	requests *prometheus.CounterVec
	duration *prometheus.HistogramVec
}

func New(pool *pgxpool.Pool) *HTTPMetrics {
	metric := &HTTPMetrics{
		requests: prometheus.NewCounterVec(prometheus.CounterOpts{
			Namespace: "studio", Subsystem: "http", Name: "requests_total", Help: "Total HTTP requests by method, route, and status.",
		}, []string{"method", "route", "status"}),
		duration: prometheus.NewHistogramVec(prometheus.HistogramOpts{
			Namespace: "studio", Subsystem: "http", Name: "request_duration_seconds", Help: "HTTP request duration by route.",
			Buckets: []float64{0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1, 2.5, 5, 10},
		}, []string{"method", "route"}),
	}
	prometheus.MustRegister(metric.requests, metric.duration)
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: "studio", Subsystem: "database", Name: "pool_acquired_connections", Help: "Current acquired PostgreSQL connections."}, func() float64 {
		return float64(pool.Stat().AcquiredConns())
	}))
	prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: "studio", Subsystem: "database", Name: "pool_idle_connections", Help: "Current idle PostgreSQL connections."}, func() float64 {
		return float64(pool.Stat().IdleConns())
	}))
	registerDatabaseGauges(pool)
	return metric
}

func registerDatabaseGauges(pool *pgxpool.Pool) {
	gauges := []struct {
		subsystem string
		name      string
		help      string
		query     string
	}{
		{"river", "jobs_ready", "River jobs available, scheduled, retryable, or pending.", `SELECT count(*)::float8 FROM river_job WHERE state IN ('available','scheduled','retryable','pending')`},
		{"river", "jobs_running", "River jobs currently running.", `SELECT count(*)::float8 FROM river_job WHERE state='running'`},
		{"river", "jobs_discarded", "River jobs in the discarded terminal state.", `SELECT count(*)::float8 FROM river_job WHERE state='discarded'`},
		{"river", "retries_24h", "River retry attempts recorded in the last 24 hours.", `SELECT COALESCE(sum(GREATEST(attempt-1,0)),0)::float8 FROM river_job WHERE attempted_at>=now()-interval '24 hours'`},
		{"river", "oldest_ready_seconds", "Age in seconds of the oldest runnable River job.", `SELECT COALESCE(EXTRACT(EPOCH FROM(now()-min(COALESCE(scheduled_at,created_at)))),0)::float8 FROM river_job WHERE state IN ('available','scheduled','retryable','pending')`},
		{"cost", "daily_usd", "Provider cost recorded since the UTC day boundary.", `SELECT COALESCE(sum(normalized_amount_usd),0)::float8 FROM cost_records WHERE occurred_at>=date_trunc('day',now())`},
		{"cost", "monthly_usd", "Provider cost recorded since the UTC month boundary.", `SELECT COALESCE(sum(normalized_amount_usd),0)::float8 FROM cost_records WHERE occurred_at>=date_trunc('month',now())`},
		{"video", "attempt_factor", "Generation tasks divided by distinct scenes.", `SELECT COALESCE(count(*)::float8/NULLIF(count(DISTINCT scene_id),0),0) FROM scene_generation_tasks`},
		{"storage", "media_bytes", "Total verified media bytes referenced in PostgreSQL.", `SELECT COALESCE(sum(file_size_bytes),0)::float8 FROM media_asset_versions`},
		{"provider", "seedance_failures_24h", "Seedance generation failures recorded in the last 24 hours.", `SELECT count(*)::float8 FROM scene_generation_tasks WHERE status='FAILED' AND updated_at>=now()-interval '24 hours'`},
		{"provider", "seedance_latency_seconds", "Average completed Seedance provider latency in the last 24 hours.", `SELECT COALESCE(avg(EXTRACT(EPOCH FROM(provider_completed_at-submitted_at))),0)::float8 FROM scene_generation_tasks WHERE provider_completed_at IS NOT NULL AND submitted_at IS NOT NULL AND provider_completed_at>=now()-interval '24 hours'`},
		{"provider", "seedance_failure_ratio_24h", "Seedance failure ratio among terminal tasks in the last 24 hours.", `SELECT COALESCE(count(*) FILTER(WHERE status='FAILED')::float8/NULLIF(count(*),0),0) FROM scene_generation_tasks WHERE status IN ('FAILED','CANCELLED','REVIEW_REQUIRED','APPROVED','REJECTED') AND updated_at>=now()-interval '24 hours'`},
		{"provider", "render_failures_24h", "Final rendering failures recorded in the last 24 hours.", `SELECT count(*)::float8 FROM render_jobs WHERE status='FAILED' AND updated_at>=now()-interval '24 hours'`},
		{"provider", "render_latency_seconds", "Average completed final render latency in the last 24 hours.", `SELECT COALESCE(avg(EXTRACT(EPOCH FROM(completed_at-started_at))),0)::float8 FROM render_jobs WHERE completed_at IS NOT NULL AND started_at IS NOT NULL AND completed_at>=now()-interval '24 hours'`},
		{"provider", "render_failure_ratio_24h", "Final render failure ratio among terminal renders in the last 24 hours.", `SELECT COALESCE(count(*) FILTER(WHERE status='FAILED')::float8/NULLIF(count(*),0),0) FROM render_jobs WHERE status IN ('FAILED','REVIEW_REQUIRED','APPROVED','REJECTED','CANCELLED') AND updated_at>=now()-interval '24 hours'`},
		{"provider", "r2_upload_failures_24h", "Object storage upload failures recorded in the last 24 hours.", `SELECT count(*)::float8 FROM media_uploads WHERE status='FAILED' AND created_at>=now()-interval '24 hours'`},
		{"provider", "meta_publish_failures_24h", "Meta publishing failures recorded in the last 24 hours.", `SELECT count(*)::float8 FROM social_posts WHERE status IN ('FAILED','PERMANENT_FAILURE') AND updated_at>=now()-interval '24 hours'`},
		{"provider", "meta_ads_failures_24h", "Meta Ads action failures recorded in the last 24 hours.", `SELECT count(*)::float8 FROM meta_ad_actions WHERE status='FAILED' AND COALESCE(completed_at,reviewed_at,requested_at)>=now()-interval '24 hours'`},
		{"provider", "webhook_failures_24h", "Provider webhook processing failures recorded in the last 24 hours.", `SELECT ((SELECT count(*) FROM webhook_events WHERE processing_error IS NOT NULL AND received_at>=now()-interval '24 hours')+(SELECT count(*) FROM meta_webhook_events WHERE processing_error IS NOT NULL AND received_at>=now()-interval '24 hours'))::float8`},
	}
	for _, item := range gauges {
		item := item
		prometheus.MustRegister(prometheus.NewGaugeFunc(prometheus.GaugeOpts{Namespace: "studio", Subsystem: item.subsystem, Name: item.name, Help: item.help}, func() float64 {
			ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
			defer cancel()
			var value float64
			if err := pool.QueryRow(ctx, item.query).Scan(&value); err != nil {
				return 0
			}
			return value
		}))
	}
	prometheus.MustRegister(newDatabaseCollector(pool))
}

type databaseCollector struct {
	pool         *pgxpool.Pool
	providerCost *prometheus.Desc
	queueJobs    *prometheus.Desc
}

func newDatabaseCollector(pool *pgxpool.Pool) *databaseCollector {
	return &databaseCollector{
		pool:         pool,
		providerCost: prometheus.NewDesc("studio_cost_provider_daily_usd", "Daily normalized provider cost by provider.", []string{"provider"}, nil),
		queueJobs:    prometheus.NewDesc("studio_river_queue_jobs", "River jobs by queue and state.", []string{"queue", "state"}, nil),
	}
}

func (collector *databaseCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- collector.providerCost
	ch <- collector.queueJobs
}

func (collector *databaseCollector) Collect(ch chan<- prometheus.Metric) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	rows, err := collector.pool.Query(ctx, `SELECT provider,sum(normalized_amount_usd)::float8 FROM cost_records WHERE occurred_at>=date_trunc('day',now()) GROUP BY provider`)
	if err == nil {
		for rows.Next() {
			var provider string
			var value float64
			if rows.Scan(&provider, &value) == nil {
				ch <- prometheus.MustNewConstMetric(collector.providerCost, prometheus.GaugeValue, value, provider)
			}
		}
		rows.Close()
	}
	rows, err = collector.pool.Query(ctx, `SELECT queue,state::text,count(*)::float8 FROM river_job GROUP BY queue,state`)
	if err == nil {
		for rows.Next() {
			var queue, state string
			var value float64
			if rows.Scan(&queue, &state, &value) == nil {
				ch <- prometheus.MustNewConstMetric(collector.queueJobs, prometheus.GaugeValue, value, queue, state)
			}
		}
		rows.Close()
	}
}

func (metric *HTTPMetrics) Middleware(c fiber.Ctx) error {
	started := time.Now()
	err := c.Next()
	route := c.Route().Path
	if route == "" {
		route = "unmatched"
	}
	status := strconv.Itoa(c.Response().StatusCode())
	metric.requests.WithLabelValues(c.Method(), route, status).Inc()
	metric.duration.WithLabelValues(c.Method(), route).Observe(time.Since(started).Seconds())
	return err
}
