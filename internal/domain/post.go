package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrPostInvalidAuthor = errors.New("post author ID cannot be empty")
	ErrPostEmptyTitle    = errors.New("post title is empty")
	ErrPostEmptyContent  = errors.New("post content is empty")
)

type Post struct {
	ID               string
	Title            string
	Content          string
	AuthorID         string
	CreatedAt        time.Time
	UpdatedAt        time.Time
	CommentsDisabled bool
}

type PostCreationParams struct {
	Title    string
	Content  string
	AuthorID string
}

func NewPost(id string, params PostCreationParams) (*Post, error) {
	params.Title = strings.TrimSpace(params.Title)
	if params.Title == "" {
		return nil, ErrPostEmptyTitle
	}

	params.Content = strings.TrimSpace(params.Content)
	if params.Content == "" {
		return nil, ErrPostEmptyContent
	}

	params.AuthorID = strings.TrimSpace(params.AuthorID)
	if params.AuthorID == "" {
		return nil, ErrPostInvalidAuthor
	}

	now := time.Now().UTC()

	return &Post{
		ID:               id,
		Title:            params.Title,
		Content:          params.Content,
		AuthorID:         params.AuthorID,
		CreatedAt:        now,
		UpdatedAt:        now,
		CommentsDisabled: false,
	}, nil
}

// DisableComments - запрещает добавлять комментарии к посту
func (p *Post) DisableComments() {
	p.CommentsDisabled = true
	p.UpdatedAt = time.Now().UTC()
}
