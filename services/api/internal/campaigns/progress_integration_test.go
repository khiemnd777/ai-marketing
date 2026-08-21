package campaigns

import (
	"context"
	"errors"
	"os"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

func TestGetProgressIntegration(t *testing.T) {
	databaseURL := os.Getenv("STUDIO_PROGRESS_DATABASE_URL")
	if databaseURL == "" {
		t.Skip("STUDIO_PROGRESS_DATABASE_URL is not configured")
	}
	ctx := context.Background()
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	var campaignID, clientID, workspaceID, actorID uuid.UUID
	err = pool.QueryRow(ctx, `SELECT id,client_id,workspace_id,created_by FROM campaigns ORDER BY created_at LIMIT 1`).Scan(&campaignID, &clientID, &workspaceID, &actorID)
	if errors.Is(err, pgx.ErrNoRows) {
		t.Skip("integration database has no campaign")
	}
	if err != nil {
		t.Fatal(err)
	}

	progress, err := NewService(pool).GetProgress(ctx, clientID, workspaceID, campaignID)
	if err != nil {
		t.Fatal(err)
	}
	if progress.CampaignID != campaignID || len(progress.Steps) != 9 {
		t.Fatalf("unexpected progress response: %#v", progress)
	}
	if _, err = NewService(pool).GetProgress(ctx, clientID, uuid.New(), campaignID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("cross-workspace progress lookup must fail closed, got %v", err)
	}

	tx, err := pool.Begin(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	conceptID := uuid.New()
	const conceptHash = "integration-progress-concept-hash"
	_, err = tx.Exec(ctx, `INSERT INTO campaign_concepts(id,campaign_id,client_id,workspace_id,title,video_format,status,payload,prompt_version,model,request_id,output_hash,locked_at,locked_by,created_by,updated_by) VALUES($1,$2,$3,$4,'Progress test','INTERVIEW_REVIEW','LOCKED','{}','test','test','test',$5,now(),$6,$6,$6)`, conceptID, campaignID, clientID, workspaceID, conceptHash, actorID)
	if err != nil {
		t.Fatal(err)
	}
	if _, err = tx.Exec(ctx, `UPDATE campaigns SET selected_concept_id=$2 WHERE id=$1`, campaignID, conceptID); err != nil {
		t.Fatal(err)
	}
	var approvalID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO approvals(client_id,workspace_id,campaign_id,entity_type,entity_id,entity_version,entity_hash,status,requested_by,decided_by,decided_at) VALUES($1,$2,$3,'CONCEPT',$4,1,$5,'APPROVED',$6,$6,now()) RETURNING id`, clientID, workspaceID, campaignID, conceptID, conceptHash, actorID).Scan(&approvalID)
	if err != nil {
		t.Fatal(err)
	}
	progress, err = getProgress(ctx, tx, clientID, workspaceID, campaignID)
	if err != nil || !progress.Steps[1].Completed {
		t.Fatalf("locked concept with a current approval must be complete: progress=%#v err=%v", progress, err)
	}
	if _, err = tx.Exec(ctx, `UPDATE approvals SET invalidated_at=now(),invalidation_reason='integration test' WHERE id=$1`, approvalID); err != nil {
		t.Fatal(err)
	}
	progress, err = getProgress(ctx, tx, clientID, workspaceID, campaignID)
	if err != nil || progress.Steps[1].Completed {
		t.Fatalf("invalidated approval must clear completion: progress=%#v err=%v", progress, err)
	}
}
