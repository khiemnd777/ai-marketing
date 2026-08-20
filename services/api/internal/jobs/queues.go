package jobs

const (
	QueueAIContent        = "ai-content"
	QueueMediaProcessing  = "media-processing"
	QueueSeedanceSubmit   = "seedance-submit"
	QueueSeedanceStatus   = "seedance-status"
	QueueSeedanceDownload = "seedance-download"
	QueueTranscription    = "transcription"
	QueueQualityCheck     = "quality-check"
	QueueRender           = "render"
	QueueSocialPublish    = "social-publish"
	QueueMetaAds          = "meta-ads"
	QueueMetricsSync      = "metrics-sync"
	QueueMaintenance      = "maintenance"
)

var RequiredQueues = []string{
	QueueAIContent,
	QueueMediaProcessing,
	QueueSeedanceSubmit,
	QueueSeedanceStatus,
	QueueSeedanceDownload,
	QueueTranscription,
	QueueQualityCheck,
	QueueRender,
	QueueSocialPublish,
	QueueMetaAds,
	QueueMetricsSync,
	QueueMaintenance,
}
