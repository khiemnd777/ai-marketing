package jobs

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/riverqueue/river"
	"github.com/riverqueue/river/riverdriver/riverpgxv5"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/telemetry"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/providerconfigs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/storage"
)

type MediaMetadataArgs struct {
	AssetID     uuid.UUID `json:"assetId" river:"unique"`
	ClientID    uuid.UUID `json:"clientId"`
	WorkspaceID uuid.UUID `json:"workspaceId"`
}

func (MediaMetadataArgs) Kind() string { return "media.metadata.extract" }

type Enqueuer struct{ client *river.Client[pgx.Tx] }

func NewEnqueuer(pool *pgxpool.Pool) (*Enqueuer, error) {
	client, err := river.NewClient(riverpgxv5.New(pool), &river.Config{Plugins: telemetry.RiverPlugins()})
	if err != nil {
		return nil, fmt.Errorf("create River enqueuer: %w", err)
	}
	return &Enqueuer{client: client}, nil
}

func (e *Enqueuer) EnqueueMediaMetadata(ctx context.Context, tx pgx.Tx, assetID, clientID, workspaceID uuid.UUID) error {
	_, err := e.client.InsertTx(ctx, tx, MediaMetadataArgs{AssetID: assetID, ClientID: clientID, WorkspaceID: workspaceID}, &river.InsertOpts{
		Queue: QueueMediaProcessing, MaxAttempts: 5,
		UniqueOpts: river.UniqueOpts{ByArgs: true, ByQueue: true},
	})
	return err
}

type MediaMetadataWorker struct {
	river.WorkerDefaults[MediaMetadataArgs]
	Pool     *pgxpool.Pool
	Store    storage.ObjectStore
	TempDir  string
	Resolver interface {
		Load(context.Context, uuid.UUID) (providerconfigs.Bundle, error)
	}
}

type mediaProbe struct {
	Width      *int32 `json:"width,omitempty"`
	Height     *int32 `json:"height,omitempty"`
	DurationMS *int64 `json:"durationMs,omitempty"`
	Codec      string `json:"codec,omitempty"`
	BitrateBPS *int64 `json:"bitrateBps,omitempty"`
}

func (w *MediaMetadataWorker) Work(ctx context.Context, job *river.Job[MediaMetadataArgs]) error {
	store := w.Store
	if w.Resolver != nil {
		bundle, resolveErr := w.Resolver.Load(ctx, job.Args.ClientID)
		if resolveErr != nil {
			return river.JobCancel(errors.New("client provider configuration is unavailable"))
		}
		resolvedStore, storeErr := storage.NewS3Store(ctx, bundle.R2)
		if storeErr != nil {
			return river.JobCancel(storeErr)
		}
		store = resolvedStore
	}
	if store == nil {
		return errors.New("media metadata worker requires object storage")
	}
	var storageKey, mimeType string
	err := w.Pool.QueryRow(ctx, `SELECT v.storage_key,v.mime_type FROM media_assets a JOIN media_asset_versions v ON v.media_asset_id=a.id AND v.version=a.current_version WHERE a.id=$1 AND a.client_id=$2 AND a.workspace_id=$3 AND a.deleted_at IS NULL`, job.Args.AssetID, job.Args.ClientID, job.Args.WorkspaceID).Scan(&storageKey, &mimeType)
	if errors.Is(err, pgx.ErrNoRows) {
		return river.JobCancel(fmt.Errorf("media asset no longer exists"))
	}
	if err != nil {
		return fmt.Errorf("load media asset: %w", err)
	}
	if err := os.MkdirAll(w.TempDir, 0o750); err != nil {
		return fmt.Errorf("create media temp root: %w", err)
	}
	dir, err := os.MkdirTemp(w.TempDir, "metadata-")
	if err != nil {
		return fmt.Errorf("create media temp directory: %w", err)
	}
	defer os.RemoveAll(dir)
	input := filepath.Join(dir, "original"+extensionForMIME(mimeType))
	if err := w.download(ctx, store, storageKey, input); err != nil {
		return err
	}

	probe := mediaProbe{}
	thumbnailKey := ""
	if strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") || strings.HasPrefix(mimeType, "audio/") {
		probe, err = probeFile(ctx, input)
		if err != nil {
			return fmt.Errorf("ffprobe media: %w", err)
		}
	}
	if strings.HasPrefix(mimeType, "image/") || strings.HasPrefix(mimeType, "video/") {
		thumbnail := filepath.Join(dir, "thumbnail.jpg")
		if err := renderThumbnail(ctx, input, thumbnail); err != nil {
			return fmt.Errorf("render media thumbnail: %w", err)
		}
		file, err := os.Open(thumbnail)
		if err != nil {
			return err
		}
		stat, err := file.Stat()
		if err != nil {
			_ = file.Close()
			return err
		}
		thumbnailKey = storage.ThumbnailObjectKey(job.Args.WorkspaceID, job.Args.AssetID)
		err = store.Put(ctx, thumbnailKey, "image/jpeg", stat.Size(), file, map[string]string{"source-asset-id": job.Args.AssetID.String()})
		closeErr := file.Close()
		if err != nil {
			return err
		}
		if closeErr != nil {
			return closeErr
		}
	}
	probeJSON, _ := json.Marshal(probe)
	tx, err := w.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	result, err := tx.Exec(ctx, `UPDATE media_asset_versions SET width=$4,height=$5,duration_ms=$6,codec=NULLIF($7,''),bitrate_bps=$8,thumbnail_storage_key=NULLIF($9,''),metadata=metadata||jsonb_build_object('probe',$10::jsonb,'processedAt',now()) WHERE media_asset_id=$1 AND client_id=$2 AND workspace_id=$3`, job.Args.AssetID, job.Args.ClientID, job.Args.WorkspaceID, probe.Width, probe.Height, probe.DurationMS, probe.Codec, probe.BitrateBPS, thumbnailKey, probeJSON)
	if err != nil {
		return err
	}
	if result.RowsAffected() != 1 {
		return river.JobCancel(fmt.Errorf("media asset version no longer exists"))
	}
	requestID := fmt.Sprintf("river:%d", job.ID)
	if err := audit.Record(ctx, db.New(tx), audit.Event{Action: "media.metadata.processed", EntityType: "media_asset", EntityID: uuid.NullUUID{UUID: job.Args.AssetID, Valid: true}, ClientID: uuid.NullUUID{UUID: job.Args.ClientID, Valid: true}, WorkspaceID: uuid.NullUUID{UUID: job.Args.WorkspaceID, Valid: true}, RequestID: requestID, Outcome: "SUCCESS", After: probe, Metadata: map[string]any{"thumbnailStorageKey": thumbnailKey}}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (w *MediaMetadataWorker) download(ctx context.Context, store storage.ObjectStore, key, destination string) error {
	body, err := store.Get(ctx, key)
	if err != nil {
		return err
	}
	defer body.Close()
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	_, copyErr := io.Copy(file, body)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("download media object: %w", copyErr)
	}
	return closeErr
}

func probeFile(ctx context.Context, filename string) (mediaProbe, error) {
	command := exec.CommandContext(ctx, "ffprobe", "-v", "error", "-show_entries", "format=duration,bit_rate:stream=codec_type,codec_name,width,height,duration,bit_rate", "-of", "json", filename)
	output, err := command.Output()
	if err != nil {
		return mediaProbe{}, err
	}
	return parseProbe(output)
}

func parseProbe(value []byte) (mediaProbe, error) {
	var document struct {
		Streams []struct {
			CodecType string `json:"codec_type"`
			CodecName string `json:"codec_name"`
			Width     int32  `json:"width"`
			Height    int32  `json:"height"`
			Duration  string `json:"duration"`
			Bitrate   string `json:"bit_rate"`
		} `json:"streams"`
		Format struct {
			Duration string `json:"duration"`
			Bitrate  string `json:"bit_rate"`
		} `json:"format"`
	}
	if err := json.Unmarshal(value, &document); err != nil {
		return mediaProbe{}, fmt.Errorf("decode ffprobe output: %w", err)
	}
	result := mediaProbe{}
	for _, stream := range document.Streams {
		if result.Codec == "" || stream.CodecType == "video" {
			result.Codec = stream.CodecName
		}
		if stream.Width > 0 && stream.Height > 0 {
			width, height := stream.Width, stream.Height
			result.Width, result.Height = &width, &height
		}
		if document.Format.Duration == "" && stream.Duration != "" {
			document.Format.Duration = stream.Duration
		}
		if document.Format.Bitrate == "" && stream.Bitrate != "" {
			document.Format.Bitrate = stream.Bitrate
		}
		if stream.CodecType == "video" {
			break
		}
	}
	if seconds, err := strconv.ParseFloat(document.Format.Duration, 64); err == nil && seconds > 0 {
		duration := int64(seconds * float64(time.Second/time.Millisecond))
		result.DurationMS = &duration
	}
	if value, err := strconv.ParseInt(document.Format.Bitrate, 10, 64); err == nil && value > 0 {
		result.BitrateBPS = &value
	}
	return result, nil
}

func renderThumbnail(ctx context.Context, input, output string) error {
	return exec.CommandContext(ctx, "ffmpeg", "-hide_banner", "-loglevel", "error", "-y", "-i", input, "-ss", "0.1", "-frames:v", "1", "-vf", "scale=480:-2", output).Run()
}

func extensionForMIME(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/webp":
		return ".webp"
	case "video/mp4":
		return ".mp4"
	case "video/quicktime":
		return ".mov"
	case "audio/mpeg":
		return ".mp3"
	case "audio/wav", "audio/x-wav":
		return ".wav"
	default:
		return ".bin"
	}
}
