package problem

import (
	"fmt"

	"github.com/gofiber/fiber/v3"
)

type FieldError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}

type Details struct {
	Type      string       `json:"type"`
	Title     string       `json:"title"`
	Status    int          `json:"status"`
	Detail    string       `json:"detail"`
	Instance  string       `json:"instance,omitempty"`
	RequestID string       `json:"requestId"`
	Errors    []FieldError `json:"errors,omitempty"`
}

func Write(c fiber.Ctx, status int, problemType, title, detail string, fieldErrors ...FieldError) error {
	requestID, _ := c.Locals("request_id").(string)
	c.Set(fiber.HeaderContentType, "application/problem+json")
	return c.Status(status).JSON(Details{
		Type:      fmt.Sprintf("https://studio.internal/problems/%s", problemType),
		Title:     title,
		Status:    status,
		Detail:    detail,
		Instance:  c.OriginalURL(),
		RequestID: requestID,
		Errors:    fieldErrors,
	})
}
