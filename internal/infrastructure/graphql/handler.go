package graphql

import (
	"net/http"
	"time"

	"github.com/99designs/gqlgen/graphql/handler"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/99designs/gqlgen/graphql/handler/transport"
	"github.com/TimofeyChernyshev/comment-post-system/internal/infrastructure/graphql/graph"
	"github.com/gorilla/websocket"
	"github.com/vektah/gqlparser/v2/ast"
)

func NewGraphQLHandler(
	resolver *graph.Resolver, MaxComplexity, QueryCacheZise, APQCacheSize int,
	handShakeTimeout, keepAlivePingInterval time.Duration,
) http.Handler {
	srv := handler.New(
		graph.NewExecutableSchema(
			graph.Config{
				Resolvers: resolver,
			},
		),
	)

	srv.SetQueryCache(lru.New[*ast.QueryDocument](QueryCacheZise))

	srv.Use(extension.FixedComplexityLimit(MaxComplexity))

	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New[string](APQCacheSize),
	})

	srv.AddTransport(transport.POST{})
	srv.AddTransport(transport.GET{})
	srv.AddTransport(
		&transport.Websocket{
			KeepAlivePingInterval: keepAlivePingInterval,
			Upgrader: websocket.Upgrader{
				HandshakeTimeout: handShakeTimeout,
				CheckOrigin: func(r *http.Request) bool {
					return true
				},
			},
		},
	)

	return srv
}
