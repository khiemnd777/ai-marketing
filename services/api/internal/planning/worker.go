package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	studioai "github.com/internal/ai-product-marketing-studio/services/api/internal/ai"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/usage"
)

type Worker struct {
	river.WorkerDefaults[jobs.AIPlanningArgs]
	Pool     *pgxpool.Pool
	Provider studioai.LLMProvider
	Config   config.Config
}

func (w *Worker) Work(ctx context.Context, job *river.Job[jobs.AIPlanningArgs]) (workErr error) {
	if w.Provider == nil {
		return river.JobCancel(errors.New("AI planning provider is not configured"))
	}
	tag, err := w.Pool.Exec(ctx, `UPDATE generation_jobs SET status='RUNNING',started_at=now(),error_code=NULL,error_message=NULL WHERE id=$1 AND client_id=$2 AND workspace_id=$3 AND campaign_id=$4 AND status='QUEUED'`, job.Args.GenerationJobID, job.Args.ClientID, job.Args.WorkspaceID, job.Args.CampaignID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return river.JobCancel(errors.New("generation job is no longer queued"))
	}
	var requestID uuid.UUID
	defer func() {
		if workErr == nil {
			return
		}
		message := workErr.Error()
		if len(message) > 500 {
			message = message[:500]
		}
		_, _ = w.Pool.Exec(context.WithoutCancel(ctx), `UPDATE generation_jobs SET status='FAILED',error_code='generation_failed',error_message=$2,completed_at=now() WHERE id=$1 AND status='RUNNING'`, job.Args.GenerationJobID, message)
		if requestID != uuid.Nil {
			_, _ = w.Pool.Exec(context.WithoutCancel(ctx), `UPDATE provider_requests SET status='FAILED',error_code='generation_failed',error_message=$2,completed_at=now(),latency_ms=EXTRACT(EPOCH FROM(now()-started_at))*1000 WHERE id=$1 AND status='PENDING'`, requestID, message)
		}
	}()

	service := NewService(w.Pool, nil, w.Config)
	planningContext, err := service.planningContext(ctx, job.Args.ClientID, job.Args.WorkspaceID, job.Args.CampaignID)
	if err != nil {
		return river.JobCancel(err)
	}
	var actorID uuid.UUID
	var inputHash string
	var estimatedCost float64
	if err = w.Pool.QueryRow(ctx, `SELECT created_by,input_hash,estimated_cost_usd::float8 FROM generation_jobs WHERE id=$1`, job.Args.GenerationJobID).Scan(&actorID, &inputHash, &estimatedCost); err != nil {
		return err
	}
	requestID = uuid.New()
	if _, err = w.Pool.Exec(ctx, `INSERT INTO provider_requests(id,client_id,workspace_id,campaign_id,provider,operation,model,prompt_version,input_hash,estimated_cost_usd,created_by) VALUES($1,$2,$3,$4,$5,$6::generation_operation,$7,$8,$9,$10,$11)`, requestID, job.Args.ClientID, job.Args.WorkspaceID, job.Args.CampaignID, providerName(w.Config), job.Args.Operation, w.Config.OpenAI.Model, studioai.PromptVersion, inputHash, estimatedCost, actorID); err != nil {
		return err
	}

	var output any
	var metadata studioai.Metadata
	switch job.Args.Operation {
	case "CONCEPTS":
		value, result, callErr := w.Provider.GenerateConcepts(ctx, studioai.ConceptInput{Context: planningContext})
		if callErr != nil {
			return callErr
		}
		if err = studioai.ValidateConcepts(value); err != nil {
			return river.JobCancel(err)
		}
		output, metadata = value, result
	case "CONTENT":
		value, result, callErr := w.Provider.GenerateContent(ctx, studioai.ContentInput{Context: planningContext})
		if callErr != nil {
			return callErr
		}
		if err = studioai.ValidateContent(value, planningContext); err != nil {
			return river.JobCancel(err)
		}
		output, metadata = value, result
	case "SCRIPT":
		value, result, callErr := w.Provider.GenerateScript(ctx, studioai.ScriptInput{Context: planningContext})
		if callErr != nil {
			return callErr
		}
		if err = studioai.ValidateScript(value, planningContext); err != nil {
			return river.JobCancel(err)
		}
		output, metadata = value, result
	case "SCENES":
		_, _, _, script, loadErr := service.currentScriptOutput(ctx, job.Args.ClientID, job.Args.WorkspaceID, job.Args.CampaignID)
		if loadErr != nil {
			return river.JobCancel(loadErr)
		}
		var primaryID, listenerID uuid.UUID
		if err = w.Pool.QueryRow(ctx, `SELECT
			(SELECT character_id FROM campaign_characters WHERE campaign_id=$1 AND role='PRIMARY'),
			(SELECT character_id FROM campaign_characters WHERE campaign_id=$1 AND role='LISTENER')`, job.Args.CampaignID).Scan(&primaryID, &listenerID); err != nil {
			return err
		}
		input := studioai.SceneInput{Context: planningContext, Script: script, SpeakerCharacterID: primaryID, ListenerCharacterID: listenerID}
		value, result, callErr := w.Provider.GenerateScenes(ctx, input)
		if callErr != nil {
			return callErr
		}
		if err = studioai.ValidateScenes(value, input); err != nil {
			return river.JobCancel(err)
		}
		output, metadata = value, result
	default:
		return river.JobCancel(ErrInvalid)
	}
	return w.persist(ctx, job, requestID, actorID, estimatedCost, output, metadata)
}

func (w *Worker) persist(ctx context.Context, job *river.Job[jobs.AIPlanningArgs], requestID, actorID uuid.UUID, estimatedCost float64, output any, metadata studioai.Metadata) error {
	hash, normalized, err := studioai.Hash(output)
	if err != nil {
		return err
	}
	actualCost := float64(metadata.InputTokens)/1_000_000*w.Config.OpenAI.InputUSDPer1M + float64(metadata.OutputTokens)/1_000_000*w.Config.OpenAI.OutputUSDPer1M
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	summary := map[string]any{}
	switch value := output.(type) {
	case studioai.ConceptOutput:
		if err = persistConcepts(ctx, tx, job.Args, value, metadata, actorID); err != nil {
			return err
		}
		summary["conceptCount"] = len(value.Concepts)
	case studioai.ContentOutput:
		if err = persistContent(ctx, tx, job.Args, value, metadata, actorID); err != nil {
			return err
		}
		summary["variantCount"] = len(value.Variants)
	case studioai.ScriptOutput:
		if err = persistScript(ctx, tx, job.Args, value, metadata, actorID); err != nil {
			return err
		}
		summary["dialogueTurnCount"] = len(value.DialogueTurns)
	case studioai.SceneOutput:
		if err = persistScenes(ctx, tx, job.Args, value, actorID); err != nil {
			return err
		}
		summary["sceneCount"] = len(value.Scenes)
		for _, scene := range value.Scenes {
			actualCost += scene.ExpectedCostUSD
		}
	default:
		return ErrInvalid
	}
	if _, err = tx.Exec(ctx, `INSERT INTO provider_outputs(provider_request_id,output_hash,normalized_output)VALUES($1,$2,$3)`, requestID, hash, normalized); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `UPDATE provider_requests SET status='SUCCEEDED',provider_request_id=$2,completed_at=now(),latency_ms=$3,input_tokens=$4,output_tokens=$5,actual_cost_usd=$6 WHERE id=$1`, requestID, metadata.RequestID, metadata.Latency.Milliseconds(), metadata.InputTokens, metadata.OutputTokens, actualCost); err != nil {
		return err
	}
	summaryRaw, _ := json.Marshal(summary)
	if _, err = tx.Exec(ctx, `UPDATE generation_jobs SET status='SUCCEEDED',provider_request_id=$2,actual_cost_usd=$3,output_summary=$4,completed_at=now() WHERE id=$1 AND status='RUNNING'`, job.Args.GenerationJobID, requestID, actualCost, summaryRaw); err != nil {
		return err
	}
	if err = audit.Record(ctx, db.New(tx), audit.Event{Action: "ai_planning.generated", EntityType: "campaign", EntityID: valid(job.Args.CampaignID), ClientID: valid(job.Args.ClientID), WorkspaceID: valid(job.Args.WorkspaceID), RequestID: fmt.Sprintf("river:%d", job.ID), Outcome: "SUCCESS", After: summary, Metadata: map[string]any{"operation": job.Args.Operation, "providerRequestId": requestID, "outputHash": hash, "actualCostUsd": actualCost}}); err != nil {
		return err
	}
	if err = usage.Record(ctx, tx, usage.Entry{Provider: providerName(w.Config), Model: metadata.Model, RequestReference: requestID.String(), Operation: job.Args.Operation, ClientID: &job.Args.ClientID, WorkspaceID: &job.Args.WorkspaceID, CampaignID: &job.Args.CampaignID, InputUnits: metadata.InputTokens, OutputUnits: metadata.OutputTokens, ProviderReportedCost: &actualCost, EstimatedCost: estimatedCost, Currency: "USD", Outcome: "SUCCESS", Category: "LLM", Metadata: map[string]any{"promptVersion": metadata.PromptVersion, "providerRequestId": metadata.RequestID}}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func persistConcepts(ctx context.Context, tx pgx.Tx, args jobs.AIPlanningArgs, output studioai.ConceptOutput, metadata studioai.Metadata, actorID uuid.UUID) error {
	for _, concept := range output.Concepts {
		hash, raw, err := studioai.Hash(concept)
		if err != nil {
			return err
		}
		id := uuid.New()
		if _, err = tx.Exec(ctx, `INSERT INTO campaign_concepts(id,campaign_id,client_id,workspace_id,title,video_format,payload,prompt_version,model,request_id,output_hash,estimated_cost_usd,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$13)`, id, args.CampaignID, args.ClientID, args.WorkspaceID, concept.Title, concept.VideoFormat, raw, metadata.PromptVersion, metadata.Model, metadata.RequestID, hash, concept.EstimatedCostUSD, actorID); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO campaign_concept_versions(concept_id,version,title,video_format,payload,output_hash,change_summary,created_by)VALUES($1,1,$2,$3,$4,$5,'AI generation',$6)`, id, concept.Title, concept.VideoFormat, raw, hash, actorID); err != nil {
			return err
		}
	}
	return nil
}
func persistContent(ctx context.Context, tx pgx.Tx, args jobs.AIPlanningArgs, output studioai.ContentOutput, metadata studioai.Metadata, actorID uuid.UUID) error {
	for _, variant := range output.Variants {
		hash, _, _ := studioai.Hash(variant.Content)
		var id uuid.UUID
		var next int32
		err := tx.QueryRow(ctx, `SELECT id,current_version+1 FROM campaign_content_variants WHERE campaign_id=$1 AND variant_key=$2 FOR UPDATE`, args.CampaignID, variant.Key).Scan(&id, &next)
		if errors.Is(err, pgx.ErrNoRows) {
			id = uuid.New()
			next = 1
			_, err = tx.Exec(ctx, `INSERT INTO campaign_content_variants(id,campaign_id,client_id,workspace_id,variant_key,platform,content,content_hash,prompt_version,model,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$11)`, id, args.CampaignID, args.ClientID, args.WorkspaceID, variant.Key, variant.Platform, variant.Content, hash, metadata.PromptVersion, metadata.Model, actorID)
		} else if err == nil {
			if err = invalidateEntity(ctx, tx, "CONTENT_VARIANT", id, "Content regenerated"); err != nil {
				return err
			}
			_, err = tx.Exec(ctx, `UPDATE campaign_content_variants SET platform=$2,content=$3,content_hash=$4,prompt_version=$5,model=$6,status='DRAFT',approved_at=NULL,approved_by=NULL,current_version=$7,version=version+1,updated_by=$8,updated_at=now() WHERE id=$1`, id, variant.Platform, variant.Content, hash, metadata.PromptVersion, metadata.Model, next, actorID)
		}
		if err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `INSERT INTO campaign_content_variant_versions(content_variant_id,version,content,content_hash,change_summary,created_by)VALUES($1,$2,$3,$4,'AI generation',$5)`, id, next, variant.Content, hash, actorID); err != nil {
			return err
		}
	}
	return nil
}
func persistScript(ctx context.Context, tx pgx.Tx, args jobs.AIPlanningArgs, output studioai.ScriptOutput, metadata studioai.Metadata, actorID uuid.UUID) error {
	hash, _, _ := studioai.Hash(output)
	var id uuid.UUID
	var next int32
	err := tx.QueryRow(ctx, `SELECT id,current_version+1 FROM scripts WHERE campaign_id=$1 FOR UPDATE`, args.CampaignID).Scan(&id, &next)
	if errors.Is(err, pgx.ErrNoRows) {
		id = uuid.New()
		next = 1
		_, err = tx.Exec(ctx, `INSERT INTO scripts(id,campaign_id,client_id,workspace_id,script_hash,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$6)`, id, args.CampaignID, args.ClientID, args.WorkspaceID, hash, actorID)
	} else if err == nil {
		if err = invalidateEntity(ctx, tx, "SCRIPT", id, "Script regenerated"); err != nil {
			return err
		}
		if err = invalidateCampaignSceneApprovals(ctx, tx, args.CampaignID, "Script regenerated"); err != nil {
			return err
		}
		if _, err = tx.Exec(ctx, `DELETE FROM scenes WHERE campaign_id=$1`, args.CampaignID); err != nil {
			return err
		}
		_, err = tx.Exec(ctx, `UPDATE scripts SET status='DRAFT',approved_at=NULL,approved_by=NULL,current_version=$2,script_hash=$3,version=version+1,updated_by=$4,updated_at=now() WHERE id=$1`, id, next, hash, actorID)
	}
	if err != nil {
		return err
	}
	if err = insertScriptVersion(ctx, tx, id, next, output, hash, "AI generation", actorID, metadata.PromptVersion, metadata.Model); err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `UPDATE campaigns SET status='SCRIPT_READY',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, args.CampaignID, actorID)
	return err
}
func persistScenes(ctx context.Context, tx pgx.Tx, args jobs.AIPlanningArgs, output studioai.SceneOutput, actorID uuid.UUID) error {
	var scriptID uuid.UUID
	if err := tx.QueryRow(ctx, `SELECT id FROM scripts WHERE campaign_id=$1 AND status='APPROVED'`, args.CampaignID).Scan(&scriptID); err != nil {
		return ErrPrerequisite
	}
	var approved int
	if err := tx.QueryRow(ctx, `SELECT count(*) FROM scenes WHERE campaign_id=$1 AND status='APPROVED'`, args.CampaignID).Scan(&approved); err != nil {
		return err
	}
	if approved > 0 {
		return ErrLocked
	}
	if _, err := tx.Exec(ctx, `DELETE FROM scenes WHERE campaign_id=$1`, args.CampaignID); err != nil {
		return err
	}
	for _, scene := range output.Scenes {
		hash, _, err := studioai.Hash(scene)
		if err != nil {
			return err
		}
		id := uuid.New()
		if _, err = tx.Exec(ctx, `INSERT INTO scenes(id,campaign_id,script_id,client_id,workspace_id,scene_key,scene_order,scene_hash,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, id, args.CampaignID, scriptID, args.ClientID, args.WorkspaceID, scene.SceneID, scene.Order, hash, actorID); err != nil {
			return err
		}
		if err = insertSceneVersion(ctx, tx, id, 1, scene, hash, "AI generation", actorID, args.ClientID, args.WorkspaceID, args.CampaignID); err != nil {
			return err
		}
	}
	_, err := tx.Exec(ctx, `UPDATE campaigns SET status='SCENE_REVIEW',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, args.CampaignID, actorID)
	return err
}
func providerName(cfg config.Config) string {
	if cfg.DemoMode {
		return "demo"
	}
	return "openai"
}
