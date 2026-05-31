package graph

// This file will not be regenerated automatically.
//
// It serves as dependency injection for your app, add any dependencies you require
// here.

type Resolver struct {
	PostService         PostService
	CommentService      CommentService
	SubscriptionManager SubscriptionManager
}

func NewResolver(postService PostService, commentService CommentService, subscriptionManager SubscriptionManager) *Resolver {
	return &Resolver{
		PostService:         postService,
		CommentService:      commentService,
		SubscriptionManager: subscriptionManager,
	}
}
