package graph

import (
	"context"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
)

type PostService interface {
	CreatePost(ctx context.Context, params domain.PostCreationParams) (*domain.Post, error)
	DisableComments(ctx context.Context, actorID, postID string) error
	GetPost(ctx context.Context, id string) (*domain.Post, error)
	ListPosts(ctx context.Context, first int, after *string) ([]*domain.Post, bool, error)
}

type CommentService interface {
	CreateComment(ctx context.Context, params domain.CommentCreationParams) (*domain.Comment, error)
	ListComments(ctx context.Context, postID string, parentID *string, first int, after *string) ([]*domain.Comment, bool, error)
	GetComment(ctx context.Context, id string) (*domain.Comment, error)
	ListChildrenBatch(ctx context.Context, parentIDs []string, firsts []int, afters []*string) (map[string][]*domain.Comment, map[string]bool, error)
}

type SubscriptionManager interface {
	SubscribeComments(postID string) (<-chan *domain.Comment, func(), error)
	PublishComment(comment *domain.Comment)
	Close()
}
