package providerconfigs

import (
	"errors"
	"net/netip"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Get(c fiber.Ctx) error {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return writeError(c, ErrInvalid)
	}
	profile, err := h.service.Get(c.Context(), clientID)
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(profile)
}

func (h *Handler) SaveMode(c fiber.Ctx) error {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return writeError(c, ErrInvalid)
	}
	var input ModeInput
	if err = c.Bind().Body(&input); err != nil {
		return writeError(c, ErrInvalid)
	}
	actor, ok := auth.PrincipalFrom(c)
	if !ok {
		return writeError(c, auth.ErrUnauthenticated)
	}
	profile, err := h.service.SaveMode(c.Context(), clientID, input, actor, requestMetadata(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(profile)
}

func (h *Handler) Save(c fiber.Ctx) error {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return writeError(c, ErrInvalid)
	}
	var input SaveInput
	if err = c.Bind().Body(&input); err != nil {
		return writeError(c, ErrInvalid)
	}
	actor, ok := auth.PrincipalFrom(c)
	if !ok {
		return writeError(c, auth.ErrUnauthenticated)
	}
	profile, err := h.service.Save(c.Context(), clientID, c.Params("provider"), input, actor, requestMetadata(c))
	if err != nil {
		return writeError(c, err)
	}
	return c.JSON(profile)
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
		return problem.Write(c, 422, "provider-configuration-invalid", "Cấu hình provider không hợp lệ", "Kiểm tra endpoint, model, version, pricing và các trường bắt buộc.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, 404, "client-not-found", "Không tìm thấy khách hàng", "Khách hàng không tồn tại.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, 409, "provider-configuration-conflict", "Cấu hình đã thay đổi", "Tải lại cấu hình trước khi lưu.")
	default:
		return problem.Write(c, 500, "provider-configuration-error", "Không thể lưu cấu hình provider", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
