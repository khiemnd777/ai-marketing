package jobs

import "testing"

func TestRequiredQueuesAreUnique(t *testing.T) {
	seen := make(map[string]struct{}, len(RequiredQueues))
	for _, queue := range RequiredQueues {
		if queue == "" {
			t.Fatal("queue name must not be empty")
		}
		if _, exists := seen[queue]; exists {
			t.Fatalf("duplicate queue %q", queue)
		}
		seen[queue] = struct{}{}
	}
}
