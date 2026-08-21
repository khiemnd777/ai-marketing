package planning_test

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"
	"github.com/riverqueue/river/rivermigrate"
	"github.com/riverqueue/river/rivertype"
	"github.com/testcontainers/testcontainers-go"
	postgrescontainer "github.com/testcontainers/testcontainers-go/modules/postgres"

	studioai "github.com/internal/ai-product-marketing-studio/services/api/internal/ai"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/analytics"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/brands"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/campaigns"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/characters"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/media"
	metaprovider "github.com/internal/ai-product-marketing-studio/services/api/internal/meta"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/metaads"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/metaconnections"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/operations"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/planning"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/publishing"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/rendering"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/storage"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/usage"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/video"
)

type fakeEnqueuer struct{ next int64 }

type fakeReferenceStore struct{}

func (fakeReferenceStore) PresignGet(_ context.Context, key string, _ time.Duration) (storage.PresignedRequest, error) {
	return storage.PresignedRequest{URL: "https://assets.example/" + key, Method: "GET"}, nil
}

func (e *fakeEnqueuer) EnqueueAIPlanning(_ context.Context, _ pgx.Tx, _ jobs.AIPlanningArgs) (int64, error) {
	e.next++
	return e.next, nil
}

func TestDemoPlanningWorkflowIntegration(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Minute)
	defer cancel()
	databaseURL := integrationDatabaseURL(t, ctx)
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(pool.Close)

	actorID, clientID, workspaceID, brandID, productID := seedPlanningFixture(t, ctx, pool)
	principal := auth.Principal{UserID: actorID, Email: "m2-integration@example.test", DisplayName: "M2 Integration", Role: db.InternalUserRoleADMIN}
	metadata := auth.ClientMetadata{RequestID: "integration-m2", UserAgent: "go-test"}
	campaign, err := campaigns.NewService(pool).Create(ctx, clientID, workspaceID, campaigns.Input{BrandID: brandID, ProductID: productID, Name: "M2 Launch " + uuid.NewString()[:8], Objective: "PRODUCT_INTRODUCTION", TargetAudience: "Frequent travelers", Market: "Vietnam", Country: "VN", Language: "vi", SocialPlatformTargets: []string{"FACEBOOK", "INSTAGRAM"}, VideoFormat: "INTERVIEW_REVIEW", DurationSeconds: 30, AspectRatio: "9:16", Tone: "Practical and trustworthy", Offer: "", CTA: "Tìm hiểu ngay", ChangeSummary: "Integration fixture"}, principal, metadata)
	if err != nil {
		t.Fatalf("create campaign: %v", err)
	}

	characterService := characters.NewService(pool)
	primary, err := characterService.Create(ctx, clientID, workspaceID, actorID, characters.Input{Name: "Demo Host " + uuid.NewString()[:6], Provider: "demo", CharacterType: "PRESET", GenderPresentation: "neutral", ApproximateAgeRange: "25-35", AppearanceDescription: "Professional travel host", Wardrobe: "Smart casual", GestureStyle: "Natural", DefaultRole: "host", SupportedLanguages: []string{"vi"}, ConsentStatus: "NOT_REQUIRED"})
	if err != nil {
		t.Fatalf("create primary character: %v", err)
	}
	listener, err := characterService.Create(ctx, clientID, workspaceID, actorID, characters.Input{Name: "Demo Traveler " + uuid.NewString()[:6], Provider: "demo", CharacterType: "TRUSTED_GENERATED", GenderPresentation: "neutral", ApproximateAgeRange: "25-35", AppearanceDescription: "Frequent traveler", Wardrobe: "Travel casual", GestureStyle: "Subtle", DefaultRole: "traveler", SupportedLanguages: []string{"vi"}, ConsentStatus: "NOT_REQUIRED"})
	if err != nil {
		t.Fatalf("create listener character: %v", err)
	}
	if _, err = characterService.Select(ctx, clientID, workspaceID, campaign.ID, primary.ID, listener.ID, actorID); err != nil {
		t.Fatalf("select campaign characters: %v", err)
	}

	cfg := config.Config{DemoMode: true, OpenAI: config.OpenAIConfig{Model: "demo-planning", InputUSDPer1M: 1, OutputUSDPer1M: 2}, Seedance: config.SeedanceConfig{USDPerSecond: 0.04}}
	enqueuer := &fakeEnqueuer{next: 90_000}
	service := planning.NewService(pool, enqueuer, cfg)
	worker := &planning.Worker{Pool: pool, Provider: studioai.NewDemoProvider(cfg.OpenAI.Model), Config: cfg}
	run := func(operation string) planning.GenerationJob {
		t.Helper()
		job, startErr := service.StartGeneration(ctx, clientID, workspaceID, campaign.ID, actorID, operation, "integration-"+operation+"-"+uuid.NewString())
		if startErr != nil {
			t.Fatalf("start %s: %v", operation, startErr)
		}
		args := jobs.AIPlanningArgs{GenerationJobID: job.ID, ClientID: clientID, WorkspaceID: workspaceID, CampaignID: campaign.ID, Operation: operation}
		if workErr := worker.Work(ctx, &river.Job[jobs.AIPlanningArgs]{JobRow: &rivertype.JobRow{ID: *job.RiverJobID}, Args: args}); workErr != nil {
			t.Fatalf("work %s: %v", operation, workErr)
		}
		completed, getErr := service.GetJob(ctx, clientID, workspaceID, campaign.ID, job.ID)
		if getErr != nil || completed.Status != "SUCCEEDED" {
			t.Fatalf("complete %s: status=%s err=%v message=%v", operation, completed.Status, getErr, completed.ErrorMessage)
		}
		return completed
	}

	run("CONCEPTS")
	concepts, err := service.ListConcepts(ctx, clientID, workspaceID, campaign.ID)
	if err != nil || len(concepts) != 2 {
		t.Fatalf("concept outputs: count=%d err=%v", len(concepts), err)
	}
	approved, err := service.DecideConcept(ctx, clientID, workspaceID, campaign.ID, concepts[0].ID, "APPROVE", concepts[0].Version, "integration review", principal, metadata)
	if err != nil {
		t.Fatalf("approve concept: %v", err)
	}
	if _, err = service.DecideConcept(ctx, clientID, workspaceID, campaign.ID, approved.ID, "LOCK", approved.Version, "integration lock", principal, metadata); err != nil {
		t.Fatalf("lock concept: %v", err)
	}

	run("CONTENT")
	content, err := service.ListContent(ctx, clientID, workspaceID, campaign.ID)
	if err != nil || len(content) != 14 {
		t.Fatalf("content outputs: count=%d err=%v", len(content), err)
	}
	if _, err = service.ApproveContent(ctx, clientID, workspaceID, campaign.ID, content[0].ID, content[0].Version, "integration review", principal); err != nil {
		t.Fatalf("approve content: %v", err)
	}

	run("SCRIPT")
	script, err := service.GetScript(ctx, clientID, workspaceID, campaign.ID)
	if err != nil || script.Output.ApproximateDurationSeconds != 30 || len(script.Output.DialogueTurns) < 2 {
		t.Fatalf("script output: duration=%d turns=%d err=%v", script.Output.ApproximateDurationSeconds, len(script.Output.DialogueTurns), err)
	}
	if _, err = service.ApproveScript(ctx, clientID, workspaceID, campaign.ID, script.Version, "integration review", principal, metadata); err != nil {
		t.Fatalf("approve script: %v", err)
	}

	run("SCENES")
	scenes, err := service.ListScenes(ctx, clientID, workspaceID, campaign.ID)
	if err != nil || len(scenes) != 4 {
		t.Fatalf("scene outputs: count=%d err=%v", len(scenes), err)
	}
	var totalDuration int32
	for _, scene := range scenes {
		totalDuration += scene.Direction.DurationSeconds
		if scene.Direction.SpeakerCharacterID != primary.ID || scene.Direction.ListenerCharacterID != listener.ID {
			t.Fatalf("scene %s lost the selected character pair", scene.SceneKey)
		}
	}
	if totalDuration != 30 {
		t.Fatalf("scene duration total=%d, want 30", totalDuration)
	}
	productReferenceID := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO media_assets(id,client_id,workspace_id,asset_type,category,name,status,usage_rights,created_by,updated_by)VALUES($1,$2,$3,'IMAGE','HERO_IMAGE','Integration product hero','APPROVED','Integration fixture',$4,$4)`, productReferenceID, clientID, workspaceID, actorID); err != nil {
		t.Fatalf("seed product reference: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO media_asset_versions(media_asset_id,client_id,workspace_id,version,storage_key,original_filename,mime_type,file_extension,file_size_bytes,checksum_sha256,width,height,verified_at,created_by)VALUES($1,$2,$3,1,$4,'hero.jpg','image/jpeg','.jpg',1024,$5,1080,1920,now(),$6)`, productReferenceID, clientID, workspaceID, "integration/"+productReferenceID.String()+"/hero.jpg", strings.Repeat("d", 64), actorID); err != nil {
		t.Fatalf("seed product reference version: %v", err)
	}
	mediaService := media.NewService(pool, nil, nil)
	productReference, err := mediaService.AttachProduct(ctx, clientID, workspaceID, productID, productReferenceID, actorID, 1)
	if err != nil || productReference.ProductID == nil || *productReference.ProductID != productID {
		t.Fatalf("attach unassigned product reference: asset=%#v err=%v", productReference, err)
	}
	otherProductID, wrongReferenceID := uuid.New(), uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO products(id,client_id,workspace_id,brand_id,name,sku,category,vertical_key,status,created_by,updated_by)VALUES($1,$2,$3,$4,'Other luggage',$5,'Travel luggage','travel-luggage','DRAFT',$6,$6)`, otherProductID, clientID, workspaceID, brandID, "OTHER-"+uuid.NewString()[:8], actorID); err != nil {
		t.Fatalf("seed other product: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO media_assets(id,client_id,workspace_id,product_id,asset_type,category,name,status,usage_rights,created_by,updated_by)VALUES($1,$2,$3,$4,'IMAGE','HERO_IMAGE','Wrong product hero','APPROVED','Integration fixture',$5,$5)`, wrongReferenceID, clientID, workspaceID, otherProductID, actorID); err != nil {
		t.Fatalf("seed wrong product reference: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO media_asset_versions(media_asset_id,client_id,workspace_id,version,storage_key,original_filename,mime_type,file_extension,file_size_bytes,checksum_sha256,width,height,verified_at,created_by)VALUES($1,$2,$3,1,$4,'wrong.jpg','image/jpeg','.jpg',1024,$5,1080,1920,now(),$6)`, wrongReferenceID, clientID, workspaceID, "integration/"+wrongReferenceID.String()+"/wrong.jpg", strings.Repeat("e", 64), actorID); err != nil {
		t.Fatalf("seed wrong product reference version: %v", err)
	}
	if _, attachErr := mediaService.AttachProduct(ctx, clientID, workspaceID, productID, wrongReferenceID, actorID, 1); !errors.Is(attachErr, media.ErrAssigned) {
		t.Fatalf("expected cross-product attachment to fail, got %v", attachErr)
	}
	wrongDirection := scenes[0].Direction
	wrongDirection.ReferenceAssetIDs = []uuid.UUID{wrongReferenceID}
	if _, updateErr := service.UpdateScene(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, wrongDirection, scenes[0].Version, principal); !errors.Is(updateErr, planning.ErrInvalid) {
		t.Fatalf("expected wrong-product reference to be rejected, got %v", updateErr)
	}
	validDirection := scenes[0].Direction
	validDirection.ReferenceAssetIDs = []uuid.UUID{productReferenceID}
	scenes[0], err = service.UpdateScene(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, validDirection, scenes[0].Version, principal)
	if err != nil {
		t.Fatalf("attach valid product reference to scene: %v", err)
	}
	if _, detachErr := mediaService.DetachProduct(ctx, clientID, workspaceID, productID, productReferenceID, actorID, productReference.Version); !errors.Is(detachErr, media.ErrInUse) {
		t.Fatalf("expected in-use product reference detach to fail, got %v", detachErr)
	}
	if scenes[0], err = service.ApproveScene(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, scenes[0].Version, "integration review", principal); err != nil {
		t.Fatalf("approve scene: %v", err)
	}
	if productReference, err = mediaService.Update(ctx, clientID, workspaceID, productReferenceID, actorID, media.UpdateInput{Category: productReference.Category, Name: "Integration product hero revised", Folder: productReference.Folder, UsageRights: productReference.UsageRights, Tags: productReference.Tags, ExpiresAt: productReference.ExpiresAt, Version: productReference.Version}); err != nil {
		t.Fatalf("update referenced product media: %v", err)
	}
	refreshedScenes, err := service.ListScenes(ctx, clientID, workspaceID, campaign.ID)
	if err != nil {
		t.Fatalf("reload scenes after media change: %v", err)
	}
	for _, scene := range refreshedScenes {
		if scene.ID == scenes[0].ID {
			scenes[0] = scene
		}
	}
	if scenes[0].Status != "DRAFT" {
		t.Fatalf("expected referenced media change to return scene to draft, got %s", scenes[0].Status)
	}
	var invalidatedSceneApprovals int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM approvals WHERE entity_type='SCENE' AND entity_id=$1 AND invalidated_at IS NOT NULL`, scenes[0].ID).Scan(&invalidatedSceneApprovals); err != nil || invalidatedSceneApprovals != 1 {
		t.Fatalf("scene approval invalidation count=%d err=%v", invalidatedSceneApprovals, err)
	}
	if scenes[0], err = service.ApproveScene(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, scenes[0].Version, "integration re-review after media change", principal); err != nil {
		t.Fatalf("reapprove scene after media change: %v", err)
	}
	jobEnqueuer, err := jobs.NewEnqueuer(pool)
	if err != nil {
		t.Fatalf("create video enqueuer: %v", err)
	}
	videoConfig := cfg
	videoConfig.Seedance = config.SeedanceConfig{Model: "demo-seedance", APIVersion: "v3", Resolution: "720p", AspectRatio: "9:16", PollInterval: time.Second, TaskTimeout: 10 * time.Minute, USDPerSecond: 0.04}
	videoService := video.NewService(pool, jobEnqueuer, video.NewDemoProvider(), videoConfig)
	generationKey := "integration-video-" + uuid.NewString()
	generation, err := videoService.Start(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, actorID, generationKey, video.StartInput{Resolution: "720p", AspectRatio: "9:16", GenerateAudio: true})
	if err != nil || generation.Status != "QUEUED" || generation.EstimatedCostUSD <= 0 {
		t.Fatalf("start video generation: generation=%#v err=%v", generation, err)
	}
	demoVideoProvider := video.NewDemoProvider()
	submitWorker := &video.SubmitWorker{Pool: pool, Provider: demoVideoProvider, Enqueuer: jobEnqueuer, Store: fakeReferenceStore{}, Config: videoConfig}
	if err = submitWorker.Work(ctx, &river.Job[jobs.SeedanceSubmitArgs]{JobRow: &rivertype.JobRow{ID: 100_001, Attempt: 1, MaxAttempts: 1}, Args: jobs.SeedanceSubmitArgs{GenerationTaskID: generation.ID}}); err != nil {
		t.Fatalf("submit Seedance task: %v", err)
	}
	statusWorker := &video.StatusWorker{Pool: pool, Provider: demoVideoProvider, Enqueuer: jobEnqueuer, Config: videoConfig}
	if err = statusWorker.Work(ctx, &river.Job[jobs.SeedanceStatusArgs]{JobRow: &rivertype.JobRow{ID: 100_002, Attempt: 1, MaxAttempts: 120}, Args: jobs.SeedanceStatusArgs{GenerationTaskID: generation.ID}}); err != nil {
		t.Fatalf("synchronize Seedance task: %v", err)
	}
	reused, err := videoService.Start(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, actorID, generationKey, video.StartInput{Resolution: "720p", AspectRatio: "9:16", GenerateAudio: true})
	if err != nil || reused.ID != generation.ID || !reused.Reused || reused.Status != "SUCCEEDED" {
		t.Fatalf("reuse video generation: generation=%#v err=%v", reused, err)
	}
	outputAssetID := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO media_assets(id,client_id,workspace_id,product_id,campaign_id,asset_type,category,name,status,usage_rights,created_by,updated_by)VALUES($1,$2,$3,$4,$5,'VIDEO','seedance-output','Integration generated take','DRAFT','Integration fixture',$6,$6)`, outputAssetID, clientID, workspaceID, productID, campaign.ID, actorID); err != nil {
		t.Fatalf("seed generated output asset: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO media_asset_versions(media_asset_id,client_id,workspace_id,version,storage_key,original_filename,mime_type,file_extension,file_size_bytes,checksum_sha256,width,height,duration_ms,codec,metadata,verified_at,created_by)VALUES($1,$2,$3,1,$4,'take.mp4','video/mp4','.mp4',1024,$5,406,720,$6::bigint,'h264',jsonb_build_object('probe',jsonb_build_object('width',406,'height',720,'durationMs',$6::bigint,'codec','h264','bitrateBps',1000,'audioStream',true)),now(),$7)`, outputAssetID, clientID, workspaceID, "integration/"+outputAssetID.String()+"/take.mp4", strings.Repeat("a", 64), int64(scenes[0].Direction.DurationSeconds)*1000, actorID); err != nil {
		t.Fatalf("seed generated output version: %v", err)
	}
	if _, err = pool.Exec(ctx, `UPDATE scene_generation_tasks SET status='VALIDATING',output_asset_id=$2 WHERE id=$1`, generation.ID, outputAssetID); err != nil {
		t.Fatalf("seed validating take: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO scene_transcriptions(generation_task_id,status,provider,model,language,transcript,segments,transcript_hash,completed_at)VALUES($1,'SUCCEEDED','demo','demo-transcribe','vi',$2,'[]','integration',now())`, generation.ID, scenes[0].Direction.Dialogue); err != nil {
		t.Fatalf("seed transcription: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO scene_quality_checks(generation_task_id,status)VALUES($1,'QUEUED')`, generation.ID); err != nil {
		t.Fatalf("seed quality check: %v", err)
	}
	qualityWorker := &video.QualityCheckWorker{Pool: pool}
	if err = qualityWorker.Work(ctx, &river.Job[jobs.QualityCheckArgs]{JobRow: &rivertype.JobRow{ID: 100_003, Attempt: 1, MaxAttempts: 3}, Args: jobs.QualityCheckArgs{GenerationTaskID: generation.ID}}); err != nil {
		t.Fatalf("complete deterministic quality checks: %v", err)
	}
	reviewReady, err := videoService.Get(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, generation.ID)
	if err != nil || reviewReady.Status != "REVIEW_REQUIRED" || reviewReady.QualityCheck == nil || reviewReady.QualityCheck.DeterministicPass == nil || !*reviewReady.QualityCheck.DeterministicPass {
		t.Fatalf("quality-check generation: generation=%#v err=%v", reviewReady, err)
	}
	trimEnd := int64(scenes[0].Direction.DurationSeconds)*1000 - 100
	edited, err := videoService.UpdateEdit(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, generation.ID, actorID, video.GenerationEdit{TrimStartMS: 100, TrimEndMS: &trimEnd, MuteAudio: false, Transition: "CROSSFADE", AttachedProductAssetIDs: []uuid.UUID{}, SubtitlePreview: true, Version: 1})
	if err != nil || edited.Edit == nil || edited.Edit.Transition != "CROSSFADE" {
		t.Fatalf("update scene generation edit: generation=%#v err=%v", edited, err)
	}
	characterCount := int32(2)
	no := false
	reviewed, err := videoService.Review(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, generation.ID, actorID, video.ReviewInput{Action: "APPROVE", Version: reviewReady.Version, Notes: "Integration human review", CharacterCount: &characterCount, DuplicateCharacter: &no, DuplicateProduct: &no, ProductColorMismatch: &no, BlurOrLowQualityWarning: &no, CropWarning: &no, SubtitleOverflow: &no, LogoOverlap: &no, CTASafeZoneViolation: &no})
	if err != nil || reviewed.Status != "APPROVED" {
		t.Fatalf("review generation: generation=%#v err=%v", reviewed, err)
	}
	selected, err := videoService.Select(ctx, clientID, workspaceID, campaign.ID, scenes[0].ID, generation.ID, actorID)
	if err != nil || !selected.Selected {
		t.Fatalf("select generation: generation=%#v err=%v", selected, err)
	}
	for _, scene := range scenes[1:] {
		assetID, taskID := uuid.New(), uuid.New()
		if _, err = pool.Exec(ctx, `INSERT INTO media_assets(id,client_id,workspace_id,product_id,campaign_id,asset_type,category,name,status,usage_rights,created_by,updated_by)VALUES($1,$2,$3,$4,$5,'VIDEO','seedance-output','Integration approved take','DRAFT','Integration fixture',$6,$6)`, assetID, clientID, workspaceID, productID, campaign.ID, actorID); err != nil {
			t.Fatalf("seed scene output asset: %v", err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO media_asset_versions(media_asset_id,client_id,workspace_id,version,storage_key,original_filename,mime_type,file_extension,file_size_bytes,checksum_sha256,width,height,duration_ms,codec,verified_at,created_by)VALUES($1,$2,$3,1,$4,'take.mp4','video/mp4','.mp4',1024,$5,720,1280,$6,'h264',now(),$7)`, assetID, clientID, workspaceID, "integration/"+assetID.String()+"/take.mp4", strings.Repeat("b", 64), int64(scene.Direction.DurationSeconds)*1000, actorID); err != nil {
			t.Fatalf("seed scene output version: %v", err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO scene_generation_tasks(id,client_id,workspace_id,campaign_id,scene_id,scene_version,provider,provider_task_id,status,idempotency_key,model,api_version,resolution,aspect_ratio,duration_seconds,generate_audio,scene_hash,prompt_hash,reference_hash,request_hash,output_asset_id,timeout_at,reviewed_at,reviewed_by,review_notes,created_by)VALUES($1,$2,$3,$4,$5,$6,'demo',$7,'APPROVED',$8,'demo-seedance','v3','720p','9:16',$9,true,$10,$10,$10,$10,$11,now()+interval '10 minutes',now(),$12,'Integration human review',$12)`, taskID, clientID, workspaceID, campaign.ID, scene.ID, scene.CurrentVersion, "demo-"+taskID.String(), "integration-"+taskID.String(), scene.Direction.DurationSeconds, strings.Repeat("c", 64), assetID, actorID); err != nil {
			t.Fatalf("seed approved scene take: %v", err)
		}
		if _, err = pool.Exec(ctx, `INSERT INTO scene_generation_edits(generation_task_id,trim_start_ms,trim_end_ms,mute_audio,transition,subtitle_preview,updated_by)VALUES($1,0,$2,false,'CUT',true,$3)`, taskID, int64(scene.Direction.DurationSeconds)*1000, actorID); err != nil {
			t.Fatalf("seed scene edit: %v", err)
		}
		if _, err = pool.Exec(ctx, `UPDATE scenes SET selected_generation_task_id=$2 WHERE id=$1`, scene.ID, taskID); err != nil {
			t.Fatalf("select scene take fixture: %v", err)
		}
	}
	var videoTasks, submitJobs, downloadJobs int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM scene_generation_tasks WHERE id=$1),(SELECT count(*) FROM river_job WHERE kind='seedance.task.submit' AND args->>'generationTaskId'=$1::text),(SELECT count(*) FROM river_job WHERE kind='seedance.output.download' AND args->>'generationTaskId'=$1::text)`, generation.ID).Scan(&videoTasks, &submitJobs, &downloadJobs); err != nil {
		t.Fatal(err)
	}
	if videoTasks != 1 || submitJobs != 1 || downloadJobs != 1 {
		t.Fatalf("video orchestration invariants tasks=%d submit_jobs=%d download_jobs=%d", videoTasks, submitJobs, downloadJobs)
	}

	logoAssetID := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO media_assets(id,client_id,workspace_id,brand_id,asset_type,category,name,status,usage_rights,created_by,updated_by)VALUES($1,$2,$3,$4,'LOGO','BRAND_LOGO','Northstar primary logo','APPROVED','Integration fixture',$5,$5)`, logoAssetID, clientID, workspaceID, brandID, actorID); err != nil {
		t.Fatalf("seed brand logo: %v", err)
	}
	if _, err = pool.Exec(ctx, `INSERT INTO media_asset_versions(media_asset_id,client_id,workspace_id,version,storage_key,original_filename,mime_type,file_extension,file_size_bytes,checksum_sha256,width,height,verified_at,created_by)VALUES($1,$2,$3,1,$4,'northstar-logo.png','image/png','.png',1024,$5,800,400,now(),$6)`, logoAssetID, clientID, workspaceID, "integration/"+logoAssetID.String()+"/logo.png", strings.Repeat("f", 64), actorID); err != nil {
		t.Fatalf("seed brand logo version: %v", err)
	}
	brandService := brands.NewService(pool)
	brandProfile, err := brandService.Get(ctx, clientID, workspaceID, brandID)
	if err != nil {
		t.Fatalf("load brand before logo selection: %v", err)
	}
	brandProfile, err = brandService.Update(ctx, clientID, workspaceID, brandID, actorID, brands.Input{Name: brandProfile.Name, LogoAssetIDs: []uuid.UUID{logoAssetID}, PrimaryLanguage: brandProfile.PrimaryLanguage, Version: brandProfile.Version, ChangeSummary: "Select integration logo"})
	if err != nil || len(brandProfile.LogoAssetIDs) != 1 || brandProfile.LogoAssetIDs[0] != logoAssetID {
		t.Fatalf("select eligible brand logo: brand=%#v err=%v", brandProfile, err)
	}
	logoAsset, err := mediaService.Get(ctx, clientID, workspaceID, logoAssetID)
	if err != nil || !logoAsset.ReadyForUse {
		t.Fatalf("load ready brand logo: asset=%#v err=%v", logoAsset, err)
	}
	if deleteErr := mediaService.SoftDelete(ctx, clientID, workspaceID, logoAssetID, actorID, logoAsset.Version); !errors.Is(deleteErr, media.ErrInUse) {
		t.Fatalf("expected selected brand logo deletion to fail, got %v", deleteErr)
	}
	if _, attachErr := mediaService.AttachProduct(ctx, clientID, workspaceID, productID, logoAssetID, actorID, logoAsset.Version); !errors.Is(attachErr, media.ErrInUse) {
		t.Fatalf("expected selected brand logo product attachment to fail, got %v", attachErr)
	}
	if _, err = pool.Exec(ctx, `UPDATE media_assets SET status='DRAFT' WHERE id=$1`, logoAssetID); err != nil {
		t.Fatalf("make logo ineligible for render precondition: %v", err)
	}
	renderService := rendering.NewService(pool, jobEnqueuer)
	renderKey := "integration-render-" + uuid.NewString()
	if _, startErr := renderService.Start(ctx, clientID, workspaceID, campaign.ID, actorID, renderKey); !errors.Is(startErr, rendering.ErrPrerequisite) {
		t.Fatalf("expected render with ineligible primary logo to fail, got %v", startErr)
	}
	if _, err = pool.Exec(ctx, `UPDATE media_assets SET status='APPROVED' WHERE id=$1`, logoAssetID); err != nil {
		t.Fatalf("restore eligible logo: %v", err)
	}
	renderJob, err := renderService.Start(ctx, clientID, workspaceID, campaign.ID, actorID, renderKey)
	if err != nil || renderJob.Status != "QUEUED" {
		t.Fatalf("start final render: render=%#v err=%v", renderJob, err)
	}
	reusedRender, err := renderService.Start(ctx, clientID, workspaceID, campaign.ID, actorID, renderKey)
	if err != nil || reusedRender.ID != renderJob.ID || !reusedRender.Reused {
		t.Fatalf("reuse final render: render=%#v err=%v", reusedRender, err)
	}
	var renderJobs, riverRenderJobs int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM render_jobs WHERE id=$1),(SELECT count(*) FROM river_job WHERE kind='video.final-render' AND args->>'renderJobId'=$1::text)`, renderJob.ID).Scan(&renderJobs, &riverRenderJobs); err != nil {
		t.Fatal(err)
	}
	if renderJobs != 1 || riverRenderJobs != 1 {
		t.Fatalf("render orchestration invariants renders=%d river_jobs=%d", renderJobs, riverRenderJobs)
	}
	if _, err = pool.Exec(ctx, `UPDATE render_jobs SET status='REVIEW_REQUIRED',output_asset_id=$2,output_hash=$3 WHERE id=$1`, renderJob.ID, outputAssetID, strings.Repeat("d", 64)); err != nil {
		t.Fatalf("seed review-ready final render: %v", err)
	}
	approvedRender, err := renderService.Review(ctx, clientID, workspaceID, campaign.ID, renderJob.ID, actorID, rendering.ReviewInput{Action: "APPROVE", Version: renderJob.Version, Notes: "Final video checked"})
	if err != nil || approvedRender.Status != "APPROVED" {
		t.Fatalf("approve final render: render=%#v err=%v", approvedRender, err)
	}
	selectedRender, err := renderService.Select(ctx, clientID, workspaceID, campaign.ID, renderJob.ID, actorID)
	if err != nil || !selectedRender.Selected {
		t.Fatalf("select final render: render=%#v err=%v", selectedRender, err)
	}

	secretCipher, err := cryptox.New(bytes.Repeat([]byte{7}, 32))
	if err != nil {
		t.Fatalf("create integration cipher: %v", err)
	}
	demoMeta := metaprovider.NewDemoProvider()
	connectionService := metaconnections.NewService(pool, secretCipher, demoMeta, "demo")
	oauth, err := connectionService.StartOAuth(ctx, clientID, workspaceID, actorID)
	if err != nil {
		t.Fatalf("start Meta OAuth: %v", err)
	}
	oauthURL, err := url.Parse(oauth.AuthorizationURL)
	if err != nil || oauthURL.Query().Get("state") == "" {
		t.Fatalf("parse Meta OAuth URL: url=%q err=%v", oauth.AuthorizationURL, err)
	}
	if _, err = connectionService.Callback(ctx, oauthURL.Query().Get("state"), "demo-code"); err != nil {
		t.Fatalf("complete Meta OAuth: %v", err)
	}
	connection, err := connectionService.Get(ctx, clientID, workspaceID)
	if err != nil || len(connection.Accounts) != 2 || len(connection.AdAccounts) != 1 || len(connection.Pixels) != 1 || len(connection.Audiences) != 1 {
		t.Fatalf("Meta discovery: connection=%#v err=%v", connection, err)
	}
	var facebookAccountID uuid.UUID
	for _, account := range connection.Accounts {
		if account.Platform == "FACEBOOK" {
			facebookAccountID, err = uuid.Parse(account.ID)
			break
		}
	}
	if err != nil || facebookAccountID == uuid.Nil {
		t.Fatalf("select Facebook account: %v", err)
	}

	publishingService := publishing.NewService(pool, jobEnqueuer)
	post, err := publishingService.Create(ctx, clientID, workspaceID, campaign.ID, actorID, "integration-publish-"+uuid.NewString(), publishing.Input{SocialAccountID: facebookAccountID, MediaAssetID: outputAssetID, Caption: "Khám phá Northstar Cabin 20 ngay hôm nay."})
	if err != nil || post.Status != "APPROVAL_REQUIRED" {
		t.Fatalf("create social post: post=%#v err=%v", post, err)
	}
	post, err = publishingService.Review(ctx, clientID, workspaceID, campaign.ID, post.ID, actorID, publishing.ReviewInput{Action: "APPROVE", Version: post.Version, Notes: "Meta publishing integration approval"})
	if err != nil || post.Status != "APPROVED" {
		t.Fatalf("approve social post: post=%#v err=%v", post, err)
	}
	publishWorker := &publishing.Worker{Pool: pool, Store: fakeReferenceStore{}, Cipher: secretCipher, Provider: demoMeta}
	if err = publishWorker.Work(ctx, &river.Job[jobs.SocialPublishArgs]{JobRow: &rivertype.JobRow{ID: 100_004, Attempt: 1, MaxAttempts: 3}, Args: jobs.SocialPublishArgs{SocialPostID: post.ID}}); err != nil {
		t.Fatalf("publish approved social post: %v", err)
	}
	posts, err := publishingService.List(ctx, clientID, workspaceID, campaign.ID)
	if err != nil || len(posts) != 1 || posts[0].Status != "PUBLISHED" || posts[0].ProviderPostID == nil {
		t.Fatalf("published social post: posts=%#v err=%v", posts, err)
	}

	adsService := metaads.NewService(pool, jobEnqueuer)
	guardrails, err := adsService.SaveGuardrails(ctx, clientID, workspaceID, actorID, metaads.Guardrails{WorkspaceSpendCapMinor: 10_000_000, DefaultCampaignSpendCapMinor: 2_000_000, MaximumBudgetIncreasePercent: 20, Currency: "VND", Version: 0})
	if err != nil || guardrails.Currency != "VND" {
		t.Fatalf("save Meta Ads guardrails: guardrails=%#v err=%v", guardrails, err)
	}
	adAccountID, parseErr := uuid.Parse(connection.AdAccounts[0].ID)
	if parseErr != nil {
		t.Fatalf("parse Meta ad account: %v", parseErr)
	}
	pixelID, parseErr := uuid.Parse(connection.Pixels[0].ID)
	if parseErr != nil {
		t.Fatalf("parse Meta Pixel: %v", parseErr)
	}
	dailyBudget := int64(100_000)
	adCampaign, err := adsService.Create(ctx, clientID, workspaceID, campaign.ID, actorID, "integration-ad-create-"+uuid.NewString(), metaads.CampaignInput{
		MetaAdAccountID: adAccountID, SocialAccountID: facebookAccountID, MetaPixelID: &pixelID,
		Name: "Northstar launch", Objective: "OUTCOME_TRAFFIC", DailyBudgetMinor: &dailyBudget,
		CampaignSpendCapMinor: 2_000_000, Currency: "VND",
		Audience:   metaads.Audience{Countries: []string{"VN"}, AgeMin: 25, AgeMax: 45, Interests: []string{"travel"}, RetargetingAudienceIDs: []string{connection.Audiences[0].ProviderAudienceID}},
		Placements: []string{"facebook_feed", "instagram_reels"}, DestinationURL: "https://example.test/products/northstar", UTMParameters: map[string]string{"utm_source": "meta", "utm_campaign": "northstar_launch"},
		Creative: metaads.CreativeInput{MediaAssetID: outputAssetID, PrimaryTextVariants: []string{"Gọn nhẹ cho mọi hành trình", "Sẵn sàng lên đường"}, HeadlineVariants: []string{"Northstar Cabin 20"}, CTAVariants: []string{"LEARN_MORE"}},
	})
	if err != nil || adCampaign.Status != "APPROVAL_REQUIRED" {
		t.Fatalf("create paused Meta campaign draft: campaign=%#v err=%v", adCampaign, err)
	}
	adCampaign, err = adsService.ReviewCreate(ctx, clientID, workspaceID, campaign.ID, adCampaign.ID, actorID, metaads.ReviewInput{Action: "APPROVE", Version: adCampaign.Version, Notes: "Budget and creative reviewed", ConfirmedBudgetMinor: dailyBudget, ConfirmationText: "CREATE PAUSED VND 100000"})
	if err != nil || adCampaign.Status != "APPROVED" {
		t.Fatalf("approve paused Meta campaign creation: campaign=%#v err=%v", adCampaign, err)
	}
	actions, err := adsService.ListActions(ctx, clientID, workspaceID, campaign.ID, adCampaign.ID)
	if err != nil || len(actions) != 1 || actions[0].Action != "CREATE_PAUSED" || actions[0].Status != "QUEUED" {
		t.Fatalf("queued Meta create action: actions=%#v err=%v", actions, err)
	}
	actionWorker := &metaads.ActionWorker{Pool: pool, Cipher: secretCipher, Provider: demoMeta, Enqueuer: jobEnqueuer}
	if err = actionWorker.Work(ctx, &river.Job[jobs.MetaAdActionArgs]{JobRow: &rivertype.JobRow{ID: 100_005, Attempt: 1, MaxAttempts: 1}, Args: jobs.MetaAdActionArgs{ActionID: actions[0].ID}}); err != nil {
		t.Fatalf("create paused Meta campaign: %v", err)
	}
	adCampaigns, err := adsService.List(ctx, clientID, workspaceID, campaign.ID)
	if err != nil || len(adCampaigns) != 1 || adCampaigns[0].Status != "PAUSED" || adCampaigns[0].ProviderCampaignID == nil {
		t.Fatalf("paused Meta campaign: campaigns=%#v err=%v", adCampaigns, err)
	}
	metricsWorker := &metaads.MetricsWorker{Pool: pool, Cipher: secretCipher, Provider: demoMeta}
	if err = metricsWorker.Work(ctx, &river.Job[jobs.MetaMetricsSyncArgs]{JobRow: &rivertype.JobRow{ID: 100_006, Attempt: 1, MaxAttempts: 5}, Args: jobs.MetaMetricsSyncArgs{AdCampaignID: adCampaign.ID}}); err != nil {
		t.Fatalf("sync Meta insights: %v", err)
	}
	var metricRows int
	if err = pool.QueryRow(ctx, `SELECT count(*) FROM ad_campaign_metrics_daily WHERE ad_campaign_id=$1`, adCampaign.ID).Scan(&metricRows); err != nil || metricRows != 1 {
		t.Fatalf("Meta insights rows=%d err=%v", metricRows, err)
	}
	activate, err := adsService.RequestAction(ctx, clientID, workspaceID, campaign.ID, adCampaign.ID, actorID, metaads.ActionInput{Action: "ACTIVATE", ConfirmationText: "ACTIVATE VND 100000", IdempotencyKey: "integration-activate-" + uuid.NewString()})
	if err != nil || activate.Status != "PENDING_APPROVAL" {
		t.Fatalf("request Meta activation: action=%#v err=%v", activate, err)
	}
	activate, err = adsService.ReviewAction(ctx, clientID, workspaceID, campaign.ID, adCampaign.ID, activate.ID, actorID, metaads.ReviewInput{Action: "APPROVE", Version: activate.Version, Notes: "Human confirmed activation budget"})
	if err != nil || activate.Status != "QUEUED" {
		t.Fatalf("approve Meta activation: action=%#v err=%v", activate, err)
	}
	if err = actionWorker.Work(ctx, &river.Job[jobs.MetaAdActionArgs]{JobRow: &rivertype.JobRow{ID: 100_007, Attempt: 1, MaxAttempts: 1}, Args: jobs.MetaAdActionArgs{ActionID: activate.ID}}); err != nil {
		t.Fatalf("activate Meta campaign: %v", err)
	}
	adCampaigns, err = adsService.List(ctx, clientID, workspaceID, campaign.ID)
	if err != nil || adCampaigns[0].Status != "ACTIVE" {
		t.Fatalf("active Meta campaign: campaigns=%#v err=%v", adCampaigns, err)
	}
	tooHighBudget := int64(130_000)
	if _, err = adsService.RequestAction(ctx, clientID, workspaceID, campaign.ID, adCampaign.ID, actorID, metaads.ActionInput{Action: "BUDGET_CHANGE", RequestedBudgetMinor: &tooHighBudget, ConfirmationText: "BUDGET VND 130000", IdempotencyKey: "integration-budget-" + uuid.NewString()}); !errors.Is(err, metaads.ErrGuardrail) {
		t.Fatalf("30 percent budget increase must be blocked, got %v", err)
	}

	analyticsService := analytics.NewService(pool)
	analyticsSummary, err := analyticsService.Summary(ctx, clientID, workspaceID, analytics.Filter{From: time.Now().UTC().AddDate(0, 0, -30), To: time.Now().UTC().AddDate(0, 0, 1), CampaignID: &campaign.ID})
	if err != nil {
		t.Fatalf("analytics summary: %v", err)
	}
	if analyticsSummary.TotalCostUSD <= 0 || analyticsSummary.Video.SceneCount < 1 || analyticsSummary.Ads.Impressions != 12_000 || analyticsSummary.Ads.Clicks != 240 || analyticsSummary.Ads.CTR != 2 || analyticsSummary.Ads.ROAS != 4.8 {
		t.Fatalf("analytics invariants: %#v", analyticsSummary)
	}
	recommendations, err := analyticsService.GenerateRecommendations(ctx, clientID, workspaceID, actorID, &campaign.ID)
	if err != nil || len(recommendations) != 1 || recommendations[0].Type != "SCALE_WINNER" || recommendations[0].Status != "DRAFT" {
		t.Fatalf("generate learning recommendation: recommendations=%#v err=%v", recommendations, err)
	}
	recommendation, err := analyticsService.ReviewRecommendation(ctx, clientID, workspaceID, recommendations[0].ID, actorID, analytics.ReviewInput{Action: "APPROVE", Version: recommendations[0].Version, Notes: "Human reviewed analytics evidence"})
	if err != nil || recommendation.Status != "APPROVED" || recommendation.ActionTaken != "" {
		t.Fatalf("review learning recommendation: recommendation=%#v err=%v", recommendation, err)
	}
	var usageRows, costRows int
	if err = pool.QueryRow(ctx, `SELECT (SELECT count(*) FROM usage_ledger WHERE campaign_id=$1),(SELECT count(*) FROM cost_records WHERE campaign_id=$1)`, campaign.ID).Scan(&usageRows, &costRows); err != nil || usageRows < 5 || costRows < 5 {
		t.Fatalf("usage ledger invariants usage=%d cost=%d err=%v", usageRows, costRows, err)
	}

	operationsApp := fiber.New()
	operationsApp.Get("/overview", operations.NewHandler(pool, cfg).Overview)
	operationsResponse, err := operationsApp.Test(httptest.NewRequest(http.MethodGet, "/overview", nil))
	if err != nil {
		t.Fatalf("operations overview: %v", err)
	}
	defer operationsResponse.Body.Close()
	if operationsResponse.StatusCode != http.StatusOK {
		t.Fatalf("operations overview status=%d", operationsResponse.StatusCode)
	}
	maintenanceWorker := &jobs.MaintenanceWorker{Queries: db.New(pool), Pool: pool}
	if err = maintenanceWorker.Work(ctx, &river.Job[jobs.MaintenanceArgs]{JobRow: &rivertype.JobRow{ID: 100_008}, Args: jobs.MaintenanceArgs{}}); err != nil {
		t.Fatalf("maintenance retention and analytics refresh: %v", err)
	}
	exchangeSnapshotID := uuid.New()
	if _, err = pool.Exec(ctx, `INSERT INTO exchange_rate_snapshots(id,base_currency,quote_currency,rate,source)VALUES($1,'VND','USD',0.00004,'integration-fixture')`, exchangeSnapshotID); err != nil {
		t.Fatalf("seed exchange-rate snapshot: %v", err)
	}
	if err = usage.Record(ctx, pool, usage.Entry{Provider: "integration", Model: "fixture", RequestReference: uuid.NewString(), Operation: "NON_USD_FIXTURE", ClientID: &clientID, WorkspaceID: &workspaceID, CampaignID: &campaign.ID, ProviderReportedCost: floatPointer(100_000), Currency: "VND", ExchangeRateSnapshotID: &exchangeSnapshotID, Outcome: "SUCCESS", Category: "OTHER"}); err != nil {
		t.Fatalf("record normalized non-USD usage: %v", err)
	}
	var normalizedUSD float64
	if err = pool.QueryRow(ctx, `SELECT normalized_amount_usd::float8 FROM cost_records WHERE provider='integration' AND campaign_id=$1 ORDER BY occurred_at DESC LIMIT 1`, campaign.ID).Scan(&normalizedUSD); err != nil || normalizedUSD != 4 {
		t.Fatalf("normalized cost=%f err=%v", normalizedUSD, err)
	}

	var requests, outputs, succeeded, approvals int
	if err = pool.QueryRow(ctx, `SELECT
		(SELECT count(*) FROM provider_requests WHERE campaign_id=$1),
		(SELECT count(*) FROM provider_outputs po JOIN provider_requests pr ON pr.id=po.provider_request_id WHERE pr.campaign_id=$1),
		(SELECT count(*) FROM generation_jobs WHERE campaign_id=$1 AND status='SUCCEEDED'),
		(SELECT count(*) FROM approvals WHERE campaign_id=$1 AND status='APPROVED')`, campaign.ID).Scan(&requests, &outputs, &succeeded, &approvals); err != nil {
		t.Fatal(err)
	}
	if requests != 4 || outputs != 4 || succeeded != 4 || approvals < 4 {
		t.Fatalf("trace invariants requests=%d outputs=%d succeeded=%d approvals=%d", requests, outputs, succeeded, approvals)
	}
}

func integrationDatabaseURL(t *testing.T, ctx context.Context) string {
	t.Helper()
	if configured := strings.TrimSpace(os.Getenv("STUDIO_INTEGRATION_DATABASE_URL")); configured != "" {
		return configured
	}
	if os.Getenv("STUDIO_USE_TESTCONTAINERS") != "true" {
		t.Skip("set STUDIO_USE_TESTCONTAINERS=true or STUDIO_INTEGRATION_DATABASE_URL to run the PostgreSQL integration workflow")
	}
	container, err := postgrescontainer.Run(
		ctx,
		"postgres:18.4-alpine",
		postgrescontainer.WithDatabase("studio_integration"),
		postgrescontainer.WithUsername("studio"),
		postgrescontainer.WithPassword("studio"),
		postgrescontainer.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start PostgreSQL 18 Testcontainer: %v", err)
	}
	testcontainers.CleanupContainer(t, container)
	databaseURL, err := container.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("resolve Testcontainer database URL: %v", err)
	}
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatalf("connect to Testcontainer database: %v", err)
	}
	defer pool.Close()

	_, testFile, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve integration test location")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", "..", "..", ".."))
	migrations, err := filepath.Glob(filepath.Join(root, "database", "migrations", "*.sql"))
	if err != nil || len(migrations) == 0 {
		t.Fatalf("discover SQL migrations: files=%d err=%v", len(migrations), err)
	}
	for _, filename := range migrations {
		contents, readErr := os.ReadFile(filename)
		if readErr != nil {
			t.Fatalf("read migration %s: %v", filepath.Base(filename), readErr)
		}
		tx, beginErr := pool.Begin(ctx)
		if beginErr != nil {
			t.Fatalf("begin migration %s: %v", filepath.Base(filename), beginErr)
		}
		if _, execErr := tx.Exec(ctx, string(contents)); execErr != nil {
			_ = tx.Rollback(ctx)
			t.Fatalf("apply migration %s: %v", filepath.Base(filename), execErr)
		}
		if commitErr := tx.Commit(ctx); commitErr != nil {
			t.Fatalf("commit migration %s: %v", filepath.Base(filename), commitErr)
		}
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	migrator, err := rivermigrate.New(riverpgxv5.New(pool), &rivermigrate.Config{Logger: logger})
	if err != nil {
		t.Fatalf("create River migrator: %v", err)
	}
	if _, err = migrator.Migrate(ctx, rivermigrate.DirectionUp, &rivermigrate.MigrateOpts{}); err != nil {
		t.Fatalf("apply River migrations: %v", err)
	}
	return databaseURL
}

func floatPointer(value float64) *float64 { return &value }

func seedPlanningFixture(t *testing.T, ctx context.Context, pool *pgxpool.Pool) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID) {
	t.Helper()
	actorID, clientID, workspaceID, brandID, productID, factID := uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New(), uuid.New()
	commands := []struct {
		sql  string
		args []any
	}{
		{`INSERT INTO internal_users(id,email,display_name,password_hash,role,requires_password_change) VALUES($1,$2,'M2 Integration','not-used','ADMIN',false)`, []any{actorID, "m2-" + uuid.NewString() + "@example.test"}},
		{`INSERT INTO clients(id,company_name,created_by,updated_by) VALUES($1,$2,$3,$3)`, []any{clientID, "M2 Client " + uuid.NewString()[:8], actorID}},
		{`INSERT INTO workspaces(id,client_id,name,slug,created_by,updated_by) VALUES($1,$2,'M2 Workspace',$3,$4,$4)`, []any{workspaceID, clientID, "m2-" + uuid.NewString()[:8], actorID}},
		{`INSERT INTO brands(id,client_id,workspace_id,name,created_by,updated_by) VALUES($1,$2,$3,'Northstar',$4,$4)`, []any{brandID, clientID, workspaceID, actorID}},
		{`INSERT INTO brand_versions(brand_id,client_id,workspace_id,version,primary_language,created_by) VALUES($1,$2,$3,1,'vi',$4)`, []any{brandID, clientID, workspaceID, actorID}},
		{`INSERT INTO products(id,client_id,workspace_id,brand_id,name,sku,category,vertical_key,status,created_by,updated_by) VALUES($1,$2,$3,$4,'Northstar Cabin 20',$5,'Travel luggage','travel-luggage','APPROVED',$6,$6)`, []any{productID, clientID, workspaceID, brandID, "SKU-" + uuid.NewString()[:8], actorID}},
		{`INSERT INTO product_facts(id,product_id,client_id,workspace_id,fact_key,label,exact_value,source_name,status,locked_value,approved_by,approved_at,created_by,updated_by) VALUES($1,$2,$3,$4,'external_dimensions','Kích thước','55 x 36 x 23 cm','Integration fixture','APPROVED',true,$5,now(),$5,$5)`, []any{factID, productID, clientID, workspaceID, actorID}},
	}
	for _, command := range commands {
		if _, err := pool.Exec(ctx, command.sql, command.args...); err != nil {
			t.Fatalf("seed fixture: %v", err)
		}
	}
	return actorID, clientID, workspaceID, brandID, productID
}
