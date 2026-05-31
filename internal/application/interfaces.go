package application

import (
	"context"
	"time"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
)

type PostRepository interface {
	Create(ctx context.Context, post *domain.Post) error
	GetByID(ctx context.Context, id string) (*domain.Post, error)
	List(ctx context.Context, first int, after *string) ([]*domain.Post, bool, error)
	Update(ctx context.Context, post *domain.Post) error
}

type CommentRepository interface {
	Create(ctx context.Context, comment *domain.Comment) error
	GetByID(ctx context.Context, id string) (*domain.Comment, error)
	ListByPost(ctx context.Context, postID string, parentID *string, fisrt int, after *string) ([]*domain.Comment, bool, error)
	ListChildrenBatch(
		ctx context.Context, parentIDs []string, firsts []int,
		hasAfter []bool, afterCreatedAt []time.Time, afterIDs []string,
	) (map[string][]*domain.Comment, map[string]bool, error)
}

type TransactionManager interface {
	WithinTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}
