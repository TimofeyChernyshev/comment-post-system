package graph

import (
	"testing"
	"time"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestToCommentConnection_NilComments(t *testing.T) {
	result := toCommentConnection(nil, false)
	require.NotNil(t, result)
	assert.Empty(t, result.Edges)
	require.NotNil(t, result.PageInfo)
	assert.False(t, result.PageInfo.HasNextPage)
	assert.Nil(t, result.PageInfo.EndCursor)
}

func TestToCommentConnection_EmptyComments(t *testing.T) {
	comments := []*domain.Comment{}
	result := toCommentConnection(comments, false)

	require.NotNil(t, result)
	assert.Empty(t, result.Edges)
	assert.NotNil(t, result.PageInfo)
	assert.False(t, result.PageInfo.HasNextPage)
	assert.Nil(t, result.PageInfo.EndCursor)
}

func TestToCommentConnection_SingleComment(t *testing.T) {
	createdAt := time.Date(2025, 5, 31, 12, 0, 0, 0, time.UTC)
	comment := &domain.Comment{
		ID:        "comment-1",
		PostID:    "post-1",
		Content:   "Hello",
		CreatedAt: createdAt,
	}
	comments := []*domain.Comment{comment}

	result := toCommentConnection(comments, true)

	require.NotNil(t, result)
	assert.True(t, result.PageInfo.HasNextPage)

	require.Len(t, result.Edges, 1)
	edge := result.Edges[0]
	assert.Equal(t, comment, edge.Node)

	cursor, err := cursorpagination.Decode(edge.Cursor)
	require.NoError(t, err)
	assert.Equal(t, "comment-1", cursor.ID)
	assert.True(t, createdAt.Equal(cursor.CreatedAt))

	require.NotNil(t, result.PageInfo.EndCursor)
	assert.Equal(t, edge.Cursor, *result.PageInfo.EndCursor)
}

func TestToCommentConnection_MultipleComments(t *testing.T) {
	now := time.Now().UTC()
	comment1 := &domain.Comment{ID: "c1", CreatedAt: now}
	comment2 := &domain.Comment{ID: "c2", CreatedAt: now.Add(time.Minute)}
	comment3 := &domain.Comment{ID: "c3", CreatedAt: now.Add(2 * time.Minute)}
	comments := []*domain.Comment{comment1, comment2, comment3}

	result := toCommentConnection(comments, false)

	require.Len(t, result.Edges, 3)
	assert.False(t, result.PageInfo.HasNextPage)

	require.NotNil(t, result.PageInfo.EndCursor)
	lastEdge := result.Edges[len(result.Edges)-1]
	assert.Equal(t, *result.PageInfo.EndCursor, lastEdge.Cursor)

	for i, edge := range result.Edges {
		cursor, err := cursorpagination.Decode(edge.Cursor)
		require.NoError(t, err)
		assert.Equal(t, comments[i].ID, cursor.ID)
		assert.True(t, comments[i].CreatedAt.Equal(cursor.CreatedAt))
	}
}

func TestToPostConnection_NilPosts(t *testing.T) {
	result := toPostConnection(nil, false)
	require.NotNil(t, result)
	assert.Empty(t, result.Edges)
	require.NotNil(t, result.PageInfo)
	assert.False(t, result.PageInfo.HasNextPage)
	assert.Nil(t, result.PageInfo.EndCursor)
}

func TestToPostConnection_EmptyPosts(t *testing.T) {
	posts := []*domain.Post{}
	result := toPostConnection(posts, false)

	require.NotNil(t, result)
	assert.Empty(t, result.Edges)
	assert.False(t, result.PageInfo.HasNextPage)
	assert.Nil(t, result.PageInfo.EndCursor)
}

func TestToPostConnection_SinglePost(t *testing.T) {
	createdAt := time.Date(2025, 5, 31, 10, 0, 0, 0, time.UTC)
	post := &domain.Post{
		ID:        "post-1",
		Title:     "Title",
		CreatedAt: createdAt,
	}
	posts := []*domain.Post{post}

	result := toPostConnection(posts, true)

	require.Len(t, result.Edges, 1)
	assert.True(t, result.PageInfo.HasNextPage)

	edge := result.Edges[0]
	assert.Equal(t, post, edge.Node)

	cursor, err := cursorpagination.Decode(edge.Cursor)
	require.NoError(t, err)
	assert.Equal(t, "post-1", cursor.ID)
	assert.True(t, createdAt.Equal(cursor.CreatedAt))

	assert.Equal(t, edge.Cursor, *result.PageInfo.EndCursor)
}

func TestToPostConnection_MultiplePosts(t *testing.T) {
	now := time.Now().UTC()
	posts := []*domain.Post{
		{ID: "p1", CreatedAt: now},
		{ID: "p2", CreatedAt: now.Add(time.Hour)},
	}

	result := toPostConnection(posts, false)

	require.Len(t, result.Edges, 2)
	assert.False(t, result.PageInfo.HasNextPage)

	lastCursor := result.Edges[len(result.Edges)-1].Cursor
	assert.Equal(t, lastCursor, *result.PageInfo.EndCursor)

	for i, edge := range result.Edges {
		cursor, err := cursorpagination.Decode(edge.Cursor)
		require.NoError(t, err)
		assert.Equal(t, posts[i].ID, cursor.ID)
	}
}
