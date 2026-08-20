package video

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/storage"
)

type TranscriptResult struct {
	Text      string
	Language  string
	Segments  json.RawMessage
	RequestID string
}

type Transcriber interface {
	Transcribe(context.Context, string, io.Reader, string) (TranscriptResult, error)
}

type OpenAITranscriber struct {
	config config.OpenAIConfig
	client *http.Client
}

func NewTranscriber(demo bool, cfg config.OpenAIConfig) (Transcriber, error) {
	if demo {
		return DemoTranscriber{}, nil
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	if strings.TrimSpace(cfg.TranscriptionModel) == "" {
		return nil, errors.New("OpenAI transcription model is required")
	}
	return &OpenAITranscriber{config: cfg, client: &http.Client{Timeout: maxDuration(cfg.Timeout, 2*time.Minute)}}, nil
}

func (t *OpenAITranscriber) Transcribe(ctx context.Context, filename string, content io.Reader, language string) (TranscriptResult, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("file", filename)
	if err != nil {
		return TranscriptResult{}, err
	}
	if _, err = io.Copy(part, content); err != nil {
		return TranscriptResult{}, err
	}
	_ = writer.WriteField("model", t.config.TranscriptionModel)
	_ = writer.WriteField("response_format", "verbose_json")
	if language != "" {
		_ = writer.WriteField("language", language)
	}
	if err = writer.Close(); err != nil {
		return TranscriptResult{}, err
	}
	request, err := http.NewRequestWithContext(ctx, http.MethodPost, strings.TrimRight(t.config.BaseURL, "/")+"/audio/transcriptions", &body)
	if err != nil {
		return TranscriptResult{}, err
	}
	request.Header.Set("Authorization", "Bearer "+t.config.APIKey)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	response, err := t.client.Do(request)
	if err != nil {
		return TranscriptResult{}, err
	}
	defer response.Body.Close()
	if response.StatusCode < 200 || response.StatusCode >= 300 {
		return TranscriptResult{}, fmt.Errorf("transcription provider returned HTTP %d", response.StatusCode)
	}
	var document struct {
		Text     string          `json:"text"`
		Language string          `json:"language"`
		Segments json.RawMessage `json:"segments"`
	}
	if err = json.NewDecoder(io.LimitReader(response.Body, 4<<20)).Decode(&document); err != nil {
		return TranscriptResult{}, err
	}
	if strings.TrimSpace(document.Text) == "" {
		return TranscriptResult{}, errors.New("transcription provider returned empty text")
	}
	if len(document.Segments) == 0 {
		document.Segments = json.RawMessage(`[]`)
	}
	return TranscriptResult{Text: document.Text, Language: document.Language, Segments: document.Segments, RequestID: firstHeader(response.Header, "x-request-id")}, nil
}

type DemoTranscriber struct{}

func (DemoTranscriber) Transcribe(_ context.Context, _ string, _ io.Reader, language string) (TranscriptResult, error) {
	return TranscriptResult{Language: language, Segments: json.RawMessage(`[]`), RequestID: "demo-transcription"}, nil
}

type TranscriptionWorker struct {
	river.WorkerDefaults[jobs.TranscriptionArgs]
	Pool        *pgxpool.Pool
	Store       storage.ObjectStore
	Transcriber Transcriber
	Enqueuer    *jobs.Enqueuer
}

func (w *TranscriptionWorker) Work(ctx context.Context, job *river.Job[jobs.TranscriptionArgs]) error {
	if w.Store == nil || w.Transcriber == nil {
		return errors.New("transcription worker is not configured")
	}
	var storageKey, language, expected, provider, model, status string
	err := w.Pool.QueryRow(ctx, `
		SELECT v.storage_key,cv.language,sv.dialogue,t.provider,t.model,t.status::text
		FROM scene_generation_tasks g
		JOIN campaigns c ON c.id=g.campaign_id JOIN campaign_versions cv ON cv.campaign_id=c.id AND cv.version=c.current_version
		JOIN scene_versions sv ON sv.scene_id=g.scene_id AND sv.version=g.scene_version
		JOIN media_assets a ON a.id=g.output_asset_id JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version
		JOIN scene_transcriptions t ON t.generation_task_id=g.id
		WHERE g.id=$1`, job.Args.GenerationTaskID).Scan(&storageKey, &language, &expected, &provider, &model, &status)
	if errors.Is(err, pgx.ErrNoRows) || status == "SUCCEEDED" {
		return river.JobCancel(errors.New("transcription is unavailable or complete"))
	}
	if err != nil {
		return err
	}
	if _, err = w.Pool.Exec(ctx, `UPDATE scene_transcriptions SET status='PROCESSING',updated_at=now() WHERE generation_task_id=$1 AND status='QUEUED'`, job.Args.GenerationTaskID); err != nil {
		return err
	}
	body, err := w.Store.Get(ctx, storageKey)
	if err != nil {
		return err
	}
	defer body.Close()
	result, err := w.Transcriber.Transcribe(ctx, "scene.mp4", body, language)
	if err != nil {
		_, _ = w.Pool.Exec(ctx, `UPDATE scene_transcriptions SET status='FAILED',error_code='transcription_failed',error_message='Scene audio could not be transcribed',updated_at=now() WHERE generation_task_id=$1`, job.Args.GenerationTaskID)
		return err
	}
	if provider == "demo" {
		result.Text = expected
	}
	hash := digest(result.Text)
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE scene_transcriptions SET status='SUCCEEDED',language=$2,transcript=$3,segments=$4,transcript_hash=$5,provider_request_id=NULLIF($6,''),completed_at=now(),updated_at=now() WHERE generation_task_id=$1`, job.Args.GenerationTaskID, language, result.Text, result.Segments, hash, result.RequestID)
	if err != nil {
		return err
	}
	if _, err = w.Enqueuer.EnqueueQualityCheck(ctx, tx, job.Args.GenerationTaskID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

type QualityCheckWorker struct {
	river.WorkerDefaults[jobs.QualityCheckArgs]
	Pool *pgxpool.Pool
}

func (w *QualityCheckWorker) Work(ctx context.Context, job *river.Job[jobs.QualityCheckArgs]) error {
	var expected, actual, resolution string
	var expectedDuration int32
	var generateAudio bool
	var metadata json.RawMessage
	err := w.Pool.QueryRow(ctx, `
		SELECT sv.dialogue,t.transcript,g.resolution,g.duration_seconds,g.generate_audio,v.metadata
		FROM scene_generation_tasks g JOIN scene_versions sv ON sv.scene_id=g.scene_id AND sv.version=g.scene_version
		JOIN scene_transcriptions t ON t.generation_task_id=g.id AND t.status='SUCCEEDED'
		JOIN media_assets a ON a.id=g.output_asset_id JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version
		WHERE g.id=$1 AND g.status='VALIDATING'`, job.Args.GenerationTaskID).Scan(&expected, &actual, &resolution, &expectedDuration, &generateAudio, &metadata)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(errors.New("generation is not ready for quality checks"))
	}
	if err != nil {
		return err
	}
	var document struct {
		Probe videoProbe `json:"probe"`
	}
	if err = json.Unmarshal(metadata, &document); err != nil {
		return err
	}
	expectedNormalized := normalizeTranscript(expected)
	actualNormalized := normalizeTranscript(actual)
	transcriptPass := expectedNormalized == actualNormalized
	durationPass := absolute(document.Probe.DurationMS-int64(expectedDuration)*1000) <= 750
	expectedHeight := map[string]int32{"480p": 480, "720p": 720, "1080p": 1080, "4k": 2160}[resolution]
	resolutionPass := document.Probe.Height >= expectedHeight
	audioPass := !generateAudio || document.Probe.AudioStream
	deterministicPass := transcriptPass && durationPass && resolutionPass && audioPass
	findings := []string{}
	if !transcriptPass {
		findings = append(findings, "Transcript differs from approved scene dialogue")
	}
	if !durationPass {
		findings = append(findings, "Video duration differs from the approved scene duration")
	}
	if !resolutionPass {
		findings = append(findings, "Video resolution is below the requested setting")
	}
	if !audioPass {
		findings = append(findings, "Generated audio was requested but no audio stream is present")
	}
	diff, _ := json.Marshal(map[string]any{"expected": expected, "actual": actual, "exactNormalizedMatch": transcriptPass})
	findingsJSON, _ := json.Marshal(findings)
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `UPDATE scene_quality_checks SET status='REVIEW_REQUIRED',deterministic_pass=$2,transcript_pass=$3,video_decodes=true,duration_pass=$4,resolution_pass=$5,audio_stream_present=$6,silence_warning=$7,transcript_diff=$8,findings=$9,completed_at=now(),updated_at=now() WHERE generation_task_id=$1`, job.Args.GenerationTaskID, deterministicPass, transcriptPass, durationPass, resolutionPass, document.Probe.AudioStream, generateAudio && !document.Probe.AudioStream, diff, findingsJSON)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE scene_generation_tasks SET status='REVIEW_REQUIRED',version=version+1,updated_at=now() WHERE id=$1 AND status='VALIDATING'`, job.Args.GenerationTaskID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns c SET status='SCENE_REVIEW',version=c.version+1,updated_at=now() FROM scene_generation_tasks g WHERE g.id=$1 AND c.id=g.campaign_id AND c.status='SCENES_GENERATING'`, job.Args.GenerationTaskID); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,source,safe_detail,metadata)VALUES($1,'VALIDATING','REVIEW_REQUIRED','WORKER','Automated QC completed',jsonb_build_object('deterministicPass',$2))`, job.Args.GenerationTaskID, deterministicPass)
	return firstError(err, tx.Commit(ctx))
}

func normalizeTranscript(value string) string {
	return strings.Join(strings.Fields(strings.ToLower(strings.TrimSpace(value))), " ")
}
func absolute(value int64) int64 {
	if value < 0 {
		return -value
	}
	return value
}
func maxDuration(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
func firstError(values ...error) error {
	for _, value := range values {
		if value != nil {
			return value
		}
	}
	return nil
}
