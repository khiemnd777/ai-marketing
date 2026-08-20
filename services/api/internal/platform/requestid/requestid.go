package requestid

import (
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"
)

const Header = "X-Request-ID"

func Middleware(c fiber.Ctx) error {
	id := strings.TrimSpace(c.Get(Header))
	if len(id) == 0 || len(id) > 200 {
		id = uuid.NewString()
	}
	c.Locals("request_id", id)
	c.Set(Header, id)
	return c.Next()
}
