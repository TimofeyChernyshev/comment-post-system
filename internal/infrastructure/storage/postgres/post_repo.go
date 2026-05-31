package postgres

import (
	"context"
	"fmt"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type PostRepository struct {
	pool *pgxpool.Pool
}

func NewPostRepository(pool *pgxpool.Pool) *PostRepository {
	return &PostRepository{
		pool: pool,
	}
}

func (r *PostRepository) db(ctx context.Context) Queryable {
	return queryableFromContext(ctx, r.pool)
}

func (r *PostRepository) Create(ctx context.Context, post *domain.Post) error {
	_, err := r.db(ctx).Exec(
		ctx,
		`
		INSERT INTO posts (
			id,
			title,
			content,
			author_id,
			comments_disabled,
			created_at,
			updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		`,
		post.ID,
		post.Title,
		post.Content,
		post.AuthorID,
		post.CommentsDisabled,
		post.CreatedAt,
		post.UpdatedAt,
	)

	return err
}

func (r *PostRepository) GetByID(ctx context.Context, id string) (*domain.Post, error) {
	var post domain.Post

	err := r.db(ctx).
		QueryRow(
			ctx,
			`
			SELECT
				id,
				title,
				content,
				author_id,
				comments_disabled,
				created_at,
				updated_at
			FROM posts
			WHERE id = $1
			`,
			id,
		).
		Scan(
			&post.ID,
			&post.Title,
			&post.Content,
			&post.AuthorID,
			&post.CommentsDisabled,
			&post.CreatedAt,
			&post.UpdatedAt,
		)

	if err != nil {
		return nil, err
	}

	return &post, nil
}

func (r *PostRepository) Update(ctx context.Context, post *domain.Post) error {
	_, err := r.db(ctx).Exec(
		ctx,
		`
		UPDATE posts
		SET
			title = $2,
			content = $3,
			comments_disabled = $4,
			updated_at = $5
		WHERE id = $1
		`,
		post.ID,
		post.Title,
		post.Content,
		post.CommentsDisabled,
		post.UpdatedAt,
	)

	return err
}

func (r *PostRepository) List(ctx context.Context, first int, after *string) ([]*domain.Post, bool, error) {
	db := r.db(ctx)

	limit := first + 1

	var (
		rows pgx.Rows
		err  error
	)

	if after == nil {
		rows, err = db.Query(
			ctx,
			`
			SELECT
				id,
				title,
				content,
				author_id,
				comments_disabled,
				created_at,
				updated_at
			FROM posts
			ORDER BY created_at DESC, id DESC
			LIMIT $1
			`,
			limit,
		)
	} else {
		cursor, err := cursorpagination.Decode(*after)
		if err != nil {
			return nil, false, fmt.Errorf("failed to decode cursor for pagination: %w", err)
		}

		rows, err = db.Query(
			ctx,
			`
			SELECT
				id,
				title,
				content,
				author_id,
				comments_disabled,
				created_at,
				updated_at
			FROM posts
			WHERE (created_at, id) < ($1, $2)
			ORDER BY created_at DESC, id DESC
			LIMIT $3
			`,
			cursor.CreatedAt,
			cursor.ID,
			limit,
		)
	}

	if err != nil {
		return nil, false, fmt.Errorf("failed to get posts: %w", err)
	}

	defer rows.Close()

	posts := make([]*domain.Post, 0, limit)

	for rows.Next() {
		var post domain.Post

		err = rows.Scan(
			&post.ID,
			&post.Title,
			&post.Content,
			&post.AuthorID,
			&post.CommentsDisabled,
			&post.CreatedAt,
			&post.UpdatedAt,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to scan rows: %w", err)
		}

		posts = append(posts, &post)
	}

	hasNextPage := len(posts) > first
	if hasNextPage {
		posts = posts[:first]
	}

	if rows.Err() != nil {
		return nil, false, fmt.Errorf("failed to list posts: %w", rows.Err())
	}

	return posts, hasNextPage, nil
}
