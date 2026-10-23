package ticket

import (
	"context"
	"fmt"
	"math/rand/v2"
)

// MockCreator — заглушка для тестов и fallback'а.
type MockCreator struct{}

func (m MockCreator) CreateTicket(ctx context.Context, summary, description string) (string, error) {
	return fmt.Sprintf("TEST-%d", rand.IntN(100000)), nil
}
