package domain

import cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"

type PostPage struct {
	Posts    []*Post
	PageInfo *cursorpagination.PageInfo
}
