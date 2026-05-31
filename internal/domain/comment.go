package domain

import (
	"errors"
	"strings"
	"time"
)

var (
	ErrCommentInvalidAuthor = errors.New("comment author ID cannot be empty")
	ErrCommentEmptyContent  = errors.New("comment content is empty")
	ErrPostIDEmpty          = errors.New("cannot comment post with empty ID")
	ErrCommentEmptyParent   = errors.New("cannot reply to comment with empty ID")
)

type Comment struct {
	ID        string
	PostID    string
	ParentID  *string // nil если корневой комментарий
	AuthorID  string
	Content   string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type CommentCreationParams struct {
	PostID   string
	ParentID *string // может быть nil
	AuthorID string
	Content  string
}

func NewComment(id string, params CommentCreationParams) (*Comment, error) {
	params.Content = strings.TrimSpace(params.Content)
	if params.Content == "" {
		return nil, ErrCommentEmptyContent
	}

	params.PostID = strings.TrimSpace(params.PostID)
	if params.PostID == "" {
		return nil, ErrPostIDEmpty
	}

	params.AuthorID = strings.TrimSpace(params.AuthorID)
	if params.AuthorID == "" {
		return nil, ErrCommentInvalidAuthor
	}

	now := time.Now().UTC()

	if params.ParentID != nil {
		*params.ParentID = strings.TrimSpace(*params.ParentID)
		if *params.ParentID == "" {
			return nil, ErrCommentEmptyParent
		}
	}

	return &Comment{
		ID:        id,
		PostID:    params.PostID,
		ParentID:  params.ParentID,
		AuthorID:  params.AuthorID,
		Content:   params.Content,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// IsRoot - проверяет, что комментарий это не ответ на другой комментарий
func (c *Comment) IsRoot() bool {
	return c.ParentID == nil
}
