package subscription

import (
	"errors"
	"sync"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
)

type SubscriptionManager struct {
	mu sync.RWMutex

	closed bool

	subscriberBufferSize int

	subscribers map[string]map[chan *domain.Comment]struct{}
}

func NewSubscriptionManager(subscriberBufferSize int) *SubscriptionManager {
	return &SubscriptionManager{
		subscriberBufferSize: subscriberBufferSize,
		subscribers:          make(map[string]map[chan *domain.Comment]struct{}),
	}
}

func (m *SubscriptionManager) SubscribeComments(postID string) (<-chan *domain.Comment, func(), error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return nil, nil, errors.New("subscription manager closed")
	}

	ch := make(chan *domain.Comment, m.subscriberBufferSize)

	if _, ok := m.subscribers[postID]; !ok {
		m.subscribers[postID] = make(map[chan *domain.Comment]struct{})
	}

	m.subscribers[postID][ch] = struct{}{}

	unsubscribe := func() {
		m.mu.Lock()
		defer m.mu.Unlock()

		if _, ok := m.subscribers[postID]; !ok {
			return
		}

		delete(m.subscribers[postID], ch)

		if len(m.subscribers[postID]) == 0 {
			delete(m.subscribers, postID)
		}
	}

	return ch, unsubscribe, nil
}

func (m *SubscriptionManager) PublishComment(comment *domain.Comment) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return
	}

	subs := m.subscribers[comment.PostID]

	for ch := range subs {
		select {
		case ch <- comment:
		default:
		}
	}
}

func (m *SubscriptionManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return
	}

	m.closed = true

	for _, subscribers := range m.subscribers {
		for ch := range subscribers {
			close(ch)
		}
	}

	m.subscribers = map[string]map[chan *domain.Comment]struct{}{}
}
