package video

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/providerconfigs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/storage"
)

type TenantResolver interface {
	Load(context.Context, uuid.UUID) (providerconfigs.Bundle, error)
}

type SubmitWorker struct {
	river.WorkerDefaults[jobs.SeedanceSubmitArgs]
	Pool     *pgxpool.Pool
	Provider Provider
	Enqueuer *jobs.Enqueuer
	Store    referenceStore
	Config   config.Config
	Resolver TenantResolver
}

type referenceStore interface {
	PresignGet(context.Context, string, time.Duration) (storage.PresignedRequest, error)
}

type submissionRecord struct {
	ClientID, WorkspaceID, CampaignID, SceneID uuid.UUID
	SceneVersionID                             uuid.UUID
	SceneVersion                               int32
	Prompt, Model, Resolution, Ratio           string
	Duration                                   int32
	GenerateAudio                              bool
	TimeoutAt                                  time.Time
}

func (w *SubmitWorker) Work(ctx context.Context, job *river.Job[jobs.SeedanceSubmitArgs]) error {
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var record submissionRecord
	err = tx.QueryRow(ctx, `
		UPDATE scene_generation_tasks g SET status='SUBMITTING',version=g.version+1,updated_at=now()
		FROM scene_versions sv
		WHERE g.id=$1 AND g.status='QUEUED' AND sv.scene_id=g.scene_id AND sv.version=g.scene_version
		RETURNING g.client_id,g.workspace_id,g.campaign_id,g.scene_id,sv.id,g.scene_version,sv.seedance_prompt,g.model,g.resolution,g.aspect_ratio,g.duration_seconds,g.generate_audio,g.timeout_at`, job.Args.GenerationTaskID).Scan(&record.ClientID, &record.WorkspaceID, &record.CampaignID, &record.SceneID, &record.SceneVersionID, &record.SceneVersion, &record.Prompt, &record.Model, &record.Resolution, &record.Ratio, &record.Duration, &record.GenerateAudio, &record.TimeoutAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(errors.New("generation is not queued"))
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,source,safe_detail)VALUES($1,'QUEUED','SUBMITTING','WORKER','Submitting once to provider')`, job.Args.GenerationTaskID); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	provider, store, seedanceConfig := w.Provider, w.Store, w.Config.Seedance
	if w.Resolver != nil {
		bundle, resolveErr := w.Resolver.Load(ctx, record.ClientID)
		if resolveErr != nil {
			return w.failSubmit(ctx, job.Args.GenerationTaskID, errors.New("client provider configuration is unavailable"))
		}
		provider, resolveErr = NewProvider(bundle.DemoMode, bundle.Seedance)
		if resolveErr != nil {
			return w.failSubmit(ctx, job.Args.GenerationTaskID, resolveErr)
		}
		resolvedStore, storeErr := storage.NewS3Store(ctx, bundle.R2)
		if storeErr != nil {
			return w.failSubmit(ctx, job.Args.GenerationTaskID, storeErr)
		}
		store, seedanceConfig = resolvedStore, bundle.Seedance
	}
	if provider == nil || store == nil {
		return w.failSubmit(ctx, job.Args.GenerationTaskID, errors.New("Seedance submit worker requires provider and object storage"))
	}

	references, err := loadReferences(ctx, w.Pool, store, record.SceneVersionID)
	if err != nil {
		return w.failSubmit(ctx, job.Args.GenerationTaskID, err)
	}
	callback := callbackURL(seedanceConfig.CallbackURL, seedanceConfig.WebhookSecret, record.ClientID)
	providerTask, err := provider.Create(ctx, CreateRequest{Prompt: record.Prompt, References: references, Model: record.Model, Resolution: record.Resolution, AspectRatio: record.Ratio, DurationSeconds: record.Duration, GenerateAudio: record.GenerateAudio, CallbackURL: callback, TimeoutSeconds: max64(1, int64(time.Until(record.TimeoutAt).Seconds()))})
	if err != nil {
		return w.failSubmit(ctx, job.Args.GenerationTaskID, err)
	}
	safe, _ := json.Marshal(providerTask.SafeResponse)
	tx, err = w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	status := normalizeProviderStatus(providerTask.Status)
	if status != "PROVIDER_QUEUED" && status != "PROVIDER_PROCESSING" && status != "SUCCEEDED" {
		status = "PROVIDER_QUEUED"
	}
	result, err := tx.Exec(ctx, `UPDATE scene_generation_tasks SET provider_task_id=$2,status=$3::scene_generation_status,sanitized_response=$4,submitted_at=now(),provider_started_at=CASE WHEN $3::text='PROVIDER_PROCESSING' THEN now() ELSE NULL END,provider_output_url=NULLIF($5,''),usage_tokens=NULLIF($6,0),provider_seed=$7,provider_fps=$8,next_poll_at=now()+$9::interval,version=version+1,updated_at=now() WHERE id=$1 AND status='SUBMITTING'`, job.Args.GenerationTaskID, providerTask.ID, status, safe, providerTask.OutputURL, providerTask.UsageTokens, providerTask.Seed, providerTask.FPS, interval(seedanceConfig.PollInterval))
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return river.JobCancel(errors.New("generation changed during submission"))
	}
	if _, err = tx.Exec(ctx, `INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,source,provider_request_id,safe_detail)VALUES($1,'SUBMITTING',$2::scene_generation_status,'WORKER',$3,'Provider task accepted')`, job.Args.GenerationTaskID, status, providerTask.ProviderRequestID); err != nil {
		return err
	}
	if status == "SUCCEEDED" {
		_, err = w.Enqueuer.EnqueueSeedanceDownload(ctx, tx, job.Args.GenerationTaskID)
	} else {
		_, err = w.Enqueuer.EnqueueSeedanceStatus(ctx, tx, job.Args.GenerationTaskID)
	}
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *SubmitWorker) failSubmit(ctx context.Context, id uuid.UUID, cause error) error {
	providerError := AsProviderError(cause)
	_, _ = w.Pool.Exec(ctx, `WITH old AS (UPDATE scene_generation_tasks SET status='FAILED',error_category=$2,error_code=$3,error_message=$4,version=version+1,updated_at=now() WHERE id=$1 AND status='SUBMITTING' RETURNING id) INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,source,safe_detail,metadata) SELECT id,'SUBMITTING','FAILED','WORKER','Provider submission failed',jsonb_build_object('category',$2,'code',$3) FROM old`, id, providerError.Category, providerError.Code, providerError.Message)
	// Create is a chargeable mutation and must never be retried automatically,
	// even when the transport outcome is ambiguous.
	return river.JobCancel(providerError)
}

type StatusWorker struct {
	river.WorkerDefaults[jobs.SeedanceStatusArgs]
	Pool     *pgxpool.Pool
	Provider Provider
	Enqueuer *jobs.Enqueuer
	Config   config.Config
	Resolver TenantResolver
}

func (w *StatusWorker) Work(ctx context.Context, job *river.Job[jobs.SeedanceStatusArgs]) error {
	var clientID uuid.UUID
	var providerTaskID, current string
	var timeoutAt time.Time
	err := w.Pool.QueryRow(ctx, `SELECT client_id,provider_task_id,status::text,timeout_at FROM scene_generation_tasks WHERE id=$1`, job.Args.GenerationTaskID).Scan(&clientID, &providerTaskID, &current, &timeoutAt)
	if errors.Is(err, pgx.ErrNoRows) || current == "CANCELLED" || current == "FAILED" || current == "REJECTED" || current == "APPROVED" {
		return river.JobCancel(errors.New("generation polling is complete"))
	}
	if err != nil {
		return err
	}
	if time.Now().After(timeoutAt) {
		_, _ = w.Pool.Exec(ctx, `WITH old AS (UPDATE scene_generation_tasks SET status='FAILED',error_category='TIMEOUT',error_code='task_timeout',error_message='Seedance task exceeded its configured deadline',version=version+1,updated_at=now() WHERE id=$1 AND status IN ('PROVIDER_QUEUED','PROVIDER_PROCESSING') RETURNING id,status) INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,source,safe_detail) SELECT id,status,'FAILED','PROVIDER_POLL','Provider task timed out' FROM old`, job.Args.GenerationTaskID)
		return river.JobCancel(errors.New("Seedance task timed out"))
	}
	provider, pollInterval := w.Provider, w.Config.Seedance.PollInterval
	if w.Resolver != nil {
		bundle, resolveErr := w.Resolver.Load(ctx, clientID)
		if resolveErr != nil {
			return river.JobCancel(errors.New("client provider configuration is unavailable"))
		}
		provider, resolveErr = NewProvider(bundle.DemoMode, bundle.Seedance)
		if resolveErr != nil {
			return river.JobCancel(resolveErr)
		}
		pollInterval = bundle.Seedance.PollInterval
	}
	if provider == nil {
		return river.JobCancel(errors.New("Seedance provider is unavailable"))
	}
	providerTask, err := provider.Get(ctx, providerTaskID)
	if err != nil {
		providerError := AsProviderError(err)
		if providerError.Retryable {
			if job.JobRow != nil && job.Attempt >= job.MaxAttempts {
				_, _ = w.Pool.Exec(ctx, `UPDATE scene_generation_tasks SET status='FAILED',error_category=$2,error_code=$3,error_message=$4,version=version+1,updated_at=now() WHERE id=$1 AND status IN ('PROVIDER_QUEUED','PROVIDER_PROCESSING')`, job.Args.GenerationTaskID, providerError.Category, providerError.Code, providerError.Message)
			}
			return err
		}
		_, _ = w.Pool.Exec(ctx, `UPDATE scene_generation_tasks SET status='FAILED',error_category=$2,error_code=$3,error_message=$4,version=version+1,updated_at=now() WHERE id=$1`, job.Args.GenerationTaskID, providerError.Category, providerError.Code, providerError.Message)
		return river.JobCancel(providerError)
	}
	status := normalizeProviderStatus(providerTask.Status)
	if status == "FAILED" {
		category := CategoryOutage
		if strings.Contains(strings.ToLower(providerTask.ErrorCode), "sensitive") || strings.Contains(strings.ToLower(providerTask.ErrorCode), "moderation") {
			category = CategoryModeration
		}
		_, _ = w.Pool.Exec(ctx, `UPDATE scene_generation_tasks SET status='FAILED',error_category=$2,error_code=NULLIF($3,''),error_message=NULLIF($4,''),sanitized_response=$5,version=version+1,updated_at=now() WHERE id=$1`, job.Args.GenerationTaskID, category, providerTask.ErrorCode, providerTask.ErrorMessage, providerTask.SafeResponse)
		return river.JobCancel(errors.New("Seedance task failed"))
	}
	if status != "PROVIDER_QUEUED" && status != "PROVIDER_PROCESSING" && status != "SUCCEEDED" {
		return fmt.Errorf("unexpected provider task status %q", providerTask.Status)
	}
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `UPDATE scene_generation_tasks SET status=$2::scene_generation_status,sanitized_response=$3,provider_output_url=NULLIF($4,''),usage_tokens=NULLIF($5,0),provider_seed=$6,provider_fps=$7,poll_count=poll_count+1,next_poll_at=CASE WHEN $2::text IN ('PROVIDER_QUEUED','PROVIDER_PROCESSING') THEN now()+$8::interval ELSE NULL END,provider_started_at=CASE WHEN $2::text='PROVIDER_PROCESSING' THEN COALESCE(provider_started_at,now()) ELSE provider_started_at END,provider_completed_at=CASE WHEN $2::text='SUCCEEDED' THEN now() ELSE provider_completed_at END,version=version+1,updated_at=now() WHERE id=$1 AND status IN ('PROVIDER_QUEUED','PROVIDER_PROCESSING')`, job.Args.GenerationTaskID, status, providerTask.SafeResponse, providerTask.OutputURL, providerTask.UsageTokens, providerTask.Seed, providerTask.FPS, interval(pollInterval))
	if err != nil {
		return err
	}
	if result.RowsAffected() == 0 {
		return river.JobCancel(errors.New("generation already advanced"))
	}
	if _, err = tx.Exec(ctx, `INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,source,provider_request_id,safe_detail)VALUES($1,$2::scene_generation_status,$3::scene_generation_status,'PROVIDER_POLL',$4,'Provider status synchronized')`, job.Args.GenerationTaskID, current, status, providerTask.ProviderRequestID); err != nil {
		return err
	}
	if status == "SUCCEEDED" {
		if _, err = w.Enqueuer.EnqueueSeedanceDownload(ctx, tx, job.Args.GenerationTaskID); err != nil {
			return err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	if status != "SUCCEEDED" {
		return river.JobSnooze(pollInterval)
	}
	return nil
}

type DownloadWorker struct {
	river.WorkerDefaults[jobs.SeedanceDownloadArgs]
	Pool       *pgxpool.Pool
	Store      storage.ObjectStore
	Enqueuer   *jobs.Enqueuer
	Config     config.Config
	HTTPClient *http.Client
	Resolver   TenantResolver
}

type downloadRecord struct {
	ClientID, WorkspaceID, CampaignID, SceneID, ProductID, CreatedBy uuid.UUID
	URL, Provider, Resolution, Ratio                                 string
	Duration                                                         int32
	GenerateAudio                                                    bool
}

func (w *DownloadWorker) Work(ctx context.Context, job *river.Job[jobs.SeedanceDownloadArgs]) error {
	var record downloadRecord
	err := w.Pool.QueryRow(ctx, `UPDATE scene_generation_tasks g SET status='DOWNLOADING',version=g.version+1,updated_at=now() FROM campaigns c WHERE g.id=$1 AND g.status='SUCCEEDED' AND c.id=g.campaign_id RETURNING g.client_id,g.workspace_id,g.campaign_id,g.scene_id,c.product_id,g.created_by,COALESCE(g.provider_output_url,''),g.provider,g.resolution,g.aspect_ratio,g.duration_seconds,g.generate_audio`, job.Args.GenerationTaskID).Scan(&record.ClientID, &record.WorkspaceID, &record.CampaignID, &record.SceneID, &record.ProductID, &record.CreatedBy, &record.URL, &record.Provider, &record.Resolution, &record.Ratio, &record.Duration, &record.GenerateAudio)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(errors.New("generation is not ready to download"))
	}
	if err != nil {
		return err
	}
	store, cfg := w.Store, w.Config
	if w.Resolver != nil {
		bundle, resolveErr := w.Resolver.Load(ctx, record.ClientID)
		if resolveErr != nil {
			return w.failDownload(ctx, job, errors.New("client provider configuration is unavailable"))
		}
		resolvedStore, storeErr := storage.NewS3Store(ctx, bundle.R2)
		if storeErr != nil {
			return w.failDownload(ctx, job, storeErr)
		}
		store = resolvedStore
		cfg.DemoMode, cfg.OpenAI, cfg.Seedance, cfg.R2 = bundle.DemoMode, bundle.OpenAI, bundle.Seedance, bundle.R2
	}
	if store == nil {
		return w.failDownload(ctx, job, errors.New("Seedance output download requires object storage"))
	}
	if err = os.MkdirAll(cfg.WorkerTempDir, 0o750); err != nil {
		return err
	}
	dir, err := os.MkdirTemp(cfg.WorkerTempDir, "seedance-")
	if err != nil {
		return err
	}
	defer os.RemoveAll(dir)
	videoFile := filepath.Join(dir, "output.mp4")
	if record.Provider == "demo" {
		err = synthesizeDemoVideo(ctx, videoFile, record.Duration, record.Resolution, record.GenerateAudio)
	} else {
		err = w.downloadProviderOutput(ctx, record.URL, videoFile)
	}
	if err != nil {
		return w.failDownload(ctx, job, err)
	}
	probe, err := probeVideo(ctx, videoFile)
	if err != nil {
		return w.failDownload(ctx, job, fmt.Errorf("validate downloaded video: %w", err))
	}
	checksum, size, err := fileDigest(videoFile)
	if err != nil {
		return err
	}
	assetID := uuid.New()
	objectKey := fmt.Sprintf("workspaces/%s/campaigns/%s/scenes/%s/generations/%s/output.mp4", record.WorkspaceID, record.CampaignID, record.SceneID, job.Args.GenerationTaskID)
	file, err := os.Open(videoFile)
	if err != nil {
		return err
	}
	err = store.Put(ctx, objectKey, "video/mp4", size, file, map[string]string{"generation-task-id": job.Args.GenerationTaskID.String(), "sha256": checksum})
	closeErr := file.Close()
	if err != nil {
		return err
	}
	if closeErr != nil {
		return closeErr
	}
	thumbnailFile := filepath.Join(dir, "thumbnail.jpg")
	thumbnailKey := ""
	if err = exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-ss", "0.1", "-i", videoFile, "-frames:v", "1", "-vf", "scale=480:-2", thumbnailFile).Run(); err == nil {
		thumbnailKey = storage.ThumbnailObjectKey(record.WorkspaceID, assetID)
		thumb, openErr := os.Open(thumbnailFile)
		if openErr != nil {
			return openErr
		}
		stat, statErr := thumb.Stat()
		if statErr != nil {
			_ = thumb.Close()
			return statErr
		}
		putErr := store.Put(ctx, thumbnailKey, "image/jpeg", stat.Size(), thumb, map[string]string{"generation-task-id": job.Args.GenerationTaskID.String()})
		_ = thumb.Close()
		if putErr != nil {
			return putErr
		}
	}
	metadata, _ := json.Marshal(map[string]any{"probe": probe, "source": record.Provider, "generationTaskId": job.Args.GenerationTaskID})
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO media_assets(id,client_id,workspace_id,product_id,campaign_id,asset_type,category,name,status,usage_rights,source_metadata,created_by,updated_by)VALUES($1,$2,$3,$4,$5,'VIDEO','seedance-output',$6,'DRAFT','AI-generated for this campaign',$7,$8,$8)`, assetID, record.ClientID, record.WorkspaceID, record.ProductID, record.CampaignID, "Seedance scene "+record.SceneID.String(), metadata, record.CreatedBy)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO media_asset_versions(media_asset_id,client_id,workspace_id,version,storage_key,original_filename,mime_type,file_extension,file_size_bytes,checksum_sha256,width,height,duration_ms,codec,bitrate_bps,thumbnail_storage_key,metadata,verified_at,created_by)VALUES($1,$2,$3,1,$4,'seedance-output.mp4','video/mp4','.mp4',$5,$6,$7,$8,$9,$10,$11,NULLIF($12,''),$13,now(),$14)`, assetID, record.ClientID, record.WorkspaceID, objectKey, size, checksum, probe.Width, probe.Height, probe.DurationMS, probe.Codec, probe.BitrateBPS, thumbnailKey, metadata, record.CreatedBy)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE scene_generation_tasks SET status='VALIDATING',provider_output_url=NULL,output_asset_id=$2,actual_cost_usd=estimated_cost_usd,downloaded_at=now(),version=version+1,updated_at=now() WHERE id=$1 AND status='DOWNLOADING'`, job.Args.GenerationTaskID, assetID)
	if err != nil {
		return err
	}
	provider := "openai"
	if cfg.DemoMode {
		provider = "demo"
	}
	_, err = tx.Exec(ctx, `INSERT INTO scene_transcriptions(generation_task_id,status,provider,model)VALUES($1,'QUEUED',$2,$3) ON CONFLICT(generation_task_id)DO NOTHING`, job.Args.GenerationTaskID, provider, cfg.OpenAI.TranscriptionModel)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO scene_quality_checks(generation_task_id,status)VALUES($1,'QUEUED') ON CONFLICT(generation_task_id)DO NOTHING`, job.Args.GenerationTaskID)
	if err != nil {
		return err
	}
	if _, err = w.Enqueuer.EnqueueTranscription(ctx, tx, job.Args.GenerationTaskID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *DownloadWorker) downloadProviderOutput(ctx context.Context, rawURL, destination string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme != "https" || parsed.Host == "" || (!strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".volces.com") && !strings.HasSuffix(strings.ToLower(parsed.Hostname()), ".bytepluses.com")) {
		return errors.New("Seedance returned an untrusted output URL")
	}
	client := w.HTTPClient
	if client == nil {
		client = &http.Client{Timeout: 2 * time.Minute}
	}
	request, _ := http.NewRequestWithContext(ctx, http.MethodGet, parsed.String(), nil)
	response, err := client.Do(request)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK {
		return fmt.Errorf("Seedance output returned HTTP %d", response.StatusCode)
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	written, copyErr := io.Copy(file, io.LimitReader(response.Body, 1_500_000_001))
	closeErr := file.Close()
	if copyErr != nil {
		return copyErr
	}
	if closeErr != nil {
		return closeErr
	}
	if written == 0 || written > 1_500_000_000 {
		return errors.New("Seedance output size is invalid")
	}
	return nil
}

func (w *DownloadWorker) failDownload(ctx context.Context, job *river.Job[jobs.SeedanceDownloadArgs], cause error) error {
	status := "SUCCEEDED"
	if job.JobRow != nil && job.Attempt >= job.MaxAttempts {
		status = "FAILED"
	}
	_, _ = w.Pool.Exec(ctx, `UPDATE scene_generation_tasks SET status=$2::scene_generation_status,error_category=CASE WHEN $2::text='FAILED' THEN 'PROVIDER_OUTAGE' ELSE NULL END,error_code=CASE WHEN $2::text='FAILED' THEN 'output_download' ELSE NULL END,error_message=CASE WHEN $2::text='FAILED' THEN 'Seedance output could not be persisted' ELSE NULL END,version=version+1,updated_at=now() WHERE id=$1 AND status='DOWNLOADING'`, job.Args.GenerationTaskID, status)
	return cause
}

type videoProbe struct {
	Width       int32  `json:"width"`
	Height      int32  `json:"height"`
	DurationMS  int64  `json:"durationMs"`
	Codec       string `json:"codec"`
	BitrateBPS  *int64 `json:"bitrateBps,omitempty"`
	AudioStream bool   `json:"audioStream"`
}

func probeVideo(ctx context.Context, filename string) (videoProbe, error) {
	output, err := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration,bit_rate:stream=codec_type,codec_name,width,height", "-of", "json", filename).Output()
	if err != nil {
		return videoProbe{}, err
	}
	var document struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int32  `json:"width"`
			Height    int32  `json:"height"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Bitrate  string `json:"bit_rate"`
		} `json:"format"`
	}
	if err = json.Unmarshal(output, &document); err != nil {
		return videoProbe{}, err
	}
	probe := videoProbe{}
	for _, stream := range document.Streams {
		if stream.CodecType == "video" {
			probe.Width, probe.Height, probe.Codec = stream.Width, stream.Height, stream.CodecName
		}
		if stream.CodecType == "audio" {
			probe.AudioStream = true
		}
	}
	var seconds float64
	_, _ = fmt.Sscan(document.Format.Duration, &seconds)
	probe.DurationMS = int64(seconds * 1000)
	var bitrate int64
	if _, scanErr := fmt.Sscan(document.Format.Bitrate, &bitrate); scanErr == nil && bitrate > 0 {
		probe.BitrateBPS = &bitrate
	}
	if probe.Width <= 0 || probe.Height <= 0 || probe.DurationMS <= 0 || probe.Codec == "" {
		return videoProbe{}, errors.New("video is missing a decodable video stream")
	}
	return probe, nil
}

func synthesizeDemoVideo(ctx context.Context, output string, seconds int32, resolution string, audio bool) error {
	size := map[string]string{"480p": "270x480", "720p": "406x720", "1080p": "608x1080", "4k": "1216x2160"}[resolution]
	if size == "" {
		size = "406x720"
	}
	args := []string{"-hide_banner", "-loglevel", "error", "-y", "-f", "lavfi", "-i", "color=c=0x17213a:s=" + size + ":r=24:d=" + fmt.Sprint(seconds)}
	if audio {
		args = append(args, "-f", "lavfi", "-i", "anullsrc=r=48000:cl=stereo", "-shortest", "-c:a", "aac")
	}
	args = append(args, "-c:v", "libx264", "-pix_fmt", "yuv420p", "-movflags", "+faststart", output)
	return exec.CommandContext(ctx, "ffmpeg", args...).Run()
}

func fileDigest(filename string) (string, int64, error) {
	file, err := os.Open(filename)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()
	hash := sha256.New()
	size, err := io.Copy(hash, file)
	return hex.EncodeToString(hash.Sum(nil)), size, err
}

func loadReferences(ctx context.Context, pool *pgxpool.Pool, store referenceStore, sceneVersionID uuid.UUID) ([]Reference, error) {
	rows, err := pool.Query(ctx, `
		SELECT role,storage_key,mime_type FROM (
		  SELECT sa.role,v.storage_key,v.mime_type,0 AS source_order,a.id
		  FROM scene_assets sa JOIN media_assets a ON a.id=sa.media_asset_id AND a.deleted_at IS NULL
		  JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version
		  WHERE sa.scene_version_id=$1
		  UNION ALL
		  SELECT CASE ca.purpose WHEN 'VOICE_REFERENCE' THEN 'AUDIO_REFERENCE' WHEN 'REFERENCE_VIDEO' THEN 'VIDEO_REFERENCE' ELSE 'CHARACTER_REFERENCE' END,
		         v.storage_key,v.mime_type,1,a.id
		  FROM scene_versions sv
		  JOIN character_assets ca ON ca.character_id IN (sv.speaker_character_id,sv.listener_character_id)
		  JOIN media_assets a ON a.id=ca.media_asset_id AND a.deleted_at IS NULL
		  JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version
		  WHERE sv.id=$1
		) refs ORDER BY source_order,role,id`, sceneVersionID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	references := []Reference{}
	for rows.Next() {
		var role, key, mime string
		if err = rows.Scan(&role, &key, &mime); err != nil {
			return nil, err
		}
		presigned, signErr := store.PresignGet(ctx, key, time.Hour)
		if signErr != nil {
			return nil, signErr
		}
		typeName := "image_url"
		if strings.HasPrefix(mime, "video/") {
			typeName = "video_url"
		} else if strings.HasPrefix(mime, "audio/") {
			typeName = "audio_url"
		}
		providerRole := "reference_image"
		if typeName == "video_url" {
			providerRole = "reference_video"
		} else if typeName == "audio_url" {
			providerRole = "reference_audio"
		}
		references = append(references, Reference{Type: typeName, URL: presigned.URL, Role: providerRole})
	}
	if err = rows.Err(); err != nil {
		return nil, err
	}
	providerRows, err := pool.Query(ctx, `SELECT provider_asset_id FROM scene_versions sv JOIN characters c ON c.id IN (sv.speaker_character_id,sv.listener_character_id) WHERE sv.id=$1 AND c.provider_asset_id LIKE 'https://%' ORDER BY c.id`, sceneVersionID)
	if err != nil {
		return nil, err
	}
	defer providerRows.Close()
	for providerRows.Next() {
		var providerAssetURL string
		if err = providerRows.Scan(&providerAssetURL); err != nil {
			return nil, err
		}
		references = append(references, Reference{Type: "image_url", URL: providerAssetURL, Role: "reference_image"})
	}
	return references, providerRows.Err()
}

func normalizeProviderStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "queued":
		return "PROVIDER_QUEUED"
	case "running", "processing":
		return "PROVIDER_PROCESSING"
	case "succeeded":
		return "SUCCEEDED"
	case "failed", "expired":
		return "FAILED"
	default:
		return ""
	}
}

func callbackURL(base, secret string, clientID uuid.UUID) string {
	if strings.TrimSpace(base) == "" || strings.TrimSpace(secret) == "" {
		return ""
	}
	parsed, err := url.Parse(base)
	if err != nil {
		return ""
	}
	query := parsed.Query()
	query.Set("token", secret)
	query.Set("clientId", clientID.String())
	parsed.RawQuery = query.Encode()
	return parsed.String()
}

func interval(duration time.Duration) string { return fmt.Sprintf("%f seconds", duration.Seconds()) }
func max64(a, b int64) int64 {
	if a > b {
		return a
	}
	return b
}
