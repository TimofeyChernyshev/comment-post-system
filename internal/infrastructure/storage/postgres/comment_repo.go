package postgres

import (
	"context"
	"fmt"
	"time"

	"github.com/TimofeyChernyshev/comment-post-system/internal/domain"
	cursorpagination "github.com/TimofeyChernyshev/comment-post-system/pkg/cursor_pagination"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type CommentRepository struct {
	pool *pgxpool.Pool
}

func NewCommentRepository(pool *pgxpool.Pool) *CommentRepository {
	return &CommentRepository{
		pool: pool,
	}
}

func (r *CommentRepository) db(ctx context.Context) Queryable {
	return queryableFromContext(ctx, r.pool)
}

func (r *CommentRepository) Create(ctx context.Context, comment *domain.Comment) error {
	_, err := r.db(ctx).Exec(
		ctx,
		`
		INSERT INTO comments (
			id,
			post_id,
			parent_id,
			author_id,
			content,
			created_at,
			updated_at
		)
		VALUES ($1,$2,$3,$4,$5,$6,$7)
		`,
		comment.ID,
		comment.PostID,
		comment.ParentID,
		comment.AuthorID,
		comment.Content,
		comment.CreatedAt,
		comment.UpdatedAt,
	)

	if err != nil {
		return fmt.Errorf("failed to create comment: %w", err)
	}

	return nil
}

func (r *CommentRepository) GetByID(ctx context.Context, id string) (*domain.Comment, error) {
	var comment domain.Comment

	err := r.db(ctx).QueryRow(
		ctx,
		`
		SELECT
			id,
			post_id,
			parent_id,
			author_id,
			content,
			created_at,
			updated_at
		FROM comments
		WHERE id = $1
		`,
		id,
	).Scan(
		&comment.ID,
		&comment.PostID,
		&comment.ParentID,
		&comment.AuthorID,
		&comment.Content,
		&comment.CreatedAt,
		&comment.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("failed to get comment: %w", err)
	}

	return &comment, nil
}

func (r *CommentRepository) ListByPost(ctx context.Context, postID string, parentID *string, first int, after *string) ([]*domain.Comment, bool, error) {
	db := r.db(ctx)

	limit := first + 1

	var (
		rows pgx.Rows
		err  error
	)

	if after != nil {
		cursor, err := cursorpagination.Decode(*after)
		if err != nil {
			return nil, false, fmt.Errorf("failed to decode cursor for pagination: %w", err)
		}

		if parentID == nil {
			rows, err = db.Query(
				ctx,
				`
				SELECT
					id,
					post_id,
					parent_id,
					author_id,
					content,
					created_at,
					updated_at
				FROM comments
				WHERE
					post_id = $1
					AND parent_id IS NULL
					AND (created_at,id) > ($2,$3)
				ORDER BY created_at ASC, id ASC
				LIMIT $4
				`,
				postID,
				cursor.CreatedAt,
				cursor.ID,
				limit,
			)
		} else {
			rows, err = db.Query(
				ctx,
				`
				SELECT
					id,
					post_id,
					parent_id,
					author_id,
					content,
					created_at,
					updated_at
				FROM comments
				WHERE
					post_id = $1
					AND parent_id = $2
					AND (created_at,id) > ($3,$4)
				ORDER BY created_at ASC, id ASC
				LIMIT $5
				`,
				postID,
				*parentID,
				cursor.CreatedAt,
				cursor.ID,
				limit,
			)
		}
	} else {
		if parentID == nil {
			rows, err = db.Query(
				ctx,
				`
				SELECT
					id,
					post_id,
					parent_id,
					author_id,
					content,
					created_at,
					updated_at
				FROM comments
				WHERE
					post_id = $1
					AND parent_id IS NULL
				ORDER BY created_at ASC, id ASC
				LIMIT $2
				`,
				postID,
				limit,
			)
		} else {
			rows, err = db.Query(
				ctx,
				`
				SELECT
					id,
					post_id,
					parent_id,
					author_id,
					content,
					created_at,
					updated_at
				FROM comments
				WHERE
					post_id = $1
					AND parent_id = $2
				ORDER BY created_at ASC, id ASC
				LIMIT $3
				`,
				postID,
				*parentID,
				limit,
			)
		}
	}

	if err != nil {
		return nil, false, fmt.Errorf("failed to get comments: %w", err)
	}

	defer rows.Close()

	comments := make([]*domain.Comment, 0, limit)

	for rows.Next() {
		var comment domain.Comment

		err = rows.Scan(
			&comment.ID,
			&comment.PostID,
			&comment.ParentID,
			&comment.AuthorID,
			&comment.Content,
			&comment.CreatedAt,
			&comment.UpdatedAt,
		)
		if err != nil {
			return nil, false, fmt.Errorf("failed to scan rows: %w", err)
		}

		comments = append(comments, &comment)
	}

	hasNextPage := len(comments) > first

	if hasNextPage {
		comments = comments[:first]
	}

	if rows.Err() != nil {
		return nil, false, fmt.Errorf("failed to list comments: %w", rows.Err())
	}

	return comments, hasNextPage, nil
}

func (r *CommentRepository) ListChildrenBatch(
	ctx context.Context, parentIDs []string, firsts []int,
	hasAfter []bool, afterCreatedAt []time.Time, afterIDs []string,
) (map[string][]*domain.Comment, map[string]bool, error) {
	firstMap := make(map[string]int)
	for i, id := range parentIDs {
		firstMap[id] = firsts[i]
	}

	rows, err := r.db(ctx).Query(
		ctx,
		`
		WITH input AS (
			SELECT
				unnest($1::text[])        AS parent_id,
				unnest($2::int[])         AS page_size,
				unnest($3::bool[])        AS has_after,
				unnest($4::timestamptz[]) AS after_created_at,
				unnest($5::text[])        AS after_id
		),
		filtered AS (
			SELECT
				c.*,
				i.page_size,

				ROW_NUMBER() OVER (
					PARTITION BY c.parent_id
					ORDER BY c.created_at ASC, c.id ASC
				) as rn

			FROM comments c
			JOIN input i
				ON i.parent_id = c.parent_id

			WHERE
				i.has_after = false
				OR (c.created_at, c.id) > (i.after_created_at, i.after_id)
		),
		ranked AS (
			SELECT *
			FROM filtered
			WHERE rn <= page_size + 1
		)
		SELECT
			id,
			post_id,
			parent_id,
			author_id,
			content,
			created_at,
			updated_at
		FROM ranked
		ORDER BY parent_id, created_at ASC, id ASC;
		`,
		parentIDs,
		firsts,
		hasAfter,
		afterCreatedAt,
		afterIDs,
	)
	if err != nil {
		return nil, nil, fmt.Errorf("batch query failed: %w", err)
	}
	defer rows.Close()

	result := make(map[string][]*domain.Comment)
	hasNextMap := make(map[string]bool)

	for rows.Next() {
		var c domain.Comment

		if err := rows.Scan(
			&c.ID,
			&c.PostID,
			&c.ParentID,
			&c.AuthorID,
			&c.Content,
			&c.CreatedAt,
			&c.UpdatedAt,
		); err != nil {
			return nil, nil, fmt.Errorf("scan failed: %w", err)
		}

		if c.ParentID != nil {
			result[*c.ParentID] = append(result[*c.ParentID], &c)
		}
	}

	for pid, items := range result {
		limit := firstMap[pid]

		if len(items) > limit {
			hasNextMap[pid] = true
			result[pid] = items[:limit]
		} else {
			hasNextMap[pid] = false
		}
	}

	return result, hasNextMap, rows.Err()
}
