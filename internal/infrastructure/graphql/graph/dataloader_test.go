package graph

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/golang/mock/gomock"
	"github.com/stretchr/testify/suite"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
)

type DataLoaderTestSuite struct {
	suite.Suite
	ctrl    *gomock.Controller
	mockSvc *MockCommentService
	loader  *Loader
}

func (s *DataLoaderTestSuite) SetupTest() {
	s.ctrl = gomock.NewController(s.T())
	s.mockSvc = NewMockCommentService(s.ctrl)
	s.loader = NewChildrenLoader(s.mockSvc, 10, 5*time.Millisecond)
}

func (s *DataLoaderTestSuite) TearDownTest() {
	s.ctrl.Finish()
}

func TestDataLoaderSuite(t *testing.T) {
	suite.Run(t, new(DataLoaderTestSuite))
}

func (s *DataLoaderTestSuite) TestBatchSuccess() {
	ctx := context.Background()
	parent1 := "p1"
	parent2 := "p2"

	keys := []ChildrenKey{
		{ParentID: parent1, First: 2},
		{ParentID: parent2, First: 1, After: nil},
	}

	expectedItems := map[string][]*domain.Comment{
		parent1: {{ID: "c1"}, {ID: "c2"}},
		parent2: {{ID: "c3"}},
	}
	expectedHasNext := map[string]bool{
		parent1: true,
		parent2: false,
	}

	s.mockSvc.EXPECT().
		ListChildrenBatch(ctx, []string{parent1, parent2}, []int{2, 1}, []*string{nil, nil}).
		Return(expectedItems, expectedHasNext, nil).
		Times(1)

	thunk1 := s.loader.commentChildren.Load(ctx, keys[0])
	thunk2 := s.loader.commentChildren.Load(ctx, keys[1])

	r1, err := thunk1()
	s.Require().NoError(err)
	s.Equal(expectedItems[parent1], r1.Items)
	s.True(r1.HasNextPage)

	r2, err := thunk2()
	s.Require().NoError(err)
	s.Equal(expectedItems[parent2], r2.Items)
	s.False(r2.HasNextPage)
}

func (s *DataLoaderTestSuite) TestBatchServiceError() {
	ctx := context.Background()
	keys := []ChildrenKey{
		{ParentID: "p1", First: 1},
	}

	s.mockSvc.EXPECT().
		ListChildrenBatch(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(nil, nil, errors.New("db error")).
		Times(1)

	thunk := s.loader.commentChildren.Load(ctx, keys[0])
	_, err := thunk()
	s.Error(err)
	s.Contains(err.Error(), "db error")
}

func (s *DataLoaderTestSuite) TestBatchPartialResults() {
	ctx := context.Background()
	parent1 := "p1"
	parent2 := "p2"

	keys := []ChildrenKey{
		{ParentID: parent1, First: 1},
		{ParentID: parent2, First: 2},
	}

	expectedItems := map[string][]*domain.Comment{
		parent1: {},
		parent2: {{ID: "c4"}},
	}
	expectedHasNext := map[string]bool{
		parent1: false,
		parent2: false,
	}

	s.mockSvc.EXPECT().
		ListChildrenBatch(ctx, []string{parent1, parent2}, []int{1, 2}, gomock.Any()).
		Return(expectedItems, expectedHasNext, nil).
		Times(1)

	thunk1 := s.loader.commentChildren.Load(ctx, keys[0])
	thunk2 := s.loader.commentChildren.Load(ctx, keys[1])

	r1, err := thunk1()
	s.Require().NoError(err)
	s.Empty(r1.Items)
	s.False(r1.HasNextPage)

	r2, err := thunk2()
	s.Require().NoError(err)
	s.Len(r2.Items, 1)
	s.False(r2.HasNextPage)
}

func (s *DataLoaderTestSuite) TestCacheKeyReuse() {
	ctx := context.Background()
	key := ChildrenKey{ParentID: "p1", First: 1}

	s.mockSvc.EXPECT().
		ListChildrenBatch(ctx, gomock.Any(), gomock.Any(), gomock.Any()).
		Return(map[string][]*domain.Comment{"p1": {{ID: "c1"}}}, map[string]bool{"p1": false}, nil).
		Times(1)

	for i := 0; i < 3; i++ {
		thunk := s.loader.commentChildren.Load(ctx, key)
		result, err := thunk()
		s.Require().NoError(err)
		s.Len(result.Items, 1)
	}
}
