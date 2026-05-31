package memory

import (
	"context"
	"sync"
)

type TransactionManager struct {
	mu sync.Mutex
}

func NewTransactionManager() *TransactionManager {
	return &TransactionManager{
		mu: sync.Mutex{},
	}
}

func (m *TransactionManager) WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	return fn(ctx)
}
