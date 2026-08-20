package internalusers

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct{ service *Service }

type createRequest struct {
	Email             string              `json:"email"`
	DisplayName       string              `json:"displayName"`
	Role              db.InternalUserRole `json:"role"`
	TemporaryPassword string              `json:"temporaryPassword"`
}

type userResponse struct {
	ID                     string     `json:"id"`
	Email                  string     `json:"email"`
	DisplayName            string     `json:"displayName"`
	Role                   string     `json:"role"`
	Status                 string     `json:"status"`
	RequiresPasswordChange bool       `json:"requiresPasswordChange"`
	LastLoginAt            *time.Time `json:"lastLoginAt"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
}

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(c fiber.Ctx) error {
	if strings.TrimSpace(c.Get("Idempotency-Key")) == "" {
		return problem.Write(c, fiber.StatusBadRequest, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc cho thao tác tạo.")
	}
	var input createRequest
	if err := c.Bind().Body(&input); err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-request", "Yêu cầu không hợp lệ", "Nội dung phải là JSON hợp lệ.")
	}
	principal, ok := auth.PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ.")
	}
	created, err := h.service.Create(c.Context(), CreateInput{Email: input.Email, DisplayName: input.DisplayName, Role: input.Role, TemporaryPassword: input.TemporaryPassword}, principal, metadata(c))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidInput):
			return problem.Write(c, fiber.StatusUnprocessableEntity, "validation", "Dữ liệu không hợp lệ", "Email, tên, vai trò hoặc mật khẩu tạm thời không hợp lệ.")
		case errors.Is(err, ErrEmailExists):
			return problem.Write(c, fiber.StatusConflict, "email-exists", "Email đã tồn tại", "Một tài khoản nội bộ đang sử dụng email này.")
		default:
			return problem.Write(c, fiber.StatusInternalServerError, "internal", "Không thể tạo tài khoản", "Hệ thống chưa thể lưu tài khoản nội bộ.")
		}
	}
	return c.Status(fiber.StatusCreated).JSON(mapResponse(created))
}

func (h *Handler) List(c fiber.Ctx) error {
	page, _ := strconv.Atoi(c.Query("page", "1"))
	pageSize, _ := strconv.Atoi(c.Query("pageSize", "25"))
	result, err := h.service.List(c.Context(), page, pageSize)
	if err != nil {
		return problem.Write(c, fiber.StatusInternalServerError, "internal", "Không thể tải tài khoản", "Hệ thống chưa thể đọc danh sách tài khoản.")
	}
	items := make([]userResponse, 0, len(result.Items))
	for _, item := range result.Items {
		items = append(items, mapResponse(item))
	}
	return c.JSON(fiber.Map{"items": items, "page": fiber.Map{"number": result.Number, "size": result.Size, "totalItems": result.TotalItems, "totalPages": result.TotalPages}})
}

func mapResponse(user User) userResponse {
	return userResponse{ID: user.ID.String(), Email: user.Email, DisplayName: user.DisplayName, Role: string(user.Role), Status: string(user.Status), RequiresPasswordChange: user.RequiresPasswordChange, LastLoginAt: user.LastLoginAt, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt}
}

func metadata(c fiber.Ctx) auth.ClientMetadata {
	requestID, _ := c.Locals("request_id").(string)
	return auth.ClientMetadata{RequestID: requestID, UserAgent: c.Get(fiber.HeaderUserAgent)}
}
