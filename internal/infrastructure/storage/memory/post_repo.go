package memory

import (
	"context"
	"errors"
	"sync"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
)

var ErrPostNotFound = errors.New("post not found")

type PostRepository struct {
	mu sync.RWMutex

	posts map[string]*domain.Post
	order []*domain.Post
}

func NewPostRepository() *PostRepository {
	return &PostRepository{
		posts: make(map[string]*domain.Post),
	}
}

func (r *PostRepository) Create(ctx context.Context, post *domain.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.posts[post.ID] = post
	r.order = append(r.order, post)

	return nil
}

func (r *PostRepository) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	post, ok := r.posts[id]
	if !ok {
		return nil, ErrPostNotFound
	}

	return post, nil
}

func (r *PostRepository) Update(ctx context.Context, post *domain.Post) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.posts[post.ID]; !ok {
		return ErrPostNotFound
	}

	r.posts[post.ID] = post

	return nil
}

func (r *PostRepository) List(ctx context.Context, first int, after *string) ([]*domain.Post, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	start := 0

	if after != nil {
		cursor, err := cursorpagination.Decode(*after)
		if err != nil {
			return nil, false, err
		}

		for i, post := range r.order {
			if post.ID == cursor.ID {
				start = i + 1
				break
			}
		}
	}

	end := start + first
	hasNextPage := false

	if end < len(r.order) {
		hasNextPage = true
	} else {
		end = len(r.order)
	}

	result := make([]*domain.Post, end-start)
	copy(result, r.order[start:end])

	return result, hasNextPage, nil
}
