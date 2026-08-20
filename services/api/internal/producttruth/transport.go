package producttruth

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func ids(c fiber.Ctx, child string) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
	a, e := uuid.Parse(c.Params("clientId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	b, e := uuid.Parse(c.Params("workspaceId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	p, e := uuid.Parse(c.Params("productId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	id := uuid.Nil
	if child != "" {
		id, e = uuid.Parse(c.Params(child))
		if e != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return a, b, p, id, nil
}
func (h *Handler) ListFacts(c fiber.Ctx) error {
	a, b, p, _, e := ids(c, "")
	if e != nil {
		return out(c, e)
	}
	v, e := h.service.ListFacts(c.Context(), a, b, p)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(fiber.Map{"items": v})
}
func (h *Handler) CreateFact(c fiber.Ctx) error {
	a, b, p, _, e := ids(c, "")
	if e != nil {
		return out(c, e)
	}
	var i FactInput
	if c.Bind().Body(&i) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.CreateFact(c.Context(), a, b, p, actor.UserID, i)
	if e != nil {
		return out(c, e)
	}
	return c.Status(201).JSON(v)
}
func (h *Handler) UpdateFact(c fiber.Ctx) error {
	a, b, p, id, e := ids(c, "factId")
	if e != nil {
		return out(c, e)
	}
	var i FactInput
	if c.Bind().Body(&i) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.UpdateFact(c.Context(), a, b, p, id, actor.UserID, i)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func (h *Handler) ApproveFact(c fiber.Ctx) error {
	a, b, p, id, e := ids(c, "factId")
	if e != nil {
		return out(c, e)
	}
	var i struct {
		Lock    bool  `json:"lock"`
		Version int64 `json:"version"`
	}
	if c.Bind().Body(&i) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.ApproveFact(c.Context(), a, b, p, id, actor, auth.MetadataFrom(c), i.Lock, i.Version)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func (h *Handler) ListClaims(c fiber.Ctx) error {
	a, b, p, _, e := ids(c, "")
	if e != nil {
		return out(c, e)
	}
	v, e := h.service.ListClaims(c.Context(), a, b, p)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(fiber.Map{"items": v})
}
func (h *Handler) CreateClaim(c fiber.Ctx) error {
	a, b, p, _, e := ids(c, "")
	if e != nil {
		return out(c, e)
	}
	var i ClaimInput
	if c.Bind().Body(&i) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.CreateClaim(c.Context(), a, b, p, actor.UserID, i)
	if e != nil {
		return out(c, e)
	}
	return c.Status(201).JSON(v)
}
func (h *Handler) UpdateClaim(c fiber.Ctx) error {
	a, b, p, id, e := ids(c, "claimId")
	if e != nil {
		return out(c, e)
	}
	var i ClaimInput
	if c.Bind().Body(&i) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.UpdateClaim(c.Context(), a, b, p, id, actor.UserID, i)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func (h *Handler) ApproveClaim(c fiber.Ctx) error {
	a, b, p, id, e := ids(c, "claimId")
	if e != nil {
		return out(c, e)
	}
	var i struct {
		Version int64 `json:"version"`
	}
	if c.Bind().Body(&i) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.ApproveClaim(c.Context(), a, b, p, id, actor, auth.MetadataFrom(c), i.Version)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func out(c fiber.Ctx, e error) error {
	switch {
	case errors.Is(e, ErrInvalid):
		return problem.Write(c, 422, "validation", "Product Truth không hợp lệ", "Kiểm tra định dạng, bằng chứng và quy tắc ngành hàng.")
	case errors.Is(e, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy Product Truth", "Bản ghi không thuộc sản phẩm hoặc workspace này.")
	case errors.Is(e, ErrConflict):
		return problem.Write(c, 409, "version-conflict", "Product Truth đã thay đổi", "Tải lại dữ liệu trước khi lưu.")
	case errors.Is(e, ErrLocked):
		return problem.Write(c, 409, "locked-fact", "Fact đã khóa", "Fact đã duyệt và khóa không thể chỉnh sửa âm thầm.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý Product Truth", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
