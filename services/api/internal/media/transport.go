package media

import (
	"errors"
	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/storage"
	"strconv"
	"strings"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{service: s} }
func scope(c fiber.Ctx, param string) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	a, e := uuid.Parse(c.Params("clientId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	b, e := uuid.Parse(c.Params("workspaceId"))
	if e != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	id := uuid.Nil
	if param != "" {
		id, e = uuid.Parse(c.Params(param))
		if e != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return a, b, id, nil
}
func (h *Handler) List(c fiber.Ctx) error {
	a, b, _, e := scope(c, "")
	if e != nil {
		return out(c, e)
	}
	var productID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("productId")); raw != "" {
		parsed, parseErr := uuid.Parse(raw)
		if parseErr != nil {
			return out(c, ErrInvalid)
		}
		productID = &parsed
	}
	items, e := h.service.List(c.Context(), a, b, c.Query("search"), c.Query("assetType"), c.Query("status"), productID)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) AttachProduct(c fiber.Ctx) error {
	a, b, assetID, e := scope(c, "assetId")
	if e != nil {
		return out(c, e)
	}
	productID, e := uuid.Parse(c.Params("productId"))
	if e != nil {
		return out(c, ErrInvalid)
	}
	var input struct {
		Version int64 `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	asset, e := h.service.AttachProduct(c.Context(), a, b, productID, assetID, actor.UserID, input.Version)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(asset)
}
func (h *Handler) DetachProduct(c fiber.Ctx) error {
	a, b, assetID, e := scope(c, "assetId")
	if e != nil {
		return out(c, e)
	}
	productID, e := uuid.Parse(c.Params("productId"))
	if e != nil {
		return out(c, ErrInvalid)
	}
	var input struct {
		Version int64 `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	asset, e := h.service.DetachProduct(c.Context(), a, b, productID, assetID, actor.UserID, input.Version)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(asset)
}
func (h *Handler) StartUpload(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc.")
	}
	a, b, _, e := scope(c, "")
	if e != nil {
		return out(c, e)
	}
	var i UploadInput
	if c.Bind().Body(&i) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.StartUpload(c.Context(), a, b, actor.UserID, i)
	if e != nil {
		return out(c, e)
	}
	return c.Status(201).JSON(v)
}
func (h *Handler) Part(c fiber.Ctx) error {
	a, b, id, e := scope(c, "uploadId")
	if e != nil {
		return out(c, e)
	}
	n, _ := strconv.Atoi(c.Params("partNumber"))
	v, e := h.service.PresignPart(c.Context(), a, b, id, int32(n))
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func (h *Handler) Complete(c fiber.Ctx) error {
	a, b, id, e := scope(c, "uploadId")
	if e != nil {
		return out(c, e)
	}
	var i struct {
		Parts []storage.UploadedPart `json:"parts"`
	}
	if c.Bind().Body(&i) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	v, e := h.service.Complete(c.Context(), a, b, id, actor.UserID, i.Parts)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func (h *Handler) Download(c fiber.Ctx) error {
	a, b, id, e := scope(c, "assetId")
	if e != nil {
		return out(c, e)
	}
	v, e := h.service.Download(c.Context(), a, b, id)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(v)
}
func (h *Handler) Update(c fiber.Ctx) error {
	a, b, id, e := scope(c, "assetId")
	if e != nil {
		return out(c, e)
	}
	var input UpdateInput
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	asset, e := h.service.Update(c.Context(), a, b, id, actor.UserID, input)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(asset)
}
func (h *Handler) SetStatus(c fiber.Ctx) error {
	a, b, id, e := scope(c, "assetId")
	if e != nil {
		return out(c, e)
	}
	var input struct {
		Status  string `json:"status"`
		Version int64  `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return out(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	asset, e := h.service.SetStatus(c.Context(), a, b, id, actor.UserID, input.Status, input.Version)
	if e != nil {
		return out(c, e)
	}
	return c.JSON(asset)
}
func (h *Handler) Delete(c fiber.Ctx) error {
	a, b, id, e := scope(c, "assetId")
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
	e = h.service.SoftDelete(c.Context(), a, b, id, actor.UserID, i.Version)
	if e != nil {
		return out(c, e)
	}
	return c.SendStatus(204)
}
func out(c fiber.Ctx, e error) error {
	switch {
	case errors.Is(e, ErrInvalid):
		return problem.Write(c, 422, "validation", "Media không hợp lệ", "Kiểm tra loại file, MIME, phần mở rộng, dung lượng và quyền sử dụng.")
	case errors.Is(e, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy media", "Asset hoặc phiên upload không thuộc workspace.")
	case errors.Is(e, ErrUnavailable):
		return problem.Write(c, 503, "provider-not-configured", "Kho đối tượng chưa cấu hình", "Cấu hình R2 hoặc MinIO trước khi upload.")
	case errors.Is(e, ErrConflict):
		return problem.Write(c, 409, "media-conflict", "Media đã thay đổi", "Tải lại asset hoặc phiên upload trước khi tiếp tục.")
	case errors.Is(e, ErrAssigned):
		return problem.Write(c, 409, "media-product-conflict", "Media đã thuộc sản phẩm khác", "Gỡ media khỏi sản phẩm hiện tại trước khi gắn lại.")
	case errors.Is(e, ErrInUse):
		return problem.Write(c, 409, "media-in-use", "Media đang được sử dụng", "Gỡ asset khỏi Brand logo, fact, claim, scene hoặc edit video trước khi hạ trạng thái, tách hay xóa.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý media", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
