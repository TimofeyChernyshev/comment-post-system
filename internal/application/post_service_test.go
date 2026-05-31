package application

import (
	"context"
	"errors"
	"testing"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"
)

const (
	testMaxTitleLenght   = 200
	testMaxContentLength = 10000
	testMaxPostPageSize  = 100
)

type PostServiceTestSuite struct {
	suite.Suite
	ctrl          *gomock.Controller
	mockPostRepo  *MockPostRepository
	mockTxManager *MockTransactionManager
	service       *PostService
}

func (s *PostServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockPostRepo = NewMockPostRepository(s.ctrl)
	s.mockTxManager = NewMockTransactionManager(s.ctrl)
	s.service = NewPostService(
		s.mockTxManager,
		s.mockPostRepo,
		testMaxTitleLenght,
		testMaxContentLength,
		testMaxPostPageSize,
	)

	s.mockTxManager.EXPECT().
		WithinTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).AnyTimes()
}

func (s *PostServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestPostServiceSuite(t *testing.T) {
	suite.Run(t, new(PostServiceTestSuite))
}

func (s *PostServiceTestSuite) TestCreatePost_Success() {
	ctx := context.Background()
	params := domain.PostCreationParams{
		Title:    "Valid Title",
		Content:  "Valid Content",
		AuthorID: "user-1",
	}

	s.mockPostRepo.EXPECT().Create(ctx, gomock.AssignableToTypeOf(&domain.Post{})).Return(nil)

	post, err := s.service.CreatePost(ctx, params)
	s.Require().NoError(err)
	s.NotEmpty(post.ID)
	s.Equal(params.Title, post.Title)
	s.Equal(params.Content, post.Content)
	s.Equal(params.AuthorID, post.AuthorID)
}

func (s *PostServiceTestSuite) TestCreatePost_TitleTooLong() {
	ctx := context.Background()
	longTitle := string(make([]byte, testMaxTitleLenght+1))
	params := domain.PostCreationParams{
		Title:    longTitle,
		Content:  "content",
		AuthorID: "user-1",
	}
	_, err := s.service.CreatePost(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "title exceeds maximum length")
}

func (s *PostServiceTestSuite) TestCreatePost_ContentTooLong() {
	ctx := context.Background()
	longContent := string(make([]byte, testMaxContentLength+1))
	params := domain.PostCreationParams{
		Title:    "title",
		Content:  longContent,
		AuthorID: "user-1",
	}
	_, err := s.service.CreatePost(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "content exceeds maximum length")
}

func (s *PostServiceTestSuite) TestCreatePost_InvalidDomainParams() {
	ctx := context.Background()
	params := domain.PostCreationParams{
		Title:    "",
		Content:  "content",
		AuthorID: "user-1",
	}
	_, err := s.service.CreatePost(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "cannot create post")
}

func (s *PostServiceTestSuite) TestCreatePost_RepoError() {
	ctx := context.Background()
	params := domain.PostCreationParams{
		Title:    "title",
		Content:  "content",
		AuthorID: "user-1",
	}
	s.mockPostRepo.EXPECT().
		Create(ctx, gomock.Any()).
		Return(errors.New("db error"))

	_, err := s.service.CreatePost(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "failed to create post")
}

func (s *PostServiceTestSuite) TestGetPost_Success() {
	ctx := context.Background()
	expectedPost := &domain.Post{ID: "post-1", Title: "Test"}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(expectedPost, nil)

	post, err := s.service.GetPost(ctx, "post-1")
	s.Require().NoError(err)
	s.Equal(expectedPost, post)
}

func (s *PostServiceTestSuite) TestGetPost_NotFound() {
	ctx := context.Background()
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(nil, errors.New("not found"))
	_, err := s.service.GetPost(ctx, "post-1")
	s.Require().Error(err)
	s.ErrorContains(err, "cannot get post")
}

func (s *PostServiceTestSuite) TestListPosts_Success() {
	ctx := context.Background()
	posts := []*domain.Post{
		{ID: "1", Title: "Post 1"},
		{ID: "2", Title: "Post 2"},
	}
	s.mockPostRepo.EXPECT().
		List(ctx, 5, gomock.Any()).
		Return(posts, true, nil)

	result, hasNext, err := s.service.ListPosts(ctx, 5, nil)
	s.Require().NoError(err)
	s.True(hasNext)
	s.Equal(posts, result)
}

func (s *PostServiceTestSuite) TestListPosts_ExceedsMaxPageSize() {
	ctx := context.Background()
	_, _, err := s.service.ListPosts(ctx, testMaxPostPageSize+1, nil)
	s.Require().Error(err)
	s.ErrorContains(err, "too big page size")
}

func (s *PostServiceTestSuite) TestListPosts_RepoError() {
	ctx := context.Background()
	s.mockPostRepo.EXPECT().
		List(ctx, 10, gomock.Any()).
		Return(nil, false, errors.New("db error"))

	_, _, err := s.service.ListPosts(ctx, 10, nil)
	s.Require().Error(err)
	s.ErrorContains(err, "cannot get posts")
}

func (s *PostServiceTestSuite) TestDisableComments_Success() {
	ctx := context.Background()
	post := &domain.Post{
		ID:               "post-1",
		AuthorID:         "user-1",
		CommentsDisabled: false,
	}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(post, nil)
	s.mockPostRepo.EXPECT().Update(ctx, gomock.AssignableToTypeOf(&domain.Post{})).
		DoAndReturn(func(_ context.Context, p *domain.Post) error {
			s.True(p.CommentsDisabled)
			return nil
		})

	err := s.service.DisableComments(ctx, "user-1", "post-1")
	s.Require().NoError(err)
}

func (s *PostServiceTestSuite) TestDisableComments_NotAuthor() {
	ctx := context.Background()
	post := &domain.Post{
		ID:       "post-1",
		AuthorID: "user-1",
	}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(post, nil)

	err := s.service.DisableComments(ctx, "user-2", "post-1")
	s.Require().Error(err)
	s.ErrorContains(err, "actor isn't post creator")
}

func (s *PostServiceTestSuite) TestDisableComments_PostNotFound() {
	ctx := context.Background()
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(nil, errors.New("not found"))
	err := s.service.DisableComments(ctx, "user-1", "post-1")
	s.Require().Error(err)
}

func (s *PostServiceTestSuite) TestDisableComments_UpdateError() {
	ctx := context.Background()
	post := &domain.Post{
		ID:       "post-1",
		AuthorID: "user-1",
	}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(post, nil)
	s.mockPostRepo.EXPECT().
		Update(ctx, gomock.Any()).
		Return(errors.New("update failed"))

	err := s.service.DisableComments(ctx, "user-1", "post-1")
	s.Require().Error(err)
	s.ErrorContains(err, "failed to update post")
}
