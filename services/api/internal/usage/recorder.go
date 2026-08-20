package usage

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type queryer interface {
	QueryRow(context.Context, string, ...any) pgx.Row
	Exec(context.Context, string, ...any) (pgconn.CommandTag, error)
}

type Entry struct {
	Provider, Model, RequestReference, Operation string
	ClientID, WorkspaceID, CampaignID, SceneID   *uuid.UUID
	VideoProjectID                               *uuid.UUID
	ExchangeRateSnapshotID                       *uuid.UUID
	InputUnits, OutputUnits                      int64
	GeneratedSeconds, AcceptedSeconds            float64
	ProviderReportedCost                         *float64
	EstimatedCost                                float64
	Currency, Outcome, Category                  string
	Reused                                       bool
	Metadata                                     map[string]any
}

func Record(ctx context.Context, q queryer, entry Entry) error {
	if entry.Provider == "" || entry.RequestReference == "" || entry.Operation == "" || (entry.Outcome != "SUCCESS" && entry.Outcome != "FAILURE") {
		return errors.New("invalid usage ledger entry")
	}
	if entry.Currency == "" {
		entry.Currency = "USD"
	}
	if entry.Metadata == nil {
		entry.Metadata = map[string]any{}
	}
	amount, estimated := entry.EstimatedCost, true
	if entry.ProviderReportedCost != nil {
		amount, estimated = *entry.ProviderReportedCost, false
	}
	if entry.Category == "" {
		entry.Category = "OTHER"
	}
	normalized := amount
	if entry.Currency != "USD" {
		if entry.ExchangeRateSnapshotID == nil {
			return errors.New("non-USD usage requires an exchange-rate snapshot")
		}
		var rate float64
		if err := q.QueryRow(ctx, `SELECT rate::float8 FROM exchange_rate_snapshots WHERE id=$1 AND base_currency=$2 AND quote_currency='USD'`, *entry.ExchangeRateSnapshotID, entry.Currency).Scan(&rate); err != nil {
			return err
		}
		normalized = amount * rate
	}
	var id uuid.UUID
	err := q.QueryRow(ctx, `INSERT INTO usage_ledger(provider,model,request_reference,operation,client_id,workspace_id,campaign_id,scene_id,video_project_id,input_units,output_units,generated_seconds,accepted_seconds,provider_reported_cost,estimated_cost,currency,exchange_rate_snapshot_id,outcome,reused,metadata)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18::usage_outcome,$19,$20)ON CONFLICT(provider,request_reference,operation)DO NOTHING RETURNING id`, entry.Provider, entry.Model, entry.RequestReference, entry.Operation, entry.ClientID, entry.WorkspaceID, entry.CampaignID, entry.SceneID, entry.VideoProjectID, entry.InputUnits, entry.OutputUnits, entry.GeneratedSeconds, entry.AcceptedSeconds, entry.ProviderReportedCost, entry.EstimatedCost, entry.Currency, entry.ExchangeRateSnapshotID, entry.Outcome, entry.Reused, entry.Metadata).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return err
	}
	_, err = q.Exec(ctx, `INSERT INTO cost_records(usage_ledger_id,client_id,workspace_id,campaign_id,category,provider,amount,currency,normalized_amount_usd,estimated,metadata)VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11)`, id, entry.ClientID, entry.WorkspaceID, entry.CampaignID, entry.Category, entry.Provider, amount, entry.Currency, normalized, estimated, entry.Metadata)
	return err
}
