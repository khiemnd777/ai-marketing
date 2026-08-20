package internalusers

import (
	"errors"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

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

type updateRequest struct {
	Email       string              `json:"email"`
	DisplayName string              `json:"displayName"`
	Role        db.InternalUserRole `json:"role"`
	Version     int64               `json:"version"`
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
	Version                int64      `json:"version"`
}

type resetPasswordRequest struct {
	TemporaryPassword string `json:"temporaryPassword"`
	Version           int64  `json:"version"`
}

type statusRequest struct {
	Status  db.InternalUserStatus `json:"status"`
	Version int64                 `json:"version"`
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
	created, err := h.service.Create(c.Context(), CreateInput{Email: input.Email, DisplayName: input.DisplayName, Role: input.Role, TemporaryPassword: input.TemporaryPassword}, principal, auth.MetadataFrom(c))
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

func (h *Handler) Update(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-id", "ID tài khoản không hợp lệ", "User ID phải là UUID hợp lệ.")
	}
	var input updateRequest
	if err = c.Bind().Body(&input); err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-request", "Yêu cầu không hợp lệ", "Nội dung cập nhật tài khoản phải là JSON hợp lệ.")
	}
	principal, ok := auth.PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ.")
	}
	updated, err := h.service.Update(c.Context(), userID, UpdateInput{Email: input.Email, DisplayName: input.DisplayName, Role: input.Role, Version: input.Version}, principal, auth.MetadataFrom(c))
	if err != nil {
		return writeLifecycleError(c, err, "Không thể cập nhật tài khoản")
	}
	return c.JSON(mapResponse(updated))
}

func (h *Handler) ResetPassword(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-id", "ID tài khoản không hợp lệ", "User ID phải là UUID hợp lệ.")
	}
	var input resetPasswordRequest
	if err = c.Bind().Body(&input); err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-request", "Yêu cầu không hợp lệ", "Nội dung reset mật khẩu phải là JSON hợp lệ.")
	}
	principal, ok := auth.PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ.")
	}
	updated, err := h.service.ResetPassword(c.Context(), userID, input.Version, input.TemporaryPassword, principal, auth.MetadataFrom(c))
	if err != nil {
		return writeLifecycleError(c, err, "Không thể reset mật khẩu")
	}
	return c.JSON(mapResponse(updated))
}

func (h *Handler) SetStatus(c fiber.Ctx) error {
	userID, err := uuid.Parse(c.Params("userId"))
	if err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-id", "ID tài khoản không hợp lệ", "User ID phải là UUID hợp lệ.")
	}
	var input statusRequest
	if err = c.Bind().Body(&input); err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-request", "Yêu cầu không hợp lệ", "Nội dung trạng thái phải là JSON hợp lệ.")
	}
	principal, ok := auth.PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ.")
	}
	updated, err := h.service.SetStatus(c.Context(), userID, input.Status, input.Version, principal, auth.MetadataFrom(c))
	if err != nil {
		return writeLifecycleError(c, err, "Không thể đổi trạng thái tài khoản")
	}
	return c.JSON(mapResponse(updated))
}

func writeLifecycleError(c fiber.Ctx, err error, fallback string) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return problem.Write(c, fiber.StatusUnprocessableEntity, "validation", "Dữ liệu không hợp lệ", "Kiểm tra mật khẩu, trạng thái và phiên bản tài khoản.")
	case errors.Is(err, ErrNotFound):
		return problem.Write(c, fiber.StatusNotFound, "user-not-found", "Không tìm thấy tài khoản", "Tài khoản không tồn tại.")
	case errors.Is(err, ErrConflict):
		return problem.Write(c, fiber.StatusConflict, "stale-user", "Tài khoản đã thay đổi", "Tải lại danh sách và thử lại.")
	case errors.Is(err, ErrEmailExists):
		return problem.Write(c, fiber.StatusConflict, "email-exists", "Email đã tồn tại", "Một tài khoản nội bộ đang sử dụng email này.")
	case errors.Is(err, ErrSelfDisable):
		return problem.Write(c, fiber.StatusConflict, "self-disable", "Không thể vô hiệu hóa chính mình", "Nhờ một quản trị viên khác thực hiện thao tác này.")
	case errors.Is(err, ErrLastAdmin):
		return problem.Write(c, fiber.StatusConflict, "last-admin", "Cần ít nhất một quản trị viên", "Không thể vô hiệu hóa quản trị viên đang hoạt động cuối cùng.")
	default:
		return problem.Write(c, fiber.StatusInternalServerError, "internal", fallback, "Hệ thống chưa thể hoàn tất thao tác.")
	}
}

func mapResponse(user User) userResponse {
	return userResponse{ID: user.ID.String(), Email: user.Email, DisplayName: user.DisplayName, Role: string(user.Role), Status: string(user.Status), RequiresPasswordChange: user.RequiresPasswordChange, LastLoginAt: user.LastLoginAt, CreatedAt: user.CreatedAt, UpdatedAt: user.UpdatedAt, Version: user.Version}
}
