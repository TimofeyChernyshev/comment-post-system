CREATE INDEX idx_posts_created ON posts(created_at DESC, id DESC);

CREATE INDEX idx_comments_post_root ON comments(post_id, created_at ASC, id ASC) WHERE parent_id IS NULL;

CREATE INDEX idx_comments_parent ON comments(parent_id, created_at ASC, id ASC);