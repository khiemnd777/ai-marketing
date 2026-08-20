package rendering

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/storage"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/usage"
)

type Worker struct {
	river.WorkerDefaults[jobs.FinalRenderArgs]
	Pool     *pgxpool.Pool
	Store    storage.ObjectStore
	Renderer *RendererClient
}

const finalRenderPersistenceGrace = 5 * time.Minute

// Timeout must outlive RendererClient's request timeout so a successful render
// still has time to persist its media asset, subtitles, usage, and review state.
func (*Worker) Timeout(*river.Job[jobs.FinalRenderArgs]) time.Duration {
	return rendererRequestTimeout + finalRenderPersistenceGrace
}

type manifestContext struct {
	ClientID, WorkspaceID, CampaignID, ProjectID, ActorID                                          uuid.UUID
	ProjectVersion                                                                                 int32
	ProjectHash, Language, Headline, LowerThird                                                    string
	Duration                                                                                       int32
	ShowPrice, ShowDiscount, ShowCTA, ShowWebsite, ShowPhone, ShowQR, ShowDisclaimer, BurnCaptions bool
	MusicAssetID                                                                                   *uuid.UUID
	MusicGain, DialogueDucking                                                                     float64
	ProductID, BrandID                                                                             uuid.UUID
	ProductName, CTA, Offer                                                                        string
	Currency, Discount, Website, Phone, Disclaimer                                                 *string
	RegularPrice, SalePrice                                                                        *float64
	LogoIDs                                                                                        []uuid.UUID
	CreatedAt                                                                                      time.Time
}

func (w *Worker) Work(ctx context.Context, job *river.Job[jobs.FinalRenderArgs]) error {
	if w.Store == nil || w.Renderer == nil {
		return errors.New("final render worker is not configured")
	}
	var data manifestContext
	err := w.Pool.QueryRow(ctx, `UPDATE render_jobs r SET status='BUILDING_MANIFEST',started_at=COALESCE(r.started_at,now()),version=r.version+1,updated_at=now() FROM video_projects p JOIN video_project_versions pv ON pv.video_project_id=p.id AND pv.version=p.current_version JOIN campaigns c ON c.id=p.campaign_id JOIN campaign_versions cv ON cv.campaign_id=c.id AND cv.version=c.current_version JOIN products pr ON pr.id=c.product_id JOIN product_versions prv ON prv.product_id=pr.id AND prv.version=pr.current_version JOIN brands b ON b.id=c.brand_id JOIN brand_versions bv ON bv.brand_id=b.id AND bv.version=b.current_version WHERE r.id=$1 AND r.status IN ('QUEUED','FAILED') AND p.id=r.video_project_id AND p.current_version=r.video_project_version RETURNING r.client_id,r.workspace_id,r.campaign_id,p.id,r.created_by,p.current_version,pv.project_hash,cv.language,cv.duration_seconds,pv.headline,pv.lower_third,pv.show_price,pv.show_discount_code,pv.show_cta,pv.show_website,pv.show_phone,pv.show_qr_code,pv.show_disclaimer,pv.burn_captions,p.music_asset_id,p.music_gain_db::float8,p.dialogue_ducking_db::float8,pr.id,b.id,pr.name,cv.cta,cv.offer,prv.currency,prv.regular_price::float8,prv.sale_price::float8,prv.discount_code,bv.website,bv.phone_number,bv.default_disclaimer,bv.logo_asset_ids,r.created_at`, job.Args.RenderJobID).Scan(&data.ClientID, &data.WorkspaceID, &data.CampaignID, &data.ProjectID, &data.ActorID, &data.ProjectVersion, &data.ProjectHash, &data.Language, &data.Duration, &data.Headline, &data.LowerThird, &data.ShowPrice, &data.ShowDiscount, &data.ShowCTA, &data.ShowWebsite, &data.ShowPhone, &data.ShowQR, &data.ShowDisclaimer, &data.BurnCaptions, &data.MusicAssetID, &data.MusicGain, &data.DialogueDucking, &data.ProductID, &data.BrandID, &data.ProductName, &data.CTA, &data.Offer, &data.Currency, &data.RegularPrice, &data.SalePrice, &data.Discount, &data.Website, &data.Phone, &data.Disclaimer, &data.LogoIDs, &data.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(errors.New("render job is no longer eligible"))
	}
	if err != nil {
		return err
	}
	manifest, err := w.buildManifest(ctx, job.Args.RenderJobID, data)
	if err != nil {
		return w.fail(ctx, job, err)
	}
	raw, err := json.Marshal(manifest)
	if err != nil {
		return w.fail(ctx, job, err)
	}
	sum := sha256.Sum256(raw)
	manifestHash := hex.EncodeToString(sum[:])
	manifestID := uuid.New()
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	err = tx.QueryRow(ctx, `INSERT INTO render_manifests(id,video_project_id,video_project_version,manifest_version,manifest_hash,manifest,created_by)VALUES($1,$2,$3,1,$4,$5,$6)ON CONFLICT(video_project_id,video_project_version,manifest_hash)DO UPDATE SET manifest=EXCLUDED.manifest RETURNING id`, manifestID, data.ProjectID, data.ProjectVersion, manifestHash, raw, data.ActorID).Scan(&manifestID)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE render_jobs SET status='RENDERING',render_manifest_id=$2,version=version+1,updated_at=now() WHERE id=$1 AND status='BUILDING_MANIFEST'`, job.Args.RenderJobID, manifestID); err != nil {
		return err
	}
	if err = tx.Commit(ctx); err != nil {
		return err
	}
	result, err := w.Renderer.Render(ctx, raw)
	if err != nil {
		return w.fail(ctx, job, err)
	}
	return w.persist(ctx, job.Args.RenderJobID, data, manifest, result)
}

func (w *Worker) buildManifest(ctx context.Context, renderID uuid.UUID, data manifestContext) (RenderManifest, error) {
	manifest := RenderManifest{RenderID: renderID.String(), ManifestVersion: 1, WorkspaceID: data.WorkspaceID.String(), CampaignID: data.CampaignID.String(), VideoProjectID: data.ProjectID.String(), VideoProjectVersion: data.ProjectVersion, VideoProjectHash: data.ProjectHash, Language: data.Language, BurnCaptions: data.BurnCaptions, SoundEffects: []SoundEffect{}, MusicGainDB: data.MusicGain, DialogueDuckingDB: data.DialogueDucking, OutputObjectKey: fmt.Sprintf("workspaces/%s/campaigns/%s/renders/%s/final.mp4", data.WorkspaceID, data.CampaignID, renderID), ThumbnailObjectKey: fmt.Sprintf("workspaces/%s/campaigns/%s/renders/%s/thumbnail.jpg", data.WorkspaceID, data.CampaignID, renderID), CreatedAt: data.CreatedAt.UTC().Format(time.RFC3339)}
	manifest.Output.Width, manifest.Output.Height, manifest.Output.FPS, manifest.Output.DurationSeconds, manifest.Output.Codec = 1080, 1920, 30, data.Duration, "h264"
	rows, err := w.Pool.Query(ctx, `SELECT s.id,s.current_version,sv.duration_seconds,a.id,v.storage_key,v.mime_type,v.checksum_sha256,e.trim_start_ms,COALESCE(e.trim_end_ms,sv.duration_seconds*1000),e.mute_audio,e.transition,e.attached_product_asset_ids,t.transcript FROM scenes s JOIN scene_versions sv ON sv.scene_id=s.id AND sv.version=s.current_version JOIN scene_generation_tasks g ON g.id=s.selected_generation_task_id AND g.status='APPROVED' JOIN scene_generation_edits e ON e.generation_task_id=g.id JOIN media_assets a ON a.id=COALESCE(e.replacement_asset_id,g.output_asset_id) AND a.deleted_at IS NULL JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version LEFT JOIN scene_transcriptions t ON t.generation_task_id=g.id AND t.status='SUCCEEDED' WHERE s.campaign_id=$1 ORDER BY s.scene_order`, data.CampaignID)
	if err != nil {
		return manifest, err
	}
	defer rows.Close()
	var timeline int64
	for rows.Next() {
		var scene ManifestScene
		var duration int32
		var assetID uuid.UUID
		var key, mime string
		var checksum *string
		var transition string
		var attached []uuid.UUID
		var transcript *string
		if err = rows.Scan(&scene.SceneID, &scene.SceneVersion, &duration, &assetID, &key, &mime, &checksum, &scene.TrimStartMS, &scene.TrimEndMS, &scene.Muted, &transition, &attached, &transcript); err != nil {
			return manifest, err
		}
		if checksum == nil || *checksum == "" {
			return manifest, ErrPrerequisite
		}
		scene.DurationMS = int64(duration) * 1000
		scene.Source = ObjectReference{ObjectKey: key, SHA256: *checksum, ContentType: mime}
		scene.Transition = map[string]string{"CUT": "cut", "CROSSFADE": "fade", "FADE_TO_BLACK": "fade"}[transition]
		if scene.Transition == "" {
			scene.Transition = "cut"
		}
		scene.ProductMedia, err = w.references(ctx, data.ClientID, data.WorkspaceID, attached)
		if err != nil {
			return manifest, err
		}
		manifest.Scenes = append(manifest.Scenes, scene)
		if transcript != nil && strings.TrimSpace(*transcript) != "" {
			manifest.Captions = append(manifest.Captions, CaptionCue{StartMS: timeline, EndMS: timeline + scene.DurationMS, Text: strings.TrimSpace(*transcript)})
		}
		timeline += scene.DurationMS
	}
	if err = rows.Err(); err != nil {
		return manifest, err
	}
	if len(manifest.Scenes) == 0 || timeline != int64(data.Duration)*1000 {
		return manifest, ErrPrerequisite
	}
	if data.MusicAssetID != nil {
		reference, refErr := w.reference(ctx, data.ClientID, data.WorkspaceID, *data.MusicAssetID)
		if refErr != nil {
			return manifest, refErr
		}
		manifest.Music = &reference
	}
	if len(data.LogoIDs) > 0 {
		reference, refErr := w.reference(ctx, data.ClientID, data.WorkspaceID, data.LogoIDs[0])
		if refErr == nil {
			manifest.Logo = &reference
		}
	}
	end := data.Duration * 30
	manifest.Overlays = append(manifest.Overlays, Overlay{Type: "headline", Value: data.Headline, StartFrame: 0, EndFrame: min32(end, 150), SafeZone: "title"})
	if data.LowerThird != "" {
		manifest.Overlays = append(manifest.Overlays, Overlay{Type: "lower_third", Value: data.LowerThird, StartFrame: 30, EndFrame: min32(end, 240), SafeZone: "action"})
	}
	if manifest.Logo != nil {
		manifest.Overlays = append(manifest.Overlays, Overlay{Type: "logo", Value: data.ProductName, StartFrame: 0, EndFrame: end, SafeZone: "title"})
	}
	start := max32(0, end-300)
	if data.ShowPrice {
		price := data.SalePrice
		if price == nil {
			price = data.RegularPrice
		}
		if price != nil {
			currency := ""
			if data.Currency != nil {
				currency = *data.Currency
			}
			manifest.Overlays = append(manifest.Overlays, Overlay{Type: "price", Value: fmt.Sprintf("%.2f %s", *price, currency), StartFrame: start, EndFrame: end, SafeZone: "action"})
		}
	}
	if data.ShowDiscount && data.Discount != nil && *data.Discount != "" {
		manifest.Overlays = append(manifest.Overlays, Overlay{Type: "discount_code", Value: *data.Discount, StartFrame: start, EndFrame: end, SafeZone: "action"})
	}
	if data.ShowCTA && data.CTA != "" {
		manifest.Overlays = append(manifest.Overlays, Overlay{Type: "cta", Value: data.CTA, StartFrame: start, EndFrame: end, SafeZone: "bottom"})
	}
	if data.ShowWebsite && data.Website != nil && *data.Website != "" {
		manifest.Overlays = append(manifest.Overlays, Overlay{Type: "website", Value: *data.Website, StartFrame: start, EndFrame: end, SafeZone: "bottom"})
	}
	if data.ShowPhone && data.Phone != nil && *data.Phone != "" {
		manifest.Overlays = append(manifest.Overlays, Overlay{Type: "phone", Value: *data.Phone, StartFrame: start, EndFrame: end, SafeZone: "bottom"})
	}
	if data.ShowQR && data.Website != nil && strings.HasPrefix(*data.Website, "http") {
		manifest.Overlays = append(manifest.Overlays, Overlay{Type: "qr_code", Value: *data.Website, StartFrame: start, EndFrame: end, SafeZone: "bottom"})
	}
	if data.ShowDisclaimer && data.Disclaimer != nil && *data.Disclaimer != "" {
		manifest.Overlays = append(manifest.Overlays, Overlay{Type: "disclaimer", Value: *data.Disclaimer, StartFrame: start, EndFrame: end, SafeZone: "bottom"})
	}
	return manifest, nil
}

func (w *Worker) reference(ctx context.Context, clientID, workspaceID, assetID uuid.UUID) (ObjectReference, error) {
	var r ObjectReference
	var checksum *string
	err := w.Pool.QueryRow(ctx, `SELECT v.storage_key,v.mime_type,v.checksum_sha256 FROM media_assets a JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version WHERE a.id=$1 AND a.client_id=$2 AND a.workspace_id=$3 AND a.deleted_at IS NULL`, assetID, clientID, workspaceID).Scan(&r.ObjectKey, &r.ContentType, &checksum)
	if errors.Is(err, pgx.ErrNoRows) {
		return r, ErrPrerequisite
	}
	if err != nil {
		return r, err
	}
	if checksum == nil || *checksum == "" {
		return r, ErrPrerequisite
	}
	r.SHA256 = *checksum
	return r, nil
}
func (w *Worker) references(ctx context.Context, clientID, workspaceID uuid.UUID, ids []uuid.UUID) ([]ObjectReference, error) {
	sort.Slice(ids, func(i, j int) bool { return ids[i].String() < ids[j].String() })
	out := make([]ObjectReference, 0, len(ids))
	for _, id := range ids {
		r, err := w.reference(ctx, clientID, workspaceID, id)
		if err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

func (w *Worker) persist(ctx context.Context, id uuid.UUID, data manifestContext, manifest RenderManifest, result RendererResult) error {
	srt, vtt := subtitles(manifest.Captions)
	srtKey := strings.TrimSuffix(manifest.OutputObjectKey, "final.mp4") + "captions.srt"
	vttKey := strings.TrimSuffix(manifest.OutputObjectKey, "final.mp4") + "captions.vtt"
	if err := w.Store.Put(ctx, srtKey, "application/x-subrip", int64(len(srt)), bytes.NewReader(srt), map[string]string{"render-job-id": id.String()}); err != nil {
		return err
	}
	if err := w.Store.Put(ctx, vttKey, "text/vtt", int64(len(vtt)), bytes.NewReader(vtt), map[string]string{"render-job-id": id.String()}); err != nil {
		return err
	}
	assetID := uuid.New()
	metadata, _ := json.Marshal(map[string]any{"rendererRequestId": result.RequestID, "reused": result.Reused, "fps": result.FPS, "audioCodec": result.AudioCodec})
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	_, err = tx.Exec(ctx, `INSERT INTO media_assets(id,client_id,workspace_id,product_id,campaign_id,asset_type,category,name,status,usage_rights,source_metadata,created_by,updated_by)VALUES($1,$2,$3,$4,$5,'VIDEO','final-render',$6,'DRAFT','Final campaign render',$7,$8,$8)`, assetID, data.ClientID, data.WorkspaceID, data.ProductID, data.CampaignID, "Final render "+id.String(), metadata, data.ActorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO media_asset_versions(media_asset_id,client_id,workspace_id,version,storage_key,original_filename,mime_type,file_extension,file_size_bytes,checksum_sha256,width,height,duration_ms,codec,thumbnail_storage_key,metadata,verified_at,created_by)VALUES($1,$2,$3,1,$4,'final.mp4','video/mp4','.mp4',$5,$6,$7,$8,$9,$10,$11,$12,now(),$13)`, assetID, data.ClientID, data.WorkspaceID, result.OutputObjectKey, result.FileSizeBytes, result.ChecksumSHA256, result.Width, result.Height, result.DurationMS, result.Codec, result.ThumbnailObjectKey, metadata, data.ActorID)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO video_outputs(render_job_id,media_asset_id,width,height,fps,duration_ms,codec,audio_codec,file_size_bytes,checksum_sha256,metadata)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, assetID, result.Width, result.Height, int(result.FPS), result.DurationMS, result.Codec, result.AudioCodec, result.FileSizeBytes, result.ChecksumSHA256, metadata)
	if err != nil {
		return err
	}
	for _, item := range []struct{ format, key, content string }{{"SRT", srtKey, string(srt)}, {"VTT", vttKey, string(vtt)}} {
		sum := sha256.Sum256([]byte(item.content))
		if _, err = tx.Exec(ctx, `INSERT INTO subtitle_outputs(render_job_id,format,language,storage_key,checksum_sha256,cue_count)VALUES($1,$2::subtitle_format,$3,$4,$5,$6)`, id, item.format, data.Language, item.key, hex.EncodeToString(sum[:]), len(manifest.Captions)); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE render_jobs SET status='REVIEW_REQUIRED',output_asset_id=$2,thumbnail_storage_key=$3,srt_storage_key=$4,vtt_storage_key=$5,output_hash=$6,renderer_request_id=$7,sanitized_response=$8,error_code=NULL,error_message=NULL,completed_at=now(),version=version+1,updated_at=now() WHERE id=$1 AND status='RENDERING'`, id, assetID, result.ThumbnailObjectKey, srtKey, vttKey, result.ChecksumSHA256, result.RequestID, metadata)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET status='FINAL_REVIEW',version=version+1,updated_at=now() WHERE id=$1`, data.CampaignID); err != nil {
		return err
	}
	if err = usage.Record(ctx, tx, usage.Entry{Provider: "renderer", Model: "remotion-4.0.513", RequestReference: id.String(), Operation: "FINAL_RENDER", ClientID: &data.ClientID, WorkspaceID: &data.WorkspaceID, CampaignID: &data.CampaignID, VideoProjectID: &data.ProjectID, GeneratedSeconds: float64(data.Duration), EstimatedCost: 0, Currency: "USD", Outcome: "SUCCESS", Category: "RENDER", Reused: result.Reused, Metadata: map[string]any{"rendererRequestId": result.RequestID, "outputHash": result.ChecksumSHA256}}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *Worker) fail(ctx context.Context, job *river.Job[jobs.FinalRenderArgs], cause error) error {
	status := "QUEUED"
	if job.JobRow != nil && job.Attempt >= job.MaxAttempts {
		status = "FAILED"
	}
	failureContext, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if _, err := w.Pool.Exec(failureContext, `UPDATE render_jobs SET status=$2::render_job_status,error_code=CASE WHEN $2::text='FAILED' THEN 'renderer_failed' ELSE NULL END,error_message=CASE WHEN $2::text='FAILED' THEN 'Final renderer could not complete the manifest' ELSE NULL END,version=version+1,updated_at=now() WHERE id=$1`, job.Args.RenderJobID, status); err != nil {
		return errors.Join(cause, fmt.Errorf("persist render failure state: %w", err))
	}
	return cause
}
func subtitles(cues []CaptionCue) ([]byte, []byte) {
	var srt, vtt strings.Builder
	vtt.WriteString("WEBVTT\n\n")
	for index, cue := range cues {
		fmt.Fprintf(&srt, "%d\n%s --> %s\n%s\n\n", index+1, stamp(cue.StartMS, ','), stamp(cue.EndMS, ','), cue.Text)
		fmt.Fprintf(&vtt, "%s --> %s\n%s\n\n", stamp(cue.StartMS, '.'), stamp(cue.EndMS, '.'), cue.Text)
	}
	if srt.Len() == 0 {
		srt.WriteByte('\n')
	}
	return []byte(srt.String()), []byte(vtt.String())
}
func stamp(ms int64, separator rune) string {
	hours := ms / 3600000
	ms %= 3600000
	minutes := ms / 60000
	ms %= 60000
	seconds := ms / 1000
	millis := ms % 1000
	return fmt.Sprintf("%02d:%02d:%02d%c%03d", hours, minutes, seconds, separator, millis)
}
func min32(a, b int32) int32 {
	if a < b {
		return a
	}
	return b
}
func max32(a, b int32) int32 {
	if a > b {
		return a
	}
	return b
}
