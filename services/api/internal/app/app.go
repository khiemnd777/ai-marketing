package app

import (
	"errors"
	"log/slog"
	"net/netip"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/gofiber/fiber/v3/middleware/adaptor"
	"github.com/gofiber/fiber/v3/middleware/recover"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/analytics"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/audit"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/brands"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/campaigns"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/characters"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/clients"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/internalusers"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/jobs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/media"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/metaads"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/metaconnections"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/operations"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/planning"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/cryptox"
	studiometrics "github.com/internal/ai-product-marketing-studio/services/api/internal/platform/metrics"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/requestid"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/security"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/products"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/producttruth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/providerconfigs"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/publishing"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/rendering"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/verticals"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/video"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/workspaces"
)

func New(cfg config.Config, logger *slog.Logger, pool *pgxpool.Pool) (*fiber.App, error) {
	secretCipher, err := cryptox.New(cfg.EncryptionKey)
	if err != nil {
		return nil, err
	}
	providerConfigService := providerconfigs.NewService(pool, secretCipher)
	providerConfigHandler := providerconfigs.NewHandler(providerConfigService)
	authService, err := auth.NewService(pool, cfg.SessionSecret, cfg.SessionTTL)
	if err != nil {
		return nil, err
	}
	authHandler := auth.NewHandler(authService, cfg.SecureCookies(), cfg.SessionTTL)
	userHandler := internalusers.NewHandler(internalusers.NewService(pool))
	clientHandler := clients.NewHandler(clients.NewService(pool))
	workspaceHandler := workspaces.NewHandler(workspaces.NewService(pool))
	verticalRegistry, err := verticals.Load(cfg.VerticalsDir)
	if err != nil {
		return nil, err
	}
	brandHandler := brands.NewHandler(brands.NewService(pool))
	campaignHandler := campaigns.NewHandler(campaigns.NewService(pool))
	characterHandler := characters.NewHandler(characters.NewService(pool))
	productHandler := products.NewHandler(products.NewService(pool, verticalRegistry))
	truthHandler := producttruth.NewHandler(producttruth.NewService(pool))
	jobEnqueuer, err := jobs.NewEnqueuer(pool)
	if err != nil {
		return nil, err
	}
	mediaHandler := media.NewHandler(media.NewTenantService(pool, providerConfigService, jobEnqueuer))
	planningHandler := planning.NewHandler(planning.NewTenantService(pool, jobEnqueuer, providerConfigService))
	videoHandler := video.NewHandler(video.NewTenantService(pool, jobEnqueuer, providerConfigService, cfg))
	renderHandler := rendering.NewHandler(rendering.NewService(pool, jobEnqueuer))
	metaConnectionHandler := metaconnections.NewHandler(metaconnections.NewTenantService(pool, secretCipher, providerConfigService), cfg.AppURL)
	publishingHandler := publishing.NewHandler(publishing.NewService(pool, jobEnqueuer))
	metaAdsHandler := metaads.NewHandler(metaads.NewService(pool, jobEnqueuer))
	operationsHandler := operations.NewHandler(pool, cfg)
	analyticsHandler := analytics.NewHandler(analytics.NewService(pool))
	loginLimiter := auth.NewLoginRateLimiter(10, 15*time.Minute)
	httpMetrics := studiometrics.New(pool)

	application := fiber.New(fiber.Config{
		AppName:      "AI Product Marketing Studio API",
		BodyLimit:    25 * 1024 * 1024,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 90 * time.Second,
		ErrorHandler: func(c fiber.Ctx, err error) error {
			var fiberError *fiber.Error
			if errors.As(err, &fiberError) {
				return problem.Write(c, fiberError.Code, "http", "Yêu cầu không thể xử lý", fiberError.Message)
			}
			logger.Error("unhandled request error", "error", err, "request_id", c.Locals("request_id"))
			return problem.Write(c, fiber.StatusInternalServerError, "internal", "Lỗi hệ thống", "Hệ thống gặp lỗi ngoài dự kiến.")
		},
	})
	application.Use(recover.New(recover.Config{EnableStackTrace: cfg.Environment != "production"}))
	application.Use(requestid.Middleware)
	application.Use(security.Headers)
	application.Use(security.OriginGuard(cfg.AppURL))
	application.Use(traceMiddleware)
	application.Use(httpMetrics.Middleware)
	application.Use(accessLog(logger))

	v1 := application.Group("/v1")
	v1.Get("/health/live", operationsHandler.Liveness)
	v1.Get("/health/ready", operationsHandler.Readiness)
	v1.Get("/metrics", adaptor.HTTPHandler(promhttp.Handler()))
	v1.Get("/auth/bootstrap/status", authHandler.BootstrapStatus)
	v1.Post("/auth/bootstrap", loginLimiter.Middleware, authHandler.BootstrapAdmin)
	v1.Post("/auth/login", loginLimiter.Middleware, authHandler.Login)
	v1.Post("/webhooks/seedance", videoHandler.Webhook)
	v1.Get("/meta/oauth/callback", metaConnectionHandler.Callback)

	protected := v1.Group("", auth.Authenticate(authService))
	protected.Get("/auth/me", authHandler.Me)
	protected.Use(auth.RequireCSRF(authService))
	protected.Put("/auth/me", authHandler.UpdateMe)
	protected.Post("/auth/logout", authHandler.Logout)
	protected.Post("/auth/change-password", authHandler.ChangePassword)
	protected.Get("/auth/sessions", authHandler.ListSessions)
	protected.Delete("/auth/sessions/:sessionId", authHandler.RevokeSession)
	protected.Use(auth.RequirePasswordChangeCleared)
	protected.Use(mutationAudit(pool, logger))
	protected.Use(operations.MaintenanceGuard(pool))
	protected.Get("/clients", clientHandler.List)
	protected.Get("/clients/:clientId", clientHandler.Get)
	protected.Post("/clients", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), clientHandler.Create)
	protected.Put("/clients/:clientId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), clientHandler.Update)
	protected.Patch("/clients/:clientId/status", auth.RequireRole(db.InternalUserRoleADMIN), clientHandler.SetStatus)
	protected.Get("/clients/:clientId/workspaces", workspaceHandler.List)
	protected.Get("/clients/:clientId/workspaces/:workspaceId", workspaceHandler.Get)
	protected.Post("/clients/:clientId/workspaces", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), workspaceHandler.Create)
	protected.Put("/clients/:clientId/workspaces/:workspaceId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), workspaceHandler.Update)
	protected.Patch("/clients/:clientId/workspaces/:workspaceId/status", auth.RequireRole(db.InternalUserRoleADMIN), workspaceHandler.SetStatus)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/brands", brandHandler.List)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/brands/:brandId", brandHandler.Get)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/brands", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), brandHandler.Create)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/brands/:brandId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), brandHandler.Update)
	protected.Patch("/clients/:clientId/workspaces/:workspaceId/brands/:brandId/status", auth.RequireRole(db.InternalUserRoleADMIN), brandHandler.SetStatus)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/products", productHandler.List)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/products/:productId", productHandler.Get)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/products/:productId/media-readiness", productHandler.MediaReadiness)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/products", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), productHandler.Create)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/products/:productId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), productHandler.Update)
	protected.Patch("/clients/:clientId/workspaces/:workspaceId/products/:productId/status", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), productHandler.SetStatus)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/products/:productId/facts", truthHandler.ListFacts)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/products/:productId/facts", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), truthHandler.CreateFact)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/products/:productId/facts/:factId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), truthHandler.UpdateFact)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/products/:productId/facts/:factId/approve", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), truthHandler.ApproveFact)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/products/:productId/claims", truthHandler.ListClaims)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/products/:productId/claims", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), truthHandler.CreateClaim)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/products/:productId/claims/:claimId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), truthHandler.UpdateClaim)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/products/:productId/claims/:claimId/approve", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), truthHandler.ApproveClaim)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/media-assets", mediaHandler.List)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/products/:productId/media-assets/:assetId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), mediaHandler.AttachProduct)
	protected.Delete("/clients/:clientId/workspaces/:workspaceId/products/:productId/media-assets/:assetId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), mediaHandler.DetachProduct)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/media-uploads", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), mediaHandler.StartUpload)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/media-uploads/:uploadId/parts/:partNumber", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), mediaHandler.Part)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/media-uploads/:uploadId/complete", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), mediaHandler.Complete)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/media-assets/:assetId/download", mediaHandler.Download)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/media-assets/:assetId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), mediaHandler.Update)
	protected.Patch("/clients/:clientId/workspaces/:workspaceId/media-assets/:assetId/status", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), mediaHandler.SetStatus)
	protected.Delete("/clients/:clientId/workspaces/:workspaceId/media-assets/:assetId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), mediaHandler.Delete)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns", campaignHandler.List)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId", campaignHandler.Get)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/progress", campaignHandler.GetProgress)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), campaignHandler.Create)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), campaignHandler.Update)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/duplicate", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), campaignHandler.Duplicate)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/characters", characterHandler.List)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/characters", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), characterHandler.Create)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/characters/:characterId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), characterHandler.Update)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/characters", characterHandler.GetSelection)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/characters", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), characterHandler.Select)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/cost-estimate", planningHandler.Estimate)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/generation-jobs", planningHandler.ListJobs)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/generation-jobs/:jobId", planningHandler.GetJob)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/generation-jobs", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), planningHandler.StartGeneration)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/concepts", planningHandler.ListConcepts)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/concepts/:conceptId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), planningHandler.UpdateConcept)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/concepts/:conceptId/decision", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), planningHandler.DecideConcept)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/content", planningHandler.ListContent)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/content/:contentId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), planningHandler.UpdateContent)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/content/:contentId/approve", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), planningHandler.ApproveContent)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/script", planningHandler.GetScript)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/script", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), planningHandler.UpdateScript)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/script/approve", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), planningHandler.ApproveScript)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes", planningHandler.ListScenes)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/reorder", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), planningHandler.ReorderScenes)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), planningHandler.UpdateScene)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/approve", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), planningHandler.ApproveScene)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/duplicate", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), planningHandler.DuplicateScene)
	protected.Delete("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), planningHandler.DeleteScene)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/generations", videoHandler.List)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/generations", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), videoHandler.Start)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/generations/:generationId", videoHandler.Get)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/generations/:generationId/cancel", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), videoHandler.Cancel)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/generations/:generationId/review", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), videoHandler.Review)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/generations/:generationId/edit", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), videoHandler.UpdateEdit)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/scenes/:sceneId/generations/:generationId/select", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), videoHandler.Select)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/video-project", renderHandler.GetProject)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/video-project", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), renderHandler.SaveProject)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/final-renders", renderHandler.List)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/final-renders", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), renderHandler.Start)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/final-renders/:renderJobId/review", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), renderHandler.Review)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/final-renders/:renderJobId/select", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), renderHandler.Select)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/meta/connection", metaConnectionHandler.Get)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/meta/oauth/start", auth.RequireRole(db.InternalUserRoleADMIN), metaConnectionHandler.Start)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/meta/sync", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), metaConnectionHandler.Sync)
	protected.Delete("/clients/:clientId/workspaces/:workspaceId/meta/connection", auth.RequireRole(db.InternalUserRoleADMIN), metaConnectionHandler.Disconnect)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/social-posts", publishingHandler.List)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/social-posts", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), publishingHandler.Create)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/social-posts/:postId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), publishingHandler.Update)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/social-posts/:postId/review", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), publishingHandler.Review)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/meta-ad-guardrails", metaAdsHandler.GetGuardrails)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/meta-ad-guardrails", auth.RequireRole(db.InternalUserRoleADMIN), metaAdsHandler.SaveGuardrails)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/meta-ad-campaigns", metaAdsHandler.List)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/meta-ad-campaigns", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), metaAdsHandler.Create)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/meta-ad-campaigns/:adCampaignId", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), metaAdsHandler.Update)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/meta-ad-campaigns/:adCampaignId/review", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), metaAdsHandler.ReviewCreate)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/meta-ad-campaigns/:adCampaignId/actions", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), metaAdsHandler.RequestAction)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/meta-ad-campaigns/:adCampaignId/actions", metaAdsHandler.ListActions)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/campaigns/:campaignId/meta-ad-campaigns/:adCampaignId/actions/:actionId/review", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), metaAdsHandler.ReviewAction)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/analytics/summary", analyticsHandler.Summary)
	protected.Get("/clients/:clientId/workspaces/:workspaceId/analytics/recommendations", analyticsHandler.ListRecommendations)
	protected.Post("/clients/:clientId/workspaces/:workspaceId/analytics/recommendations", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleOPERATOR), analyticsHandler.GenerateRecommendations)
	protected.Put("/clients/:clientId/workspaces/:workspaceId/analytics/recommendations/:recommendationId/review", auth.RequireRole(db.InternalUserRoleADMIN, db.InternalUserRoleREVIEWER), analyticsHandler.ReviewRecommendation)

	admins := protected.Group("", auth.RequireRole(db.InternalUserRoleADMIN))
	admins.Get("/internal-users", userHandler.List)
	admins.Post("/internal-users", userHandler.Create)
	admins.Put("/internal-users/:userId", userHandler.Update)
	admins.Post("/internal-users/:userId/reset-password", userHandler.ResetPassword)
	admins.Patch("/internal-users/:userId/status", userHandler.SetStatus)
	admins.Get("/clients/:clientId/provider-configuration", providerConfigHandler.Get)
	admins.Put("/clients/:clientId/provider-configuration/mode", providerConfigHandler.SaveMode)
	admins.Put("/clients/:clientId/provider-configuration/:provider", providerConfigHandler.Save)
	admins.Get("/operations/overview", operationsHandler.Overview)
	admins.Post("/operations/jobs/:jobId/retry", operationsHandler.RetryJob)
	admins.Post("/operations/jobs/:jobId/cancel", operationsHandler.CancelJob)
	admins.Put("/operations/maintenance", operationsHandler.SetMaintenance)

	return application, nil
}

func mutationAudit(pool *pgxpool.Pool, logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		method := c.Method()
		if method != fiber.MethodPost && method != fiber.MethodPut && method != fiber.MethodPatch && method != fiber.MethodDelete {
			return c.Next()
		}
		err := c.Next()
		principal, ok := auth.PrincipalFrom(c)
		if !ok {
			return err
		}
		status := c.Response().StatusCode()
		outcome := "SUCCESS"
		if status == fiber.StatusUnauthorized || status == fiber.StatusForbidden {
			outcome = "DENIED"
		} else if status >= 400 || err != nil {
			outcome = "FAILURE"
		}
		requestID, _ := c.Locals("request_id").(string)
		var address *netip.Addr
		if parsed, parseErr := netip.ParseAddr(strings.TrimSpace(c.IP())); parseErr == nil {
			address = &parsed
		}
		clientID := nullableParam(c, "clientId")
		workspaceID := nullableParam(c, "workspaceId")
		entityType, entityID := mutationEntity(c)
		event := audit.Event{ActorID: uuid.NullUUID{UUID: principal.UserID, Valid: true}, Action: "http." + strings.ToLower(method) + "." + strings.TrimPrefix(c.Route().Path, "/v1/"), EntityType: entityType, EntityID: entityID, ClientID: clientID, WorkspaceID: workspaceID, RequestID: requestID, IPAddress: address, UserAgent: c.Get(fiber.HeaderUserAgent), Outcome: outcome, Metadata: map[string]any{"route": c.Route().Path, "status": status}}
		if auditErr := audit.Record(c.Context(), db.New(pool), event); auditErr != nil {
			logger.Error("write mutation audit log", "error", auditErr, "request_id", requestID, "route", c.Route().Path)
		}
		return err
	}
}

func mutationEntity(c fiber.Ctx) (string, uuid.NullUUID) {
	for _, candidate := range []struct{ param, entity string }{{"renderJobId", "final_render"}, {"generationId", "scene_generation"}, {"factId", "product_fact"}, {"claimId", "product_claim"}, {"assetId", "media_asset"}, {"uploadId", "media_upload"}, {"conceptId", "campaign_concept"}, {"sceneId", "scene"}, {"campaignId", "campaign"}, {"brandId", "brand"}, {"productId", "product"}, {"workspaceId", "workspace"}, {"clientId", "client"}} {
		if id := nullableParam(c, candidate.param); id.Valid {
			return candidate.entity, id
		}
	}
	return "", uuid.NullUUID{}
}

func nullableParam(c fiber.Ctx, name string) uuid.NullUUID {
	id, err := uuid.Parse(c.Params(name))
	return uuid.NullUUID{UUID: id, Valid: err == nil}
}

func traceMiddleware(c fiber.Ctx) error {
	tracer := otel.Tracer("studio-api/http")
	ctx, span := tracer.Start(c.Context(), c.Method()+" "+c.Route().Path)
	defer span.End()
	c.SetContext(ctx)
	span.SetAttributes(attribute.String("http.request.method", c.Method()), attribute.String("url.path", c.Path()))
	err := c.Next()
	span.SetAttributes(attribute.Int("http.response.status_code", c.Response().StatusCode()))
	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, "request failed")
	}
	return err
}

func accessLog(logger *slog.Logger) fiber.Handler {
	return func(c fiber.Ctx) error {
		started := time.Now()
		err := c.Next()
		logger.Info("http request",
			"request_id", c.Locals("request_id"),
			"method", c.Method(),
			"path", c.Route().Path,
			"status", c.Response().StatusCode(),
			"duration_ms", time.Since(started).Milliseconds(),
		)
		return err
	}
}
