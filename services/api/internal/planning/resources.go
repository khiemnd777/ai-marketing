package planning

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	studioai "github.com/internal/ai-product-marketing-studio/services/api/internal/ai"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
)

type Concept struct {
	ID               uuid.UUID                 `json:"id"`
	CampaignID       uuid.UUID                 `json:"campaignId"`
	Status           string                    `json:"status"`
	Payload          studioai.ConceptCandidate `json:"payload"`
	CurrentVersion   int32                     `json:"currentVersion"`
	PromptVersion    string                    `json:"promptVersion"`
	Model            string                    `json:"model"`
	RequestID        string                    `json:"requestId"`
	OutputHash       string                    `json:"outputHash"`
	EstimatedCostUSD float64                   `json:"estimatedCostUsd"`
	Version          int64                     `json:"version"`
	LockedAt         *time.Time                `json:"lockedAt"`
	CreatedAt        time.Time                 `json:"createdAt"`
	UpdatedAt        time.Time                 `json:"updatedAt"`
}

type ContentVariant struct {
	ID             uuid.UUID  `json:"id"`
	CampaignID     uuid.UUID  `json:"campaignId"`
	Key            string     `json:"key"`
	Platform       string     `json:"platform"`
	Content        string     `json:"content"`
	Status         string     `json:"status"`
	CurrentVersion int32      `json:"currentVersion"`
	ContentHash    string     `json:"contentHash"`
	PromptVersion  string     `json:"promptVersion"`
	Model          string     `json:"model"`
	Version        int64      `json:"version"`
	ApprovedAt     *time.Time `json:"approvedAt"`
	UpdatedAt      time.Time  `json:"updatedAt"`
}

type Script struct {
	ID             uuid.UUID             `json:"id"`
	CampaignID     uuid.UUID             `json:"campaignId"`
	Status         string                `json:"status"`
	CurrentVersion int32                 `json:"currentVersion"`
	ScriptHash     string                `json:"scriptHash"`
	Version        int64                 `json:"version"`
	ApprovedAt     *time.Time            `json:"approvedAt"`
	Output         studioai.ScriptOutput `json:"output"`
	UpdatedAt      time.Time             `json:"updatedAt"`
}

type Scene struct {
	ID             uuid.UUID               `json:"id"`
	CampaignID     uuid.UUID               `json:"campaignId"`
	ScriptID       uuid.UUID               `json:"scriptId"`
	SceneKey       string                  `json:"sceneKey"`
	Order          int32                   `json:"order"`
	Status         string                  `json:"status"`
	CurrentVersion int32                   `json:"currentVersion"`
	SceneHash      string                  `json:"sceneHash"`
	Version        int64                   `json:"version"`
	ApprovedAt     *time.Time              `json:"approvedAt"`
	Direction      studioai.SceneDirection `json:"direction"`
	UpdatedAt      time.Time               `json:"updatedAt"`
}

func (s *Service) ListConcepts(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) ([]Concept, error) {
	rows, err := s.pool.Query(ctx, `SELECT cc.id,cc.campaign_id,cc.status::text,cc.payload,cc.current_version,cc.prompt_version,cc.model,cc.request_id,cc.output_hash,cc.estimated_cost_usd::float8,cc.version,cc.locked_at,cc.created_at,cc.updated_at FROM campaign_concepts cc JOIN campaigns c ON c.id=cc.campaign_id WHERE cc.campaign_id=$1 AND c.client_id=$2 AND c.workspace_id=$3 ORDER BY cc.created_at`, campaignID, clientID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Concept{}
	for rows.Next() {
		item, err := scanConcept(rows)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	return items, rows.Err()
}

func (s *Service) UpdateConcept(ctx context.Context, clientID, workspaceID, campaignID, conceptID uuid.UUID, payload studioai.ConceptCandidate, version int64, actor auth.Principal, metadata auth.ClientMetadata) (Concept, error) {
	if err := validateConcept(payload); err != nil || version < 1 {
		return Concept{}, ErrInvalid
	}
	hash, raw, err := studioai.Hash(payload)
	if err != nil {
		return Concept{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Concept{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := scanConcept(tx.QueryRow(ctx, `SELECT cc.id,cc.campaign_id,cc.status::text,cc.payload,cc.current_version,cc.prompt_version,cc.model,cc.request_id,cc.output_hash,cc.estimated_cost_usd::float8,cc.version,cc.locked_at,cc.created_at,cc.updated_at FROM campaign_concepts cc JOIN campaigns c ON c.id=cc.campaign_id WHERE cc.id=$1 AND cc.campaign_id=$2 AND c.client_id=$3 AND c.workspace_id=$4 FOR UPDATE OF cc`, conceptID, campaignID, clientID, workspaceID))
	if err != nil {
		return Concept{}, err
	}
	if before.Status == "LOCKED" {
		return Concept{}, ErrLocked
	}
	if before.Version != version {
		return Concept{}, ErrConflict
	}
	var next int32
	err = tx.QueryRow(ctx, `UPDATE campaign_concepts SET title=$5,video_format=$6,payload=$7,status='DRAFT',locked_at=NULL,locked_by=NULL,current_version=current_version+1,version=version+1,output_hash=$8,updated_by=$9,updated_at=now() WHERE id=$1 AND campaign_id=$2 AND client_id=$3 AND workspace_id=$4 RETURNING current_version`, conceptID, campaignID, clientID, workspaceID, payload.Title, payload.VideoFormat, raw, hash, actor.UserID).Scan(&next)
	if err != nil {
		return Concept{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO campaign_concept_versions(concept_id,version,title,video_format,payload,output_hash,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6,'Operator edit',$7)`, conceptID, next, payload.Title, payload.VideoFormat, raw, hash, actor.UserID); err != nil {
		return Concept{}, err
	}
	if err = invalidateEntity(ctx, tx, "CONCEPT", conceptID, "Concept changed"); err != nil {
		return Concept{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET selected_concept_id=NULL,status='DRAFT',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1 AND selected_concept_id=$3`, campaignID, actor.UserID, conceptID); err != nil {
		return Concept{}, err
	}
	if err = invalidateCampaignDownstream(ctx, tx, campaignID, actor.UserID, "Concept changed"); err != nil {
		return Concept{}, err
	}
	after, err := scanConcept(tx.QueryRow(ctx, `SELECT id,campaign_id,status::text,payload,current_version,prompt_version,model,request_id,output_hash,estimated_cost_usd::float8,version,locked_at,created_at,updated_at FROM campaign_concepts WHERE id=$1`, conceptID))
	if err != nil {
		return Concept{}, err
	}
	if err = audit.Record(ctx, db.New(tx), audit.Event{ActorID: valid(actor.UserID), Action: "campaign_concept.updated", EntityType: "campaign_concept", EntityID: valid(conceptID), ClientID: valid(clientID), WorkspaceID: valid(workspaceID), RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: after}); err != nil {
		return Concept{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Concept{}, err
	}
	return after, nil
}

func (s *Service) DecideConcept(ctx context.Context, clientID, workspaceID, campaignID, conceptID uuid.UUID, action string, version int64, notes string, actor auth.Principal, metadata auth.ClientMetadata) (Concept, error) {
	action = strings.ToUpper(strings.TrimSpace(action))
	if !map[string]bool{"APPROVE": true, "REJECT": true, "LOCK": true}[action] || version < 1 {
		return Concept{}, ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Concept{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	before, err := scanConcept(tx.QueryRow(ctx, `SELECT cc.id,cc.campaign_id,cc.status::text,cc.payload,cc.current_version,cc.prompt_version,cc.model,cc.request_id,cc.output_hash,cc.estimated_cost_usd::float8,cc.version,cc.locked_at,cc.created_at,cc.updated_at FROM campaign_concepts cc JOIN campaigns c ON c.id=cc.campaign_id WHERE cc.id=$1 AND cc.campaign_id=$2 AND c.client_id=$3 AND c.workspace_id=$4 FOR UPDATE OF cc`, conceptID, campaignID, clientID, workspaceID))
	if err != nil {
		return Concept{}, err
	}
	if before.Version != version {
		return Concept{}, ErrConflict
	}
	switch action {
	case "APPROVE":
		if before.Status == "LOCKED" {
			return Concept{}, ErrLocked
		}
		if before.Status == "APPROVED" {
			return Concept{}, ErrConflict
		}
		if _, err = tx.Exec(ctx, `WITH changed AS (
			UPDATE approvals SET invalidated_at=now(),invalidation_reason='Another concept approved'
			WHERE campaign_id=$1 AND entity_type='CONCEPT' AND invalidated_at IS NULL
			RETURNING id,entity_version,entity_hash
		) INSERT INTO approval_events(approval_id,event_type,entity_version,entity_hash,notes)
		SELECT id,'INVALIDATED',entity_version,entity_hash,'Another concept approved' FROM changed`, campaignID); err != nil {
			return Concept{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE campaign_concepts SET status=CASE WHEN id=$2 THEN 'APPROVED'::concept_status ELSE CASE WHEN status='APPROVED' THEN 'REJECTED'::concept_status ELSE status END END,version=version+1,updated_by=$3,updated_at=now() WHERE campaign_id=$1 AND (id=$2 OR status='APPROVED')`, campaignID, conceptID, actor.UserID); err != nil {
			return Concept{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE campaigns SET selected_concept_id=$2,version=version+1,updated_by=$3,updated_at=now() WHERE id=$1`, campaignID, conceptID, actor.UserID); err != nil {
			return Concept{}, err
		}
		if err = recordApproval(ctx, tx, clientID, workspaceID, campaignID, "CONCEPT", conceptID, before.Version+1, before.OutputHash, notes, actor.UserID); err != nil {
			return Concept{}, err
		}
	case "REJECT":
		if before.Status == "LOCKED" {
			return Concept{}, ErrLocked
		}
		if before.Status == "REJECTED" {
			return Concept{}, ErrConflict
		}
		if err = invalidateEntity(ctx, tx, "CONCEPT", conceptID, "Concept rejected"); err != nil {
			return Concept{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE campaign_concepts SET status='REJECTED',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, conceptID, actor.UserID); err != nil {
			return Concept{}, err
		}
		if _, err = tx.Exec(ctx, `UPDATE campaigns SET selected_concept_id=NULL,status='DRAFT',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1 AND selected_concept_id=$3`, campaignID, actor.UserID, conceptID); err != nil {
			return Concept{}, err
		}
	case "LOCK":
		if before.Status != "APPROVED" {
			return Concept{}, ErrPrerequisite
		}
		if _, err = tx.Exec(ctx, `UPDATE campaign_concepts SET status='LOCKED',locked_at=now(),locked_by=$2,version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, conceptID, actor.UserID); err != nil {
			return Concept{}, err
		}
	}
	after, err := scanConcept(tx.QueryRow(ctx, `SELECT id,campaign_id,status::text,payload,current_version,prompt_version,model,request_id,output_hash,estimated_cost_usd::float8,version,locked_at,created_at,updated_at FROM campaign_concepts WHERE id=$1`, conceptID))
	if err != nil {
		return Concept{}, err
	}
	if err = audit.Record(ctx, db.New(tx), audit.Event{ActorID: valid(actor.UserID), Action: "campaign_concept." + strings.ToLower(action), EntityType: "campaign_concept", EntityID: valid(conceptID), ClientID: valid(clientID), WorkspaceID: valid(workspaceID), RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", Before: before, After: after, Metadata: map[string]any{"notes": notes}}); err != nil {
		return Concept{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Concept{}, err
	}
	return after, nil
}

func (s *Service) ListContent(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) ([]ContentVariant, error) {
	rows, err := s.pool.Query(ctx, `SELECT cv.id,cv.campaign_id,cv.variant_key,cv.platform,cv.content,cv.status::text,cv.current_version,cv.content_hash,cv.prompt_version,cv.model,cv.version,cv.approved_at,cv.updated_at FROM campaign_content_variants cv JOIN campaigns c ON c.id=cv.campaign_id WHERE cv.campaign_id=$1 AND c.client_id=$2 AND c.workspace_id=$3 ORDER BY cv.variant_key`, campaignID, clientID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []ContentVariant{}
	for rows.Next() {
		var i ContentVariant
		if err = rows.Scan(&i.ID, &i.CampaignID, &i.Key, &i.Platform, &i.Content, &i.Status, &i.CurrentVersion, &i.ContentHash, &i.PromptVersion, &i.Model, &i.Version, &i.ApprovedAt, &i.UpdatedAt); err != nil {
			return nil, err
		}
		items = append(items, i)
	}
	return items, rows.Err()
}

func (s *Service) UpdateContent(ctx context.Context, clientID, workspaceID, campaignID, id uuid.UUID, content string, version int64, actor auth.Principal) (ContentVariant, error) {
	content = strings.TrimSpace(content)
	if content == "" || version < 1 {
		return ContentVariant{}, ErrInvalid
	}
	planningContext, err := s.planningContext(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return ContentVariant{}, err
	}
	var current ContentVariant
	err = s.pool.QueryRow(ctx, `SELECT cv.id,cv.campaign_id,cv.variant_key,cv.platform,cv.content,cv.status::text,cv.current_version,cv.content_hash,cv.prompt_version,cv.model,cv.version,cv.approved_at,cv.updated_at FROM campaign_content_variants cv JOIN campaigns c ON c.id=cv.campaign_id WHERE cv.id=$1 AND cv.campaign_id=$2 AND c.client_id=$3 AND c.workspace_id=$4`, id, campaignID, clientID, workspaceID).Scan(&current.ID, &current.CampaignID, &current.Key, &current.Platform, &current.Content, &current.Status, &current.CurrentVersion, &current.ContentHash, &current.PromptVersion, &current.Model, &current.Version, &current.ApprovedAt, &current.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ContentVariant{}, ErrNotFound
	}
	if err != nil {
		return ContentVariant{}, err
	}
	variants, err := s.ListContent(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return ContentVariant{}, err
	}
	output := studioai.ContentOutput{Variants: make([]studioai.ContentVariant, 0, len(variants))}
	for _, item := range variants {
		value := item.Content
		if item.ID == id {
			value = content
		}
		output.Variants = append(output.Variants, studioai.ContentVariant{Key: item.Key, Platform: item.Platform, Content: value})
	}
	if err = studioai.ValidateContent(output, planningContext); err != nil {
		return ContentVariant{}, ErrInvalid
	}
	hash, _, _ := studioai.Hash(content)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ContentVariant{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var next int32
	err = tx.QueryRow(ctx, `UPDATE campaign_content_variants SET content=$5,content_hash=$6,status='DRAFT',approved_at=NULL,approved_by=NULL,current_version=current_version+1,version=version+1,updated_by=$7,updated_at=now() WHERE id=$1 AND campaign_id=$2 AND client_id=$3 AND workspace_id=$4 AND version=$8 RETURNING current_version`, id, campaignID, clientID, workspaceID, content, hash, actor.UserID, version).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return ContentVariant{}, ErrConflict
	}
	if err != nil {
		return ContentVariant{}, err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO campaign_content_variant_versions(content_variant_id,version,content,content_hash,change_summary,created_by)VALUES($1,$2,$3,$4,'Operator edit',$5)`, id, next, content, hash, actor.UserID); err != nil {
		return ContentVariant{}, err
	}
	if err = invalidateEntity(ctx, tx, "CONTENT_VARIANT", id, "Content changed"); err != nil {
		return ContentVariant{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ContentVariant{}, err
	}
	items, err := s.ListContent(ctx, clientID, workspaceID, campaignID)
	for _, item := range items {
		if item.ID == id {
			return item, err
		}
	}
	return ContentVariant{}, ErrNotFound
}

func (s *Service) ApproveContent(ctx context.Context, clientID, workspaceID, campaignID, id uuid.UUID, version int64, notes string, actor auth.Principal) (ContentVariant, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return ContentVariant{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var item ContentVariant
	err = tx.QueryRow(ctx, `UPDATE campaign_content_variants SET status='APPROVED',approved_at=now(),approved_by=$5,version=version+1,updated_by=$5,updated_at=now() WHERE id=$1 AND campaign_id=$2 AND client_id=$3 AND workspace_id=$4 AND version=$6 AND status='DRAFT' RETURNING id,campaign_id,variant_key,platform,content,status::text,current_version,content_hash,prompt_version,model,version,approved_at,updated_at`, id, campaignID, clientID, workspaceID, actor.UserID, version).Scan(&item.ID, &item.CampaignID, &item.Key, &item.Platform, &item.Content, &item.Status, &item.CurrentVersion, &item.ContentHash, &item.PromptVersion, &item.Model, &item.Version, &item.ApprovedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return ContentVariant{}, ErrConflict
	}
	if err != nil {
		return ContentVariant{}, err
	}
	if err = recordApproval(ctx, tx, clientID, workspaceID, campaignID, "CONTENT_VARIANT", id, item.Version, item.ContentHash, notes, actor.UserID); err != nil {
		return ContentVariant{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ContentVariant{}, err
	}
	return item, nil
}

func (s *Service) GetScript(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) (Script, error) {
	id, version, hash, output, err := s.currentScriptOutput(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return Script{}, err
	}
	var item Script
	err = s.pool.QueryRow(ctx, `SELECT id,campaign_id,status::text,current_version,script_hash,version,approved_at,updated_at FROM scripts WHERE id=$1`, id).Scan(&item.ID, &item.CampaignID, &item.Status, &item.CurrentVersion, &item.ScriptHash, &item.Version, &item.ApprovedAt, &item.UpdatedAt)
	if err != nil {
		return Script{}, err
	}
	item.Version = version
	item.ScriptHash = hash
	item.Output = output
	return item, nil
}

func (s *Service) UpdateScript(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID, output studioai.ScriptOutput, version int64, actor auth.Principal) (Script, error) {
	cfg, configErr := s.effectiveConfig(ctx, clientID)
	if configErr != nil {
		return Script{}, configErr
	}
	planningContext, err := s.planningContext(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return Script{}, err
	}
	if version < 1 || studioai.ValidateScript(output, planningContext) != nil {
		return Script{}, ErrInvalid
	}
	scriptID, currentVersion, _, _, err := s.currentScriptOutput(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return Script{}, err
	}
	if currentVersion != version {
		return Script{}, ErrConflict
	}
	hash, _, _ := studioai.Hash(output)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Script{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var next int32
	err = tx.QueryRow(ctx, `UPDATE scripts SET status='DRAFT',approved_at=NULL,approved_by=NULL,current_version=current_version+1,script_hash=$2,version=version+1,updated_by=$3,updated_at=now() WHERE id=$1 AND version=$4 RETURNING current_version`, scriptID, hash, actor.UserID, version).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return Script{}, ErrConflict
	}
	if err != nil {
		return Script{}, err
	}
	if err = insertScriptVersion(ctx, tx, scriptID, next, output, hash, "Operator edit", actor.UserID, studioai.PromptVersion, cfg.OpenAI.Model); err != nil {
		return Script{}, err
	}
	if err = invalidateEntity(ctx, tx, "SCRIPT", scriptID, "Script changed"); err != nil {
		return Script{}, err
	}
	if err = invalidateCampaignSceneApprovals(ctx, tx, campaignID, "Script changed"); err != nil {
		return Script{}, err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM scenes WHERE campaign_id=$1`, campaignID); err != nil {
		return Script{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET status='SCRIPT_READY',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, campaignID, actor.UserID); err != nil {
		return Script{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Script{}, err
	}
	return s.GetScript(ctx, clientID, workspaceID, campaignID)
}

func (s *Service) ApproveScript(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID, version int64, notes string, actor auth.Principal, metadata auth.ClientMetadata) (Script, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Script{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	var hash string
	var next int64
	err = tx.QueryRow(ctx, `UPDATE scripts SET status='APPROVED',approved_at=now(),approved_by=$4,version=version+1,updated_by=$4,updated_at=now() WHERE campaign_id=$1 AND client_id=$2 AND workspace_id=$3 AND version=$5 AND status='DRAFT' RETURNING id,script_hash,version`, campaignID, clientID, workspaceID, actor.UserID, version).Scan(&id, &hash, &next)
	if errors.Is(err, pgx.ErrNoRows) {
		return Script{}, ErrConflict
	}
	if err != nil {
		return Script{}, err
	}
	if err = recordApproval(ctx, tx, clientID, workspaceID, campaignID, "SCRIPT", id, next, hash, notes, actor.UserID); err != nil {
		return Script{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET status='SCRIPT_APPROVED',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, campaignID, actor.UserID); err != nil {
		return Script{}, err
	}
	if err = audit.Record(ctx, db.New(tx), audit.Event{ActorID: valid(actor.UserID), Action: "script.approved", EntityType: "script", EntityID: valid(id), ClientID: valid(clientID), WorkspaceID: valid(workspaceID), RequestID: metadata.RequestID, IPAddress: metadata.IPAddress, UserAgent: metadata.UserAgent, Outcome: "SUCCESS", After: map[string]any{"version": next, "hash": hash, "notes": notes}}); err != nil {
		return Script{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Script{}, err
	}
	return s.GetScript(ctx, clientID, workspaceID, campaignID)
}

func (s *Service) ListScenes(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) ([]Scene, error) {
	rows, err := s.pool.Query(ctx, `SELECT sc.id,sc.campaign_id,sc.script_id,sc.scene_key,sc.scene_order,sc.status::text,sc.current_version,sc.scene_hash,sc.version,sc.approved_at,sc.updated_at,sv.duration_seconds,sv.generation_method,coalesce(sv.speaker_character_id,'00000000-0000-0000-0000-000000000000'::uuid),coalesce(sv.listener_character_id,'00000000-0000-0000-0000-000000000000'::uuid),sv.dialogue,sv.speaker_action,sv.listener_action,sv.camera,sv.environment,sv.product_placement,sv.expected_cost_usd::float8,sv.seedance_prompt,
		coalesce((SELECT array_agg(sa.media_asset_id ORDER BY sa.media_asset_id) FROM scene_assets sa WHERE sa.scene_version_id=sv.id),'{}'::uuid[]),
		coalesce((SELECT array_agg(sf.product_fact_id ORDER BY sf.product_fact_id) FROM scene_required_facts sf WHERE sf.scene_version_id=sv.id),'{}'::uuid[])
		FROM scenes sc JOIN scene_versions sv ON sv.scene_id=sc.id AND sv.version=sc.current_version JOIN campaigns c ON c.id=sc.campaign_id WHERE sc.campaign_id=$1 AND c.client_id=$2 AND c.workspace_id=$3 ORDER BY sc.scene_order`, campaignID, clientID, workspaceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	items := []Scene{}
	for rows.Next() {
		var i Scene
		d := &i.Direction
		err = rows.Scan(&i.ID, &i.CampaignID, &i.ScriptID, &i.SceneKey, &i.Order, &i.Status, &i.CurrentVersion, &i.SceneHash, &i.Version, &i.ApprovedAt, &i.UpdatedAt, &d.DurationSeconds, &d.GenerationMethod, &d.SpeakerCharacterID, &d.ListenerCharacterID, &d.Dialogue, &d.SpeakerAction, &d.ListenerAction, &d.Camera, &d.Environment, &d.ProductPlacement, &d.ExpectedCostUSD, &d.SeedancePrompt, &d.ReferenceAssetIDs, &d.RequiredProductFactIDs)
		if err != nil {
			return nil, err
		}
		d.SceneID = i.SceneKey
		d.Order = i.Order
		items = append(items, i)
	}
	return items, rows.Err()
}

func (s *Service) UpdateScene(ctx context.Context, clientID, workspaceID, campaignID, sceneID uuid.UUID, direction studioai.SceneDirection, version int64, actor auth.Principal) (Scene, error) {
	scenes, err := s.ListScenes(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return Scene{}, err
	}
	var current *Scene
	for index := range scenes {
		if scenes[index].ID == sceneID {
			current = &scenes[index]
		}
	}
	if current == nil {
		return Scene{}, ErrNotFound
	}
	if current.Status == "APPROVED" {
		return Scene{}, ErrLocked
	}
	if current.Version != version {
		return Scene{}, ErrConflict
	}
	direction.SceneID = current.SceneKey
	direction.Order = current.Order
	if err = s.validateEditedScenes(ctx, clientID, workspaceID, campaignID, sceneID, direction, scenes); err != nil {
		return Scene{}, err
	}
	hash, _, err := studioai.Hash(direction)
	if err != nil {
		return Scene{}, err
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Scene{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var next int32
	err = tx.QueryRow(ctx, `UPDATE scenes SET current_version=current_version+1,scene_hash=$5,status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$6,updated_at=now() WHERE id=$1 AND campaign_id=$2 AND client_id=$3 AND workspace_id=$4 AND version=$7 RETURNING current_version`, sceneID, campaignID, clientID, workspaceID, hash, actor.UserID, version).Scan(&next)
	if errors.Is(err, pgx.ErrNoRows) {
		return Scene{}, ErrConflict
	}
	if err != nil {
		return Scene{}, err
	}
	if err = insertSceneVersion(ctx, tx, sceneID, next, direction, hash, "Operator edit", actor.UserID, clientID, workspaceID, campaignID); err != nil {
		return Scene{}, err
	}
	if err = invalidateEntity(ctx, tx, "SCENE", sceneID, "Scene direction changed"); err != nil {
		return Scene{}, err
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET status='SCENE_REVIEW',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, campaignID, actor.UserID); err != nil {
		return Scene{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Scene{}, err
	}
	updated, err := s.ListScenes(ctx, clientID, workspaceID, campaignID)
	for _, item := range updated {
		if item.ID == sceneID {
			return item, err
		}
	}
	return Scene{}, ErrNotFound
}

func (s *Service) ApproveScene(ctx context.Context, clientID, workspaceID, campaignID, sceneID uuid.UUID, version int64, notes string, actor auth.Principal) (Scene, error) {
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Scene{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var id uuid.UUID
	var hash string
	var next int64
	err = tx.QueryRow(ctx, `UPDATE scenes SET status='APPROVED',approved_at=now(),approved_by=$5,version=version+1,updated_by=$5,updated_at=now() WHERE id=$1 AND campaign_id=$2 AND client_id=$3 AND workspace_id=$4 AND version=$6 AND status='DRAFT' RETURNING id,scene_hash,version`, sceneID, campaignID, clientID, workspaceID, actor.UserID, version).Scan(&id, &hash, &next)
	if errors.Is(err, pgx.ErrNoRows) {
		return Scene{}, ErrConflict
	}
	if err != nil {
		return Scene{}, err
	}
	if err = recordApproval(ctx, tx, clientID, workspaceID, campaignID, "SCENE", id, next, hash, notes, actor.UserID); err != nil {
		return Scene{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Scene{}, err
	}
	items, err := s.ListScenes(ctx, clientID, workspaceID, campaignID)
	for _, item := range items {
		if item.ID == sceneID {
			return item, err
		}
	}
	return Scene{}, ErrNotFound
}

func (s *Service) ReorderScenes(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID, orderedIDs []uuid.UUID, actorID uuid.UUID) ([]Scene, error) {
	if len(orderedIDs) < 2 {
		return nil, ErrInvalid
	}
	items, err := s.ListScenes(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return nil, err
	}
	byID := make(map[uuid.UUID]Scene, len(items))
	for _, item := range items {
		byID[item.ID] = item
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var total, editable int
	if err = tx.QueryRow(ctx, `SELECT count(*),count(*) FILTER(WHERE id=ANY($4::uuid[]) AND status<>'APPROVED') FROM scenes WHERE campaign_id=$1 AND client_id=$2 AND workspace_id=$3`, campaignID, clientID, workspaceID, orderedIDs).Scan(&total, &editable); err != nil {
		return nil, err
	}
	if total != len(orderedIDs) || editable != len(orderedIDs) {
		return nil, ErrLocked
	}
	if _, err = tx.Exec(ctx, `UPDATE scenes SET scene_order=scene_order+10000 WHERE campaign_id=$1 AND client_id=$2 AND workspace_id=$3`, campaignID, clientID, workspaceID); err != nil {
		return nil, err
	}
	for index, id := range orderedIDs {
		item, exists := byID[id]
		if !exists {
			return nil, ErrInvalid
		}
		direction := item.Direction
		direction.Order = int32(index + 1)
		hash, _, hashErr := studioai.Hash(direction)
		if hashErr != nil {
			return nil, hashErr
		}
		var next int32
		if err = tx.QueryRow(ctx, `UPDATE scenes SET scene_order=$2,current_version=current_version+1,scene_hash=$3,version=version+1,updated_by=$4,updated_at=now() WHERE id=$1 AND campaign_id=$5 AND current_version=$6 AND version=$7 RETURNING current_version`, id, index+1, hash, actorID, campaignID, item.CurrentVersion, item.Version).Scan(&next); err != nil {
			return nil, err
		}
		if err = insertSceneVersion(ctx, tx, id, next, direction, hash, "Scene reordered", actorID, clientID, workspaceID, campaignID); err != nil {
			return nil, err
		}
	}
	if err = tx.Commit(ctx); err != nil {
		return nil, err
	}
	return s.ListScenes(ctx, clientID, workspaceID, campaignID)
}

func (s *Service) DuplicateScene(ctx context.Context, clientID, workspaceID, campaignID, sceneID, actorID uuid.UUID) (Scene, error) {
	items, err := s.ListScenes(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return Scene{}, err
	}
	var source *Scene
	maxOrder := int32(0)
	for index := range items {
		if items[index].ID == sceneID {
			source = &items[index]
		}
		if items[index].Order > maxOrder {
			maxOrder = items[index].Order
		}
	}
	if source == nil {
		return Scene{}, ErrNotFound
	}
	if source.Status == "APPROVED" {
		return Scene{}, ErrLocked
	}
	if len(items) >= 8 {
		return Scene{}, ErrInvalid
	}
	direction := source.Direction
	direction.Order = maxOrder + 1
	direction.SceneID = fmt.Sprintf("scene-%02d", maxOrder+1)
	hash, _, err := studioai.Hash(direction)
	if err != nil {
		return Scene{}, err
	}
	id := uuid.New()
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return Scene{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err = tx.Exec(ctx, `INSERT INTO scenes(id,campaign_id,script_id,client_id,workspace_id,scene_key,scene_order,scene_hash,created_by,updated_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$9)`, id, campaignID, source.ScriptID, clientID, workspaceID, direction.SceneID, direction.Order, hash, actorID); err != nil {
		return Scene{}, err
	}
	if err = insertSceneVersion(ctx, tx, id, 1, direction, hash, "Duplicated scene", actorID, clientID, workspaceID, campaignID); err != nil {
		return Scene{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return Scene{}, err
	}
	updated, err := s.ListScenes(ctx, clientID, workspaceID, campaignID)
	for _, item := range updated {
		if item.ID == id {
			return item, err
		}
	}
	return Scene{}, ErrNotFound
}

func (s *Service) DeleteScene(ctx context.Context, clientID, workspaceID, campaignID, sceneID uuid.UUID, version int64, actorID uuid.UUID) error {
	items, err := s.ListScenes(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return err
	}
	if len(items) <= 2 {
		return ErrInvalid
	}
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	tag, err := tx.Exec(ctx, `DELETE FROM scenes WHERE id=$1 AND campaign_id=$2 AND client_id=$3 AND workspace_id=$4 AND version=$5 AND status<>'APPROVED'`, sceneID, campaignID, clientID, workspaceID, version)
	if err != nil {
		return err
	}
	if tag.RowsAffected() != 1 {
		return ErrConflict
	}
	if _, err = tx.Exec(ctx, `UPDATE scenes SET scene_order=scene_order+10000 WHERE campaign_id=$1`, campaignID); err != nil {
		return err
	}
	order := 0
	for _, item := range items {
		if item.ID == sceneID {
			continue
		}
		order++
		direction := item.Direction
		direction.Order = int32(order)
		hash, _, hashErr := studioai.Hash(direction)
		if hashErr != nil {
			return hashErr
		}
		var next int32
		if err = tx.QueryRow(ctx, `UPDATE scenes SET scene_order=$2,current_version=current_version+1,scene_hash=$3,version=version+1,updated_by=$4,updated_at=now() WHERE id=$1 AND campaign_id=$5 AND current_version=$6 AND version=$7 RETURNING current_version`, item.ID, order, hash, actorID, campaignID, item.CurrentVersion, item.Version).Scan(&next); err != nil {
			return err
		}
		if err = insertSceneVersion(ctx, tx, item.ID, next, direction, hash, "Scene renumbered after deletion", actorID, clientID, workspaceID, campaignID); err != nil {
			return err
		}
	}
	_, err = tx.Exec(ctx, `UPDATE campaigns SET status='SCENE_REVIEW',version=version+1,updated_by=$2,updated_at=now() WHERE id=$1`, campaignID, actorID)
	if err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func recordApproval(ctx context.Context, tx pgx.Tx, clientID, workspaceID, campaignID uuid.UUID, entityType string, entityID uuid.UUID, version int64, hash, notes string, actorID uuid.UUID) error {
	var id uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO approvals(client_id,workspace_id,campaign_id,entity_type,entity_id,entity_version,entity_hash,status,requested_by,decided_by,decided_at,notes)VALUES($1,$2,$3,$4,$5,$6,$7,'APPROVED',$8,$8,now(),$9) RETURNING id`, clientID, workspaceID, campaignID, entityType, entityID, version, hash, actorID, strings.TrimSpace(notes)).Scan(&id)
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO approval_events(approval_id,event_type,actor_id,entity_version,entity_hash,notes)VALUES($1,'APPROVED',$2,$3,$4,$5)`, id, actorID, version, hash, strings.TrimSpace(notes))
	return err
}
func invalidateEntity(ctx context.Context, tx pgx.Tx, entityType string, entityID uuid.UUID, reason string) error {
	_, err := tx.Exec(ctx, `WITH changed AS (UPDATE approvals SET invalidated_at=now(),invalidation_reason=$3 WHERE entity_type=$1 AND entity_id=$2 AND invalidated_at IS NULL RETURNING id,entity_version,entity_hash) INSERT INTO approval_events(approval_id,event_type,entity_version,entity_hash,notes) SELECT id,'INVALIDATED',entity_version,entity_hash,$3 FROM changed`, entityType, entityID, reason)
	return err
}
func invalidateCampaignSceneApprovals(ctx context.Context, tx pgx.Tx, campaignID uuid.UUID, reason string) error {
	_, err := tx.Exec(ctx, `WITH changed AS (UPDATE approvals SET invalidated_at=now(),invalidation_reason=$2 WHERE campaign_id=$1 AND entity_type='SCENE' AND invalidated_at IS NULL RETURNING id,entity_version,entity_hash) INSERT INTO approval_events(approval_id,event_type,entity_version,entity_hash,notes) SELECT id,'INVALIDATED',entity_version,entity_hash,$2 FROM changed`, campaignID, reason)
	return err
}
func invalidateCampaignDownstream(ctx context.Context, tx pgx.Tx, campaignID, actorID uuid.UUID, reason string) error {
	if _, err := tx.Exec(ctx, `WITH changed AS (UPDATE approvals SET invalidated_at=now(),invalidation_reason=$2 WHERE campaign_id=$1 AND entity_type IN('CONTENT_VARIANT','SCRIPT','SCENE') AND invalidated_at IS NULL RETURNING id,entity_version,entity_hash) INSERT INTO approval_events(approval_id,event_type,entity_version,entity_hash,notes) SELECT id,'INVALIDATED',entity_version,entity_hash,$2 FROM changed`, campaignID, reason); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE campaign_content_variants SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status='APPROVED'`, campaignID, actorID); err != nil {
		return err
	}
	if _, err := tx.Exec(ctx, `UPDATE scripts SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status='APPROVED'`, campaignID, actorID); err != nil {
		return err
	}
	_, err := tx.Exec(ctx, `UPDATE scenes SET status='DRAFT',approved_at=NULL,approved_by=NULL,version=version+1,updated_by=$2,updated_at=now() WHERE campaign_id=$1 AND status='APPROVED'`, campaignID, actorID)
	return err
}
func insertScriptVersion(ctx context.Context, tx pgx.Tx, scriptID uuid.UUID, version int32, output studioai.ScriptOutput, hash, summary string, actorID uuid.UUID, promptVersion, model string) error {
	roles, _ := json.Marshal(output.CharacterRoles)
	var versionID uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO script_versions(script_id,version,hook,introduction,problem,product_solution,product_features,benefits,cta,closing,approximate_duration_seconds,character_roles,spoken_language,script_hash,prompt_version,model,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18) RETURNING id`, scriptID, version, output.Hook, output.Introduction, output.Problem, output.ProductSolution, output.ProductFeatures, output.Benefits, output.CTA, output.Closing, output.ApproximateDurationSeconds, roles, output.SpokenLanguage, hash, promptVersion, model, summary, actorID).Scan(&versionID)
	if err != nil {
		return err
	}
	for _, turn := range output.DialogueTurns {
		if _, err = tx.Exec(ctx, `INSERT INTO script_dialogue_turns(script_version_id,turn_order,character_role,dialogue,estimated_duration_ms)VALUES($1,$2,$3,$4,$5)`, versionID, turn.Order, turn.CharacterRole, turn.Dialogue, turn.EstimatedDurationMS); err != nil {
			return err
		}
	}
	return nil
}
func insertSceneVersion(ctx context.Context, tx pgx.Tx, sceneID uuid.UUID, version int32, d studioai.SceneDirection, hash, summary string, actorID, clientID, workspaceID, campaignID uuid.UUID) error {
	var versionID uuid.UUID
	err := tx.QueryRow(ctx, `INSERT INTO scene_versions(scene_id,version,duration_seconds,generation_method,speaker_character_id,listener_character_id,dialogue,speaker_action,listener_action,camera,environment,product_placement,expected_cost_usd,seedance_prompt,scene_hash,change_summary,created_by)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17) RETURNING id`, sceneID, version, d.DurationSeconds, d.GenerationMethod, d.SpeakerCharacterID, d.ListenerCharacterID, d.Dialogue, d.SpeakerAction, d.ListenerAction, d.Camera, d.Environment, d.ProductPlacement, d.ExpectedCostUSD, d.SeedancePrompt, hash, summary, actorID).Scan(&versionID)
	if err != nil {
		return err
	}
	for _, id := range d.ReferenceAssetIDs {
		tag, insertErr := tx.Exec(ctx, `INSERT INTO scene_assets(scene_version_id,media_asset_id,role)
			SELECT $1,id,'PRODUCT_REFERENCE' FROM media_assets
			WHERE id=$2 AND client_id=$3 AND workspace_id=$4 AND deleted_at IS NULL`, versionID, id, clientID, workspaceID)
		if insertErr != nil {
			return insertErr
		}
		if tag.RowsAffected() != 1 {
			return ErrInvalid
		}
	}
	for _, id := range d.RequiredProductFactIDs {
		tag, insertErr := tx.Exec(ctx, `INSERT INTO scene_required_facts(scene_version_id,product_fact_id)
			SELECT $1,pf.id FROM product_facts pf JOIN campaigns c ON c.product_id=pf.product_id
			WHERE pf.id=$2 AND c.id=$3 AND pf.client_id=$4 AND pf.workspace_id=$5 AND pf.status='APPROVED'
			AND (pf.effective_from IS NULL OR pf.effective_from<=now()) AND (pf.expires_at IS NULL OR pf.expires_at>now())`, versionID, id, campaignID, clientID, workspaceID)
		if insertErr != nil {
			return insertErr
		}
		if tag.RowsAffected() != 1 {
			return ErrInvalid
		}
	}
	return nil
}

func (s *Service) validateEditedScenes(ctx context.Context, clientID, workspaceID, campaignID, editedID uuid.UUID, edited studioai.SceneDirection, scenes []Scene) error {
	if edited.DurationSeconds < 3 || edited.DurationSeconds > 15 || edited.ExpectedCostUSD < 0 ||
		!map[string]bool{"seedance": true, "product_footage": true, "still_image": true}[edited.GenerationMethod] ||
		strings.TrimSpace(edited.Camera) == "" || strings.TrimSpace(edited.Environment) == "" || strings.TrimSpace(edited.ProductPlacement) == "" ||
		hasDuplicateUUIDs(edited.ReferenceAssetIDs) || hasDuplicateUUIDs(edited.RequiredProductFactIDs) {
		return ErrInvalid
	}
	planningContext, err := s.planningContext(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		return err
	}
	var primaryID, listenerID uuid.UUID
	if err = s.pool.QueryRow(ctx, `SELECT
		(SELECT character_id FROM campaign_characters WHERE campaign_id=$1 AND role='PRIMARY'),
		(SELECT character_id FROM campaign_characters WHERE campaign_id=$1 AND role='LISTENER')`, campaignID).Scan(&primaryID, &listenerID); err != nil {
		return ErrPrerequisite
	}
	output := studioai.SceneOutput{Scenes: make([]studioai.SceneDirection, 0, len(scenes))}
	for index, scene := range scenes {
		direction := scene.Direction
		if scene.ID == editedID {
			direction = edited
		}
		// Stable scene keys survive drag-and-drop; normalize only the validation copy.
		direction.SceneID = fmt.Sprintf("scene-%02d", index+1)
		direction.Order = int32(index + 1)
		output.Scenes = append(output.Scenes, direction)
	}
	input := studioai.SceneInput{Context: planningContext, SpeakerCharacterID: primaryID, ListenerCharacterID: listenerID}
	if err = studioai.ValidateScenes(output, input); err != nil {
		return ErrInvalid
	}
	return nil
}

func hasDuplicateUUIDs(values []uuid.UUID) bool {
	seen := make(map[uuid.UUID]struct{}, len(values))
	for _, id := range values {
		if id == uuid.Nil {
			return true
		}
		if _, exists := seen[id]; exists {
			return true
		}
		seen[id] = struct{}{}
	}
	return false
}
func scanConcept(row scanner) (Concept, error) {
	var item Concept
	var raw []byte
	err := row.Scan(&item.ID, &item.CampaignID, &item.Status, &raw, &item.CurrentVersion, &item.PromptVersion, &item.Model, &item.RequestID, &item.OutputHash, &item.EstimatedCostUSD, &item.Version, &item.LockedAt, &item.CreatedAt, &item.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return Concept{}, ErrNotFound
	}
	if err == nil {
		err = json.Unmarshal(raw, &item.Payload)
	}
	return item, err
}
func validateConcept(c studioai.ConceptCandidate) error {
	if strings.TrimSpace(c.Title) == "" || strings.TrimSpace(c.Hook) == "" || len(c.CharacterRoles) != 2 || c.CharacterRoles[0] == c.CharacterRoles[1] || !map[string]bool{"INTERVIEW_REVIEW": true, "PROBLEM_SOLUTION": true}[c.VideoFormat] || c.ExpectedSceneCount < 2 || c.ExpectedSceneCount > 8 || c.EstimatedCostUSD < 0 {
		return ErrInvalid
	}
	return nil
}
func valid(id uuid.UUID) uuid.NullUUID { return uuid.NullUUID{UUID: id, Valid: true} }
