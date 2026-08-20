package metaconnections

import (
	"errors"
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/meta"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct {
	service *Service
	appURL  string
}

func NewHandler(service *Service, appURL string) *Handler {
	return &Handler{service: service, appURL: strings.TrimRight(appURL, "/")}
}

func metaScope(c fiber.Ctx) (uuid.UUID, uuid.UUID, error) {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalid
	}
	workspaceID, err := uuid.Parse(c.Params("workspaceId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, ErrInvalid
	}
	return clientID, workspaceID, nil
}
func (h *Handler) Start(c fiber.Ctx) error {
	clientID, workspaceID, err := metaScope(c)
	if err != nil {
		return connectionError(c, err)
	}
	principal, _ := auth.PrincipalFrom(c)
	result, err := h.service.StartOAuth(c.Context(), clientID, workspaceID, principal.UserID)
	if err != nil {
		return connectionError(c, err)
	}
	return c.JSON(result)
}
func (h *Handler) Get(c fiber.Ctx) error {
	clientID, workspaceID, err := metaScope(c)
	if err != nil {
		return connectionError(c, err)
	}
	result, err := h.service.Get(c.Context(), clientID, workspaceID)
	if err != nil {
		return connectionError(c, err)
	}
	return c.JSON(result)
}
func (h *Handler) Sync(c fiber.Ctx) error {
	clientID, workspaceID, err := metaScope(c)
	if err != nil {
		return connectionError(c, err)
	}
	result, err := h.service.Sync(c.Context(), clientID, workspaceID)
	if err != nil {
		return connectionError(c, err)
	}
	return c.JSON(result)
}
func (h *Handler) Disconnect(c fiber.Ctx) error {
	clientID, workspaceID, err := metaScope(c)
	if err != nil {
		return connectionError(c, err)
	}
	principal, _ := auth.PrincipalFrom(c)
	if err = h.service.Disconnect(c.Context(), clientID, workspaceID, principal.UserID); err != nil {
		return connectionError(c, err)
	}
	return c.SendStatus(fiber.StatusNoContent)
}
func (h *Handler) Callback(c fiber.Ctx) error {
	result, err := h.service.Callback(c.Context(), strings.TrimSpace(c.Query("state")), strings.TrimSpace(c.Query("code")))
	if err != nil {
		return connectionError(c, err)
	}
	query := url.Values{"clientId": {result.ClientID.String()}, "workspaceId": {result.WorkspaceID.String()}, "connected": {"1"}}
	c.Set(fiber.HeaderLocation, h.appURL+"/settings/meta?"+query.Encode())
	return c.SendStatus(fiber.StatusSeeOther)
}
func connectionError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalid):
		return problem.Write(c, 422, "meta-invalid", "Kết nối Meta không hợp lệ", "OAuth state, code hoặc token metadata không hợp lệ.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, 404, "meta-not-found", "Chưa có kết nối Meta", "Workspace chưa kết nối Meta hoặc kết nối đã ngắt.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, 409, "meta-conflict", "Kết nối Meta đã thay đổi", "Bắt đầu lại OAuth để tạo một kết nối mới.")
	case errors.Is(err, meta.ErrUnauthorized):
		return problem.Write(c, 401, "meta-token-expired", "Meta token hết hiệu lực", "Kết nối lại Meta để tiếp tục.")
	case errors.Is(err, meta.ErrConfiguration):
		return problem.Write(c, 503, "meta-unconfigured", "Meta chưa được cấu hình", "Admin cần cấu hình Meta app và API version.")
	default:
		return problem.Write(c, 502, "meta-provider", "Meta chưa thể xử lý yêu cầu", "Thử lại hoặc kiểm tra quyền ứng dụng Meta.")
	}
}
