package application

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
)

const (
	testMaxCommentLegth    = 2000
	testMaxCommentPageSize = 100
)

type CommentServiceTestSuite struct {
	suite.Suite
	ctrl            *gomock.Controller
	mockPostRepo    *MockPostRepository
	mockCommentRepo *MockCommentRepository
	mockTxManager   *MockTransactionManager
	service         *CommentService
}

func (s *CommentServiceTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockPostRepo = NewMockPostRepository(s.ctrl)
	s.mockCommentRepo = NewMockCommentRepository(s.ctrl)
	s.mockTxManager = NewMockTransactionManager(s.ctrl)
	s.service = NewCommentService(
		s.mockTxManager,
		s.mockPostRepo,
		s.mockCommentRepo,
		testMaxCommentLegth,
		testMaxCommentPageSize,
	)

	s.mockTxManager.EXPECT().
		WithinTransaction(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context) error) error {
			return fn(ctx)
		}).AnyTimes()
}

func (s *CommentServiceTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestCommentServiceSuite(t *testing.T) {
	suite.Run(t, new(CommentServiceTestSuite))
}

func (s *CommentServiceTestSuite) TestCreateComment_SuccessRoot() {
	ctx := context.Background()
	params := domain.CommentCreationParams{
		PostID:   "post-1",
		ParentID: nil,
		AuthorID: "user-1",
		Content:  "Great post!",
	}
	post := &domain.Post{ID: "post-1", CommentsDisabled: false}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(post, nil)
	s.mockCommentRepo.EXPECT().Create(ctx, gomock.AssignableToTypeOf(&domain.Comment{})).Return(nil)

	comment, err := s.service.CreateComment(ctx, params)
	s.Require().NoError(err)
	s.NotEmpty(comment.ID)
	s.Nil(comment.ParentID)
}

func (s *CommentServiceTestSuite) TestCreateComment_ContentTooLong() {
	ctx := context.Background()
	params := domain.CommentCreationParams{
		PostID:   "post-1",
		AuthorID: "user-1",
		Content:  string(make([]byte, testMaxCommentLegth+1)),
	}
	_, err := s.service.CreateComment(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "exceeds maximum length")
}

func (s *CommentServiceTestSuite) TestCreateComment_CommentsDisabled() {
	ctx := context.Background()
	params := domain.CommentCreationParams{
		PostID:   "post-1",
		AuthorID: "user-1",
		Content:  "Hello",
	}
	post := &domain.Post{ID: "post-1", CommentsDisabled: true}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(post, nil)

	_, err := s.service.CreateComment(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "comments disabled")
}

func (s *CommentServiceTestSuite) TestCreateComment_ParentNotFound() {
	ctx := context.Background()
	parentID := "parent-1"
	params := domain.CommentCreationParams{
		PostID:   "post-1",
		ParentID: &parentID,
		AuthorID: "user-1",
		Content:  "Reply",
	}
	post := &domain.Post{ID: "post-1", CommentsDisabled: false}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(post, nil)
	s.mockCommentRepo.EXPECT().GetByID(ctx, parentID).Return(nil, errors.New("not found"))

	_, err := s.service.CreateComment(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "failed to get parent comment")
}

func (s *CommentServiceTestSuite) TestCreateComment_ParentWrongPost() {
	ctx := context.Background()
	parentID := "parent-1"
	params := domain.CommentCreationParams{
		PostID:   "post-1",
		ParentID: &parentID,
		AuthorID: "user-1",
		Content:  "Reply",
	}
	post := &domain.Post{ID: "post-1", CommentsDisabled: false}
	parentComment := &domain.Comment{ID: parentID, PostID: "post-2"}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(post, nil)
	s.mockCommentRepo.EXPECT().GetByID(ctx, parentID).Return(parentComment, nil)

	_, err := s.service.CreateComment(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "parent belongs to another post")
}

func (s *CommentServiceTestSuite) TestCreateComment_RepoCreateError() {
	ctx := context.Background()
	params := domain.CommentCreationParams{
		PostID:   "post-1",
		AuthorID: "user-1",
		Content:  "text",
	}
	post := &domain.Post{ID: "post-1", CommentsDisabled: false}
	s.mockPostRepo.EXPECT().GetByID(ctx, "post-1").Return(post, nil)
	s.mockCommentRepo.EXPECT().Create(ctx, gomock.Any()).Return(errors.New("db error"))

	_, err := s.service.CreateComment(ctx, params)
	s.Require().Error(err)
	s.ErrorContains(err, "failed to create comment")
}

func (s *CommentServiceTestSuite) TestGetComment_Success() {
	ctx := context.Background()
	expected := &domain.Comment{ID: "comment-1", Content: "test"}
	s.mockCommentRepo.EXPECT().GetByID(ctx, "comment-1").Return(expected, nil)

	c, err := s.service.GetComment(ctx, "comment-1")
	s.Require().NoError(err)
	s.Equal(expected, c)
}

func (s *CommentServiceTestSuite) TestGetComment_NotFound() {
	ctx := context.Background()
	s.mockCommentRepo.EXPECT().GetByID(ctx, "comment-1").Return(nil, errors.New("not found"))
	_, err := s.service.GetComment(ctx, "comment-1")
	s.Require().Error(err)
	s.ErrorContains(err, "failed to get comment")
}

func (s *CommentServiceTestSuite) TestListComments_Success() {
	ctx := context.Background()
	comments := []*domain.Comment{
		{ID: "1", Content: "comment1"},
		{ID: "2", Content: "comment2"},
	}

	first := 10
	s.mockCommentRepo.EXPECT().ListByPost(ctx, "post-1", gomock.Any(), first, gomock.Any()).Return(comments, true, nil)

	result, hasNext, err := s.service.ListComments(ctx, "post-1", nil, first, nil)
	s.Require().NoError(err)
	s.True(hasNext)
	s.Equal(comments, result)
}

func (s *CommentServiceTestSuite) TestListComments_ExceedsMaxPageSize() {
	ctx := context.Background()
	_, _, err := s.service.ListComments(ctx, "post-1", nil, testMaxCommentPageSize+1, nil)
	s.Require().Error(err)
	s.ErrorContains(err, "too big page size")
}

func (s *CommentServiceTestSuite) TestListComments_RepoError() {
	ctx := context.Background()

	first := 10
	s.mockCommentRepo.EXPECT().ListByPost(ctx, "post-1", gomock.Any(), first, gomock.Any()).Return(nil, false, errors.New("db error"))

	_, _, err := s.service.ListComments(ctx, "post-1", nil, first, nil)
	s.Require().Error(err)
	s.ErrorContains(err, "failed to list comments")
}

func (s *CommentServiceTestSuite) TestInvalidBatchArguments() {
	ctx := context.Background()
	_, _, err := s.service.ListChildrenBatch(ctx,
		[]string{"p1"},
		[]int{1, 2},
		[]*string{nil},
	)
	s.Require().Error(err)
	s.ErrorContains(err, "invalid batch arguments")
}

func (s *CommentServiceTestSuite) TestSuccessWithoutCursors() {
	ctx := context.Background()
	parentIDs := []string{"p1", "p2"}
	firsts := []int{5, 10}
	afters := []*string{nil, nil}

	expectedComments := map[string][]*domain.Comment{
		"p1": {{ID: "c1", Content: "hello"}},
		"p2": {},
	}
	expectedHasNext := map[string]bool{
		"p1": true,
		"p2": false,
	}

	s.mockCommentRepo.EXPECT().ListChildrenBatch(ctx, parentIDs, firsts, []bool{false, false}, []time.Time{{}, {}}, []string{"", ""}).
		Return(expectedComments, expectedHasNext, nil)

	comments, hasNext, err := s.service.ListChildrenBatch(ctx, parentIDs, firsts, afters)
	s.Require().NoError(err)
	s.Equal(expectedComments, comments)
	s.Equal(expectedHasNext, hasNext)
}

func (s *CommentServiceTestSuite) TestSuccessWithMixedCursors() {
	ctx := context.Background()
	now := time.Date(2025, 5, 31, 12, 0, 0, 0, time.UTC)
	later := now.Add(time.Hour)

	cursor1 := cursorpagination.Encode(now, "c1")
	cursor2 := cursorpagination.Encode(later, "c2")

	parentIDs := []string{"p1", "p2", "p3"}
	firsts := []int{2, 3, 4}
	afters := []*string{&cursor1, nil, &cursor2}

	expectedComments := map[string][]*domain.Comment{
		"p1": {{ID: "c1"}},
		"p3": {{ID: "c2"}, {ID: "c3"}},
	}
	expectedHasNext := map[string]bool{
		"p1": false,
		"p2": false,
		"p3": true,
	}

	s.mockCommentRepo.EXPECT().
		ListChildrenBatch(
			ctx,
			parentIDs,
			firsts,
			[]bool{true, false, true},
			[]time.Time{now, {}, later},
			[]string{"c1", "", "c2"},
		).
		Return(expectedComments, expectedHasNext, nil)

	comments, hasNext, err := s.service.ListChildrenBatch(ctx, parentIDs, firsts, afters)
	s.Require().NoError(err)
	s.Equal(expectedComments, comments)
	s.Equal(expectedHasNext, hasNext)
}

func (s *CommentServiceTestSuite) TestBadCursorDecodeError() {
	ctx := context.Background()
	badCursor := "not-a-valid-cursor"
	afters := []*string{&badCursor}

	_, _, err := s.service.ListChildrenBatch(ctx,
		[]string{"p1"},
		[]int{1},
		afters,
	)
	s.Require().Error(err)
	s.ErrorContains(err, "bad cursor")
}

func (s *CommentServiceTestSuite) TestRepositoryError() {
	ctx := context.Background()
	parentIDs := []string{"p1"}
	firsts := []int{3}
	afters := []*string{nil}

	s.mockCommentRepo.EXPECT().ListChildrenBatch(ctx, parentIDs, firsts, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil, errors.New("db error"))

	_, _, err := s.service.ListChildrenBatch(ctx, parentIDs, firsts, afters)
	s.Require().Error(err)
	s.ErrorContains(err, "failed to list children batch")
}
