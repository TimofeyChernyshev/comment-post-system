package domain

import cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"

type CommentPage struct {
	Comments []*Comment
	PageInfo *cursorpagination.PageInfo
}
