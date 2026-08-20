package video

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/google/uuid"

	"github.com/internal/ai-product-marketing-studio/services/api/internal/platform/config"
)

type DemoProvider struct {
	mu    sync.Mutex
	tasks map[string]Task
}

func NewDemoProvider() *DemoProvider { return &DemoProvider{tasks: map[string]Task{}} }

func (p *DemoProvider) Create(_ context.Context, input CreateRequest) (Task, error) {
	if err := validateCreate(input); err != nil {
		return Task{}, err
	}
	p.mu.Lock()
	defer p.mu.Unlock()
	id := "demo-" + uuid.NewString()
	now := time.Now().UTC()
	task := Task{ID: id, Model: input.Model, Status: "queued", Resolution: input.Resolution, AspectRatio: input.AspectRatio, DurationSeconds: input.DurationSeconds, GenerateAudio: input.GenerateAudio, ProviderRequestID: "demo-create-" + id, CreatedAt: &now, UpdatedAt: &now, SafeResponse: map[string]any{"id": id, "status": "queued", "demo": true}}
	p.tasks[id] = task
	return task, nil
}
func (p *DemoProvider) Get(_ context.Context, id string) (Task, error) {
	p.mu.Lock()
	defer p.mu.Unlock()
	task, ok := p.tasks[id]
	if !ok {
		return Task{}, &ProviderError{Category: CategoryNotFound, Code: "demo_not_found", Message: "Demo Seedance task was not found"}
	}
	task.Status, task.ProviderRequestID = "succeeded", "demo-get-"+id
	task.UsageTokens = int64(task.DurationSeconds) * 1000
	task.SafeResponse = map[string]any{"id": id, "status": "succeeded", "demo": true, "outputAvailable": true}
	now := time.Now().UTC()
	task.UpdatedAt = &now
	p.tasks[id] = task
	return task, nil
}
func (p *DemoProvider) Cancel(_ context.Context, id string) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	task, ok := p.tasks[id]
	if !ok {
		return &ProviderError{Category: CategoryNotFound, Code: "demo_not_found", Message: "Demo Seedance task was not found"}
	}
	if task.Status != "queued" {
		return &ProviderError{Category: CategoryInvalid, Code: "not_queued", Message: "Only queued Seedance tasks can be cancelled"}
	}
	task.Status = "cancelled"
	task.SafeResponse = map[string]any{"id": id, "status": "cancelled", "demo": true}
	p.tasks[id] = task
	return nil
}

func NewProvider(demo bool, cfg config.SeedanceConfig) (Provider, error) {
	if demo {
		return NewDemoProvider(), nil
	}
	provider, err := NewBytePlusProvider(cfg)
	if err != nil {
		return nil, fmt.Errorf("configure BytePlus ModelArk: %w", err)
	}
	return provider, nil
}
