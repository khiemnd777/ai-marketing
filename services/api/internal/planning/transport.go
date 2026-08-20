package planning

import (
	"errors"
	"strings"

	"github.com/gofiber/fiber/v3"
	"github.com/google/uuid"

	studioai "github.com/internal/ai-product-marketing-studio/services/api/internal/ai"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/auth"
	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/problem"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func planningScope(c fiber.Ctx, resource string) (uuid.UUID, uuid.UUID, uuid.UUID, uuid.UUID, error) {
	clientID, err := uuid.Parse(c.Params("clientId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	workspaceID, err := uuid.Parse(c.Params("workspaceId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	campaignID, err := uuid.Parse(c.Params("campaignId"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
	}
	resourceID := uuid.Nil
	if resource != "" {
		resourceID, err = uuid.Parse(c.Params(resource))
		if err != nil {
			return uuid.Nil, uuid.Nil, uuid.Nil, uuid.Nil, ErrInvalid
		}
	}
	return clientID, workspaceID, campaignID, resourceID, nil
}
func actor(c fiber.Ctx) auth.Principal { value, _ := auth.PrincipalFrom(c); return value }

func (h *Handler) Estimate(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	item, e := h.service.Estimate(c.Context(), a, b, p, c.Query("operation"))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) StartGeneration(c fiber.Ctx) error {
	key := strings.TrimSpace(c.Get("Idempotency-Key"))
	if key == "" {
		return problem.Write(c, 400, "idempotency-key", "Thiếu khóa chống trùng", "Idempotency-Key là bắt buộc cho tác vụ AI.")
	}
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Operation string `json:"operation"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.StartGeneration(c.Context(), a, b, p, actor(c).UserID, input.Operation, key)
	if e != nil {
		return planningError(c, e)
	}
	return c.Status(202).JSON(item)
}
func (h *Handler) ListJobs(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	items, e := h.service.ListJobs(c.Context(), a, b, p)
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) GetJob(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "jobId")
	if e != nil {
		return planningError(c, e)
	}
	item, e := h.service.GetJob(c.Context(), a, b, p, id)
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}

func (h *Handler) ListConcepts(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	items, e := h.service.ListConcepts(c.Context(), a, b, p)
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) UpdateConcept(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "conceptId")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Payload studioai.ConceptCandidate `json:"payload"`
		Version int64                     `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.UpdateConcept(c.Context(), a, b, p, id, input.Payload, input.Version, actor(c), auth.MetadataFrom(c))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) DecideConcept(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "conceptId")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Action  string `json:"action"`
		Version int64  `json:"version"`
		Notes   string `json:"notes"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.DecideConcept(c.Context(), a, b, p, id, input.Action, input.Version, input.Notes, actor(c), auth.MetadataFrom(c))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}

func (h *Handler) ListContent(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	items, e := h.service.ListContent(c.Context(), a, b, p)
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) UpdateContent(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "contentId")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Content string `json:"content"`
		Version int64  `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.UpdateContent(c.Context(), a, b, p, id, input.Content, input.Version, actor(c))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) ApproveContent(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "contentId")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Version int64  `json:"version"`
		Notes   string `json:"notes"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.ApproveContent(c.Context(), a, b, p, id, input.Version, input.Notes, actor(c))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}

func (h *Handler) GetScript(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	item, e := h.service.GetScript(c.Context(), a, b, p)
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) UpdateScript(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Output  studioai.ScriptOutput `json:"output"`
		Version int64                 `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.UpdateScript(c.Context(), a, b, p, input.Output, input.Version, actor(c))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) ApproveScript(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Version int64  `json:"version"`
		Notes   string `json:"notes"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.ApproveScript(c.Context(), a, b, p, input.Version, input.Notes, actor(c), auth.MetadataFrom(c))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}

func (h *Handler) ListScenes(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	items, e := h.service.ListScenes(c.Context(), a, b, p)
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) UpdateScene(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "sceneId")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Direction studioai.SceneDirection `json:"direction"`
		Version   int64                   `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.UpdateScene(c.Context(), a, b, p, id, input.Direction, input.Version, actor(c))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) ApproveScene(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "sceneId")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Version int64  `json:"version"`
		Notes   string `json:"notes"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	item, e := h.service.ApproveScene(c.Context(), a, b, p, id, input.Version, input.Notes, actor(c))
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(item)
}
func (h *Handler) ReorderScenes(c fiber.Ctx) error {
	a, b, p, _, e := planningScope(c, "")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		SceneIDs []uuid.UUID `json:"sceneIds"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	items, e := h.service.ReorderScenes(c.Context(), a, b, p, input.SceneIDs, actor(c).UserID)
	if e != nil {
		return planningError(c, e)
	}
	return c.JSON(fiber.Map{"items": items})
}
func (h *Handler) DuplicateScene(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "sceneId")
	if e != nil {
		return planningError(c, e)
	}
	item, e := h.service.DuplicateScene(c.Context(), a, b, p, id, actor(c).UserID)
	if e != nil {
		return planningError(c, e)
	}
	return c.Status(201).JSON(item)
}
func (h *Handler) DeleteScene(c fiber.Ctx) error {
	a, b, p, id, e := planningScope(c, "sceneId")
	if e != nil {
		return planningError(c, e)
	}
	var input struct {
		Version int64 `json:"version"`
	}
	if c.Bind().Body(&input) != nil {
		return planningError(c, ErrInvalid)
	}
	if e = h.service.DeleteScene(c.Context(), a, b, p, id, input.Version, actor(c).UserID); e != nil {
		return planningError(c, e)
	}
	return c.SendStatus(204)
}

func planningError(c fiber.Ctx, e error) error {
	switch {
	case errors.Is(e, ErrInvalid):
		return problem.Write(c, 422, "validation", "Dữ liệu planning không hợp lệ", "Kiểm tra schema, Product Truth, thời lượng, nhân vật và phiên bản.")
	case errors.Is(e, ErrNotFound):
		return problem.Write(c, 404, "not-found", "Không tìm thấy dữ liệu planning", "Tài nguyên không thuộc campaign hoặc workspace.")
	case errors.Is(e, ErrConflict):
		return problem.Write(c, 409, "version-conflict", "Dữ liệu đã thay đổi hoặc job đang chạy", "Tải lại dữ liệu trước khi thử lại.")
	case errors.Is(e, ErrPrerequisite):
		return problem.Write(c, 409, "approval-required", "Chưa đủ điều kiện", "Khóa concept, duyệt script và chọn hai nhân vật phù hợp trước khi tiếp tục.")
	case errors.Is(e, ErrLocked):
		return problem.Write(c, 409, "locked", "Tài nguyên đã khóa", "Không thể chỉnh sửa tài nguyên đã duyệt hoặc khóa; tạo phiên bản mới theo workflow.")
	default:
		return problem.Write(c, 500, "internal", "Không thể xử lý AI planning", "Hệ thống chưa thể hoàn tất thao tác.")
	}
}
