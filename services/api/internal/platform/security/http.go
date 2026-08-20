package security

import (
	"net/url"
	"strings"

	"github.com/gofiber/fiber/v3"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

func Headers(c fiber.Ctx) error {
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("X-Frame-Options", "DENY")
	c.Set("Referrer-Policy", "strict-origin-when-cross-origin")
	c.Set("Permissions-Policy", "camera=(), microphone=(), geolocation=()")
	c.Set("Cache-Control", "no-store")
	return c.Next()
}

func OriginGuard(appURL string) fiber.Handler {
	parsed, _ := url.Parse(appURL)
	expectedOrigin := parsed.Scheme + "://" + parsed.Host
	return func(c fiber.Ctx) error {
		if c.Method() == fiber.MethodGet || c.Method() == fiber.MethodHead || c.Method() == fiber.MethodOptions {
			return c.Next()
		}
		if site := strings.ToLower(strings.TrimSpace(c.Get("Sec-Fetch-Site"))); site == "cross-site" {
			return problem.Write(c, fiber.StatusForbidden, "cross-site", "Yêu cầu bị từ chối", "Không chấp nhận yêu cầu thay đổi dữ liệu từ trang khác.")
		}
		if origin := strings.TrimSpace(c.Get(fiber.HeaderOrigin)); origin != "" && origin != expectedOrigin {
			return problem.Write(c, fiber.StatusForbidden, "origin", "Yêu cầu bị từ chối", "Origin không nằm trong phạm vi ứng dụng nội bộ.")
		}
		return c.Next()
	}
}
