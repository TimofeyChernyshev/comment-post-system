package graph

import (
	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/graphql/graph/model"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
)

func toCommentConnection(comments []*domain.Comment, hasNextPage bool) *model.CommentConnection {
	if comments == nil {
		return nil
	}

	edges := make([]*model.CommentEdge, 0, len(comments))

	for _, comment := range comments {
		cursor := cursorpagination.Encode(comment.CreatedAt, comment.ID)

		edges = append(edges, &model.CommentEdge{
			Cursor: cursor,
			Node:   comment,
		})
	}

	var endCursor *string
	if len(comments) > 0 {
		cursor := cursorpagination.Encode(comments[len(comments)-1].CreatedAt, comments[len(comments)-1].ID)
		endCursor = &cursor
	}

	return &model.CommentConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			HasNextPage: hasNextPage,
			EndCursor:   endCursor,
		},
	}
}

func toPostConnection(posts []*domain.Post, hasNextPage bool) *model.PostConnection {
	if posts == nil {
		return nil
	}

	edges := make([]*model.PostEdge, 0, len(posts))

	for _, post := range posts {
		cursor := cursorpagination.Encode(post.CreatedAt, post.ID)

		edges = append(edges, &model.PostEdge{
			Cursor: cursor,
			Node:   post,
		})
	}

	var endCursor *string
	if len(posts) > 0 {
		cursor := cursorpagination.Encode(posts[len(posts)-1].CreatedAt, posts[len(posts)-1].ID)
		endCursor = &cursor
	}

	return &model.PostConnection{
		Edges: edges,
		PageInfo: &model.PageInfo{
			HasNextPage: hasNextPage,
			EndCursor:   endCursor,
		},
	}
}
