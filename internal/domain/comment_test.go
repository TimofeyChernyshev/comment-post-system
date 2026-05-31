package domain

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewComment_ValidRootComment(t *testing.T) {
	params := CommentCreationParams{
		PostID:   "post-1",
		ParentID: nil,
		AuthorID: "user-1",
		Content:  "comment text",
	}
	id := "comment-1"

	c, err := NewComment(id, params)
	require.NoError(t, err)

	assert.Equal(t, id, c.ID)
	assert.Equal(t, params.PostID, c.PostID)
	assert.Nil(t, c.ParentID)
	assert.Equal(t, params.AuthorID, c.AuthorID)
	assert.Equal(t, params.Content, c.Content)
	assert.NotZero(t, c.CreatedAt)
	assert.NotZero(t, c.UpdatedAt)

	isRoot := c.IsRoot()
	assert.True(t, isRoot)
}

func TestNewComment_ValidReply(t *testing.T) {
	parentID := "comment-1"
	params := CommentCreationParams{
		PostID:   "post-1",
		ParentID: &parentID,
		AuthorID: "user-2",
		Content:  "reply text",
	}
	id := "comment-2"

	c, err := NewComment(id, params)
	require.NoError(t, err)

	require.NotNil(t, c.ParentID)
	assert.Equal(t, parentID, *c.ParentID)

	isRoot := c.IsRoot()
	assert.False(t, isRoot)
}

func TestNewComment_TrimsWhitespace(t *testing.T) {
	parentID := "  comment-1  "
	params := CommentCreationParams{
		PostID:   "  post-1  ",
		ParentID: &parentID,
		AuthorID: "  user-1  ",
		Content:  "  Hello world  ",
	}
	id := "comment-1"

	c, err := NewComment(id, params)
	require.NoError(t, err)

	assert.Equal(t, "Hello world", c.Content)
	assert.Equal(t, "post-1", c.PostID)
	assert.Equal(t, "user-1", c.AuthorID)
	require.NotNil(t, c.ParentID)
	assert.Equal(t, "comment-1", *c.ParentID)
}

func TestNewComment_EmptyContent(t *testing.T) {
	params := CommentCreationParams{
		PostID:   "post-1",
		AuthorID: "user-1",
		Content:  "   ",
	}
	_, err := NewComment("id", params)
	require.ErrorIs(t, err, ErrCommentEmptyContent)
}

func TestNewComment_EmptyPostID(t *testing.T) {
	params := CommentCreationParams{
		PostID:   "  ",
		AuthorID: "user-1",
		Content:  "text",
	}
	_, err := NewComment("id", params)
	require.ErrorIs(t, err, ErrPostIDEmpty)
}

func TestNewComment_EmptyAuthor(t *testing.T) {
	params := CommentCreationParams{
		PostID:   "post-1",
		AuthorID: "   ",
		Content:  "text",
	}
	_, err := NewComment("id", params)
	require.ErrorIs(t, err, ErrCommentInvalidAuthor)
}

func TestNewComment_EmptyParentID(t *testing.T) {
	parentID := "   "
	params := CommentCreationParams{
		PostID:   "post-1",
		ParentID: &parentID,
		AuthorID: "user-1",
		Content:  "text",
	}
	_, err := NewComment("id", params)
	require.ErrorIs(t, err, ErrCommentEmptyParent)
}

func TestComment_IsRoot(t *testing.T) {
	parentId := "parent"
	root := &Comment{ParentID: nil}
	reply := &Comment{ParentID: &parentId}

	assert.True(t, root.IsRoot())
	assert.False(t, reply.IsRoot())
}
