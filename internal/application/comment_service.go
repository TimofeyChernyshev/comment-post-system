package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"
	"unicode/utf8"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
	"github.com/google/uuid"
)

type CommentService struct {
	txManager   TransactionManager
	postRepo    PostRepository
	commentRepo CommentRepository
	maxLength   int
	maxPageSize int
}

func NewCommentService(txManager TransactionManager, postRepo PostRepository, commentRepo CommentRepository, maxCommentLength, maxPageSize int) *CommentService {
	return &CommentService{
		txManager:   txManager,
		postRepo:    postRepo,
		commentRepo: commentRepo,
		maxLength:   maxCommentLength,
		maxPageSize: maxPageSize,
	}
}

func (s *CommentService) CreateComment(ctx context.Context, params domain.CommentCreationParams) (*domain.Comment, error) {
	if utf8.RuneCountInString(params.Content) > s.maxLength {
		return nil, fmt.Errorf("comment content exceeds maximum length of %d characters", s.maxLength)
	}

	comment, err := domain.NewComment(uuid.NewString(), params)
	if err != nil {
		return nil, fmt.Errorf("cannot create comment: %w", err)
	}

	txManagerErr := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		post, err := s.postRepo.GetByID(ctx, params.PostID)
		if err != nil {
			return fmt.Errorf("failed to get post: %w", err)
		}

		if post.CommentsDisabled {
			return errors.New("comments disabled")
		}

		if params.ParentID != nil {
			parent, err := s.commentRepo.GetByID(ctx, *params.ParentID)
			if err != nil {
				return fmt.Errorf("failed to get parent comment: %w", err)
			}

			if parent.PostID != params.PostID {
				return errors.New("parent belongs to another post")
			}
		}

		if err = s.commentRepo.Create(ctx, comment); err != nil {
			return fmt.Errorf("failed to create comment: %w", err)
		}

		return nil
	})

	if txManagerErr != nil {
		slog.Error("failed to do transaction", "error", txManagerErr)
		return nil, fmt.Errorf("failed to do transaction: %w", txManagerErr)
	}

	return comment, nil
}

func (s *CommentService) ListComments(ctx context.Context, postID string, parentID *string, first int, after *string) ([]*domain.Comment, bool, error) {
	if first > s.maxPageSize {
		return nil, false, fmt.Errorf("too big page size provided: %d > %d", first, s.maxPageSize)
	}

	comments, hasNextPage, err := s.commentRepo.ListByPost(ctx, postID, parentID, first, after)
	if err != nil {
		return nil, false, fmt.Errorf("failed to list comments: %w", err)
	}

	return comments, hasNextPage, nil
}

func (s *CommentService) GetComment(ctx context.Context, id string) (*domain.Comment, error) {
	comment, err := s.commentRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	return comment, nil
}

func (s *CommentService) ListChildrenBatch(ctx context.Context, parentIDs []string, firsts []int, afters []*string) (map[string][]*domain.Comment, map[string]bool, error) {
	if len(parentIDs) != len(firsts) || len(parentIDs) != len(afters) {
		return nil, nil, errors.New("invalid batch arguments")
	}

	decodedAfterIDs := make([]string, len(parentIDs))
	decodedAfterCreated := make([]time.Time, len(parentIDs))
	hasAfter := make([]bool, len(parentIDs))

	for i, a := range afters {
		if a == nil {
			continue
		}

		cur, err := cursorpagination.Decode(*a)
		if err != nil {
			return nil, nil, fmt.Errorf("bad cursor: %w", err)
		}

		decodedAfterIDs[i] = cur.ID
		decodedAfterCreated[i] = cur.CreatedAt
		hasAfter[i] = true
	}

	comments, hasNext, err := s.commentRepo.ListChildrenBatch(ctx, parentIDs, firsts, hasAfter, decodedAfterCreated, decodedAfterIDs)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to list children batch: %w", err)
	}

	return comments, hasNext, nil
}
