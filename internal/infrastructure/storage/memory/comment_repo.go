package memory

import (
	"context"
	"errors"
	"sync"
	"time"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
)

var ErrCommentNotFound = errors.New("comment not found")

type CommentRepository struct {
	mu sync.RWMutex

	comments     map[string]*domain.Comment
	postComments map[string][]*domain.Comment
	children     map[string][]*domain.Comment
}

func NewCommentRepository() *CommentRepository {
	return &CommentRepository{
		comments:     make(map[string]*domain.Comment),
		postComments: make(map[string][]*domain.Comment),
		children:     make(map[string][]*domain.Comment),
	}
}

func (r *CommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.comments[comment.ID] = comment

	if comment.ParentID == nil {
		r.postComments[comment.PostID] = append(r.postComments[comment.PostID], comment)
	} else {
		parentID := *comment.ParentID
		r.children[parentID] = append(r.children[parentID], comment)
	}

	return nil
}

func (r *CommentRepository) GetByID(ctx context.Context, id string) (*domain.Comment, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	comment, ok := r.comments[id]
	if !ok {
		return nil, ErrCommentNotFound
	}

	return comment, nil
}

func (r *CommentRepository) ListByPost(ctx context.Context, postID string, parentID *string, first int, after *string) ([]*domain.Comment, bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var source []*domain.Comment

	if parentID == nil {
		source = r.postComments[postID]
	} else {
		source = r.children[*parentID]
	}

	return paginateComments(source, first, after)
}

func (r *CommentRepository) ListChildrenBatch(
	ctx context.Context, parentIDs []string, firsts []int,
	hasAfter []bool, afterCreatedAt []time.Time, afterIDs []string,
) (map[string][]*domain.Comment, map[string]bool, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string][]*domain.Comment, len(parentIDs))
	hasNext := make(map[string]bool, len(parentIDs))

	for i, parentID := range parentIDs {
		comments := r.children[parentID]

		start := 0

		if hasAfter[i] {
			for j, c := range comments {
				if c.ID == afterIDs[i] {
					start = j + 1
					break
				}
			}
		}

		limit := firsts[i]

		end := start + limit + 1

		if end > len(comments) {
			end = len(comments)
		}

		page := comments[start:end]

		if len(page) > limit {
			hasNext[parentID] = true
			page = page[:limit]
		}

		result[parentID] = page
	}

	return result, hasNext, nil
}

func paginateComments(comments []*domain.Comment, first int, after *string) ([]*domain.Comment, bool, error) {
	start := 0

	if after != nil {
		cursor, err := cursorpagination.Decode(*after)
		if err != nil {
			return nil, false, err
		}

		for i, comment := range comments {
			if comment.ID == cursor.ID {
				start = i + 1
				break
			}
		}
	}

	end := start + first

	hasNextPage := false

	if end < len(comments) {
		hasNextPage = true
	} else {
		end = len(comments)
	}

	result := make([]*domain.Comment, end-start)
	copy(result, comments[start:end])

	return result, hasNextPage, nil
}
