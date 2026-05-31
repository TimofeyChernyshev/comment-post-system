package application

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"unicode/utf8"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	"github.com/google/uuid"
)

type PostService struct {
	txManager        TransactionManager
	postRepo         PostRepository
	maxTitleLength   int
	maxContentLength int
	maxPageSize      int
}

func NewPostService(txManager TransactionManager, postRepo PostRepository, maxTitleLength, maxContentLength, maxPageSize int) *PostService {
	return &PostService{
		txManager:        txManager,
		postRepo:         postRepo,
		maxTitleLength:   maxTitleLength,
		maxContentLength: maxContentLength,
		maxPageSize:      maxPageSize,
	}
}

func (s *PostService) CreatePost(ctx context.Context, params domain.PostCreationParams) (*domain.Post, error) {
	if utf8.RuneCountInString(params.Title) > s.maxTitleLength {
		return nil, fmt.Errorf("title exceeds maximum length of %d characters", s.maxTitleLength)
	}
	if utf8.RuneCountInString(params.Content) > s.maxContentLength {
		return nil, fmt.Errorf("content exceeds maximum length of %d characters", s.maxContentLength)
	}

	post, err := domain.NewPost(uuid.NewString(), params)
	if err != nil {
		return nil, fmt.Errorf("cannot create post: %w", err)
	}

	if err = s.postRepo.Create(ctx, post); err != nil {
		return nil, fmt.Errorf("failed to create post: %w", err)
	}

	return post, nil
}

func (s *PostService) DisableComments(ctx context.Context, actorID, postID string) error {
	txManagerErr := s.txManager.WithinTransaction(ctx, func(ctx context.Context) error {
		post, err := s.postRepo.GetByID(ctx, postID)
		if err != nil {
			return fmt.Errorf("failed to get post: %w", err)
		}

		if post.AuthorID != actorID {
			return errors.New("actor isn't post creator")
		}

		post.DisableComments()

		if err = s.postRepo.Update(ctx, post); err != nil {
			return fmt.Errorf("failed to update post: %w", err)
		}

		return nil
	})

	if txManagerErr != nil {
		slog.Error("failed to do transaction", "error", txManagerErr)
		return fmt.Errorf("failed to do transaction: %w", txManagerErr)
	}

	return nil
}

func (s *PostService) GetPost(ctx context.Context, id string) (*domain.Post, error) {
	post, err := s.postRepo.GetByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("cannot get post: %w", err)
	}

	return post, nil
}

func (s *PostService) ListPosts(ctx context.Context, first int, after *string) ([]*domain.Post, bool, error) {
	if first > s.maxPageSize {
		return nil, false, fmt.Errorf("too big page size provided: %d > %d", first, s.maxPageSize)
	}

	posts, hasNextPage, err := s.postRepo.List(ctx, first, after)
	if err != nil {
		return nil, false, fmt.Errorf("cannot get posts: %w", err)
	}

	return posts, hasNextPage, nil
}
