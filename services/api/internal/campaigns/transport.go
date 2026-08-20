package campaigns

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

func campaignScope(c fiber.Ctx, withID bool) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	workspaceID, err := uuid.Parse(c.Params("workspaceId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	id := uuid.Nil
	if withID {
		id, err = uuid.Parse(c.Params("campaignId"))
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return clientID, workspaceID, id, nil
}

func (h *Handler) List(c fiber.Ctx) error {
	a, b, _, err := campaignScope(c, false)
	if err != nil {
		return writeError(c, err)
	}
	items, err := h.service.List(c.Context(), a, b, c.Query("search"), c.Query("status"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"items": items})
}

func (h *Handler) Get(c fiber.Ctx) error {
	a, b, id, err := campaignScope(c, true)
	if err != nil {
		return writeError(c, err)
	}
	item, err := h.service.Get(c.Context(), a, b, id)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(item)
}

func (h *Handler) Create(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc.")
	}
	a, b, _, err := campaignScope(c, false)
	if err != nil {
		return writeError(c, err)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return writeError(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, err := h.service.Create(c.Context(), a, b, input, actor, auth.MetadataFrom(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.Status(201).JSON(item)
}

func (h *Handler) Update(c fiber.Ctx) error {
	a, b, id, err := campaignScope(c, true)
	if err != nil {
		return writeError(c, err)
	}
	var input Input
	if c.Bind().Body(&input) != nil {
		return writeError(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, err := h.service.Update(c.Context(), a, b, id, input, actor, auth.MetadataFrom(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(item)
}

func (h *Handler) Duplicate(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc.")
	}
	a, b, id, err := campaignScope(c, true)
	if err != nil {
		return writeError(c, err)
	}
	var input struct {
		Name string `json:"name"`
	}
	if c.Bind().Body(&input) != nil {
		return writeError(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, err := h.service.Duplicate(c.Context(), a, b, id, input.Name, actor, auth.MetadataFrom(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.Status(201).JSON(item)
}

func writeError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, ErrInvalid):
		return problem.Write(c, 422, "validation", "Campaign không hợp lệ", "Kiểm tra brief, định dạng video, nền tảng, thời lượng, ngân sách và ngày chạy.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy campaign", "Campaign hoặc tài nguyên tham chiếu không thuộc workspace.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, 409, "version-conflict", "Campaign đã thay đổi", "Tải lại dữ liệu trước khi lưu.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý campaign", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
