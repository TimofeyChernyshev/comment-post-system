package memory

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/suite"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
)

type MemoryRepoTestSuite struct {
	suite.Suite
	ctx   context.Context
	post  *PostRepository
	cmnt  *CommentRepository
	txMgr *TransactionManager
}

func (s *MemoryRepoTestSuite) SetupTest() {
	s.ctx = context.Background()
	s.post = NewPostRepository()
	s.cmnt = NewCommentRepository()
	s.txMgr = NewTransactionManager()
}

func TestMemoryRepoTestSuite(t *testing.T) {
	suite.Run(t, new(MemoryRepoTestSuite))
}

func (s *MemoryRepoTestSuite) TestPost_CreateAndGetByID() {
	post := &domain.Post{ID: "post-1", Title: "Test", Content: "c", AuthorID: "a"}
	err := s.post.Create(s.ctx, post)
	s.Require().NoError(err)

	retrieved, err := s.post.GetByID(s.ctx, "post-1")
	s.Require().NoError(err)
	s.Equal(post, retrieved)
}

func (s *MemoryRepoTestSuite) TestPost_GetByID_NotFound() {
	_, err := s.post.GetByID(s.ctx, "missing")
	s.ErrorIs(err, ErrPostNotFound)
}

func (s *MemoryRepoTestSuite) TestPost_Update() {
	post := &domain.Post{ID: "p1", Title: "original", Content: "c", AuthorID: "a"}
	s.Require().NoError(s.post.Create(s.ctx, post))

	post.Title = "new title"
	err := s.post.Update(s.ctx, post)
	s.Require().NoError(err)

	updated, _ := s.post.GetByID(s.ctx, "p1")
	s.Equal("new title", updated.Title)
}

func (s *MemoryRepoTestSuite) TestPost_Update_NotFound() {
	err := s.post.Update(s.ctx, &domain.Post{ID: "ghost"})
	s.ErrorIs(err, ErrPostNotFound)
}

func (s *MemoryRepoTestSuite) TestPost_List_Pagination() {
	now := time.Now().UTC()
	for i := 1; i <= 5; i++ {
		post := &domain.Post{
			ID:        fmt.Sprintf("post-%d", i),
			Title:     fmt.Sprintf("Post %d", i),
			Content:   "c",
			AuthorID:  "a",
			CreatedAt: now.Add(time.Duration(6-i) * time.Minute),
		}
		s.Require().NoError(s.post.Create(s.ctx, post))
	}

	posts, hasNext, err := s.post.List(s.ctx, 3, nil)
	s.Require().NoError(err)
	s.Len(posts, 3)
	s.True(hasNext)
	s.Equal("post-1", posts[0].ID)
}

func (s *MemoryRepoTestSuite) TestPost_List_WithCursor() {
	now := time.Now().UTC()
	for i := 1; i <= 3; i++ {
		post := &domain.Post{
			ID:        fmt.Sprintf("post-%d", i),
			Title:     fmt.Sprintf("Post %d", i),
			Content:   "c",
			AuthorID:  "a",
			CreatedAt: now.Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.post.Create(s.ctx, post))
	}
	page1, hasNext, err := s.post.List(s.ctx, 1, nil)
	s.Require().NoError(err)
	s.True(hasNext)
	cursor := cursorpagination.Encode(page1[0].CreatedAt, page1[0].ID)
	page2, hasNext, err := s.post.List(s.ctx, 1, &cursor)
	s.Require().NoError(err)
	s.True(hasNext)
	s.Equal("post-2", page2[0].ID)
}

func (s *MemoryRepoTestSuite) TestComment_CreateAndGetByID() {
	comment := &domain.Comment{ID: "c1", PostID: "p1", AuthorID: "a", Content: "Hi"}
	err := s.cmnt.Create(s.ctx, comment)
	s.Require().NoError(err)

	ret, err := s.cmnt.GetByID(s.ctx, "c1")
	s.Require().NoError(err)
	s.Equal(comment, ret)
}

func (s *MemoryRepoTestSuite) TestComment_GetByID_NotFound() {
	_, err := s.cmnt.GetByID(s.ctx, "no")
	s.ErrorIs(err, ErrCommentNotFound)
}

func (s *MemoryRepoTestSuite) TestComment_ListByPost_Roots() {
	postID := "post-1"
	for i := 1; i <= 4; i++ {
		c := &domain.Comment{
			ID:        fmt.Sprintf("c%d", i),
			PostID:    postID,
			AuthorID:  "a",
			Content:   fmt.Sprintf("root %d", i),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.cmnt.Create(s.ctx, c))
	}
	comments, hasNext, err := s.cmnt.ListByPost(s.ctx, postID, nil, 2, nil)
	s.Require().NoError(err)
	s.Len(comments, 2)
	s.True(hasNext)
	s.Equal("c1", comments[0].ID)
}

func (s *MemoryRepoTestSuite) TestComment_ListByPost_Children() {
	root := &domain.Comment{ID: "r1", PostID: "p1", AuthorID: "a", Content: "root"}
	s.Require().NoError(s.cmnt.Create(s.ctx, root))
	for i := 1; i <= 3; i++ {
		child := &domain.Comment{
			ID:        fmt.Sprintf("child%d", i),
			PostID:    "p1",
			ParentID:  &root.ID,
			AuthorID:  "a",
			Content:   fmt.Sprintf("child %d", i),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.cmnt.Create(s.ctx, child))
	}
	children, hasNext, err := s.cmnt.ListByPost(s.ctx, "p1", &root.ID, 2, nil)
	s.Require().NoError(err)
	s.Len(children, 2)
	s.True(hasNext)
	s.Equal("child1", children[0].ID)
}

func (s *MemoryRepoTestSuite) TestComment_ListChildrenBatch() {
	parent1 := &domain.Comment{ID: "p1", PostID: "post1", AuthorID: "a", Content: "p1"}
	parent2 := &domain.Comment{ID: "p2", PostID: "post1", AuthorID: "a", Content: "p2"}
	s.Require().NoError(s.cmnt.Create(s.ctx, parent1))
	s.Require().NoError(s.cmnt.Create(s.ctx, parent2))

	for i := 1; i <= 3; i++ {
		child := &domain.Comment{
			ID:        fmt.Sprintf("c1-%d", i),
			PostID:    "post1",
			ParentID:  &parent1.ID,
			AuthorID:  "a",
			Content:   fmt.Sprintf("child %d", i),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.cmnt.Create(s.ctx, child))
	}
	for i := 1; i <= 2; i++ {
		child := &domain.Comment{
			ID:        fmt.Sprintf("c2-%d", i),
			PostID:    "post1",
			ParentID:  &parent2.ID,
			AuthorID:  "a",
			Content:   fmt.Sprintf("child %d", i),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.cmnt.Create(s.ctx, child))
	}

	results, hasNextMap, err := s.cmnt.ListChildrenBatch(
		s.ctx,
		[]string{"p1", "p2"},
		[]int{2, 1},
		[]bool{false, false},
		[]time.Time{{}, {}},
		[]string{"", ""},
	)
	s.Require().NoError(err)
	s.Len(results["p1"], 2)
	s.Len(results["p2"], 1)
	s.True(hasNextMap["p1"])
	s.True(hasNextMap["p2"])
}

func (s *MemoryRepoTestSuite) TestTransaction_Simple() {
	err := s.txMgr.WithinTransaction(s.ctx, func(ctx context.Context) error {
		post := &domain.Post{ID: "tx-post", Title: "tx", Content: "c", AuthorID: "a"}
		return s.post.Create(ctx, post)
	})
	s.NoError(err)

	_, err = s.post.GetByID(s.ctx, "tx-post")
	s.NoError(err)
}
