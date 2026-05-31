package domain

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPost_Valid(t *testing.T) {
	params := PostCreationParams{
		Title:    "Мой первый пост",
		Content:  "Содержание поста",
		AuthorID: "user-1",
	}
	id := "post-1"

	p, err := NewPost(id, params)
	require.NoError(t, err)

	assert.Equal(t, id, p.ID)
	assert.Equal(t, params.Title, p.Title)
	assert.Equal(t, params.Content, p.Content)
	assert.Equal(t, params.AuthorID, p.AuthorID)
	assert.False(t, p.CommentsDisabled, "new post must allow comments")
	assert.False(t, p.CreatedAt.IsZero(), "created at must be set")
	assert.False(t, p.UpdatedAt.IsZero(), "updated at must be set")
}

func TestNewPost_TrimsWhitespace(t *testing.T) {
	params := PostCreationParams{
		Title:    "  Заголовок  ",
		Content:  "  Контент  ",
		AuthorID: "  user-1  ",
	}
	p, err := NewPost("id", params)
	require.NoError(t, err)

	assert.Equal(t, "Заголовок", p.Title)
	assert.Equal(t, "Контент", p.Content)
	assert.Equal(t, "user-1", p.AuthorID)
}

func TestNewPost_EmptyTitle(t *testing.T) {
	params := PostCreationParams{
		Title:    "   ",
		Content:  "text",
		AuthorID: "user-1",
	}
	_, err := NewPost("id", params)
	require.ErrorIs(t, err, ErrPostEmptyTitle)
}

func TestNewPost_EmptyContent(t *testing.T) {
	params := PostCreationParams{
		Title:    "title",
		Content:  "  ",
		AuthorID: "user-1",
	}
	_, err := NewPost("id", params)
	require.ErrorIs(t, err, ErrPostEmptyContent)
}

func TestNewPost_EmptyAuthor(t *testing.T) {
	params := PostCreationParams{
		Title:    "title",
		Content:  "content",
		AuthorID: "  ",
	}
	_, err := NewPost("id", params)
	require.ErrorIs(t, err, ErrPostInvalidAuthor)
}

func TestPost_DisableComments(t *testing.T) {
	oldTime := time.Now().Add(-time.Hour)
	p := &Post{
		ID:               "post-1",
		UpdatedAt:        oldTime,
		CommentsDisabled: false,
	}
	time.Sleep(time.Millisecond) // гарантируем, что время изменится
	p.DisableComments()

	assert.True(t, p.CommentsDisabled)
	assert.True(t, p.UpdatedAt.After(oldTime), "UpdatedAt should be refreshed")
}
