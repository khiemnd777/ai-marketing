package video

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func generationScope(c fiber.Ctx, withGeneration bool) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
	values := make([]uuid.UUID, 0, 5)
	params := []string{"clientId", "workspaceId", "campaignId", "sceneId"}
	if withGeneration {
		params = append(params, "generationId")
	}
	for _, param := range params {
		value, err := uuid.Parse(c.Params(param))
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
		values = append(values, value)
	}
	if !withGeneration {
		values = append(values, uuid.Nil)
	}
	return values[0], values[1], values[2], values[3], values[4], nil
}

func generationActor(c fiber.Ctx) uuid.UUID {
	principal, _ := auth.PrincipalFrom(c)
	return principal.UserID
}

func (h *Handler) Start(c fiber.Ctx) error {
	clientID, workspaceID, campaignID, sceneID, _, err := generationScope(c, false)
	if err != nil {
		return generationError(c, err)
	}
	key := strings.TrimSpace(c.Get("Idempotency-Key"))
	if key == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc cho mỗi yêu cầu tạo video.")
	}
	var input StartInput
	if err = c.Bind().Body(&input); err != nil {
		return generationError(c, ErrInvalid)
	}
	item, err := h.service.Start(c.Context(), clientID, workspaceID, campaignID, sceneID, generationActor(c), key, input)
	if err != nil {
		return generationError(c, err)
	}
	return c.Status(fiber.StatusAccepted).JSON(item)
}

func (h *Handler) List(c fiber.Ctx) error {
	clientID, workspaceID, campaignID, sceneID, _, err := generationScope(c, false)
	if err != nil {
		return generationError(c, err)
	}
	items, err := h.service.List(c.Context(), clientID, workspaceID, campaignID, sceneID)
	if err != nil {
		return generationError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) Get(c fiber.Ctx) error {
	clientID, workspaceID, campaignID, sceneID, generationID, err := generationScope(c, true)
	if err != nil {
		return generationError(c, err)
	}
	item, err := h.service.Get(c.Context(), clientID, workspaceID, campaignID, sceneID, generationID)
	if err != nil {
		return generationError(c, err)
	}
	return c.JSON(item)
}

func (h *Handler) Cancel(c fiber.Ctx) error {
	clientID, workspaceID, campaignID, sceneID, generationID, err := generationScope(c, true)
	if err != nil {
		return generationError(c, err)
	}
	item, err := h.service.Cancel(c.Context(), clientID, workspaceID, campaignID, sceneID, generationID, generationActor(c))
	if err != nil {
		return generationError(c, err)
	}
	return c.JSON(item)
}

func (h *Handler) Review(c fiber.Ctx) error {
	clientID, workspaceID, campaignID, sceneID, generationID, err := generationScope(c, true)
	if err != nil {
		return generationError(c, err)
	}
	var input ReviewInput
	if err = c.Bind().Body(&input); err != nil {
		return generationError(c, ErrInvalid)
	}
	item, err := h.service.Review(c.Context(), clientID, workspaceID, campaignID, sceneID, generationID, generationActor(c), input)
	if err != nil {
		return generationError(c, err)
	}
	return c.JSON(item)
}

func (h *Handler) Select(c fiber.Ctx) error {
	clientID, workspaceID, campaignID, sceneID, generationID, err := generationScope(c, true)
	if err != nil {
		return generationError(c, err)
	}
	item, err := h.service.Select(c.Context(), clientID, workspaceID, campaignID, sceneID, generationID, generationActor(c))
	if err != nil {
		return generationError(c, err)
	}
	return c.JSON(item)
}

func (h *Handler) UpdateEdit(c fiber.Ctx) error {
	clientID, workspaceID, campaignID, sceneID, generationID, err := generationScope(c, true)
	if err != nil {
		return generationError(c, err)
	}
	var input GenerationEdit
	if err = c.Bind().Body(&input); err != nil {
		return generationError(c, ErrInvalid)
	}
	item, err := h.service.UpdateEdit(c.Context(), clientID, workspaceID, campaignID, sceneID, generationID, generationActor(c), input)
	if err != nil {
		return generationError(c, err)
	}
	return c.JSON(item)
}

func generationError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalid):
		return problem.Write(c, 422, "validation", "Dữ liệu tạo video không hợp lệ", "Kiểm tra định dạng, phiên bản, thông số cắt và checklist review.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy generation", "Generation không thuộc scene hoặc workspace này.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, 409, "generation-conflict", "Trạng thái đã thay đổi", "Tải lại generation trước khi thử lại; job tính phí không được tự động gửi lại.")
	case errors.Is(err, ErrPrerequisite):
		return problem.Write(c, 409, "approval-required", "Chưa đủ điều kiện", "Scene phải được duyệt, QC phải đạt và checklist người duyệt phải hoàn tất.")
	case errors.Is(err, ErrUnavailable):
		return problem.Write(c, 503, "provider-unavailable", "Seedance chưa sẵn sàng", "Kiểm tra cấu hình provider hoặc thử lại sau.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý video", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
