package auth

import (
	"errors"
	"net/mail"
	"net/netip"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/gen/db"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

const principalLocal = "auth_principal"

type Handler struct {
	service       *Service
	secureCookies bool
	sessionTTL    time.Duration
}

type loginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}

type sessionResponse struct {
	ID         string    `json:"id"`
	IPAddress  string    `json:"ipAddress"`
	UserAgent  string    `json:"userAgent"`
	ExpiresAt  time.Time `json:"expiresAt"`
	LastSeenAt time.Time `json:"lastSeenAt"`
	CreatedAt  time.Time `json:"createdAt"`
	Current    bool      `json:"current"`
}

type UserResponse struct {
	ID                     string     `json:"id"`
	Email                  string     `json:"email"`
	DisplayName            string     `json:"displayName"`
	Role                   string     `json:"role"`
	Status                 string     `json:"status"`
	RequiresPasswordChange bool       `json:"requiresPasswordChange"`
	LastLoginAt            *time.Time `json:"lastLoginAt"`
	CreatedAt              time.Time  `json:"createdAt"`
	UpdatedAt              time.Time  `json:"updatedAt"`
	CSRFToken              string     `json:"csrfToken,omitempty"`
}

func NewHandler(service *Service, secureCookies bool, sessionTTL time.Duration) *Handler {
	return &Handler{service: service, secureCookies: secureCookies, sessionTTL: sessionTTL}
}

func (h *Handler) Login(c fiber.Ctx) error {
	var input loginRequest
	if err := c.Bind().Body(&input); err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-request", "Yêu cầu không hợp lệ", "Nội dung đăng nhập phải là JSON hợp lệ.")
	}
	input.Email = strings.ToLower(strings.TrimSpace(input.Email))
	if _, err := mail.ParseAddress(input.Email); err != nil || len(input.Email) > 320 {
		return problem.Write(c, fiber.StatusUnprocessableEntity, "validation", "Dữ liệu không hợp lệ", "Vui lòng kiểm tra email.", problem.FieldError{Field: "email", Code: "email", Message: "Email không hợp lệ"})
	}
	if len(input.Password) < 10 || len(input.Password) > 200 {
		return problem.Write(c, fiber.StatusUnprocessableEntity, "validation", "Dữ liệu không hợp lệ", "Vui lòng kiểm tra mật khẩu.", problem.FieldError{Field: "password", Code: "length", Message: "Mật khẩu phải có từ 10 đến 200 ký tự"})
	}
	result, err := h.service.Login(c.Context(), input.Email, input.Password, metadataFrom(c))
	if err != nil {
		switch {
		case errors.Is(err, ErrInvalidCredentials):
			return problem.Write(c, fiber.StatusUnauthorized, "invalid-credentials", "Đăng nhập thất bại", "Email hoặc mật khẩu không đúng.")
		case errors.Is(err, ErrAccountDisabled):
			return problem.Write(c, fiber.StatusForbidden, "account-disabled", "Tài khoản bị vô hiệu hóa", "Liên hệ quản trị viên để được hỗ trợ.")
		case errors.Is(err, ErrAccountLocked):
			return problem.Write(c, fiber.StatusTooManyRequests, "account-locked", "Tài khoản tạm khóa", "Thử lại sau 15 phút hoặc liên hệ quản trị viên.")
		default:
			return problem.Write(c, fiber.StatusInternalServerError, "internal", "Không thể đăng nhập", "Hệ thống chưa thể hoàn tất đăng nhập.")
		}
	}
	h.setSessionCookies(c, result)
	c.Set("X-CSRF-Token", result.CSRFToken)
	response := principalResponse(result.Principal)
	response.CSRFToken = result.CSRFToken
	return c.JSON(response)
}

func (h *Handler) Me(c fiber.Ctx) error {
	principal, ok := PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
	}
	return c.JSON(principalResponse(principal))
}

func (h *Handler) Logout(c fiber.Ctx) error {
	principal, ok := PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
	}
	if err := h.service.Logout(c.Context(), principal, metadataFrom(c)); err != nil {
		return problem.Write(c, fiber.StatusInternalServerError, "internal", "Không thể đăng xuất", "Hệ thống chưa thể thu hồi phiên đăng nhập.")
	}
	h.clearSessionCookies(c)
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ChangePassword(c fiber.Ctx) error {
	principal, ok := PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
	}
	var input changePasswordRequest
	if err := c.Bind().Body(&input); err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-request", "Yêu cầu không hợp lệ", "Nội dung đổi mật khẩu phải là JSON hợp lệ.")
	}
	if err := h.service.ChangePassword(c.Context(), principal, input.CurrentPassword, input.NewPassword, metadataFrom(c)); err != nil {
		switch {
		case errors.Is(err, ErrInvalidPassword):
			return problem.Write(c, fiber.StatusUnprocessableEntity, "invalid-password", "Mật khẩu hiện tại không đúng", "Kiểm tra lại mật khẩu hiện tại.", problem.FieldError{Field: "currentPassword", Code: "invalid", Message: "Mật khẩu hiện tại không đúng"})
		case errors.Is(err, ErrPasswordPolicy):
			return problem.Write(c, fiber.StatusUnprocessableEntity, "password-policy", "Mật khẩu mới không hợp lệ", "Mật khẩu mới phải có từ 14 đến 200 ký tự và khác mật khẩu hiện tại.", problem.FieldError{Field: "newPassword", Code: "policy", Message: "Dùng mật khẩu mới từ 14 đến 200 ký tự"})
		case errors.Is(err, ErrConflict):
			return problem.Write(c, fiber.StatusConflict, "stale-user", "Tài khoản đã thay đổi", "Tải lại trang và thử lại.")
		default:
			return problem.Write(c, fiber.StatusInternalServerError, "internal", "Không thể đổi mật khẩu", "Hệ thống chưa thể cập nhật mật khẩu.")
		}
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) ListSessions(c fiber.Ctx) error {
	principal, ok := PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
	}
	items, err := h.service.ListSessions(c.Context(), principal)
	if err != nil {
		return problem.Write(c, fiber.StatusInternalServerError, "internal", "Không thể tải phiên", "Hệ thống chưa thể đọc danh sách phiên đăng nhập.")
	}
	response := make([]sessionResponse, 0, len(items))
	for _, item := range items {
		response = append(response, sessionResponse{ID: item.ID.String(), IPAddress: item.IPAddress, UserAgent: item.UserAgent, ExpiresAt: item.ExpiresAt, LastSeenAt: item.LastSeenAt, CreatedAt: item.CreatedAt, Current: item.Current})
	}
	return c.JSON(fiber.Map{"items": response})
}

func (h *Handler) RevokeSession(c fiber.Ctx) error {
	principal, ok := PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
	}
	sessionID, err := uuid.Parse(c.Params("sessionId"))
	if err != nil {
		return problem.Write(c, fiber.StatusBadRequest, "invalid-id", "ID phiên không hợp lệ", "Session ID phải là UUID hợp lệ.")
	}
	current, err := h.service.RevokeOwnSession(c.Context(), principal, sessionID, metadataFrom(c))
	if errors.Is(err, ErrSessionNotFound) {
		return problem.Write(c, fiber.StatusNotFound, "session-not-found", "Không tìm thấy phiên", "Phiên không tồn tại, đã hết hạn hoặc đã bị thu hồi.")
	}
	if err != nil {
		return problem.Write(c, fiber.StatusInternalServerError, "internal", "Không thể thu hồi phiên", "Hệ thống chưa thể thu hồi phiên đăng nhập.")
	}
	if current {
		h.clearSessionCookies(c)
	}
	return c.SendStatus(fiber.StatusNoContent)
}

func (h *Handler) setSessionCookies(c fiber.Ctx, result LoginResult) {
	maxAge := int(h.sessionTTL.Seconds())
	c.Cookie(&fiber.Cookie{
		Name: "studio_session", Value: result.SessionToken, Path: "/", HTTPOnly: true,
		Secure: h.secureCookies, SameSite: fiber.CookieSameSiteStrictMode, MaxAge: maxAge, Expires: result.ExpiresAt,
	})
	c.Cookie(&fiber.Cookie{
		Name: "studio_csrf", Value: result.CSRFToken, Path: "/", HTTPOnly: false,
		Secure: h.secureCookies, SameSite: fiber.CookieSameSiteStrictMode, MaxAge: maxAge, Expires: result.ExpiresAt,
	})
}

func (h *Handler) clearSessionCookies(c fiber.Ctx) {
	expired := time.Unix(0, 0).UTC()
	for _, name := range []string{"studio_session", "studio_csrf"} {
		c.Cookie(&fiber.Cookie{Name: name, Value: "", Path: "/", HTTPOnly: name == "studio_session", Secure: h.secureCookies, SameSite: fiber.CookieSameSiteStrictMode, MaxAge: -1, Expires: expired})
	}
}

func principalResponse(principal Principal) UserResponse {
	return UserResponse{
		ID: principal.UserID.String(), Email: principal.Email, DisplayName: principal.DisplayName,
		Role: string(principal.Role), Status: "ACTIVE", RequiresPasswordChange: principal.RequiresPasswordChange,
		LastLoginAt: principal.LastLoginAt, CreatedAt: principal.CreatedAt, UpdatedAt: principal.UpdatedAt,
	}
}

func Authenticate(service *Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		principal, err := service.Authenticate(c.Context(), c.Cookies("studio_session"))
		if err != nil {
			if errors.Is(err, ErrUnauthenticated) {
				return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
			}
			return problem.Write(c, fiber.StatusInternalServerError, "internal", "Không thể xác thực", "Hệ thống chưa thể xác thực phiên đăng nhập.")
		}
		c.Locals(principalLocal, principal)
		return c.Next()
	}
}

func RequireCSRF(service *Service) fiber.Handler {
	return func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead || c.Method() == fiber.MethodOptions {
			return c.Next()
		}
		principal, ok := PrincipalFrom(c)
		if !ok {
			return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
		}
		header := c.Get("X-CSRF-Token")
		cookie := c.Cookies("studio_csrf")
		if header == "" || cookie == "" || len(header) != len(cookie) || subtleEqual(header, cookie) == false || service.ValidateCSRF(principal, header) != nil {
			return problem.Write(c, fiber.StatusForbidden, "csrf", "Yêu cầu bị từ chối", "CSRF token không hợp lệ.")
		}
		return c.Next()
	}
}

func RequireRole(roles ...db.InternalUserRole) fiber.Handler {
	allowed := make(map[db.InternalUserRole]struct{}, len(roles))
	for _, role := range roles {
		allowed[role] = struct{}{}
	}
	return func(c fiber.Ctx) error {
		principal, ok := PrincipalFrom(c)
		if !ok {
			return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
		}
		if _, ok := allowed[principal.Role]; !ok {
			return problem.Write(c, fiber.StatusForbidden, "forbidden", "Không đủ quyền", "Vai trò hiện tại không được phép thực hiện thao tác này.")
		}
		return c.Next()
	}
}

func RequirePasswordChangeCleared(c fiber.Ctx) error {
	principal, ok := PrincipalFrom(c)
	if !ok {
		return problem.Write(c, fiber.StatusUnauthorized, "unauthenticated", "Cần đăng nhập", "Phiên đăng nhập không hợp lệ hoặc đã hết hạn.")
	}
	if principal.RequiresPasswordChange {
		return problem.Write(c, fiber.StatusPreconditionRequired, "password-change-required", "Cần đổi mật khẩu", "Đổi mật khẩu tạm thời trước khi tiếp tục sử dụng hệ thống.")
	}
	return c.Next()
}

func PrincipalFrom(c fiber.Ctx) (Principal, bool) {
	principal, ok := c.Locals(principalLocal).(Principal)
	return principal, ok
}

func metadataFrom(c fiber.Ctx) ClientMetadata {
	requestID, _ := c.Locals("request_id").(string)
	var address *netip.Addr
	if parsed, err := netip.ParseAddr(strings.TrimSpace(c.IP())); err == nil {
		address = &parsed
	}
	return ClientMetadata{RequestID: requestID, IPAddress: address, UserAgent: c.Get(fiber.HeaderUserAgent)}
}

func MetadataFrom(c fiber.Ctx) ClientMetadata { return metadataFrom(c) }

func subtleEqual(left, right string) bool {
	if len(left) != len(right) {
		return false
	}
	var difference byte
	for index := range len(left) {
		difference |= left[index] ^ right[index]
	}
	return difference == 0
}

type loginRateEntry struct {
	window time.Time
	count  int
}

type LoginRateLimiter struct {
	mu      sync.Mutex
	entries map[string]loginRateEntry
	limit   int
	window  time.Duration
}

func NewLoginRateLimiter(limit int, window time.Duration) *LoginRateLimiter {
	return &LoginRateLimiter{entries: make(map[string]loginRateEntry), limit: limit, window: window}
}

func (limiter *LoginRateLimiter) Middleware(c fiber.Ctx) error {
	now := time.Now().UTC()
	key := c.IP()
	limiter.mu.Lock()
	entry := limiter.entries[key]
	if entry.window.IsZero() || now.Sub(entry.window) >= limiter.window {
		entry = loginRateEntry{window: now, count: 0}
	}
	entry.count++
	limiter.entries[key] = entry
	allowed := entry.count <= limiter.limit
	remaining := max(limiter.limit-entry.count, 0)
	limiter.mu.Unlock()
	c.Set("X-RateLimit-Limit", strconv.Itoa(limiter.limit))
	c.Set("X-RateLimit-Remaining", strconv.Itoa(remaining))
	if !allowed {
		return problem.Write(c, fiber.StatusTooManyRequests, "rate-limit", "Quá nhiều yêu cầu", "Vui lòng thử đăng nhập lại sau.")
	}
	return c.Next()
}
