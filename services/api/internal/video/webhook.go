package video

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (h *Handler) Webhook(c fiber.Ctx) error {
	clientID, parseErr := uuid.Parse(c.Query("clientId"))
	if parseErr != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	cfg, configErr := h.service.effectiveConfig(c.Context(), clientID)
	if configErr != nil {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	secret := cfg.Seedance.WebhookSecret
	supplied := c.Query("token")
	if secret == "" || len(secret) != len(supplied) || subtle.ConstantTimeCompare([]byte(secret), []byte(supplied)) != 1 {
		return c.SendStatus(fiber.StatusUnauthorized)
	}
	payload := c.Body()
	if len(payload) == 0 || len(payload) > maxProviderResponseBytes {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	requestID, _ := c.Locals("request_id").(string)
	task, err := ParseBytePlusTask(payload, requestID)
	if err != nil {
		return c.SendStatus(fiber.StatusBadRequest)
	}
	if err = h.service.ProcessWebhook(c.Context(), clientID, task, payload, requestID); err != nil {
		if errors.Is(err, ErrNotFound) {
			// A valid provider callback may race the transaction that persists its
			// task ID. Polling remains the authoritative fallback.
			return c.SendStatus(fiber.StatusAccepted)
		}
		return c.SendStatus(fiber.StatusInternalServerError)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (s *Service) ProcessWebhook(ctx context.Context, clientID uuid.UUID, task Task, raw []byte, requestID string) error {
	payloadHash := sha256.Sum256(raw)
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	var deliveryID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO provider_webhook_deliveries(provider,provider_task_id,payload_hash,request_id,signature_valid)VALUES('byteplus-modelark',$1,$2,$3,true)ON CONFLICT(provider,provider_task_id,payload_hash)DO NOTHING RETURNING id`, task.ID, hex.EncodeToString(payloadHash[:]), requestID).Scan(&deliveryID)
	if errors.Is(err, pgx.ErrNoRows) {
		return tx.Commit(ctx)
	}
	if err != nil {
		return err
	}
	var generationID uuid.UUID
	var current string
	err = tx.QueryRow(ctx, `SELECT id,status::text FROM scene_generation_tasks WHERE client_id=$1 AND provider='byteplus-modelark' AND provider_task_id=$2 FOR UPDATE`, clientID, task.ID).Scan(&generationID, &current)
	if errors.Is(err, pgx.ErrNoRows) {
		_, _ = tx.Exec(ctx, `UPDATE provider_webhook_deliveries SET status_code=202,processing_error='Generation task ID is not visible yet',processed_at=now() WHERE id=$1`, deliveryID)
		if commitErr := tx.Commit(ctx); commitErr != nil {
			return commitErr
		}
		return ErrNotFound
	}
	if err != nil {
		return err
	}
	status := normalizeProviderStatus(task.Status)
	if status == "" {
		return ErrInvalid
	}
	if !mayAdvance(current, status) {
		_, err = tx.Exec(ctx, `UPDATE provider_webhook_deliveries SET status_code=204,processed_at=now() WHERE id=$1`, deliveryID)
		if err != nil {
			return err
		}
		return tx.Commit(ctx)
	}
	category := ""
	if status == "FAILED" {
		category = string(CategoryOutage)
		lower := strings.ToLower(task.ErrorCode)
		if strings.Contains(lower, "sensitive") || strings.Contains(lower, "moderation") || strings.Contains(lower, "safety") {
			category = string(CategoryModeration)
		}
	}
	_, err = tx.Exec(ctx, `UPDATE scene_generation_tasks SET status=$2::scene_generation_status,sanitized_response=$3,provider_output_url=NULLIF($4,''),usage_tokens=NULLIF($5,0),provider_seed=$6,provider_fps=$7,error_category=NULLIF($8,''),error_code=NULLIF($9,''),error_message=NULLIF($10,''),provider_started_at=CASE WHEN $2::text='PROVIDER_PROCESSING' THEN COALESCE(provider_started_at,now()) ELSE provider_started_at END,provider_completed_at=CASE WHEN $2::text IN ('SUCCEEDED','FAILED') THEN now() ELSE provider_completed_at END,next_poll_at=CASE WHEN $2::text IN ('PROVIDER_QUEUED','PROVIDER_PROCESSING') THEN next_poll_at ELSE NULL END,version=version+1,updated_at=now() WHERE id=$1`, generationID, status, task.SafeResponse, task.OutputURL, task.UsageTokens, task.Seed, task.FPS, category, task.ErrorCode, task.ErrorMessage)
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `INSERT INTO scene_generation_events(generation_task_id,from_status,to_status,source,provider_request_id,safe_detail)VALUES($1,$2::scene_generation_status,$3::scene_generation_status,'PROVIDER_WEBHOOK',$4,'Provider callback synchronized')`, generationID, current, status, task.ProviderRequestID); err != nil {
		return err
	}
	if status == "SUCCEEDED" {
		if _, err = s.enqueuer.EnqueueSeedanceDownload(ctx, tx, generationID); err != nil {
			return err
		}
	}
	if _, err = tx.Exec(ctx, `UPDATE provider_webhook_deliveries SET status_code=204,processed_at=now() WHERE id=$1`, deliveryID); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func mayAdvance(current, next string) bool {
	if current == "FAILED" || current == "CANCELLED" || current == "REJECTED" || current == "APPROVED" || current == "DOWNLOADING" || current == "VALIDATING" || current == "REVIEW_REQUIRED" {
		return false
	}
	rank := map[string]int{"SUBMITTING": 0, "PROVIDER_QUEUED": 1, "PROVIDER_PROCESSING": 2, "SUCCEEDED": 3, "FAILED": 3}
	return rank[next] >= rank[current]
}
