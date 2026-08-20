package jobs

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
)

type MaintenanceArgs struct{}

func (MaintenanceArgs) Kind() string { return "maintenance.cleanup" }

type MaintenanceWorker struct {
	river.WorkerDefaults[MaintenanceArgs]
	Queries *db.Queries
	Pool    *pgxpool.Pool
}

func (worker *MaintenanceWorker) Work(ctx context.Context, _ *river.Job[MaintenanceArgs]) error {
	if _, err := worker.Queries.DeleteExpiredSessions(ctx); err != nil {
		return fmt.Errorf("delete expired sessions: %w", err)
	}
	statements := []struct {
		name string
		sql  string
	}{
		{"expired idempotency keys", `DELETE FROM idempotency_keys WHERE expires_at<now()`},
		{"expired Meta OAuth states", `DELETE FROM meta_oauth_states WHERE expires_at<now() OR consumed_at<now()-interval '1 day'`},
		{"expired webhook payloads", `UPDATE webhook_events SET sanitized_payload=NULL WHERE sanitized_payload IS NOT NULL AND received_at<now()-interval '90 days'`},
		{"expired Meta webhook payloads", `UPDATE meta_webhook_events SET normalized_events='[]'::jsonb WHERE normalized_events<>'[]'::jsonb AND received_at<now()-interval '90 days'`},
		{"old completed River jobs", `DELETE FROM river_job WHERE state IN ('completed','cancelled') AND finalized_at<now()-interval '30 days'`},
		{"old discarded River jobs", `DELETE FROM river_job WHERE state='discarded' AND finalized_at<now()-interval '90 days'`},
		{"Meta expiry notifications", `INSERT INTO notifications(client_id,workspace_id,severity,notification_type,title,message,entity_type,entity_id)
			SELECT c.client_id,c.workspace_id,'WARNING','META_TOKEN_EXPIRING','Meta connection requires attention','Meta access or data-access token expires within 7 days. Reconnect before publishing or Ads operations.','meta_connection',c.id
			FROM meta_connections c WHERE c.disconnected_at IS NULL AND (c.token_expires_at<now()+interval '7 days' OR c.data_access_expires_at<now()+interval '7 days')
			AND NOT EXISTS(SELECT 1 FROM notifications n WHERE n.notification_type='META_TOKEN_EXPIRING' AND n.entity_id=c.id AND n.created_at>=now()-interval '1 day')`},
		{"cost anomaly notifications", `WITH daily AS (SELECT client_id,workspace_id,date_trunc('day',occurred_at) d,sum(normalized_amount_usd) cost FROM cost_records WHERE occurred_at>=now()-interval '8 days' GROUP BY 1,2,3), anomaly AS (SELECT t.client_id,t.workspace_id,t.cost FROM daily t WHERE t.d=date_trunc('day',now()) AND t.cost>1 AND t.cost>2*COALESCE((SELECT avg(h.cost) FROM daily h WHERE h.client_id=t.client_id AND h.workspace_id=t.workspace_id AND h.d<t.d),0)) INSERT INTO notifications(client_id,workspace_id,severity,notification_type,title,message) SELECT a.client_id,a.workspace_id,'WARNING','DAILY_COST_ANOMALY','Provider cost anomaly','Today provider cost is more than twice the recent daily average. Review usage before continuing paid work.' FROM anomaly a WHERE NOT EXISTS(SELECT 1 FROM notifications n WHERE n.notification_type='DAILY_COST_ANOMALY' AND n.client_id=a.client_id AND n.workspace_id=a.workspace_id AND n.created_at>=date_trunc('day',now()))`},
	}
	for _, statement := range statements {
		if _, err := worker.Pool.Exec(ctx, statement.sql); err != nil {
			return fmt.Errorf("maintain %s: %w", statement.name, err)
		}
	}
	if _, err := worker.Pool.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY analytics_workspace_daily`); err != nil {
		return fmt.Errorf("refresh analytics workspace daily: %w", err)
	}
	return nil
}
