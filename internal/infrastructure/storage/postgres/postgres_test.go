package postgres

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stretchr/testify/suite"
	testcontainers "github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
)

type PostgresRepoTestSuite struct {
	suite.Suite
	ctx     context.Context
	pool    *pgxpool.Pool
	post    *PostRepository
	cmnt    *CommentRepository
	txMgr   *TransactionManager
	cleanup func()
}

func (s *PostgresRepoTestSuite) SetupSuite() {
	s.ctx = context.Background()

	testcontainers.SkipIfProviderIsNotHealthy(s.T())

	req := testcontainers.ContainerRequest{
		Image:        "postgres:15",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "test",
			"POSTGRES_PASSWORD": "test",
		},
		WaitingFor: wait.ForAll(
			wait.ForListeningPort("5432/tcp"),
			wait.ForLog("database system is ready to accept connections"),
		).WithDeadline(30 * time.Second),
	}

	container, err := testcontainers.GenericContainer(s.ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
		Reuse:            false,
	})
	s.Require().NoError(err)
	s.cleanup = func() { container.Terminate(s.ctx) }

	host, _ := container.Host(s.ctx)
	port, _ := container.MappedPort(s.ctx, "5432")
	dsn := fmt.Sprintf("postgres://test:test@%s:%s/testdb?sslmode=disable", host, port.Port())

	pool, err := pgxpool.New(s.ctx, dsn)
	s.Require().NoError(err)
	s.pool = pool

	s.post = NewPostRepository(pool)
	s.cmnt = NewCommentRepository(pool)
	s.txMgr = NewTransactionManager(pool)

	s.applyMigrations()
}

func (s *PostgresRepoTestSuite) TearDownSuite() {
	s.pool.Close()
	s.cleanup()
}

func (s *PostgresRepoTestSuite) SetupTest() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_, err := s.pool.Exec(ctx, "TRUNCATE comments, posts RESTART IDENTITY CASCADE")
	if err != nil {
		s.T().Logf("cleanup failed: %v", err)
	}
}

func (s *PostgresRepoTestSuite) applyMigrations() {
	for _, m := range []string{
		`CREATE TABLE IF NOT EXISTS posts (
			id            TEXT PRIMARY KEY,
			title         TEXT NOT NULL,
			content       TEXT NOT NULL,
			author_id     TEXT NOT NULL,
			comments_disabled BOOLEAN NOT NULL DEFAULT FALSE,
			created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE TABLE IF NOT EXISTS comments (
			id         TEXT PRIMARY KEY,
			post_id    TEXT NOT NULL REFERENCES posts(id) ON DELETE CASCADE,
			parent_id  TEXT REFERENCES comments(id),
			author_id  TEXT NOT NULL,
			content    TEXT NOT NULL,
			created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
			updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
		)`,
		`CREATE INDEX IF NOT EXISTS idx_comments_post_parent ON comments(post_id, parent_id, created_at, id)`,
	} {
		_, err := s.pool.Exec(s.ctx, m)
		s.Require().NoError(err)
	}
}

func TestPostgresRepoTestSuite(t *testing.T) {
	suite.Run(t, new(PostgresRepoTestSuite))
}

func (s *PostgresRepoTestSuite) TestPost_CreateAndGetByID() {
	post := &domain.Post{
		ID:               "post-1",
		Title:            "Test Post",
		Content:          "content",
		AuthorID:         "author-1",
		CommentsDisabled: false,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	err := s.post.Create(s.ctx, post)
	s.Require().NoError(err)

	retrieved, err := s.post.GetByID(s.ctx, post.ID)
	s.Require().NoError(err)
	s.Equal(post.ID, retrieved.ID)
	s.Equal(post.Title, retrieved.Title)
	s.Equal(post.Content, retrieved.Content)
	s.Equal(post.AuthorID, retrieved.AuthorID)
	s.Equal(post.CommentsDisabled, retrieved.CommentsDisabled)
	s.WithinDuration(post.CreatedAt, retrieved.CreatedAt, time.Millisecond)
}

func (s *PostgresRepoTestSuite) TestPost_GetByID_NotFound() {
	_, err := s.post.GetByID(s.ctx, "not-exist")
	s.Require().Error(err)
	s.ErrorContains(err, "no rows in result set")
}

func (s *PostgresRepoTestSuite) TestPost_Update() {
	post := &domain.Post{
		ID:               "post-1",
		Title:            "Original",
		Content:          "original",
		AuthorID:         "author-1",
		CommentsDisabled: false,
		CreatedAt:        time.Now().UTC(),
		UpdatedAt:        time.Now().UTC(),
	}
	s.Require().NoError(s.post.Create(s.ctx, post))

	post.Title = "Updated"
	post.CommentsDisabled = true
	post.UpdatedAt = time.Now().UTC()
	err := s.post.Update(s.ctx, post)
	s.Require().NoError(err)

	updated, err := s.post.GetByID(s.ctx, post.ID)
	s.Require().NoError(err)
	s.Equal("Updated", updated.Title)
	s.True(updated.CommentsDisabled)
	s.WithinDuration(post.UpdatedAt, updated.UpdatedAt, time.Millisecond)
}

func (s *PostgresRepoTestSuite) TestPost_List_WithoutCursor() {
	now := time.Now().UTC()
	for i := 1; i <= 5; i++ {
		post := &domain.Post{
			ID:               fmt.Sprintf("post-%d", i),
			Title:            fmt.Sprintf("Post %d", i),
			Content:          "content",
			AuthorID:         "author-1",
			CreatedAt:        now.Add(time.Duration(i) * time.Minute),
			UpdatedAt:        now.Add(time.Duration(i) * time.Minute),
			CommentsDisabled: false,
		}
		s.Require().NoError(s.post.Create(s.ctx, post))
		time.Sleep(time.Millisecond)
	}

	posts, hasNext, err := s.post.List(s.ctx, 2, nil)
	s.Require().NoError(err)
	s.Len(posts, 2)
	s.True(hasNext)

	s.Equal("post-5", posts[0].ID)
	s.Equal("post-4", posts[1].ID)
}

func (s *PostgresRepoTestSuite) TestPost_List_WithCursor() {
	now := time.Now().UTC()
	for i := 1; i <= 3; i++ {
		post := &domain.Post{
			ID:               fmt.Sprintf("post-%d", i),
			Title:            fmt.Sprintf("Post %d", i),
			Content:          "content",
			AuthorID:         "author-1",
			CreatedAt:        now.Add(time.Duration(i) * time.Minute),
			UpdatedAt:        now.Add(time.Duration(i) * time.Minute),
			CommentsDisabled: false,
		}
		s.Require().NoError(s.post.Create(s.ctx, post))
		time.Sleep(time.Millisecond)
	}

	page1, hasNext, err := s.post.List(s.ctx, 1, nil)
	s.Require().NoError(err)
	s.True(hasNext)
	s.Equal("post-3", page1[0].ID)
	cursor := cursorpagination.Encode(page1[0].CreatedAt, page1[0].ID)

	page2, hasNext, err := s.post.List(s.ctx, 1, &cursor)
	s.Require().NoError(err)
	s.True(hasNext)
	s.Equal("post-2", page2[0].ID)
	cursor = cursorpagination.Encode(page2[0].CreatedAt, page2[0].ID)

	page3, hasNext, err := s.post.List(s.ctx, 1, &cursor)
	s.Require().NoError(err)
	s.False(hasNext)
	s.Equal("post-1", page3[0].ID)
}

func (s *PostgresRepoTestSuite) TestComment_CreateAndGetByID() {
	post := &domain.Post{ID: "post-1", Title: "p", Content: "c", AuthorID: "a1"}
	s.Require().NoError(s.post.Create(s.ctx, post))

	comment := &domain.Comment{
		ID:        "c1",
		PostID:    "post-1",
		ParentID:  nil,
		AuthorID:  "author-1",
		Content:   "Nice!",
		CreatedAt: time.Now().UTC(),
		UpdatedAt: time.Now().UTC(),
	}
	err := s.cmnt.Create(s.ctx, comment)
	s.Require().NoError(err)

	retrieved, err := s.cmnt.GetByID(s.ctx, comment.ID)
	s.Require().NoError(err)
	s.Equal(comment.Content, retrieved.Content)
	s.Nil(retrieved.ParentID)
}

func (s *PostgresRepoTestSuite) TestComment_GetByID_NotFound() {
	_, err := s.cmnt.GetByID(s.ctx, "not-exist")
	s.Require().Error(err)
}

func (s *PostgresRepoTestSuite) TestComment_ListByPost_Roots() {
	post := &domain.Post{ID: "post-1", Title: "p", Content: "c", AuthorID: "a1"}
	s.Require().NoError(s.post.Create(s.ctx, post))

	for i := 1; i <= 3; i++ {
		c := &domain.Comment{
			ID:        fmt.Sprintf("c%d", i),
			PostID:    "post-1",
			AuthorID:  "a",
			Content:   fmt.Sprintf("root %d", i),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.cmnt.Create(s.ctx, c))
	}

	comments, hasNext, err := s.cmnt.ListByPost(s.ctx, "post-1", nil, 2, nil)
	s.Require().NoError(err)
	s.Len(comments, 2)
	s.True(hasNext)

	s.Equal("c1", comments[0].ID)
	s.Equal("c2", comments[1].ID)
}

func (s *PostgresRepoTestSuite) TestComment_ListByPost_Children() {
	post := &domain.Post{ID: "post-1", Title: "p", Content: "c", AuthorID: "a1"}
	s.Require().NoError(s.post.Create(s.ctx, post))

	root := &domain.Comment{ID: "c-root", PostID: "post-1", AuthorID: "a", Content: "root", CreatedAt: time.Now()}
	s.Require().NoError(s.cmnt.Create(s.ctx, root))

	for i := 1; i <= 4; i++ {
		child := &domain.Comment{
			ID:        fmt.Sprintf("c-child-%d", i),
			PostID:    "post-1",
			ParentID:  &root.ID,
			AuthorID:  "a",
			Content:   fmt.Sprintf("child %d", i),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.cmnt.Create(s.ctx, child))
	}

	children, hasNext, err := s.cmnt.ListByPost(s.ctx, "post-1", &root.ID, 3, nil)
	s.Require().NoError(err)
	s.Len(children, 3)
	s.True(hasNext)
	s.Equal("c-child-1", children[0].ID)
}

func (s *PostgresRepoTestSuite) TestComment_ListByPost_Cursor() {
	post := &domain.Post{ID: "post-1", Title: "p", Content: "c", AuthorID: "a1"}
	s.Require().NoError(s.post.Create(s.ctx, post))

	for i := 1; i <= 3; i++ {
		c := &domain.Comment{
			ID:        fmt.Sprintf("c%d", i),
			PostID:    "post-1",
			AuthorID:  "a",
			Content:   fmt.Sprintf("comment %d", i),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.cmnt.Create(s.ctx, c))
	}

	page1, hasNext, err := s.cmnt.ListByPost(s.ctx, "post-1", nil, 1, nil)
	s.Require().NoError(err)
	s.True(hasNext)
	s.Equal("c1", page1[0].ID)

	cursor := cursorpagination.Encode(page1[0].CreatedAt, page1[0].ID)
	page2, hasNext, err := s.cmnt.ListByPost(s.ctx, "post-1", nil, 1, &cursor)
	s.Require().NoError(err)
	s.True(hasNext)
	s.Equal("c2", page2[0].ID)
}

func (s *PostgresRepoTestSuite) TestComment_ListChildrenBatch() {
	post := &domain.Post{ID: "post-1", Title: "p", Content: "c", AuthorID: "a1"}
	s.Require().NoError(s.post.Create(s.ctx, post))

	parent1 := &domain.Comment{ID: "p1", PostID: "post-1", AuthorID: "a", Content: "parent1", CreatedAt: time.Now()}
	parent2 := &domain.Comment{ID: "p2", PostID: "post-1", AuthorID: "a", Content: "parent2", CreatedAt: time.Now()}
	s.Require().NoError(s.cmnt.Create(s.ctx, parent1))
	s.Require().NoError(s.cmnt.Create(s.ctx, parent2))

	for i := 1; i <= 3; i++ {
		child := &domain.Comment{
			ID:        fmt.Sprintf("c1-%d", i),
			PostID:    "post-1",
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
			PostID:    "post-1",
			ParentID:  &parent2.ID,
			AuthorID:  "a",
			Content:   fmt.Sprintf("child %d", i),
			CreatedAt: time.Now().UTC().Add(time.Duration(i) * time.Minute),
		}
		s.Require().NoError(s.cmnt.Create(s.ctx, child))
	}

	parentIDs := []string{"p1", "p2"}
	firsts := []int{2, 3}
	afters := []*string{nil, nil}

	var hasAfter []bool
	var afterCreatedAt []time.Time
	var afterIDs []string
	for _, a := range afters {
		hasAfter = append(hasAfter, a != nil)
		afterCreatedAt = append(afterCreatedAt, time.Time{})
		afterIDs = append(afterIDs, "")
	}

	results, hasNextMap, err := s.cmnt.ListChildrenBatch(s.ctx, parentIDs, firsts, hasAfter, afterCreatedAt, afterIDs)
	s.Require().NoError(err)

	s.Len(results["p1"], 2)
	s.Len(results["p2"], 2)
	s.True(hasNextMap["p1"])
	s.False(hasNextMap["p2"])
}

func (s *PostgresRepoTestSuite) TestTransaction_Commit() {
	err := s.txMgr.WithinTransaction(s.ctx, func(ctx context.Context) error {
		post := &domain.Post{ID: "tx-post", Title: "tx", Content: "c", AuthorID: "a"}
		if err := s.post.Create(ctx, post); err != nil {
			return err
		}
		return nil
	})
	s.Require().NoError(err)

	_, err = s.post.GetByID(s.ctx, "tx-post")
	s.NoError(err)
}

func (s *PostgresRepoTestSuite) TestTransaction_Rollback() {
	err := s.txMgr.WithinTransaction(s.ctx, func(ctx context.Context) error {
		post := &domain.Post{ID: "tx-rollback", Title: "tx", Content: "c", AuthorID: "a"}
		if err := s.post.Create(ctx, post); err != nil {
			return err
		}
		return fmt.Errorf("some error")
	})
	s.Error(err)

	_, err = s.post.GetByID(s.ctx, "tx-rollback")
	s.Error(err)
}
