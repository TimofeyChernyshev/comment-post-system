package httpserver

import (
	"context"
	"fmt"
	"net/http"
	"time"
)

type Server struct {
	httpServer *http.Server
}

func New(port string, graphqlHandler http.Handler, ReadTimeout, WriteTimeout, IdleTimeout time.Duration) *Server {
	mux := http.NewServeMux()

	mux.Handle("/graphql", graphqlHandler)

	return &Server{
		httpServer: &http.Server{
			Addr:         port,
			Handler:      mux,
			ReadTimeout:  ReadTimeout,
			WriteTimeout: WriteTimeout,
			IdleTimeout:  IdleTimeout,
		},
	}
}

func (s *Server) Start() error {
	err := s.httpServer.ListenAndServe()
	if err != nil {
		return fmt.Errorf("cannot listen and serve server: %w", err)
	}

	return nil
}

func (s *Server) Stop(ctx context.Context) error {
	err := s.httpServer.Shutdown(ctx)
	if err != nil {
		return fmt.Errorf("cannot shutdown server: %w", err)
	}

	return nil
}
