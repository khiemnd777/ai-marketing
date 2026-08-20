package planning

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	studioai "github.com/internal/ai-product-marketing-studio/services/api/internal/ai"
)

func (s *Service) planningContext(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) (studioai.PlanningContext, error) {
	var value studioai.PlanningContext
	value.CampaignID = campaignID
	err := s.pool.QueryRow(ctx, `SELECT c.name,v.objective::text,v.target_audience,v.market,v.language,v.video_format,v.duration_seconds,v.tone,v.offer,v.cta,b.name,p.name FROM campaigns c JOIN campaign_versions v ON v.campaign_id=c.id AND v.version=c.current_version JOIN brands b ON b.id=c.brand_id JOIN products p ON p.id=c.product_id WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 AND c.status<>'ARCHIVED'`, campaignID, clientID, workspaceID).Scan(&value.CampaignName, &value.Objective, &value.Audience, &value.Market, &value.Language, &value.VideoFormat, &value.DurationSeconds, &value.Tone, &value.Offer, &value.CTA, &value.BrandName, &value.ProductName)
	if errors.Is(err, pgx.ErrNoRows) {
		return studioai.PlanningContext{}, ErrNotFound
	}
	if err != nil {
		return studioai.PlanningContext{}, err
	}
	facts, err := s.pool.Query(ctx, `SELECT f.id,f.fact_key,f.exact_value,f.locked_value FROM campaigns c JOIN product_facts f ON f.product_id=c.product_id WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 AND f.status='APPROVED' AND (f.effective_from IS NULL OR f.effective_from<=now()) AND (f.expires_at IS NULL OR f.expires_at>now()) ORDER BY f.fact_key`, campaignID, clientID, workspaceID)
	if err != nil {
		return studioai.PlanningContext{}, err
	}
	defer facts.Close()
	value.ProductTruth = []studioai.ProductFact{}
	for facts.Next() {
		var fact studioai.ProductFact
		if err = facts.Scan(&fact.ID, &fact.Key, &fact.ExactValue, &fact.Locked); err != nil {
			return studioai.PlanningContext{}, err
		}
		value.ProductTruth = append(value.ProductTruth, fact)
	}
	if err = facts.Err(); err != nil {
		return studioai.PlanningContext{}, err
	}
	claims, err := s.pool.Query(ctx, `SELECT pc.claim_text FROM campaigns c JOIN product_claims pc ON pc.product_id=c.product_id WHERE c.id=$1 AND c.client_id=$2 AND c.workspace_id=$3 AND pc.claim_kind='PROHIBITED' ORDER BY pc.created_at`, campaignID, clientID, workspaceID)
	if err != nil {
		return studioai.PlanningContext{}, err
	}
	defer claims.Close()
	value.ProhibitedClaims = []string{}
	for claims.Next() {
		var claim string
		if err = claims.Scan(&claim); err != nil {
			return studioai.PlanningContext{}, err
		}
		value.ProhibitedClaims = append(value.ProhibitedClaims, claim)
	}
	return value, claims.Err()
}

func (s *Service) currentScriptOutput(ctx context.Context, clientID, workspaceID, campaignID uuid.UUID) (uuid.UUID, int64, string, studioai.ScriptOutput, error) {
	var scriptID uuid.UUID
	var version int64
	var hash string
	var output studioai.ScriptOutput
	var rolesRaw []byte
	err := s.pool.QueryRow(ctx, `SELECT s.id,s.version,s.script_hash,sv.hook,sv.introduction,sv.problem,sv.product_solution,sv.product_features,sv.benefits,sv.cta,sv.closing,sv.approximate_duration_seconds,sv.character_roles,sv.spoken_language FROM scripts s JOIN script_versions sv ON sv.script_id=s.id AND sv.version=s.current_version WHERE s.campaign_id=$1 AND s.client_id=$2 AND s.workspace_id=$3`, campaignID, clientID, workspaceID).Scan(&scriptID, &version, &hash, &output.Hook, &output.Introduction, &output.Problem, &output.ProductSolution, &output.ProductFeatures, &output.Benefits, &output.CTA, &output.Closing, &output.ApproximateDurationSeconds, &rolesRaw, &output.SpokenLanguage)
	if errors.Is(err, pgx.ErrNoRows) {
		return uuid.Nil, 0, "", studioai.ScriptOutput{}, ErrNotFound
	}
	if err != nil {
		return uuid.Nil, 0, "", studioai.ScriptOutput{}, err
	}
	if err = json.Unmarshal(rolesRaw, &output.CharacterRoles); err != nil {
		return uuid.Nil, 0, "", studioai.ScriptOutput{}, err
	}
	rows, err := s.pool.Query(ctx, `SELECT turn_order,character_role,dialogue,estimated_duration_ms FROM script_dialogue_turns WHERE script_version_id=(SELECT id FROM script_versions WHERE script_id=$1 AND version=(SELECT current_version FROM scripts WHERE id=$1)) ORDER BY turn_order`, scriptID)
	if err != nil {
		return uuid.Nil, 0, "", studioai.ScriptOutput{}, err
	}
	defer rows.Close()
	output.DialogueTurns = []studioai.DialogueTurn{}
	for rows.Next() {
		var turn studioai.DialogueTurn
		if err = rows.Scan(&turn.Order, &turn.CharacterRole, &turn.Dialogue, &turn.EstimatedDurationMS); err != nil {
			return uuid.Nil, 0, "", studioai.ScriptOutput{}, err
		}
		output.DialogueTurns = append(output.DialogueTurns, turn)
	}
	return scriptID, version, hash, output, rows.Err()
}
