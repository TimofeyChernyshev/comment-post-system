package graphql

import (
	"net/http"
	"time"

	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/graphql/graph"
)

func LoaderMiddleware(commentService graph.CommentService, next http.Handler, loaderBatchCapacity int, loaderWaitTime time.Duration) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		loader := graph.NewChildrenLoader(commentService, loaderBatchCapacity, loaderWaitTime)
		ctx := graph.WithLoaders(r.Context(), loader)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}
