package clients

import (
	"errors"
	"net/netip"
	"strconv"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) List(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	size, _ := strconv.Atoi(c.Query("pageSize", "25"))
	result, err := h.service.List(c.Context(), page, size, c.Query("search"), c.Query("status"))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(fiber.Map{"items": result.Items, "page": fiber.Map{"number": result.Number, "size": result.Size, "totalItems": result.TotalItems, "totalPages": result.TotalPages}})
}
func (h *Handler) Get(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return writeError(c, ErrInvalid)
	}
	item, err := h.service.Get(c.Context(), id)
	if err != nil {
		return writeError(c, err)
	}
	c.Set("ETag", `W/"`+strconv.FormatInt(item.Version, 10)+`"`)
	return c.JSON(item)
}
func (h *Handler) Create(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc.")
	}
	var input Input
	if err := c.Bind().Body(&input); err != nil {
		return writeError(c, ErrInvalid)
	}
	actor, ok := auth.PrincipalFrom(c)
	if !ok {
		return writeError(c, auth.ErrUnauthenticated)
	}
	item, err := h.service.Create(c.Context(), input, actor, requestMetadata(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.Status(201).JSON(item)
}
func (h *Handler) Update(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return writeError(c, ErrInvalid)
	}
	var input Input
	if err := c.Bind().Body(&input); err != nil {
		return writeError(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, err := h.service.Update(c.Context(), id, input, actor, requestMetadata(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(item)
}
func (h *Handler) SetStatus(c fiber.Ctx) error {
	id, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return writeError(c, ErrInvalid)
	}
	var input struct {
		Status  string `json:"status"`
		Version int64  `json:"version"`
	}
	if err := c.Bind().Body(&input); err != nil {
		return writeError(c, ErrInvalid)
	}
	actor, _ := auth.PrincipalFrom(c)
	item, err := h.service.SetStatus(c.Context(), id, input.Status, input.Version, actor, requestMetadata(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(item)
}
func requestMetadata(c fiber.Ctx) auth.ClientMetadata {
	requestID, _ := c.Locals("request_id").(string)
	var address *netip.Addr
	if parsed, err := netip.ParseAddr(strings.TrimSpace(c.IP())); err == nil {
		address = &parsed
	}
	return auth.ClientMetadata{RequestID: requestID, IPAddress: address, UserAgent: c.Get(fiber.HeaderUserAgent)}
}
func writeError(c fiber.Ctx, err error) error {
	switch {
	case errors.Is(err, auth.ErrUnauthenticated):
		return problem.Write(c, 401, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ.")
	case errors.Is(err, ErrInvalid):
		return problem.Write(c, 422, "validation", "Dữ liệu không hợp lệ", "Vui lòng kiểm tra dữ liệu khách hàng.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy khách hàng", "Khách hàng không tồn tại.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, 409, "version-conflict", "Dữ liệu đã thay đổi", "Tải lại dữ liệu trước khi lưu.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý khách hàng", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
